package types

import (
	"fmt"

	"github.com/jackspirou/chip/parser/token"
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

// Token returns the token.Type.
func (a *Array) Token() token.Type {
	return a.basic.Token()
}

// Value returns the array type value.
func (a *Array) Value() Typer {
	return a.basic
}

// String satisfies the fmt.Stringer interface.
func (a Array) String() string {
	return fmt.Sprintf("%s array", a.name)
}
