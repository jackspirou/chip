package node

import (
	"github.com/JackSpirou/chip/ssa"
	"github.com/JackSpirou/chip/typ"
)

// A Node that describes a register
type RegNode struct {
	typ typ.Typ
	reg *ssa.Register
}

func NewRegNode(typ typ.Typ, reg *ssa.Register) *RegNode {
	return &RegNode{
		typ: typ,
		reg: reg,
	}
}

func (r *RegNode) Type() typ.Typ {
	return r.typ
}

func (r *RegNode) String() string {
	return "[RegNode " + r.typ.String() + " " + r.reg.String() + "]"
}

func (r *RegNode) Reg() *ssa.Register {
	return r.reg
}
