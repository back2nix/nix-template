package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Читаем SHELL_HTTP_PORT
	port := os.Getenv("SHELL_HTTP_PORT")
	if port == "" {
		port = "9002"
	}

	staticDir := os.Getenv("SHELL_STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	// 1. Настраиваем файловый сервер
	fs := http.FileServer(http.Dir(staticDir))

	// 2. Добавляем endpoint для метрик (Prometheus)
	http.Handle("/metrics", promhttp.Handler())

	// 3. Оборачиваем статику, чтобы она работала на корне, но не перехватывала /metrics
	// Если /metrics обрабатывается выше, то http.Handle("/") поймает всё остальное
	http.Handle("/", fs)

	log.Printf("🚀 Shell (Host) listening at :%s", port)
	log.Printf("📈 Metrics available at :%s/metrics", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
