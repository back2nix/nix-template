package application

import (
	"context"
	"encoding/json"
	"fmt"

	"chat/internal/domain"
)

// --- Ports (Interfaces) ---

// EventBus - порт для публикации доменных событий (Kafka).
type EventBus interface {
	Publish(ctx context.Context, key string, payload []byte) error
}

// --- CQRS: WRITE SIDE (Commands) ---

// PostMessageCommand - команда на отправку сообщения.
type PostMessageCommand struct {
	AuthorID string
	Content  string
}

// PostMessageHandler - обработчик команды.
type PostMessageHandler struct {
	eventBus EventBus
}

func NewPostMessageHandler(eventBus EventBus) *PostMessageHandler {
	return &PostMessageHandler{
		eventBus: eventBus,
	}
}

// Handle выполняет бизнес-логику и публикует события.
func (h *PostMessageHandler) Handle(ctx context.Context, cmd PostMessageCommand) (string, error) {
	// 1. Domain Logic: Создание агрегата
	_, events, err := domain.NewMessage(cmd.AuthorID, cmd.Content)
	if err != nil {
		return "", fmt.Errorf("domain error: %w", err)
	}

	// 2. Persistence: Сохранение событий (Event Store / Bus)
	// В данном случае мы сразу пишем в EventBus (Kafka), который выступает и стором, и брокером.
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return "", fmt.Errorf("failed to marshal event: %w", err)
		}

		// Используем EventName как ключ (Topic/Key)
		if err := h.eventBus.Publish(ctx, event.EventName(), payload); err != nil {
			return "", fmt.Errorf("failed to publish event: %w", err)
		}
	}

	return "Message processed async", nil
}
