package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"shell/pkg/telemetry"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// 1. Конфигурация
	port := os.Getenv("SHELL_HTTP_PORT")
	if port == "" {
		port = "9002"
	}

	staticDir := os.Getenv("SHELL_STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	// Адрес коллектора Tempo (OTLP gRPC)
	// В docker-compose это host.docker.internal:4317 или tempo:4317 (если в одной сети)
	otelCollector := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelCollector == "" {
		// Fallback для локального запуска вне контейнера
		otelCollector = "127.0.0.1:4317"
	}

	// 2. Инициализация Observability (Tracing)
	ctx := context.Background()
	shutdownTracer, err := telemetry.InitTracer(ctx, "shell-service", otelCollector)
	if err != nil {
		log.Printf("⚠️ Failed to init tracer: %v", err)
	} else {
		log.Printf("✅ Tracing initialized (sending to %s)", otelCollector)
		defer func() {
			_ = shutdownTracer(ctx)
		}()
	}

	// 3. Роутер
	mux := http.NewServeMux()

	// Метрики (обычно не трейсим)
	mux.Handle("/metrics", promhttp.Handler())

	// Файловый сервер
	fs := http.FileServer(http.Dir(staticDir))

	// Оборачиваем раздачу статики в OpenTelemetry Middleware.
	// otelhttp автоматически извлечет контекст трейса из заголовков Envoy.
	otelHandler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Логируем для отладки (в реальном проде лучше использовать структурный логгер)
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		fs.ServeHTTP(w, r)
	}), "HTTP Static Content")

	// Регистрируем на корневой путь
	mux.Handle("/", otelHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🚀 Shell (Host) listening at :%s", port)
	log.Printf("📈 Metrics available at :%s/metrics", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
