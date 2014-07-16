package node

import (
	"github.com/JackSpirou/chip/ssa"
	"github.com/JackSpirou/chip/typ"
)

type LabelNode struct {
	typ   typ.Typ
	label *ssa.Label
}

func NewLabelNode(typ typ.Typ, label *ssa.Label) *LabelNode {
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

func (l *LabelNode) Type() typ.Typ {
	return l.typ
}

func (l *LabelNode) String() string {
	return "[LabelNode " + l.typ.String() + " " + l.label.String() + "]"
}
