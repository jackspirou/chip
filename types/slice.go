package types

import (
	"fmt"

	"github.com/jackspirou/chip/parser/token"
)

// Slice describes an slice type.
type Slice struct {
	basic Basic
	name  string
}

// NewSlice returns a slice object.
func NewSlice(basic Basic) *Slice {
	return &Slice{basic, basic.String()}
}

// Token returns the token.Type.
func (s *Slice) Token() token.Type {
	return s.basic.Token()
}

// Value returns the slice type value.
func (s *Slice) Value() Typer {
	return s.basic
}

// String satisfies the fmt.Stringer interface.
func (s Slice) String() string {
	return fmt.Sprintf("%s array", s.name)
}
