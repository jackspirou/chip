package parser

import "github.com/jackspirou/chip/token"

// nextConjunction parses a conjunction.
func (p *Parser) nextConjunction() {
	p.enter()
	p.nextComparison()
	for p.tok.Type == token.LAND {
		p.next() // skip '&&'
		p.nextComparison()
	}
	p.exit()
}
