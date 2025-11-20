package config

import (
	"fmt"
	"greeter/pkg/config"
)

type Config struct {
	Server ServerConfig
	Log    LogConfig
}

type ServerConfig struct {
	HTTPPort  string
	GRPCPort  string
	StaticDir string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	// Загружаем env файл
	loader := config.NewLoader()
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			HTTPPort:  config.GetEnv("GREETER_HTTP_PORT", "8081"),
			GRPCPort:  config.GetEnv("GREETER_GRPC_PORT", "50051"),
			StaticDir: config.GetEnv("SHELL_STATIC_DIR", ""),
		},
		Log: LogConfig{
			Level:  config.GetEnv("LOG_LEVEL", "info"),
			Format: config.GetEnv("LOG_FORMAT", "text"),
		},
	}

	// Валидация
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	validator := config.NewValidator()

	if err := validator.ValidatePort(c.Server.HTTPPort); err != nil {
		return fmt.Errorf("HTTP port: %w", err)
	}

	if err := validator.ValidatePort(c.Server.GRPCPort); err != nil {
		return fmt.Errorf("gRPC port: %w", err)
	}

	allowedLogLevels := []string{"debug", "info", "warn", "error"}
	if err := validator.ValidateOneOf(c.Log.Level, allowedLogLevels, "log level"); err != nil {
		return err
	}

	allowedLogFormats := []string{"text", "json"}
	if err := validator.ValidateOneOf(c.Log.Format, allowedLogFormats, "log format"); err != nil {
		return err
	}

	return nil
}
