package types

import "github.com/jackspirou/chip/src/chip/token"

// Typer describes a type.
type Typer interface {
	Type() token.Type
	String() string
}
