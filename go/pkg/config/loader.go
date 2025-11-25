package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// Loader загружает конфигурацию с использованием Viper
type Loader struct {
	v   *viper.Viper
	env string
	prefix string
}

// NewLoader создает новый загрузчик.
func NewLoader(serviceName string) *Loader {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if serviceName != "" {
		v.SetEnvPrefix(serviceName)
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	return &Loader{
		v:      v,
		env:    env,
		prefix: serviceName,
	}
}

// Load загружает конфигурацию из файла (если есть)
func (l *Loader) Load() error {
	configPath := findConfigPath()
	configName := l.env

	l.v.SetConfigName(configName)
	l.v.SetConfigType("env")
	l.v.AddConfigPath(configPath)

	// Пытаемся прочитать файл, но не умираем, если его нет (полагаемся на ENV)
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Если файла нет, это нормально, идем дальше
	} else {
		fmt.Printf("✅ Loaded config from %s\n", l.v.ConfigFileUsed())
	}

	return nil
}

// Unmarshal десериализует конфигурацию и биндит ENV переменные
func (l *Loader) Unmarshal(cfg interface{}) error {
	// ВАЖНО: Явно биндим ENV переменные для всех полей структуры
	if err := l.bindEnvs(cfg); err != nil {
		return err
	}
	return l.v.Unmarshal(cfg)
}

// bindEnvs рекурсивно проходит по полям структуры и делает v.BindEnv
func (l *Loader) bindEnvs(iface interface{}, parts ...string) error {
	ifv := reflect.ValueOf(iface)
	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
	}

	ift := ifv.Type()
	for i := 0; i < ift.NumField(); i++ {
		field := ift.Field(i)
		tv, ok := field.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}

		// Обрабатываем вложенные структуры (например, ServerConfig)
		if field.Type.Kind() == reflect.Struct {
			if err := l.bindEnvs(ifv.Field(i).Interface(), append(parts, tv)...); err != nil {
				return err
			}
			continue
		}

		// Формируем ключ: server.http_port
		key := strings.Join(append(parts, tv), ".")
		if err := l.v.BindEnv(key); err != nil {
			return err
		}

		// Debug log (optional, enabled for troubleshooting)
		// envKey := strings.ToUpper(l.prefix + "_" + strings.ReplaceAll(key, ".", "_"))
		// fmt.Printf("🔧 Binding Config Key '%s' -> Env '%s'\n", key, envKey)
	}
	return nil
}

func findConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		configPath := filepath.Join(dir, "configs")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
