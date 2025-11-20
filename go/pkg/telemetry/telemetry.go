package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer инициализирует глобальный TracerProvider с экспортом в OTLP (Tempo)
func InitTracer(ctx context.Context, serviceName string, collectorAddr string) (func(context.Context) error, error) {
	// 1. Создаем ресурс (описание сервиса)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 2. Настраиваем экспортер (по умолчанию gRPC на порт 4317)
	// Используем Insecure, так как внутри кластера/локально
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorAddr),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 3. Настраиваем провайдер трейсов (BatchProcessor эффективнее для продакшена)
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // В продакшене лучше использовать TraceIDRatioBased
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 4. Устанавливаем глобальный провайдер
	otel.SetTracerProvider(tracerProvider)

	// 5. Настраиваем пропогацию контекста (W3C Trace Context для связки Envoy -> Go -> Go)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Возвращаем функцию для корректного завершения работы
	return tracerProvider.Shutdown, nil
}
