package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Term. Parse a term.
func (p *Parser) nextTerm() {
	p.enter()
	p.nextUnit()
	for p.tok == token.SUB || p.tok == token.NOT {
		p.next() // skip '-' or '!'
		p.nextUnit()
	}
	p.exit()
}
