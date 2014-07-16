package typ

import "github.com/JackSpirou/chip/token"

type ArrayType struct {
	basicType BasicType
	name      string
}

func NewArrayType(basicType BasicType) *ArrayType {
	return &ArrayType{
		basicType: basicType,
		name:      basicType.String(),
	}
}

func (a *ArrayType) Type() token.Tokint {
	return a.basicType.Type()
}

func (a *ArrayType) String() string {
	return a.name + " array"
}
