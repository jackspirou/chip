package node

import (
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/types"
)

// A Node that describes a name
type NameNode interface {
	Type() types.Typ
	String() string
	Label() *ssa.Label
	Lvalue() string
	Rvalue() string
}
