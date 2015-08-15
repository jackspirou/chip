package parser

import "github.com/jackspirou/chip/token"

// nextProduct parses a product.
func (p *Parser) nextProduct() {
	p.enter()
	p.nextTerm()
	for p.tok.Type == token.MUL || p.tok.Type == token.QUO || p.tok.Type == token.REM {
		p.next() // skip '*' or '/'
		p.nextTerm()
	}
	p.exit()
}