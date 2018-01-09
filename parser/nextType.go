package parser

import (
	"log"
	"strings"

	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/typ"
)

// nextType parses a type token.
func (p *Parser) nextType() typ.Type {
	p.enter()
	var t typ.Type
	switch strings.ToUpper(p.tok.String()) {
	case token.INT.String():
		t = typ.NewBasic(token.INT)
	case token.STRING.String():
		t = typ.NewBasic(token.STRING)
	default:
		log.Fatalf("unsupported type '%s'", p.tok)
	}
	p.next()
	p.exit()
	return t
}
