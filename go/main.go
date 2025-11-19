package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"

	// Импортируем сгенерированный код
	// В реальном проекте путь будет github.com/user/repo/proto/helloworld
	pb "my-go-app/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version = "dev"

// server реализует интерфейс GreeterServer
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello реализует метод, описанный в proto файле
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello from Nix " + in.GetName()}, nil
}

func main() {
	fmt.Printf("Starting App... Version: %s, Go: %s\n", Version, runtime.Version())

	// Настраиваем слушатель
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Создаем gRPC сервер
	s := grpc.NewServer()

	// Регистрируем наш сервис
	pb.RegisterGreeterServer(s, &server{})

	// Включаем Reflection API (чтобы можно было дебажить через grpcurl или Postman)
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
