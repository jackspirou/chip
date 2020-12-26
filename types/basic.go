package types

import (
	"github.com/jackspirou/chip/token"
)

// Integer
type Integer struct{}

func (t Integer) Token() token.Type {
	return token.INT
}

func (t Integer) String() string {
	return token.INT.String()
}

// String
type String struct{}

func (t String) Token() token.Type {
	return token.STRING
}

func (t String) String() string {
	return token.STRING.String()
}
