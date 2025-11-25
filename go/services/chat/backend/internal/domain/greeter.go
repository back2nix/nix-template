package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// --- DDD Aggregate Root: Message ---
// Файл исторически называется greeter.go, но мы рефакторим его в Message (Chat Domain).
// В реальном проекте файл стоит переименовать в message.go.

// Message - агрегат, представляющий сообщение в чате.
// Использует Event Sourcing подход: состояние восстанавливается/изменяется через события.
type Message struct {
	id        string
	content   string
	authorID  string
	timestamp time.Time
	// Версия агрегата для оптимистической блокировки (в будущем)
	version int
}

// DomainEvent - контракт для всех событий домена.
type DomainEvent interface {
	EventName() string
}

// MessagePostedEvent - событие: сообщение было опубликовано.
// Single Source of Truth: это событие является фактом того, что случилось.
type MessagePostedEvent struct {
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MessagePostedEvent) EventName() string {
	return "chat.message_posted"
}

// NewMessage - Factory method (Command handler logic usage).
// Создает агрегат и возвращает несохраненные события.
func NewMessage(authorID, content string) (*Message, []DomainEvent, error) {
	if content == "" {
		return nil, nil, fmt.Errorf("message content cannot be empty")
	}
	if authorID == "" {
		return nil, nil, fmt.Errorf("authorID cannot be empty")
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	event := MessagePostedEvent{
		MessageID: id,
		Content:   content,
		AuthorID:  authorID,
		Timestamp: now,
	}

	msg := &Message{}
	// Apply event to state
	msg.Apply(event)

	return msg, []DomainEvent{event}, nil
}

// Apply - меняет состояние агрегата на основе события.
// Этот метод используется как при создании, так и при восстановлении (Rehydration) из Event Store.
func (m *Message) Apply(event DomainEvent) {
	switch e := event.(type) {
	case MessagePostedEvent:
		m.id = e.MessageID
		m.content = e.Content
		m.authorID = e.AuthorID
		m.timestamp = e.Timestamp
		m.version++
	}
}

// Getters для Read Model (если нужно, но в CQRS мы обычно используем проекции)

func (m *Message) ID() string {
	return m.id
}

func (m *Message) Content() string {
	return m.content
}
