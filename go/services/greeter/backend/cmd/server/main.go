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
	grpcHandler "greeter/internal/infrastructure/grpc"
	httpHandler "greeter/internal/infrastructure/http"
	"greeter/pkg/config"
	"greeter/pkg/logger"
	"greeter/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	// Загружаем конфигурацию
	loader := config.NewLoader()

	// Устанавливаем дефолтные значения
	loader.SetDefault("GREETER_HTTP_PORT", "8081")
	loader.SetDefault("GREETER_GRPC_PORT", "50051")
	loader.SetDefault("LOG_LEVEL", "info")
	loader.SetDefault("LOG_FORMAT", "text")
	loader.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "127.0.0.1:4317")

	if err := loader.Load(); err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	v := loader.GetViper()

	// Валидация критичных параметров
	validator := config.NewValidator()
	httpPort := v.GetString("GREETER_HTTP_PORT")
	grpcPort := v.GetString("GREETER_GRPC_PORT")

	if err := validator.ValidatePort(httpPort); err != nil {
		log.Fatalf("❌ Invalid HTTP port: %v", err)
	}
	if err := validator.ValidatePort(grpcPort); err != nil {
		log.Fatalf("❌ Invalid gRPC port: %v", err)
	}

	logLevel := v.GetString("LOG_LEVEL")
	if err := validator.ValidateOneOf(logLevel, []string{"debug", "info", "warn", "error"}, "log level"); err != nil {
		log.Fatalf("❌ Invalid log level: %v", err)
	}

	// Инициализация логгера
	logger.Init("greeter-service", logLevel)
	ctx := context.Background()
	logger.Info(ctx, "🚀 Starting Greeter Service",
		"env", v.GetString("APP_ENV"),
		"http_port", httpPort,
		"grpc_port", grpcPort)

	// Инициализация трейсинга
	otelCollector := v.GetString("OTEL_EXPORTER_OTLP_ENDPOINT")
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

	// Создаём use case и серверы
	greeterUseCase := application.NewGreeterUseCase()

	grpcOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
	grpcServer := grpcHandler.NewServer(greeterUseCase, grpcOpts...)

	// Для HTTP сервера создаём минимальную конфигурацию
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			HTTPPort:  httpPort,
			GRPCPort:  grpcPort,
			StaticDir: v.GetString("SHELL_STATIC_DIR"),
		},
		Log: config.LogConfig{
			Level:  logLevel,
			Format: v.GetString("LOG_FORMAT"),
		},
	}
	httpServer := httpHandler.NewServer(cfg, greeterUseCase)

	// Запускаем серверы
	grpcAddr := "0.0.0.0:" + grpcPort
	httpAddr := "0.0.0.0:" + httpPort

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Error(ctx, "Failed to listen gRPC", "error", err)
			os.Exit(1)
		}
		logger.Info(ctx, "✅ Greeter gRPC listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error(ctx, "Failed to serve gRPC", "error", err)
		}
	}()

	go func() {
		logger.Info(ctx, "✅ Greeter HTTP listening", "addr", httpAddr)
		httpServer.SetAddr(httpAddr)
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
