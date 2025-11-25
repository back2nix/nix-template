package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

var Log *slog.Logger

// Init инициализирует логгер.
// Приложение пишет ТОЛЬКО в stdout (JSON).
// Сбор логов делает OTel Collector (docker-compose) или Fluent Bit (k8s).
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
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)

	// КРИТИЧНО: используем "service.name" (стандарт OpenTelemetry)
	Log = slog.New(jsonHandler).With(
		slog.String("service.name", serviceName),
	)

	slog.SetDefault(Log)
}

// withTrace добавляет TraceID и SpanID из контекста в поля лога
// Это позволяет связывать логи с трейсами в Grafana (Trace to Logs)
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

func Info(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).InfoContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).ErrorContext(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).DebugContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	withTrace(ctx).WarnContext(ctx, msg, args...)
}
