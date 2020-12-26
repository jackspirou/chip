package ssa

import "github.com/jackspirou/chip/typ"

// Name describes a name node.
type Name interface {
	Type() typ.Type
	String() string
	Label() *Label
	Lvalue() string
	Rvalue() string
}
