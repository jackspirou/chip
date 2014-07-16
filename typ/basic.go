package typ

import "github.com/JackSpirou/chip/token"

type BasicType struct {
	tok  token.Tokint
	name string
}

func NewBasicType(tok token.Tokint) *BasicType {
	return &BasicType{
		tok:  tok,
		name: tok.String(),
	}
}

func (b *BasicType) Type() token.Tokint {
	return b.tok
}

func (b *BasicType) String() string {
	return b.name
}
