package ssa

import "github.com/jackspirou/chip/types"

// Node represents a Node descriptor.
type Node interface {
	Type() types.Typer
	String() string
}
