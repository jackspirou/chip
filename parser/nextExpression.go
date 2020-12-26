package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextExpression parses an expression.
func (p *Parser) nextExpression() (expression ast.Node, err error) {
	p.enterNext()

	expression, err = p.nextConjunction()
	for p.tok.Type == token.LOR {
		p.next() // skip '||'
		expression, err = p.nextConjunction()
	}

	p.exitNext()
	return expression, nil
}
