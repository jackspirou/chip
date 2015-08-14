package types

import "github.com/jackspirou/chip/src/chip/token"

// Param describes a parameter type.
type Param struct {
	typ  Typer
	name string
	next *Param
}

// newParam returns a new parameter of the provided type.
func newParam(typ Typer) *Param {
	return &Param{
		typ:  typ,
		name: typ.String(),
	}
}

// Type returns the parameter's type.
func (p *Param) Type() token.Type {
	return p.typ.Type()
}

// String impliments the fmt.Stringer interface and returns the name of the
// parameter.
func (p *Param) String() string {
	return p.name
}

// Next returns the next parameter or nil.
func (p *Param) Next() *Param {
	return p.next
}
