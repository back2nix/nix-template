package grpc

import (
	"context"
	"log"

	"greeter/internal/application"
	pb "greeter/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGreeterServer
	useCase *application.GreeterUseCase
}

func NewServer(useCase *application.GreeterUseCase) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{useCase: useCase})
	reflection.Register(s)
	return s
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("[gRPC] Received: %v", in.GetName())

	message, err := s.useCase.GreetUser(ctx, in.GetName())
	if err != nil {
		return nil, err
	}

	return &pb.HelloReply{Message: message}, nil
}
