package node

import (
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/types"
)

// Name describes a name node.
type Name interface {
	Type() types.Typer
	String() string
	Label() *ssa.Label
	Lvalue() string
	Rvalue() string
}
