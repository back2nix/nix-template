package main

import (
	"log"
	"net/http"

	"gateway/internal/config"
	"gateway/internal/middleware"
	"gateway/internal/proxy"
)

func main() {
	cfg := config.Load()

	router := http.NewServeMux()

	proxy.RegisterRoutes(router, cfg)

	handler := middleware.Logger(
		middleware.CORS(
			middleware.RateLimit(router),
		),
	)

	log.Printf("🚀 API Gateway listening at :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}
