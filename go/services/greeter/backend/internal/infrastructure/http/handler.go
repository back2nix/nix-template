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

	// Observability
	mux.Handle("/metrics", promhttp.Handler())

	handleGreet := http.HandlerFunc(s.HandleGreet)
	mux.Handle("/api/hello", otelhttp.NewHandler(handleGreet, "HTTP /api/hello"))

	handleHealth := http.HandlerFunc(s.HandleHealth)
	mux.Handle("/health", otelhttp.NewHandler(handleHealth, "HTTP /health"))

	if cfg.Server.StaticDir != "" {
		fs := http.FileServer(http.Dir(cfg.Server.StaticDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, filepath.Join(cfg.Server.StaticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}

	handler := middleware.CORS(mux)

	// Слушаем на 0.0.0.0, чтобы было видно из Docker
	addr := "0.0.0.0:" + cfg.Server.HTTPPort

	s.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return s
}

// SetAddr позволяет изменить адрес прослушивания (хелпер для main)
func (s *Server) SetAddr(addr string) {
	s.server.Addr = addr
}

func (s *Server) HandleGreet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Логируем с контекстом, чтобы TraceID попал в логи (если логгер поддерживает)
	logger.Info(ctx, "Handling Greet Request", "method", r.Method, "url", r.URL.String())

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
