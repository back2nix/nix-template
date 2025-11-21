package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shell/pkg/logger"
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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// 2. Инициализируем структурированный логгер (Loki friendly)
	logger.Init("shell-service", logLevel)
	ctx := context.Background()
	logger.Info(ctx, "🚀 Starting Shell Service",
		"env", os.Getenv("APP_ENV"),
		"port", port,
		"static_dir", staticDir,
	)

	// 3. Адрес коллектора Tempo (OTLP gRPC)
	otelCollector := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelCollector == "" {
		otelCollector = "127.0.0.1:4317"
	}

	// 4. Инициализация Observability (Tracing)
	shutdownTracer, err := telemetry.InitTracer(ctx, "shell-service", otelCollector)
	if err != nil {
		logger.Error(ctx, "Failed to init tracer", "error", err)
	} else {
		logger.Info(ctx, "✅ Tracing initialized", "collector", otelCollector)
		defer func() {
			if err := shutdownTracer(ctx); err != nil {
				logger.Error(ctx, "Failed to shutdown tracer", "error", err)
			}
		}()
	}

	// 5. Роутер
	mux := http.NewServeMux()

	// Метрики (для VictoriaMetrics)
	mux.Handle("/metrics", promhttp.Handler())

	// Health check с логированием
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.Context(), "Health check request",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"shell"}`))
	})
	mux.Handle("/health", otelhttp.NewHandler(healthHandler, "HTTP /health"))

	// Файловый сервер с трейсингом и логированием
	fs := http.FileServer(http.Dir(staticDir))
	otelHandler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Логируем каждый запрос (для статики можно использовать debug level)
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			logger.Info(r.Context(), "Serving static content",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
		} else {
			logger.Debug(r.Context(), "Serving static asset",
				"method", r.Method,
				"path", r.URL.Path,
			)
		}
		fs.ServeHTTP(w, r)
	}), "HTTP Static Content")

	mux.Handle("/", otelHandler)

	// 6. HTTP Server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 7. Запуск сервера в горутине
	go func() {
		logger.Info(ctx, "✅ Shell HTTP listening", "port", port)
		logger.Info(ctx, "📈 Metrics available", "endpoint", "http://localhost:"+port+"/metrics")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP shutdown error", "error", err)
	}

	logger.Info(ctx, "Server stopped")
}
