package types

import "github.com/jackspirou/chip/token"

// Proc describes a procedure type.
type Proc struct {
	arity int    //  Number of parameters for this procedure.
	first *Param //  Head node of the parameter list.
	last  *Param //  Last node of the parameter list.
	value Typer  //  The type this procedure returns.
}

// NewProc returns a new Proc object.
func NewProc() *Proc {
	p := &Param{}
	return &Proc{arity: 0, first: p, last: p}
}

// AddParam adds a new parameter type to the end of the parameter list.
func (p *Proc) AddParam(typ Typer) {
	p.arity++
	p.last.next = newParam(typ)
	p.last = p.last.next
}

// AddValue adds the value type.
func (p *Proc) AddValue(value Typer) {
	p.value = value
}

// Value returns the procedures type.
func (p *Proc) Value() Typer {
	return p.value
}

// Arity returns the number of parameters in this procedure.
func (p *Proc) Arity() int {
	return p.arity
}

// Param returns the first parameter of this procedure.
func (p *Proc) Param() *Param {
	return p.first.next
}

// Type returns the token type of this procedure.
func (p *Proc) Type() token.Type {
	return p.value.Type()
}

// String satisfies the fmt.Stringer interface.
func (p *Proc) String() string {
	return "PROC TYPE"
}
