package parser

import (
	"log"
	"strings"

	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/types"
)

// nextType parses a type token.
func (p *Parser) nextType() types.Type {
	p.enterNext()

	var t types.Type
	switch strings.ToUpper(p.tok.String()) {
	case token.INT.String():
		t = types.Integer{}
	case token.STRING.String():
		t = types.String{}
	default:
		log.Fatalf("unsupported type '%s'", p.tok)
	}
	p.next()

	p.exitNext()
	return t
}
