package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"greeter/internal/application"
	"greeter/internal/config"
	"greeter/internal/middleware"
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

	// API routes
	mux.HandleFunc("/api/hello", s.HandleGreet)
	mux.HandleFunc("/health", s.HandleHealth)

	// Static files
	if cfg.Server.StaticDir != "" {
		log.Printf("📁 Serving static files from: %s", cfg.Server.StaticDir)
		fs := http.FileServer(http.Dir(cfg.Server.StaticDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, filepath.Join(cfg.Server.StaticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}

	// Оборачиваем в CORS middleware
	handler := middleware.CORS(mux)

	s.server = &http.Server{
		Addr:    ":" + cfg.Server.HTTPPort,
		Handler: handler,
	}

	return s
}

func (s *Server) HandleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	message, err := s.useCase.GreetUser(r.Context(), name)
	if err != nil {
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
