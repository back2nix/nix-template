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

	"chat/internal/application"
	grpc_implementation "chat/internal/infrastructure/grpc"
	http_implementation "chat/internal/infrastructure/http"
	"chat/internal/infrastructure/queue"
	"chat/pkg/config"
	"chat/pkg/logger"
	"chat/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
)

func main() {
	// 1. Загрузка конфигурации
	loader := config.NewLoader("CHAT")
	if err := loader.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var cfg config.AppConfig
	if err := loader.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// 2. Логгер
	logger.Init("chat-service", cfg.Log.Level)

	// 3. Статика - ИСПРАВЛЕНИЕ: сначала резолвим, потом преобразуем в абсолютный путь
	resolvedStaticDir := resolveStaticDir(cfg.Server.StaticDir)

	// КРИТИЧНО: Преобразуем в абсолютный путь
	absStaticDir, err := filepath.Abs(resolvedStaticDir)
	if err != nil {
		logger.Error(context.Background(), "Failed to get absolute path", "path", resolvedStaticDir, "error", err)
		absStaticDir = resolvedStaticDir
	}

	cfg.Server.StaticDir = absStaticDir
	logger.Info(context.Background(), "📂 Serving static files", "dir", cfg.Server.StaticDir)

	// Проверка remoteEntry.js
	remoteEntryPath := filepath.Join(cfg.Server.StaticDir, "remoteEntry.js")
	if _, err := os.Stat(remoteEntryPath); err != nil {
		logger.Error(context.Background(), "❌ remoteEntry.js NOT FOUND", "path", remoteEntryPath, "error", err)
	} else {
		logger.Info(context.Background(), "✅ remoteEntry.js found", "path", remoteEntryPath)
	}

	// 4. Трейсинг
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdownTracer, err := telemetry.InitTracer(context.Background(), "chat-service", cfg.Telemetry.OtelEndpoint)
	if err != nil {
		logger.Error(context.Background(), "⚠️ Failed to init tracer", "error", err)
		shutdownTracer = func(context.Context) error { return nil }
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	metricsHandler, err := telemetry.InitMetrics("chat-service")
	if err != nil {
		logger.Error(context.Background(), "Failed to init metrics", "error", err)
	}

	// 5. Infrastructure: Kafka Producer
	brokers := cfg.Kafka.Brokers
	if len(brokers) == 0 {
		logger.Error(context.Background(), "❌ CHAT_KAFKA_BROKERS is required but not set")
		os.Exit(1)
	}
	logger.Info(context.Background(), "📡 Kafka Brokers", "brokers", brokers)

	kafkaProducer := queue.NewKafkaProducer(brokers, cfg.Kafka.Topic)
	defer kafkaProducer.Close()

	// 6. Application Layer
	postMessageHandler := application.NewPostMessageHandler(kafkaProducer)

	// 7. Presentation Layer: HTTP Server
	httpServer := http_implementation.NewServer(&cfg, kafkaProducer)

	http.Handle("/metrics", metricsHandler)

	errChan := make(chan error, 1)
	go func() {
		logger.Info(context.Background(), "🚀 HTTP Server listening", "port", cfg.Server.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 8. Presentation Layer: gRPC Server
	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		logger.Error(context.Background(), "Failed to listen gRPC", "error", err)
		return
	}

	grpcServer := grpc_implementation.NewServer(postMessageHandler, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "🛑 Shutting down server...")
	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
}

func resolveStaticDir(configPath string) string {
	if configPath == "" {
		return ""
	}

	// Если путь уже существует - используем его
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
		filepath.Join(cwd, "services/chat/frontend/dist"),
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	return configPath
}
