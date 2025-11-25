package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"landing/internal/application"
	"landing/internal/middleware"
	"landing/pkg/config"
	"landing/pkg/logger"
)

type Server struct {
	server  *http.Server
	useCase *application.GreeterUseCase
	config  *config.AppConfig
}

func NewServer(cfg *config.AppConfig, useCase *application.GreeterUseCase) *Server {
	mux := http.NewServeMux()
	s := &Server{
		useCase: useCase,
		config:  cfg,
	}

	mux.Handle("/metrics", promhttp.Handler())

	handleGreet := http.HandlerFunc(s.HandleGreet)
	// Используем otelhttp для замеров задержек HTTP уровня
	mux.Handle("/hello", otelhttp.NewHandler(handleGreet, "HTTP /hello"))

	handleHealth := http.HandlerFunc(s.HandleHealth)
	mux.Handle("/health", handleHealth)

	// Статика
	if cfg.Server.StaticDir != "" {
		fs := http.FileServer(http.Dir(cfg.Server.StaticDir))

		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(cfg.Server.StaticDir, r.URL.Path)

			// 1. Проверяем существование файла
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}

			// 2. Корень -> index.html
			if r.URL.Path == "/" {
				http.ServeFile(w, r, filepath.Join(cfg.Server.StaticDir, "index.html"))
				return
			}

			// 3. Fallback или 404
			http.NotFound(w, r)
		}))
	}

	handler := middleware.CORS(mux)

	s.server = &http.Server{
		Addr:              "0.0.0.0:" + cfg.Server.HTTPPort,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second, // G112: Protection against Slowloris
	}

	return s
}

func (s *Server) SetAddr(addr string) {
	s.server.Addr = addr
}

func (s *Server) HandleGreet(w http.ResponseWriter, r *http.Request) {
	// otelhttp уже извлек контекст из заголовков Envoy
	ctx := r.Context()

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Guest"
	}
	message, err := s.useCase.GreetUser(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message, "source": "Landing Service"}); err != nil {
		// Используем структурный логгер вместо log.Printf
		logger.Error(ctx, "Failed to encode response", "error", err)
	}
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
		// Используем структурный логгер
		// В HealthCheck контекст часто короткоживущий, но все же передаем
		logger.Error(r.Context(), "Failed to encode health response", "error", err)
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
