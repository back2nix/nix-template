package config

import (
	"fmt"
	"net/url"
	"strconv"
)

// Validator предоставляет методы для валидации конфигурации
type Validator struct{}

// NewValidator создает новый валидатор
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePort проверяет что порт в допустимом диапазоне
func (v *Validator) ValidatePort(port string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", port)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", portNum)
	}

	return nil
}

// ValidateURL проверяет что URL валидный
func (v *Validator) ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must have a scheme (http/https): %s", rawURL)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host: %s", rawURL)
	}

	return nil
}

// ValidateRequired проверяет что значение не пустое
func (v *Validator) ValidateRequired(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateOneOf проверяет что значение входит в список допустимых
func (v *Validator) ValidateOneOf(value string, allowed []string, fieldName string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of %v, got: %s", fieldName, allowed, value)
}
