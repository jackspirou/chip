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

// Lvalue returns a string representing the left value of the label.
func (l *Label) Lvalue() string {
	return "Variable Node Lvalue()"
}

// Rvalue returns a string representing the right value of the label.
func (l *Label) Rvalue() string {
	return "Variable Node Rvalue()"
}

func (l *Label) Label() *ssa.Label {
	return l.label
}

func (l *Label) Type() types.Typer {
	return l.typ
}

// String satisfies the fmt.Stringer interface.
func (l *Label) String() string {
	return "[LabelNode " + l.typ.String() + " " + l.label.String() + "]"
}
