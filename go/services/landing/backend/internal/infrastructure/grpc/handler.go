package grpc

import (
	"context"

	"landing/internal/application"
	"landing/pkg/logger"
	pb "landing/pkg/proto/helloworld"

	// Убрали импорт "google.golang.org/grpc" и "reflection", так как сервер создается в main
)

// server реализует интерфейс gRPC сервера, сгенерированный protoc
type server struct {
	pb.UnimplementedGreeterServer
	useCase *application.GreeterUseCase
}

// NewHandler создает экземпляр обработчика gRPC.
// Сам *grpc.Server создается в main.go (Composition Root).
func NewHandler(useCase *application.GreeterUseCase) pb.GreeterServer {
	return &server{
		useCase: useCase,
	}
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	// Используем slog через адаптер logger
	logger.Info(ctx, "gRPC SayHello called", "name", in.GetName())

	message, err := s.useCase.GreetUser(ctx, in.GetName())
	if err != nil {
		logger.Error(ctx, "UseCase error", "error", err)
		return nil, err
	}

	return &pb.HelloReply{Message: message}, nil
}
