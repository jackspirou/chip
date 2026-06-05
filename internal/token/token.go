// Package token defines lexical tokens.
package token

// Token describes a lexical token.
type Token struct {
	Type Type   // token type
	lit  string // literal string value, e.g. "23.2"
	pos  Pos    // position of the token's first character
	end  Pos    // position one past the token's last character
}

// Pos describes a tokens position in a source file.
// A pos value is valid if Line > 0.
type Pos struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
	Offset int // 0-based byte offset from the start of the source
}

// New creates a new Token.
func New(typ Type, lit string, pos Pos) Token {
	return Token{Type: typ, lit: lit, pos: pos}
}

// NewEOF returns an EOF Token.
func NewEOF() Token {
	return New(EOF, "EOF", Pos{})
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

// Offset returns the Token's 0-based byte offset in the source file.
func (t Token) Offset() int {
	return t.pos.Offset
}

// Pos returns the Token's position in the source file.
func (t Token) Pos() Pos {
	return t.pos
}

// End returns the position one past the token's last character, so the
// half-open byte range [Pos().Offset, End().Offset) is the token's exact span.
// The scanner records it via WithEnd; tokens built elsewhere have the zero Pos.
func (t Token) End() Pos {
	return t.end
}

// WithEnd returns a copy of t with its end position set. The scanner uses this
// to record a token's byte-exact span; other callers read End instead.
func (t Token) WithEnd(end Pos) Token {
	t.end = end
	return t
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
