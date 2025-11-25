package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"landing/internal/application"
	grpc_handler "landing/internal/infrastructure/grpc"
	http_handler "landing/internal/infrastructure/http"
	"landing/pkg/config"
	"landing/pkg/logger"
	pb "landing/pkg/proto/helloworld"
	"landing/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Config (Prefix: LANDING)
	loader := config.NewLoader("LANDING")
	if err := loader.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var cfg config.AppConfig
	if err := loader.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	serviceName := cfg.Telemetry.ServiceName
	if serviceName == "" {
		serviceName = "landing-service"
	}

	// ИСПРАВЛЕНИЕ: Удален аргумент OTel Endpoint.
	logger.Init(serviceName, cfg.Log.Level)
	logger.Info(context.Background(), "🚀 Logger initialized", "level", cfg.Log.Level)

	// --- FIX: Resolve Static Directory ---
	resolvedStaticDir := resolveStaticDir(cfg.Server.StaticDir)
	if resolvedStaticDir != cfg.Server.StaticDir {
		logger.Info(
			context.Background(),
			"🔄 Static dir path resolved automatically",
			"original",
			cfg.Server.StaticDir,
			"resolved",
			resolvedStaticDir,
		)
		cfg.Server.StaticDir = resolvedStaticDir
	}

	// 2. Telemetry
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	shutdownTracer, err := telemetry.InitTracer(
		context.Background(),
		serviceName,
		cfg.Telemetry.OtelEndpoint,
	)
	if err != nil {
		logger.Error(context.Background(), "Failed to init tracer", "error", err)
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	_ = telemetry.InitProfiler(serviceName, cfg.Telemetry.PyroscopeEndpoint)

	// --- NEW: Metrics ---
	metricsHandler, err := telemetry.InitMetrics(serviceName)
	if err != nil {
		logger.Error(context.Background(), "Failed to init metrics", "error", err)
	}

	logger.Info(context.Background(), "Starting service",
		"http_port", cfg.Server.HTTPPort,
		"grpc_port", cfg.Server.GRPCPort,
		"static_dir", cfg.Server.StaticDir,
	)

	// 4. Application Core
	greeter := application.NewGreeterUseCase()

	// 5. HTTP Server
	http.Handle("/metrics", metricsHandler)

	httpSrv := http_handler.NewServer(&cfg, greeter)

	errChan := make(chan error, 1)

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	// 6. gRPC Server
	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		logger.Error(context.Background(), "Failed to listen TCP", "error", err)
		return
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	grpcServerHandler := grpc_handler.NewHandler(greeter)

	pb.RegisterGreeterServer(s, grpcServerHandler)
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	// 7. Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info(context.Background(), "Shutting down servers...")
	case err := <-errChan:
		logger.Error(context.Background(), "Server startup failed", "error", err)
	}

	s.GracefulStop()
	if err := httpSrv.Shutdown(context.Background()); err != nil {
		logger.Error(context.Background(), "HTTP server shutdown error", "error", err)
	}
}

func resolveStaticDir(configPath string) string {
	if configPath == "" {
		return ""
	}
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	cwd, err := os.Getwd()
	if err != nil {
		return configPath
	}
	candidates := []string{
		filepath.Join(cwd, configPath),
		filepath.Join(cwd, "../frontend/dist"),
		filepath.Join(cwd, "../../frontend/dist"),
		filepath.Join(cwd, "services/landing/frontend/dist"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return configPath
}
