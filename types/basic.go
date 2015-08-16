package types

import "github.com/jackspirou/chip/token"

// Basic describes a basic type.
type Basic struct {
	tok  token.Type
	name string
}

// NewBasic returns a Basic object.
func NewBasic(tok token.Type) *Basic {
	return &Basic{tok, tok.String()}
}

// Type returns the token.Type.
func (b *Basic) Type() token.Type {
	return b.tok
}

// String satisfies the fmt.Stringer interface.
func (b Basic) String() string {
	return b.name
}
