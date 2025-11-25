package application

import (
	"context"
	"landing/internal/domain"
)

// GreeterUseCase - application service
type GreeterUseCase struct {
	// В будущем добавишь repository, external services, etc.
}

func NewGreeterUseCase() *GreeterUseCase {
	return &GreeterUseCase{}
}

func (uc *GreeterUseCase) GreetUser(ctx context.Context, name string) (string, error) {
	landing, err := domain.NewGreeter(name)
	if err != nil {
		return "", err
	}

	// Тут может быть логика сохранения в БД, отправка события и т.д.

	return landing.GenerateGreeting(), nil
}
