package typ

import "github.com/JackSpirou/chip/token"

// Parameter
type Parameter struct {
	typ  Typ //  This parameter's type.
	name string
	next *Parameter //  The parameter after this one.
}

//  Make a new PARAMETER that has TYPE.
func newParameter(typ Typ) *Parameter {
	return &Parameter{
		typ:  typ,
		name: typ.String(),
	}
}

//  GET TYPE. Return this parameter's type.
func (p *Parameter) Type() token.Tokint {
	return p.typ.Type()
}

func (p *Parameter) String() string {
	return p.name
}

//  GET NEXT. Return the parameter after this one, or NULL.
func (p *Parameter) Next() *Parameter {
	return p.next
}
