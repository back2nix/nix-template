package domain

import "errors"

var (
	ErrEmptyName = errors.New("name cannot be empty")
)
