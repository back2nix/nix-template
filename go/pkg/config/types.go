package config

// DatabaseConfig общая структура для конфигурации БД
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// MongoConfig конфигурация MongoDB
type MongoConfig struct {
	URI      string
	Database string
}

// LogConfig конфигурация логирования
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// ServerConfig базовая конфигурация HTTP сервера
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  int // seconds
	WriteTimeout int // seconds
}
