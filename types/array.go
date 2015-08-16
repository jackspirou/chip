package types

import (
	"fmt"

	"github.com/jackspirou/chip/token"
)

// Array describes an array type.
type Array struct {
	basic Basic
	name  string
}

// NewArray returns an Array object.
func NewArray(basic Basic) *Array {
	return &Array{basic, basic.String()}
}

// Type returns the token.Type.
func (a *Array) Type() token.Type {
	return a.basic.Type()
}

// String satisfies the fmt.Stringer interface.
func (a Array) String() string {
	return fmt.Sprintf("%s array", a.name)
}
