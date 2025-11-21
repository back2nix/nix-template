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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	logger.Init("greeter-service", cfg.Log.Level)
	ctx := context.Background()
	logger.Info(ctx, "🚀 Starting Greeter Service", "env", os.Getenv("APP_ENV"))

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

	greeterUseCase := application.NewGreeterUseCase()

	grpcOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
	grpcServer := grpcHandler.NewServer(greeterUseCase, grpcOpts...)
	httpServer := httpHandler.NewServer(cfg, greeterUseCase)

	// --- FIX: Listen on 0.0.0.0 explicitly ---
	// Это позволит Docker-контейнеру (Collector) видеть метрики сервиса
	grpcAddr := "0.0.0.0:" + cfg.Server.GRPCPort
	httpAddr := "0.0.0.0:" + cfg.Server.HTTPPort

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
		// Принудительно перезаписываем Addr в сервере
		httpServer.SetAddr(httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

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
