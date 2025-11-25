package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"shell/pkg/config"
	"shell/pkg/logger"
	"shell/pkg/telemetry"
)

func main() {
	// 1. Config (Prefix: SHELL)
	loader := config.NewLoader("SHELL")
	if err := loader.Load(); err != nil {
		// Используем стандартный логгер, т.к. наш еще не инициализирован
		// Но так как logger.Error использует глобальную переменную Log, которая может быть nil,
		// лучше сначала инициализировать логгер дефолтными значениями или использовать fmt/log.
		// В данном случае logger.Log по умолчанию инициализирован (обычно), но для надежности:
		logger.Init("shell-bootstrap", "info")
		logger.Error(context.Background(), "Failed to load config", "error", err)
		os.Exit(1)
	}

	var cfg config.AppConfig
	if err := loader.Unmarshal(&cfg); err != nil {
		logger.Init("shell-bootstrap", "info")
		logger.Error(context.Background(), "Failed to unmarshal config", "error", err)
		os.Exit(1)
	}

	// 2. Logger
	serviceName := cfg.Telemetry.ServiceName
	if serviceName == "" {
		serviceName = "shell-service"
	}

	// ИСПРАВЛЕНИЕ: Удален аргумент OtelEndpoint и вызов Shutdown
	logger.Init(serviceName, "info")

	// 3. Telemetry
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

	metricsHandler, err := telemetry.InitMetrics(serviceName)
	if err != nil {
		logger.Error(context.Background(), "Failed to init metrics", "error", err)
	}

	// --- FIX: Robust Static Dir Resolution ---
	staticDir := cfg.Server.StaticDir
	if staticDir == "" {
		staticDir = "../frontend/dist"
	}
	resolvedStaticDir := resolveStaticDir(staticDir)

	logger.Info(context.Background(), "Starting service",
		"http_port", cfg.Server.HTTPPort,
		"static_dir", resolvedStaticDir,
	)

	// 4. HTTP Server (Static Files + API)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"shell"}`))
	})

	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}

	// Static Files
	fs := http.FileServer(http.Dir(resolvedStaticDir))
	mux.Handle("/", fs)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Server.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrChan := make(chan error, 1)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// 5. Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info(context.Background(), "Shutdown signal received")
	case err := <-serverErrChan:
		logger.Error(context.Background(), "HTTP server failed", "error", err)
	}

	if err := httpServer.Shutdown(context.Background()); err != nil {
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
		filepath.Join(cwd, "services/shell/frontend/dist"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return configPath
}
