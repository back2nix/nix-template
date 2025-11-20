package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"greeter/internal/application"
	"greeter/internal/config"
	grpcHandler "greeter/internal/infrastructure/grpc"
	httpHandler "greeter/internal/infrastructure/http"
	"greeter/pkg/logger"
	"greeter/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	// 1. Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		// Используем стандартный логгер, пока наш не инициализирован
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// 2. Инициализируем структурированный логгер (Loki friendly)
	logger.Init("greeter-service", cfg.Log.Level)
	ctx := context.Background()
	logger.Info(ctx, "🚀 Starting Greeter Service", "env", os.Getenv("APP_ENV"))

	// 3. Инициализируем Telemetry (Tracing for Tempo)
	// Адрес коллектора берем из ENV или используем дефолт (для локального docker-compose)
	otelCollector := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelCollector == "" {
		otelCollector = "127.0.0.1:4317"
	}

	shutdownTracer, err := telemetry.InitTracer(ctx, "greeter-service", otelCollector)
	if err != nil {
		logger.Error(ctx, "Failed to init tracer", "error", err)
	} else {
		defer func() {
			if err := shutdownTracer(ctx); err != nil {
				logger.Error(ctx, "Failed to shutdown tracer", "error", err)
			}
		}()
		logger.Info(ctx, "✅ Tracing initialized", "collector", otelCollector)
	}

	// 4. Инициализация бизнес-логики
	greeterUseCase := application.NewGreeterUseCase()

	// 5. Настройка gRPC сервера с интерцепторами OpenTelemetry
	// Мы передаем ServerOption в NewServer (понадобится модификация grpcHandler)
	// или создаем сервер здесь, если сигнатура позволяет.
	// Модифицируем вызов: передаем опции инструментирования.
	grpcOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
	grpcServer := grpcHandler.NewServer(greeterUseCase, grpcOpts...)

	// 6. Настройка HTTP сервера
	httpServer := httpHandler.NewServer(cfg, greeterUseCase)

	// Запуск gRPC
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
		if err != nil {
			logger.Error(ctx, "Failed to listen gRPC", "error", err)
			os.Exit(1)
		}
		logger.Info(ctx, "✅ Greeter gRPC listening", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error(ctx, "Failed to serve gRPC", "error", err)
		}
	}()

	// Запуск HTTP
	go func() {
		logger.Info(ctx, "✅ Greeter HTTP listening", "port", cfg.Server.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP shutdown error", "error", err)
	}

	logger.Info(ctx, "Servers stopped")
}
