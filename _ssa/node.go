package ssa

import "github.com/jackspirou/chip/typ"

// Node represents a Node descriptor.
type Node interface {
	Type() typ.Type
	String() string
}
