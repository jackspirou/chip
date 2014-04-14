package tokens

// Item represents a token returned from the scanner.
type Token struct {
	tok Tokint // Type, such as itemNumber.
	lit string // Value, such as "23.2".
	pos Position
	err error
}

func NewToken(tok Tokint, lit string, pos Position, err error) *Token {
	return &Token{tok: tok, lit: lit, pos:pos, err: err}
}

func (t *Token) String() string {
	return t.tok.String() + ": " + t.lit
}

func (t *Token) Int() Tokint {
	return t.tok
}
