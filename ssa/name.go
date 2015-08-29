package ssa

import (
	"github.com/jackspirou/chip/types"
)

// Name describes a name node.
type Name interface {
	Type() types.Typer
	String() string
	Label() *Label
	Lvalue() string
	Rvalue() string
}
