package http

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"

	"greeter/internal/application"
	"greeter/internal/config"
	"greeter/internal/middleware"
	"greeter/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	server  *http.Server
	useCase *application.GreeterUseCase
	config  *config.Config
}

func NewServer(cfg *config.Config, useCase *application.GreeterUseCase) *Server {
	mux := http.NewServeMux()

	s := &Server{
		useCase: useCase,
		config:  cfg,
	}

	// --- OBSERVABILITY ---
	// 1. Metrics Endpoint (для VictoriaMetrics)
	mux.Handle("/metrics", promhttp.Handler())

	// 2. Tracing Middleware (Оборачиваем хендлеры)
	// Обертка добавляет Span в трейс
	handleGreet := http.HandlerFunc(s.HandleGreet)
	mux.Handle("/api/hello", otelhttp.NewHandler(handleGreet, "HTTP /api/hello"))

	handleHealth := http.HandlerFunc(s.HandleHealth)
	mux.Handle("/health", otelhttp.NewHandler(handleHealth, "HTTP /health"))

	// --- STATIC FILES ---
	if cfg.Server.StaticDir != "" {
		logger.Info(context.Background(), "📁 Serving static files", "dir", cfg.Server.StaticDir)
		fs := http.FileServer(http.Dir(cfg.Server.StaticDir))

		// Для статики трейсинг обычно не нужен, но можно добавить при желании
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, filepath.Join(cfg.Server.StaticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}

	// --- GLOBAL MIDDLEWARE ---
	// CORS (можно тоже обернуть в otelhttp.NewHandler, если нужно трейсить весь пайплайн)
	handler := middleware.CORS(mux)

	s.server = &http.Server{
		Addr:    ":" + cfg.Server.HTTPPort,
		Handler: handler,
	}

	return s
}

func (s *Server) HandleGreet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Логируем событие (trace_id добавится автоматически через middleware логгера,
	// если мы его напишем, или можно вытащить вручную. Пока просто структурный лог)
	logger.Info(ctx, "Handling Greet Request",
		"method", r.Method,
		"url", r.URL.String(),
	)

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	message, err := s.useCase.GreetUser(ctx, name)
	if err != nil {
		logger.Error(ctx, "Greeting failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
