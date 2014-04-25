package token

// Item represents a token returned from the scanner.
type Tok struct {
	typ Tokint // Type, such as itemNumber.
	lit string // Value, such as "23.2".
	pos Position
}

func NewTok(typ Tokint, lit string, pos Position) *Tok {
	return &Tok{typ: typ, lit: lit, pos: pos}
}

func NewEndTok() *Tok {
	typ := EOF
	lit := "EOF"
	pos := NewPosition(0, 0)
	return NewTok(typ, lit, pos)
}

func (t *Tok) String() string {
	return t.typ.String() + ": " + t.lit
}

func (t *Tok) Typ() Tokint {
	return t.typ
}
