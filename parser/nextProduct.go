package parser

import (
	"github.com/jackspirou/chip/ssa"

	"github.com/jackspirou/chip/token"
)

// nextProduct parses a product.
func (p *Parser) nextProduct() ssa.Node {
	p.enter()

	p.nextTerm()
	for p.tok.Type == token.MUL || p.tok.Type == token.QUO || p.tok.Type == token.REM {

		tok := p.tok
		p.next() // skip '*' or '/'
		p.nextTerm()

		if tok.Type == token.MUL {
		} else {
		}
	}

	p.exit()
	return nil
}
