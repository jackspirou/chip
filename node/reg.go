package node

import (
	"fmt"

	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/types"
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

// String impliments the fmt.Stringer interface.
func (r Reg) String() string {
	return fmt.Sprintf("[RegNode %s %s ]", r.typ, r.reg)
}

func (r *Reg) Reg() *ssa.Register {
	return r.reg
}
