package parser

import (
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
)

// nextExpression parses an expression.
func (p *Parser) nextExpression() ssa.Node {
	p.enter()

	p.nextConjunction()
	for p.tok.Type == token.LOR {
		p.next() // skip '||'
		p.nextConjunction()
	}

	p.exit()
	return nil
}
