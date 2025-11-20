package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader загружает конфигурацию из env файлов и переменных окружения
type Loader struct {
	env string
}

// NewLoader создает новый загрузчик конфигурации
func NewLoader() *Loader {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	return &Loader{env: env}
}

// findProjectRoot ищет корень проекта (где находится flake.nix)
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Идём вверх по директориям пока не найдём flake.nix
	for {
		if _, err := os.Stat(filepath.Join(dir, "flake.nix")); err == nil {
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
// 1. Дефолтные значения (в структуре конфига сервиса)
// 2. Файл configs/{APP_ENV}.env
// 3. OS environment variables (переопределяют всё)
func (l *Loader) Load() error {
	// Находим корень проекта
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("⚠️  Could not find project root: %v, using only environment variables\n", err)
		return nil
	}

	configPath := filepath.Join(projectRoot, "configs", fmt.Sprintf("%s.env", l.env))

	// Проверяем существование файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Файл не существует - продолжаем только с OS env
		fmt.Printf("⚠️  Config file %s not found, using only environment variables\n", configPath)
		return nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Парсим KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format in %s at line %d: %s", configPath, lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Убираем кавычки если есть
		value = strings.Trim(value, "\"'")

		// Устанавливаем переменную окружения ТОЛЬКО если она еще не установлена
		// Это даёт приоритет OS environment variables
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file %s: %w", configPath, err)
	}

	fmt.Printf("✅ Loaded config from %s\n", configPath)
	return nil
}

// GetEnv получает значение переменной окружения с fallback
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// MustGetEnv получает значение переменной окружения, паникует если не найдена
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

// GetEnvBool получает boolean значение из env
func GetEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}
