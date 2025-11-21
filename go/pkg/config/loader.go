package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader загружает конфигурацию с использованием Viper
type Loader struct {
	v   *viper.Viper
	env string
}

// NewLoader создает новый загрузчик конфигурации
func NewLoader() *Loader {
	v := viper.New()

	// Настраиваем чтение из environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	env := v.GetString("APP_ENV")
	if env == "" {
		env = "dev"
	}

	return &Loader{
		v:   v,
		env: env,
	}
}

// findProjectRoot ищет корень проекта (где находится flake.nix)
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Идём вверх по директориям пока не найдём flake.nix
	for {
		flakePath := filepath.Join(dir, "flake.nix")
		if _, err := os.Stat(flakePath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Дошли до корня файловой системы
			return "", fmt.Errorf("project root not found (no flake.nix)")
		}
		dir = parent
	}
}

// Load загружает конфигурацию с приоритетами:
// 1. Дефолтные значения (установленные через SetDefault)
// 2. Файл configs/{APP_ENV}.env
// 3. OS environment variables (переопределяют всё)
func (l *Loader) Load() error {
	// Находим корень проекта
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("⚠️  Could not find project root: %v, using only environment variables\n", err)
		return nil
	}

	configPath := filepath.Join(projectRoot, "configs")
	configName := l.env

	// Настраиваем Viper для чтения конфиг файла
	l.v.SetConfigName(configName)
	l.v.SetConfigType("env")
	l.v.AddConfigPath(configPath)

	// Пытаемся прочитать конфиг файл
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("⚠️  Config file %s.env not found in %s, using only environment variables\n", configName, configPath)
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	fmt.Printf("✅ Loaded config from %s\n", l.v.ConfigFileUsed())
	return nil
}

// GetViper возвращает экземпляр Viper для прямого доступа
func (l *Loader) GetViper() *viper.Viper {
	return l.v
}

// SetDefault устанавливает дефолтное значение
func (l *Loader) SetDefault(key string, value interface{}) {
	l.v.SetDefault(key, value)
}

// Unmarshal десериализует конфигурацию в структуру
func (l *Loader) Unmarshal(cfg interface{}) error {
	return l.v.Unmarshal(cfg)
}

// UnmarshalKey десериализует конкретный ключ в структуру
func (l *Loader) UnmarshalKey(key string, cfg interface{}) error {
	return l.v.UnmarshalKey(key, cfg)
}
