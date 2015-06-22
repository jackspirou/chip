package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Expression. Parse an expression.
func (p *Parser) nextExpression() {
	p.enter()
	p.nextConjunction()
	for p.tok == token.LOR {
		p.next() // skip '||'
		p.nextConjunction()
	}
	p.exit()
}
