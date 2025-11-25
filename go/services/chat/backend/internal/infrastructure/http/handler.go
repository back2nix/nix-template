package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chat/internal/application"
	"chat/internal/middleware"
	"chat/pkg/config"
	"chat/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	server   *http.Server
	eventBus application.EventBus
	config   *config.AppConfig
}

type MessageDTO struct {
	Headers map[string]string `json:"headers"`
	Text    string            `json:"text"`
}

func NewServer(cfg *config.AppConfig, eventBus application.EventBus) *Server {
	mux := http.NewServeMux()

	s := &Server{
		eventBus: eventBus,
		config:   cfg,
	}

	// ВАЖНО: Регистрируем API endpoints ПЕРЕД static handler
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/health", http.HandlerFunc(s.HandleHealth))

	handlePostMessage := http.HandlerFunc(s.HandlePostMessage)
	mux.Handle("/messages", otelhttp.NewHandler(handlePostMessage, "POST /messages"))

	// Static Files Handler - регистрируем ПОСЛЕДНИМ чтобы не перехватывал API
	if cfg.Server.StaticDir != "" {
		logger.Info(context.Background(), "🗂️  Static dir absolute path", "path", cfg.Server.StaticDir)

		// Листинг файлов в директории для отладки
		entries, err := os.ReadDir(cfg.Server.StaticDir)
		if err == nil {
			logger.Info(context.Background(), "📁 Files in static dir:")
			for _, e := range entries {
				logger.Info(context.Background(), fmt.Sprintf("  - %s", e.Name()))
			}
		}

		// КРИТИЧНО: Static handler регистрируется ПОСЛЕДНИМ
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// API endpoints обрабатываются выше, сюда попадает только статика
			logger.Info(r.Context(), "🌐 Static request",
				"method", r.Method,
				"path", r.URL.Path,
			)

			path := r.URL.Path
			path = strings.TrimPrefix(path, "/api/chat")
			path = filepath.Clean(path)

			logger.Info(r.Context(), "🔍 Path after cleanup", "path", path)

			fullPath := filepath.Join(cfg.Server.StaticDir, path)

			// 1. Прямая проверка файла
			info, err := os.Stat(fullPath)
			if err == nil && !info.IsDir() {
				logger.Info(r.Context(), "✅ Serving file", "path", fullPath)
				http.ServeFile(w, r, fullPath)
				return
			}

			// 2. SPA fallback для корня
			if path == "/" || path == "/index.html" {
				indexPath := filepath.Join(cfg.Server.StaticDir, "index.html")
				logger.Info(r.Context(), "📄 Serving index.html", "path", indexPath)
				http.ServeFile(w, r, indexPath)
				return
			}

			// 3. Пробуем без начального слеша
			if strings.HasPrefix(path, "/") {
				altPath := filepath.Join(cfg.Server.StaticDir, path[1:])
				info, err = os.Stat(altPath)
				if err == nil && !info.IsDir() {
					logger.Info(r.Context(), "✅ Serving file (no leading slash)", "path", altPath)
					http.ServeFile(w, r, altPath)
					return
				}
			}

			logger.Error(r.Context(), "❌ 404 Not Found",
				"path", path,
				"fullPath", fullPath,
				"statError", err,
			)
			http.NotFound(w, r)
		}))
	}

	handler := middleware.CORS(mux)

	s.server = &http.Server{
		Addr:    "0.0.0.0:" + cfg.Server.HTTPPort,
		Handler: handler,
	}

	return s
}

func (s *Server) SetAddr(addr string) {
	s.server.Addr = addr
}

func (s *Server) HandlePostMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	tracer := otel.Tracer("chat-http-handler")
	spanCtx, span := tracer.Start(ctx, "publish_chat_message",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystem("kafka"),
			semconv.MessagingDestinationName("chat-events"),
			attribute.String("messaging.operation", "publish"),
		),
	)
	defer span.End()

	var msg MessageDTO
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logger.Info(spanCtx, "📩 Message received via HTTP", "text", msg.Text)

	eventPayload, _ := json.Marshal(map[string]interface{}{
		"text":   msg.Text,
		"sender": "http_user",
		"ts":     time.Now(),
	})

	err := s.eventBus.Publish(spanCtx, "chat.message_posted", eventPayload)
	if err != nil {
		span.RecordError(err)
		logger.Error(spanCtx, "Failed to publish to EventBus", "error", err)
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	json.NewEncoder(w).Encode(map[string]string{
		"status": "queued",
		"event":  "chat.message_posted",
	})
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
