package queue

import (
	"context"
	"time"

	"chat/pkg/logger"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// --- Trace Propagation Helpers ---

type kafkaHeaderCarrier struct {
	msg *kafka.Message
}

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeaderCarrier) Set(key string, value string) {
	c.msg.Headers = append(c.msg.Headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

func (c *kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		keys = append(keys, h.Key)
	}
	return keys
}

// --- Producer ---

type KafkaProducer struct {
	writer *kafka.Writer
	tracer trace.Tracer
	topic  string
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
		BatchTimeout:           10 * time.Millisecond,
		WriteTimeout:           10 * time.Second,
		RequiredAcks:           kafka.RequireOne,
	}
	return &KafkaProducer{
		writer: w,
		topic:  topic,
		tracer: otel.Tracer("kafka-producer"),
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, key string, payload []byte) error {
	ctx, span := p.tracer.Start(ctx, key+" send",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystem("kafka"),
			semconv.MessagingDestinationName(p.topic),
			semconv.MessagingKafkaMessageKey(key),
		),
	)
	defer span.End()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now(),
	}

	// Внедряем traceparent и другие заголовки в сообщение Kafka
	carrier := &kafkaHeaderCarrier{msg: &msg}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		span.RecordError(err)
		logger.Error(ctx, "❌ [Kafka] Failed to publish", "error", err)
		return err
	}
	logger.Info(ctx, "📤 [Kafka] Event Published", "event_type", key, "size", len(payload))
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
