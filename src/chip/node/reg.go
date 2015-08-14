package node

import (
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/types"
)

// Reg describes a register node.
type Reg struct {
	typ types.Typer
	reg *ssa.Register
}

func NewReg(typ types.Typer, reg *ssa.Register) *Reg {
	return &Reg{typ, reg}
}

func (r *Reg) Type() types.Typer {
	return r.typ
}

func (r *Reg) String() string {
	return "[RegNode " + r.typ.String() + " " + r.reg.String() + "]"
}

func (r *Reg) Reg() *ssa.Register {
	return r.reg
}
