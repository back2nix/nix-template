package main

import (
	"context"
	"encoding/json" // <--- Добавлен импорт для JSON
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

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

// --- HTTP API Handler ---
// Простой обработчик для REST запросов от Vue
func apiHelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Hello from Go Backend! 🚀",
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Printf("Starting App... Version: %s\n", Version)

	forever := make(chan bool)

	// 1. gRPC Server
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

	// 2. HTTP Server (Vue + API)
	go func() {
		staticDir := os.Getenv("SERVER_STATIC_DIR")
		if staticDir == "" {
			staticDir = "./static"
		}

		absPath, _ := filepath.Abs(staticDir)

		// -- 1. Регистрируем API хендлеры (до FileServer!) --
		http.HandleFunc("/api/hello", apiHelloHandler)

		// -- 2. Раздаем статику Vue --
		if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
			// FileServer обрабатывает всё остальное
			http.Handle("/", http.FileServer(http.Dir(staticDir)))
			log.Printf("✅ Serving Vue from: %s (Abs: %s)", staticDir, absPath)
		} else {
			log.Printf("⚠️  No static files found at: %s (Abs: %s). API only.", staticDir, absPath)
		}

		log.Println("✅ HTTP listening at :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	<-forever
}
