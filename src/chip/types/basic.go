package types

import "github.com/jackspirou/chip/src/chip/token"

type BasicType struct {
	tok  token.Type
	name string
}

func NewBasicType(tok token.Type) *BasicType {
	return &BasicType{
		tok:  tok,
		name: tok.String(),
	}
}

func (b *BasicType) Type() token.Type {
	return b.tok
}

func (b *BasicType) String() string {
	return b.name
}
