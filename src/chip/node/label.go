package node

import (
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/types"
)

type LabelNode struct {
	typ   types.Typer
	label *ssa.Label
}

func NewLabelNode(typ types.Typer, label *ssa.Label) *LabelNode {
	return &LabelNode{
		typ:   typ,
		label: label,
	}
}

func (l *LabelNode) Lvalue() string {
	return "Variable Node Lvalue()"
}

func (l *LabelNode) Rvalue() string {
	return "Variable Node Rvalue()"
}

func (l *LabelNode) Label() *ssa.Label {
	return l.label
}

func (l *LabelNode) Type() types.Typer {
	return l.typ
}

func (l *LabelNode) String() string {
	return "[LabelNode " + l.types.String() + " " + l.label.String() + "]"
}
