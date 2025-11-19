package config

import "os"

type Config struct {
	HTTPPort   string
	GRPCPort   string
	StaticDir  string
}

func Load() *Config {
	return &Config{
		HTTPPort:  getEnv("HTTP_PORT", "8081"),
		GRPCPort:  getEnv("GRPC_PORT", "50051"),
		StaticDir: os.Getenv("SERVER_STATIC_DIR"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
