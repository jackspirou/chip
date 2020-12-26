package types

import "github.com/jackspirou/chip/token"

// Func describes a function type.
type Func struct {
	arity int    //  Number of parameters for this function.
	first *Param //  Head node of the parameter list.
	last  *Param //  Last node of the parameter list.
	value Type   //  The type this function returns.
}

// NewFunc returns a new Func object.
func NewFunc() *Func {
	p := &Param{}
	return &Func{arity: 0, first: p, last: p}
}

// AddParam adds a new parameter type to the end of the parameter list.
func (f *Func) AddParam(t Type) {
	f.arity++
	f.last.next = newParam(t)
	f.last = f.last.next
}

// AddValue adds the value type.
func (f *Func) AddValue(value Type) {
	f.value = value
}

// Value returns the function return type.
func (f *Func) Value() Type {
	return f.value
}

// Token returns the function token.Type.
func (f *Func) Token() token.Type {
	return token.FUNC
}

// Arity returns the number of parameters in this function.
func (f *Func) Arity() int {
	return f.arity
}

// Param returns the first parameter of this function.
func (f *Func) Param() *Param {
	return f.first.next
}

// String satisfies the fmt.Stringer interface.
func (f Func) String() string {
	return "FUNC TYPE"
}
