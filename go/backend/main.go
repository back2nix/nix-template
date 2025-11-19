package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	// Путь к сгенерированному коду (замени на свой модуль)
	pb "my-go-app/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version = "dev"

// --- gRPC Server Implementation ---
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("gRPC Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello from Nix/gRPC " + in.GetName()}, nil
}

func main() {
	fmt.Printf("Starting App... Version: %s\n", Version)

	// Канал, чтобы программа не завершилась
	forever := make(chan bool)

	// 1. Запускаем gRPC сервер в горутине
	go func() {
		port := ":50051"
		lis, err := net.Listen("tcp", port)
		if err != nil {
			log.Fatalf("failed to listen gRPC: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterGreeterServer(s, &server{})
		reflection.Register(s)
		log.Printf("✅ gRPC server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 2. Запускаем HTTP сервер (Vue frontend + API gateway если надо)
	go func() {
		staticDir := os.Getenv("SERVER_STATIC_DIR")
		if staticDir == "" { staticDir = "./static" }

		if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
			http.Handle("/", http.FileServer(http.Dir(staticDir)))
			log.Printf("✅ Serving Vue form: %s", staticDir)
		} else {
			log.Println("⚠️  No static files found. API only.")
		}

		log.Println("✅ HTTP listening at :8080")
		http.ListenAndServe(":8080", nil)
	}()

	// Ждем вечно
	<-forever
}
