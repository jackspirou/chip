package parser

import (
	"strings"

	"github.com/jackspirou/chip/src/chip/token"
	"github.com/jackspirou/chip/src/chip/types"
)

// Next Type.  Parse the next type token.
func (p *Parser) nextType() types.Typ {
	p.enter()
	var t types.Typ
	switch strings.ToUpper(p.lit) {
	case token.INT.String():
		t = types.NewBasicType(token.INT)
	case token.STRING.String():
		t = types.NewBasicType(token.STRING)
	default:
		panic(token.INT.String() + " vs " + strings.ToLower(p.lit) + " | unsupported type: " + p.lit)
	}
	p.next()
	p.exit()
	return t
}
