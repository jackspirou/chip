// Package token defines constants representing the lexical tokens of the chip
// programming language and basic operations on tokens (printing, predicates).
//
package token

// Token represents a chip token.
type Token struct {
	typ Type   // token type
	lit string // token string value, e.g. "23.2"
	pos Pos    // token postion in the source file
}

// New takes a token type, string literal, and position.  It returns a chip token.
func New(typ Type, lit string, pos Pos) *Token {
	return &Token{typ: typ, lit: lit, pos: pos}
}

// NewEOF returns an EOF token.
func NewEOF() *Token {
	return New(EOF, "EOF", NewPos(0, 0))
}

// String turns token.Token into fmt.Stringer.
func (t *Token) String() string {
	return t.lit
}

// Type returns a token type.
func (t *Token) Type() Type {
	return t.typ
}

// Pos represents a position in a source file.  A Pos value is valid if Line > 0.
type Pos struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
}

// NewPos takes a line and column.  It returns a Pos.
func NewPos(line int, column int) Pos {
	return Pos{Line: line, Column: column}
}

// Valid checks if the Pos is valid.
func (pos Pos) Valid() bool { return pos.Line > 0 }

// Lookup finds a keyword token based on an identifier.  If no keyword found it defaults to token.IDENT.
func Lookup(ident string) Type {
	if tok, keyword := keywords[ident]; keyword {
		return tok
	}
	return IDENT
}
