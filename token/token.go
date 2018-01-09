// Package token defines lexical tokens.
package token

// Token describes a lexical token.
type Token struct {
	Type Type   // token type
	lit  string // literal string value, e.g. "23.2"
	pos  Pos    // postion in the source file
}

// Pos describes a tokens position in a source file.
// A pos value is valid if Line > 0.
type Pos struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
}

// New creates a new Token.
func New(typ Type, lit string, pos Pos) Token {
	return Token{typ, lit, pos}
}

// NewEOF returns an EOF Token.
func NewEOF() Token {
	return New(EOF, "EOF", Pos{0, 0})
}

// String impliments the fmt.Stringer interface.
func (t Token) String() string {
	return t.lit
}

// Error impliments the errors interface.
func (t Token) Error() string {
	return t.String()
}

// Line returns the Token line number.
func (t Token) Line() int {
	return t.pos.Line
}

// Column returns the Token column number.
func (t Token) Column() int {
	return t.pos.Column
}

// Valid validates a Token.
func (t Token) Valid() bool {
	return t.pos.Line > 0
}

// Lookup finds a keyword token based on an identifier.
// If no keyword found it defaults to token.IDENT.
func Lookup(ident string) Type {
	if tok, keyword := keywords[ident]; keyword {
		return tok
	}
	return IDENT
}
