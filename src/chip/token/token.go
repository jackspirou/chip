// Package token defines constants and types representing lexical tokens.
//
package token

// Token describes a lexical token.
type Token struct {
	typ Type   // token type
	lit string // string value, e.g. "23.2"
	pos Pos    // postion in the source file
}

// New creates a new Token.
func New(typ Type, lit string, pos Pos) Token {
	return Token{typ: typ, lit: lit, pos: pos}
}

// NewEOF returns an EOF Token.
func NewEOF() Token {
	return New(EOF, "EOF", NewPos(0, 0))
}

// String makes Token impliment the fmt.Stringer interface.
func (t Token) String() string {
	return t.lit
}

// Type returns a tokens type.
func (t Token) Type() Type {
	return t.typ
}

// Pos describes a tokens position in a source file.
//
// A Pos value is valid if Line > 0.
type Pos struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
}

// NewPos returns a new Pos object.
func NewPos(line int, column int) Pos {
	return Pos{Line: line, Column: column}
}

// Valid validates a Pos object.
func (pos Pos) Valid() bool { return pos.Line > 0 }

// Lookup finds a keyword token based on an identifier.
// If no keyword found it defaults to token.IDENT.
func Lookup(ident string) Type {
	if tok, keyword := keywords[ident]; keyword {
		return tok
	}
	return IDENT
}
