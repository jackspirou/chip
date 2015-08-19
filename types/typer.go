package types

import "github.com/jackspirou/chip/token"

// Typer represents a type.
type Typer interface {
	Token() token.Type
	Value() Typer
	String() string
}
