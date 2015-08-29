package ssa

import (
	"fmt"

	"github.com/jackspirou/chip/types"
)

// RegNode describes a register node.
type RegNode struct {
	typ types.Typer
	reg *Register
}

func NewRegNode(typ types.Typer, reg *Register) *RegNode {
	return &RegNode{typ, reg}
}

func (r *RegNode) Type() types.Typer {
	return r.typ
}

// String impliments the fmt.Stringer interface.
func (r RegNode) String() string {
	return fmt.Sprintf("[RegNode %s %s ]", r.typ, r.reg)
}

func (r *RegNode) Reg() *Register {
	return r.reg
}
