package config

// DatabaseConfig общая структура для конфигурации БД
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// MongoConfig конфигурация MongoDB
type MongoConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

// LogConfig конфигурация логирования
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// TelemetryConfig конфигурация observability
type TelemetryConfig struct {
	OtelEndpoint      string `mapstructure:"otel_endpoint"`
	PyroscopeEndpoint string `mapstructure:"pyroscope_endpoint"`
	ServiceName       string `mapstructure:"service_name"` // Имя сервиса для трейсинга
}

// ServerConfig базовая конфигурация HTTP/GRPC сервера
type ServerConfig struct {
	HTTPPort     string `mapstructure:"http_port"`
	GRPCPort     string `mapstructure:"grpc_port"`
	StaticDir    string `mapstructure:"static_dir"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// KafkaConfig конфигурация для брокера сообщений
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
	GroupID string   `mapstructure:"group_id"`
}

// ServicesConfig адреса зависимых микросервисов (Service Discovery)
type ServicesConfig struct {
	NotificationEndpoint string `mapstructure:"notification_endpoint"`
	ChatEndpoint         string `mapstructure:"chat_endpoint"`
	LandingEndpoint      string `mapstructure:"landing_endpoint"`
}

// AppConfig общая конфигурация приложения
type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Log       LogConfig       `mapstructure:"log"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Services  ServicesConfig  `mapstructure:"services"`
	// Можно добавлять специфичные секции, если нужно
}
