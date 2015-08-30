// Package token defines lexical tokens.
package token

// Token describes a lexical token.
type Token struct {
	Type Type   // token type
	lit  string // string value, e.g. "23.2"
	Pos  Pos    // postion in the source file
}

// New creates a new Token.
func New(typ Type, lit string, pos Pos) Token {
	return Token{typ, lit, pos}
}

// NewEOF returns an EOF Token.
func NewEOF() Token {
	return New(EOF, "EOF", NewPos(0, 0))
}

// String makes Token impliment the fmt.Stringer interface.
func (t Token) String() string {
	return t.lit
}

// Error makes Token impliment the errors interface.
func (t Token) Error() string {
	return t.String()
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
	return Pos{line, column}
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
