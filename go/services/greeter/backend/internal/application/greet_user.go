package application

import (
	"context"
	"greeter/internal/domain"
)

// GreeterUseCase - application service
type GreeterUseCase struct {
	// В будущем добавишь repository, external services, etc.
}

func NewGreeterUseCase() *GreeterUseCase {
	return &GreeterUseCase{}
}

func (uc *GreeterUseCase) GreetUser(ctx context.Context, name string) (string, error) {
	greeter, err := domain.NewGreeter(name)
	if err != nil {
		return "", err
	}

	// Тут может быть логика сохранения в БД, отправка события и т.д.

	return greeter.GenerateGreeting(), nil
}
