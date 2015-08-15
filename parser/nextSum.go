package parser

import "github.com/jackspirou/chip/token"

// nextSum parse a sum.
func (p *Parser) nextSum() {
	p.enter()
	p.nextProduct()
	for p.tok.Type == token.ADD || p.tok.Type == token.SUB {
		p.next() // skip '+' or '-'
		p.nextProduct()
	}
	p.exit()
}
