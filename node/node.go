package node

import "github.com/jackspirou/chip/types"

// Node represents a general descriptor.
type Node interface {
	Type() types.Typer
	String() string
}
