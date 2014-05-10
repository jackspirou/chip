package parser

import "github.com/jackspirou/chip/token"

// Next Product. Parse a product.
func (p *Parser) nextProduct() {
	p.enter()
	p.nextTerm()
	for p.tok == token.MUL || p.tok == token.QUO || p.tok == token.REM {
		p.next() // skip '*' or '/'
		p.nextTerm()
	}
	p.exit()
}
