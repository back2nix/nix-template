package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"notification/pkg/config"
	"notification/pkg/logger"
	notification_pb "notification/pkg/proto/notification"
	"notification/pkg/telemetry"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- Hub Implementation ---

type NotificationServer struct {
	notification_pb.UnimplementedNotificationServiceServer

	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewNotificationServer() *NotificationServer {
	return &NotificationServer{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (s *NotificationServer) AddClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[conn] = true
	logger.Info(context.Background(), "Client connected", "total", len(s.clients))
}

func (s *NotificationServer) RemoveClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[conn]; ok {
		delete(s.clients, conn)
		if err := conn.Close(); err != nil {
			logger.Error(context.Background(), "Error closing connection", "error", err)
		}
		logger.Info(context.Background(), "Client disconnected", "total", len(s.clients))
	}
}

func (s *NotificationServer) Broadcast(ctx context.Context, payload []byte) {
	s.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(s.clients))
	for client := range s.clients {
		conns = append(conns, client)
	}
	s.mu.RUnlock()

	// Логируем попытку бродкаста, чтобы видеть, доходит ли вообще дело до сюда
	logger.Info(ctx, "📢 Broadcasting message to clients", "count", len(conns))

	messageType := websocket.TextMessage
	for _, conn := range conns {
		err := conn.WriteMessage(messageType, payload)
		if err != nil {
			logger.Error(ctx, "Error broadcasting to client", "error", err)
			s.RemoveClient(conn)
		}
	}
}

func (s *NotificationServer) Send(
	ctx context.Context,
	req *notification_pb.SendRequest,
) (*notification_pb.SendReply, error) {
	s.Broadcast(ctx, req.PayloadJson)
	return &notification_pb.SendReply{Success: true}, nil
}

// --- Kafka Implementation (Consumer) ---

type MessagePostedEvent struct {
	// DDD Fields
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	Timestamp time.Time `json:"timestamp"`

	// Actual Kafka Dump Fields (Fallback)
	Sender string    `json:"sender"`
	Text   string    `json:"text"`
	Ts     time.Time `json:"ts"`
}

type kafkaHeaderCarrier struct {
	msg *kafka.Message
}

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h.Key == key {
			val := string(h.Value)
			if len(val) > 2 && val[0] == '"' && val[len(val)-1] == '"' {
				return val[1 : len(val)-1]
			}
			return val
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

type KafkaConsumer struct {
	reader *kafka.Reader
	hub    *NotificationServer
	tracer trace.Tracer
	topic  string
}

func NewKafkaConsumer(brokers []string, topic, groupID string, hub *NotificationServer) *KafkaConsumer {
	// Включаем подробное логирование ВНУТРИ драйвера Kafka
	// Это покажет, почему он молчит (Connect Timeout, DNS error, Rebalancing...)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.FirstOffset,
		// ВАЖНО: Логгеры для диагностики
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// Логируем INFO от драйвера кафки
			logger.Info(context.Background(), fmt.Sprintf("KAFKA DRIVER: "+msg, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// Логируем ERROR от драйвера кафки (СЮДА СМОТРЕТЬ ПРИ ОШИБКАХ)
			logger.Error(context.Background(), fmt.Sprintf("KAFKA DRIVER ERROR: "+msg, args...))
		}),
	})

	return &KafkaConsumer{
		reader: r,
		hub:    hub,
		topic:  topic,
		tracer: otel.Tracer("kafka-consumer"),
	}
}

func (c *KafkaConsumer) Start(ctx context.Context) {
	logger.Info(ctx, "📥 [Kafka] Consumer loop starting...", "topic", c.topic)
	for {
		// Блокирующий вызов. Если драйвер не может соединиться, он будет висеть здесь
		// и кидать ошибки в ErrorLogger (который мы добавили выше).
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info(ctx, "📥 [Kafka] Context cancelled, stopping consumer")
				return
			}
			logger.Error(ctx, "❌ [Kafka] ReadMessage returned error", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// 1. Trace Propagation
		carrier := &kafkaHeaderCarrier{msg: &m}
		propagator := otel.GetTextMapPropagator()
		extractedCtx := propagator.Extract(ctx, carrier)

		eventName := string(m.Key)
		spanCtx, span := c.tracer.Start(extractedCtx, eventName+" process",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				semconv.MessagingSystem("kafka"),
				semconv.MessagingDestinationName(c.topic),
				semconv.MessagingKafkaMessageKey(eventName),
				semconv.MessagingOperationProcess,
				attribute.Int64("kafka.offset", m.Offset),
			),
		)

		// Логируем факт получения пакета (даже если не сможем распарсить)
		logger.Info(spanCtx, "📥 [Kafka] Packet received", "key", eventName, "offset", m.Offset)

		// 2. Logic
		if eventName == "chat.message_posted" {
			var event MessagePostedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				logger.Error(spanCtx, "Failed to unmarshal event", "error", err, "raw", string(m.Value))
				span.RecordError(err)
			} else {
				author := event.AuthorID
				if author == "" {
					author = event.Sender
				}
				content := event.Content
				if content == "" {
					content = event.Text
				}
				ts := event.Timestamp
				if ts.IsZero() {
					ts = event.Ts
				}
				id := event.MessageID
				if id == "" {
					id = "gen-" + author // fallback
				}

				wsPayload := map[string]interface{}{
					"id":     id,
					"msg":    content,
					"sender": author,
					"ts":     ts,
				}

				data, _ := json.Marshal(wsPayload)
				c.hub.Broadcast(spanCtx, data)
			}
		} else {
			logger.Info(spanCtx, "⚠️ Ignored event key", "key", eventName)
		}

		span.End()
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}

// --- Main ---

func main() {
	loader := config.NewLoader("NOTIFICATION")
	if err := loader.Load(); err != nil {
		logger.Init("notification-bootstrap", "info")
		logger.Error(context.Background(), "Failed to load config", "error", err)
		os.Exit(1)
	}

	var cfg config.AppConfig
	if err := loader.Unmarshal(&cfg); err != nil {
		logger.Init("notification-bootstrap", "info")
		logger.Error(context.Background(), "Failed to unmarshal config", "error", err)
		os.Exit(1)
	}

	serviceName := cfg.Telemetry.ServiceName
	if serviceName == "" {
		serviceName = "notification-service"
	}

	logger.Init(serviceName, "info")

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdownTracer, err := telemetry.InitTracer(
		context.Background(),
		serviceName,
		cfg.Telemetry.OtelEndpoint,
	)
	if err != nil {
		logger.Error(context.Background(), "Failed to init tracer", "error", err)
		shutdownTracer = func(context.Context) error { return nil }
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	_ = telemetry.InitProfiler(serviceName, cfg.Telemetry.PyroscopeEndpoint)

	metricsHandler, err := telemetry.InitMetrics(serviceName)
	if err != nil {
		logger.Error(context.Background(), "Failed to init metrics", "error", err)
	}

	// Инициализируем Hub
	srv := NewNotificationServer()

	// Kafka Setup
	brokers := cfg.Kafka.Brokers
	// ЛОГИРУЕМ КОНФИГ ПРИ СТАРТЕ - Проверь эти логи!
	logger.Info(context.Background(), "🔌 Kafka Config", "brokers", brokers, "topic", cfg.Kafka.Topic)

	if len(brokers) == 0 {
		logger.Error(context.Background(), "❌ Kafka Brokers list is EMPTY! check configs/staging.env or local.env")
	} else {
		kafkaConsumer := NewKafkaConsumer(
			brokers,
			cfg.Kafka.Topic,
			"notification-group",
			srv,
		)
		defer kafkaConsumer.Close()

		// Запускаем консьюмер
		go func() {
			kafkaConsumer.Start(context.Background())
		}()
	}

	// HTTP Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(context.Background(), "Upgrade error", "error", err)
			return
		}
		srv.AddClient(conn)
		// Не закрываем здесь defer, так как у нас свой цикл чтения
		// defer srv.RemoveClient(conn) вызывается при выходе из цикла
		defer srv.RemoveClient(conn)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"notification"}`))
	})

	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Server.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// gRPC Setup
	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		logger.Error(context.Background(), "Failed to listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	notification_pb.RegisterNotificationServiceServer(grpcServer, srv)

	errChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	go func() {
		logger.Info(context.Background(), "gRPC server listening", "addr", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	logger.Info(context.Background(), "✅ Notification Service Started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info(context.Background(), "Shutting down...")
	case err := <-errChan:
		logger.Error(context.Background(), "Server failed", "error", err)
	}

	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
}
