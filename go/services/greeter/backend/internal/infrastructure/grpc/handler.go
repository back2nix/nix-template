package grpc

import (
	"context"

	"greeter/internal/application"
	"greeter/pkg/logger"
	pb "greeter/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGreeterServer
	useCase *application.GreeterUseCase
}

// NewServer теперь принимает опции gRPC для внедрения интерцепторов (Observability)
func NewServer(useCase *application.GreeterUseCase, opts ...grpc.ServerOption) *grpc.Server {
	s := grpc.NewServer(opts...)
	pb.RegisterGreeterServer(s, &server{useCase: useCase})
	reflection.Register(s)
	return s
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	// Используем slog вместо стандартного log
	logger.Info(ctx, "gRPC SayHello called", "name", in.GetName())

	message, err := s.useCase.GreetUser(ctx, in.GetName())
	if err != nil {
		logger.Error(ctx, "UseCase error", "error", err)
		return nil, err
	}

	return &pb.HelloReply{Message: message}, nil
}
