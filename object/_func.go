package object

import (
	"fmt"

	"github.com/jackspirou/chip/types"
)

// FuncNode describes a node label for SSA intermediate representation.
type FuncNode struct {
	typ   types.Type
	label *Label
}

// NewFuncNode creates a new label for a node.
func NewFuncNode(typ types.Type, label *Label) *FuncNode {
	return &FuncNode{typ, label}
}

// Lvalue returns a string representing the left value of the label.
func (l *FuncNode) Lvalue() string {
	return "Variable Node Lvalue()"
}

// Rvalue returns a string representing the right value of the label.
func (l *FuncNode) Rvalue() string {
	return "Variable Node Rvalue()"
}

// Label returns the Label.
func (l FuncNode) Label() *Label {
	return l.label
}

// Type returns the FuncNode Label.
func (l *FuncNode) Type() types.Type {
	return l.typ
}

// String satisfies the fmt.Stringer interface.
func (l FuncNode) String() string {
	return fmt.Sprintf("[FuncNode %s %s ]", l.typ, l.label)
}
