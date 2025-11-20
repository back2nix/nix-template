package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Log - глобальный инстанс логгера
var Log *slog.Logger

// Init инициализирует логгер в формате JSON для удобного парсинга в Loki
func Init(serviceName string, level string) {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		// AddSource: true, // Можно включить, если нужны строки кода, но это дороже
	}

	// JSONHandler обязателен для качественного логирования в Loki
	handler := slog.NewJSONHandler(os.Stdout, opts)

	Log = slog.New(handler).With(
		slog.String("service", serviceName),
	)

	slog.SetDefault(Log)
}

// Info логирует информационное сообщение с контекстом
func Info(ctx context.Context, msg string, args ...any) {
	// В будущем здесь можно вытаскивать TraceID из ctx и добавлять в логи
	Log.InfoContext(ctx, msg, args...)
}

// Error логирует ошибку с контекстом
func Error(ctx context.Context, msg string, args ...any) {
	Log.ErrorContext(ctx, msg, args...)
}

// Debug логирует отладочное сообщение
func Debug(ctx context.Context, msg string, args ...any) {
	Log.DebugContext(ctx, msg, args...)
}
