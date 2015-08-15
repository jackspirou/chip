package parser

import (
	"strings"

	"github.com/jackspirou/chip/token"
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
		panic(token.INT.String() + " vs " + strings.ToLower(p.tok.String()) + " | unsupported type: " + p.tok.String())
	}
	p.next()
	p.exit()
	return t
}
