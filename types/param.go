package types

// Param describes a parameter type.
type Param struct {
	t    Type
	name string
	next *Param
}

// newParam returns a new parameter of the provided type.
func newParam(t Type) *Param {
	return &Param{t: t, name: t.String()}
}

// Value returns the parameter's token type.
func (p Param) Value() Type {
	return p.t
}

// String impliments the fmt.Stringer interface and returns the name of the
// parameter.
func (p Param) String() string {
	return p.name
}

// Next returns the next parameter or nil.
func (p *Param) Next() *Param {
	return p.next
}
