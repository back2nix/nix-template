package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitTracer инициализирует глобальный TracerProvider с экспортом в OTLP (OTel Collector)
func InitTracer(ctx context.Context, serviceName string, collectorAddr string) (func(context.Context) error, error) {
	// 1. Создаем ресурс (описание сервиса)
	// Убрали TelemetrySdkLanguageGo, так как он вызывает ошибки компиляции в разных версиях semconv
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.1.0"), // Версию лучше прокидывать из build-args
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 2. Настраиваем соединение с коллектором
	// Используем таймаут для соединения, чтобы не висеть вечно, если коллектор недоступен
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, collectorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector at %s: %w", collectorAddr, err)
	}

	// 3. Настраиваем экспортер
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 4. Настраиваем провайдер трейсов
	// BatchSpanProcessor эффективнее для производительности, чем SimpleSpanProcessor
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 5. Устанавливаем глобальные провайдеры
	otel.SetTracerProvider(tracerProvider)

	// W3C Trace Context (стандарт) + Baggage (для передачи кастомных данных)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Возвращаем функцию для шатдауна
	return func(ctx context.Context) error {
		// Сначала останавливаем провайдер (сбрасываем батчи)
		if err := tracerProvider.Shutdown(ctx); err != nil {
			return err
		}
		// Затем закрываем соединение
		return conn.Close()
	}, nil
}
