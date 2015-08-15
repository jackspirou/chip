package types

import "github.com/jackspirou/chip/token"

// Typer represents a type.
type Typer interface {
	Type() token.Type
	String() string
}
