package domain

import "fmt"

// Greeter - domain entity
type Greeter struct {
	name string
}

func NewGreeter(name string) (*Greeter, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	return &Greeter{name: name}, nil
}

func (g *Greeter) GenerateGreeting() string {
	return fmt.Sprintf("Hello %s from Greeter Domain!", g.name)
}

func (g *Greeter) Name() string {
	return g.name
}
