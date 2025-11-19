package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"greeter/internal/application"
	"greeter/internal/config"
	grpcHandler "greeter/internal/infrastructure/grpc"
	httpHandler "greeter/internal/infrastructure/http"
)

func main() {
	cfg := config.Load()

	greeterUseCase := application.NewGreeterUseCase()

	grpcServer := grpcHandler.NewServer(greeterUseCase)
	httpServer := httpHandler.NewServer(cfg, greeterUseCase)

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			log.Fatalf("Failed to listen gRPC: %v", err)
		}
		log.Printf("✅ Greeter gRPC listening at :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("✅ Greeter HTTP listening at :%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	httpServer.Shutdown(ctx)

	log.Println("Servers stopped")
}
