package parser

import (
	"github.com/JackSpirou/chip/token"
	"github.com/JackSpirou/chip/typ"
	"strings"
)

// Next Type.  Parse the next type token.
func (p *Parser) nextType() typ.Typ {
	p.enter()
	var t typ.Typ
	switch strings.ToUpper(p.lit) {
	case token.INT.String():
		t = typ.NewBasicType(token.INT)
	case token.STRING.String():
		t = typ.NewBasicType(token.STRING)
	default:
		panic(token.INT.String() + " vs " + strings.ToLower(p.lit) + " | unsupported type: " + p.lit)
	}
	p.next()
	p.exit()
	return t
}
