package types

import "github.com/jackspirou/chip/token"

// Func describes a function type.
type Func struct {
	arity int    //  Number of parameters for this function.
	first *Param //  Head node of the parameter list.
	last  *Param //  Last node of the parameter list.
	value Typer  //  The type this function returns.
}

// NewFunc returns a new Func object.
func NewFunc() *Func {
	p := &Param{}
	return &Func{arity: 0, first: p, last: p}
}

// AddParam adds a new parameter type to the end of the parameter list.
func (f *Func) AddParam(typ Typer) {
	f.arity++
	f.last.next = newParam(typ)
	f.last = f.last.next
}

// AddValue adds the value type.
func (f *Func) AddValue(value Typer) {
	f.value = value
}

// Value returns the function return type.
func (f *Func) Value() Typer {
	return f.value
}

// Arity returns the number of parameters in this function.
func (f *Func) Arity() int {
	return f.arity
}

// Param returns the first parameter of this function.
func (f *Func) Param() *Param {
	return f.first.next
}

// Type returns the token type of this function.
func (f *Func) Type() token.Type {
	return f.value.Type()
}

// String satisfies the fmt.Stringer interface.
func (f Func) String() string {
	return "FUNC TYPE"
}
