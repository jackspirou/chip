package parser

import "github.com/jackspirou/chip/token"

// nextTerm parses a term.
func (p *Parser) nextTerm() {
	p.enter()
	p.nextUnit()
	for p.tok.Type == token.SUB || p.tok.Type == token.NOT {
		p.next() // skip '-' or '!'
		p.nextUnit()
	}
	p.exit()
}
