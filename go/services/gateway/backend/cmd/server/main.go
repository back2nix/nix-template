package main

import (
	"log"
	"net/http"
	"time"

	"gateway/internal/config"
	"gateway/internal/middleware"
	"gateway/internal/proxy"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	log.Printf("📋 Starting Gateway in %s mode", cfg.Log.Level)

	router := http.NewServeMux()

	proxy.RegisterRoutes(router, cfg)

	handler := middleware.Logger(
		middleware.CORS(
			middleware.RateLimit(router),
		),
	)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	log.Printf("🚀 API Gateway listening at :%s", cfg.Server.Port)
	log.Fatal(server.ListenAndServe())
}
