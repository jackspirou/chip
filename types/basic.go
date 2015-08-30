package types

import "github.com/jackspirou/chip/parser/token"

var Int *Basic
var String *Basic

func init() {
	Int = NewBasic(token.INT)
	String = NewBasic(token.STRING)
}

// Basic describes a basic type.
type Basic struct {
	tok  token.Type
	name string
}

// NewBasic returns a Basic object.
func NewBasic(tok token.Type) *Basic {
	return &Basic{tok, tok.String()}
}

// Token returns the token.Type.
func (b Basic) Token() token.Type {
	return b.tok
}

// Value returns the return values type.
func (b Basic) Value() Typer {
	return b
}

// String satisfies the fmt.Stringer interface.
func (b Basic) String() string {
	return b.name
}
