package config

import (
	"os"
)

type Config struct {
	Port     string
	Services ServiceConfig
}

type ServiceConfig struct {
	Greeter ServiceEndpoint
	// User    ServiceEndpoint
	// Order   ServiceEndpoint
}

type ServiceEndpoint struct {
	URL string
}

func Load() *Config {
	return &Config{
		Port: getEnv("HTTP_PORT", "8080"),
		Services: ServiceConfig{
			Greeter: ServiceEndpoint{
				URL: getEnv("GREETER_URL", "http://localhost:8081"),
			},
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
