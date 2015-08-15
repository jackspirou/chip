package parser

import "github.com/jackspirou/chip/token"

// nextExpression parses an expression.
func (p *Parser) nextExpression() {
	p.enter()
	p.nextConjunction()
	for p.tok.Type == token.LOR {
		p.next() // skip '||'
		p.nextConjunction()
	}
	p.exit()
}
