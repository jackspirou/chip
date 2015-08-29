package types

import "github.com/jackspirou/chip/parser/token"

// Typer represents a type.
type Typer interface {
	Token() token.Type
	Value() Typer
	// Subtype(t Typer) bool
	String() string
}
