package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Log - глобальный инстанс логгера
var Log *slog.Logger

// Init инициализирует логгер в формате JSON для удобного парсинга в OTel/VictoriaLogs
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
		// AddSource: true, // Можно включить для отладки
	}

	// ОБЯЗАТЕЛЬНО JSONHandler для структурированных логов
	handler := slog.NewJSONHandler(os.Stdout, opts)

	Log = slog.New(handler).With(
		slog.String("service", serviceName),
	)

	slog.SetDefault(Log)
}

// withTrace добавляет TraceID и SpanID из контекста в поля лога
func withTrace(ctx context.Context) *slog.Logger {
	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if spanContext.IsValid() {
		return Log.With(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	return Log
}

// Info логирует информационное сообщение с контекстом
func Info(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).InfoContext(ctx, msg, args...)
}

// Error логирует ошибку с контекстом
func Error(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).ErrorContext(ctx, msg, args...)
}

// Debug логирует отладочное сообщение
func Debug(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).DebugContext(ctx, msg, args...)
}

// Warn логирует предупреждение
func Warn(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).WarnContext(ctx, msg, args...)
}
