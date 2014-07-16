package node

import (
	"github.com/JackSpirou/chip/ssa"
	"github.com/JackSpirou/chip/typ"
)

// A Node that describes a name
type NameNode interface {
	Type() typ.Typ
	String() string
	Label() *ssa.Label
	Lvalue() string
	Rvalue() string
}
