package ssa

import (
	"fmt"

	"github.com/jackspirou/chip/types"
)

// LabelNode describes a node label for SSA intermediate representation.
type LabelNode struct {
	typ   types.Typer
	label *Label
}

// NewNodeLabel creates a new label for a node.
func NewLabelNode(typ types.Typer, label *Label) *LabelNode {
	return &LabelNode{typ, label}
}

// Lvalue returns a string representing the left value of the label.
func (l *LabelNode) Lvalue() string {
	return "Variable Node Lvalue()"
}

// Rvalue returns a string representing the right value of the label.
func (l *LabelNode) Rvalue() string {
	return "Variable Node Rvalue()"
}

// Label returns the Label.
func (l LabelNode) Label() *Label {
	return l.label
}

// Type returns the Label.
func (l *LabelNode) Type() types.Typer {
	return l.typ
}

// String satisfies the fmt.Stringer interface.
func (l LabelNode) String() string {
	return fmt.Sprintf("[NodeLabel %s %s ]", l.typ, l.label)
}
