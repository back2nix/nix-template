package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	// ЗАМЕНИ my-go-app НА АКТУАЛЬНЫЙ ПУТЬ ИЗ GO.MOD, ЕСЛИ ОН ДРУГОЙ
	pb "my-go-app/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// --- CORS Middleware (Критично для загрузки JS с другого порта) ---
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // В проде лучше конкретный домен
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- gRPC Logic ---
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("[Greeter] gRPC request: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName() + " (from gRPC Service)"}, nil
}

func apiHelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Data from Greeter REST API",
	})
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" { port = "8081" } // Greeter по умолчанию на 8081

	// 1. gRPC Server (Port 50051)
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil { log.Fatalf("failed to listen gRPC: %v", err) }
		s := grpc.NewServer()
		pb.RegisterGreeterServer(s, &server{})
		reflection.Register(s)
		log.Printf("✅ Greeter gRPC listening at :50051")
		s.Serve(lis)
	}()

	// 2. HTTP Server (Vue + Assets + API)
	staticDir := os.Getenv("SERVER_STATIC_DIR")
	if staticDir == "" { staticDir = "./static" } // Для локального запуска

	mux := http.NewServeMux()
	mux.HandleFunc("/api/hello", apiHelloHandler)

	// Раздача статики
	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		absPath, _ := filepath.Abs(staticDir)
		log.Printf("✅ Serving Greeter Assets from: %s", absPath)
		// Важно! Module Federation ищет файлы, поэтому просто раздаем всё
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	log.Printf("✅ Greeter HTTP listening at :%s", port)
	http.ListenAndServe(":"+port, corsMiddleware(mux))
}
