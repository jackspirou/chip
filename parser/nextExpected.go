package parser

import (
	"log"

	"github.com/jackspirou/chip/token"
)

// nextExpected expects the next token to match the token.Type provided.
func (p *Parser) nextExpected(expected token.Type) {
	if p.tok.Type == expected {
		p.next()
		return
	}
	log.Fatalf("expected '%s', got '%s'", expected.String(), p.tok.String())
}
