package node

import (
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/types"
)

// A Node that describes a register
type RegNode struct {
	typ types.Typer
	reg *ssa.Register
}

func NewRegNode(typ types.Typer, reg *ssa.Register) *RegNode {
	return &RegNode{
		typ: typ,
		reg: reg,
	}
}

func (r *RegNode) Type() types.Typer {
	return r.typ
}

func (r *RegNode) String() string {
	return "[RegNode " + r.typ.String() + " " + r.reg.String() + "]"
}

func (r *RegNode) Reg() *ssa.Register {
	return r.reg
}
