package ast

import (
	"bytes"

	"github.com/jackspirou/chip/object"
	"github.com/jackspirou/chip/token"
)

const (
	STATEMENT  = "STATEMENT"
	EXPRESSION = "EXPRESSION"
	PROGRAM    = "PROGRAM"
)

const (
	INTEGER Type = iota
	BOOLEAN
	FLOAT
	STRING
	IDENT
	OPERATOR
	COMPARISON
)

type Type int

type Node interface {
	Type() Type
	Token() token.Token
	TokenLiteral() string
	SetValue(Value)
	StringValue() string
	IntegerValue() int
	FloatValue() float64
	String() string
}

// Value takes an value and returns an error.
type Value func(*value)

type value struct {
	i   int
	b   bool
	str string
	f   float64
}

// Integer sets the integer value.
func Integer(i int) Value {
	return func(v *value) {
		v.i = i
	}
}

// Boolean sets the boolean value.
func Boolean(b bool) Value {
	return func(v *value) {
		v.b = b
	}
}

// Float sets the float value.
func Float(f float64) Value {
	return func(v *value) {
		v.f = f
	}
}

// String sets the string value.
func String(s string) Value {
	return func(v *value) {
		v.str = s
	}
}

type node struct {
	t     Type
	tok   token.Token
	value value // option settings
}

func (n node) Type() Type {
	return n.t
}

func (n node) Token() token.Token {
	return n.tok
}

func (n node) TokenLiteral() string {
	return n.tok.String()
}

func (n node) String() string {
	return n.tok.String()
}

func (n node) SetValue(v Value) {
	if n.value != (value{}) {
		panic("SetValue: you can't set two values in a Node")
	}
	// set the value
	v(&n.value)
}

func (n node) IntegerValue() int {
	return n.value.i
}

func (n node) FloatValue() float64 {
	return n.value.f
}

func (n node) BooleanValue() bool {
	return n.value.b
}

func (n node) StringValue() string {
	return n.value.str
}

func NewNode(t Type, tok token.Token, val ...Value) Node {

	// new node
	n := &node{
		t:   t,
		tok: tok,
	}

	// range through the potential possible values given for this node.
	for _, v := range val {
		// only one value per node is allowed
		if n.value != (value{}) {
			panic("NewNode: you can't set two values in a Node")
		}
		// set the value
		v(&n.value)
	}

	return n
}

type Statement struct {
	Node
	Object object.Object
}

func (s Statement) String() string {
	return STATEMENT
}

func (s Statement) StringLiteral() string {
	return STATEMENT
}

type Expression struct {
	Node
}

func (e Expression) String() string {
	return EXPRESSION
}

type Program struct {
	Node
	Statements []Node
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}
func (p *Program) String() string {
	var out bytes.Buffer

	for _, stmt := range p.Statements {
		out.WriteString(stmt.String())
	}

	return out.String()
}
