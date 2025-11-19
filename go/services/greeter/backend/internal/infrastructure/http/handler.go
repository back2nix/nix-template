package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"greeter/internal/application"
	"greeter/internal/config"
)

type Server struct {
	server  *http.Server
	useCase *application.GreeterUseCase
}

func NewServer(cfg *config.Config, useCase *application.GreeterUseCase) *Server {
	mux := http.NewServeMux()

	s := &Server{
		useCase: useCase,
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: corsMiddleware(mux),
		},
	}

	mux.HandleFunc("/api/hello", s.handleHello)
	mux.HandleFunc("/health", s.handleHealth)

	// Static files
	if cfg.StaticDir != "" {
		if _, err := os.Stat(cfg.StaticDir); !os.IsNotExist(err) {
			absPath, _ := filepath.Abs(cfg.StaticDir)
			fs := http.FileServer(http.Dir(absPath))
			mux.Handle("/", fs)
		}
	}

	return s
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Guest"
	}

	message, err := s.useCase.GreetUser(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}
