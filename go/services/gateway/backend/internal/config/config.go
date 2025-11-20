package config

import (
	"fmt"
	"gateway/pkg/config"
)

type Config struct {
	Server   ServerConfig
	Services ServiceConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
}

type ServiceConfig struct {
	Greeter ServiceEndpoint
}

type ServiceEndpoint struct {
	URL string
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
			Port:         config.GetEnv("GATEWAY_HTTP_PORT", "8080"),
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Services: ServiceConfig{
			Greeter: ServiceEndpoint{
				URL: config.GetEnv("GREETER_URL", "http://localhost:8081"),
			},
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

	if err := validator.ValidatePort(c.Server.Port); err != nil {
		return fmt.Errorf("server port: %w", err)
	}

	if err := validator.ValidateURL(c.Services.Greeter.URL); err != nil {
		return fmt.Errorf("greeter URL: %w", err)
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
