package parser

import (
	"log"
	"strings"

	"github.com/jackspirou/chip/parser/token"
	"github.com/jackspirou/chip/types"
)

// nextType parses a type token.
func (p *Parser) nextType() types.Typer {
	p.enter()
	var t types.Typer
	switch strings.ToUpper(p.tok.String()) {
	case token.INT.String():
		t = types.NewBasic(token.INT)
	case token.STRING.String():
		t = types.NewBasic(token.STRING)
	default:
		log.Fatalf("unsupported type '%s'", p.tok)
	}
	p.next()
	p.exit()
	return t
}
