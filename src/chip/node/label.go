package node

import (
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/types"
)

// Label describes a node label for SSA intermediate representation.
type Label struct {
	typ   types.Typer
	label *ssa.Label
}

// NewLabel creates a new label for a node.
func NewLabel(typ types.Typer, label *ssa.Label) *Label {
	return &Label{
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
