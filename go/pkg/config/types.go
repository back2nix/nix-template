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

// ServerConfig базовая конфигурация HTTP сервера
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	HTTPPort     string `mapstructure:"http_port"`
	GRPCPort     string `mapstructure:"grpc_port"`
	StaticDir    string `mapstructure:"static_dir"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// AppConfig общая конфигурация приложения
type AppConfig struct {
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
}
