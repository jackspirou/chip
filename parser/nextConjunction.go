package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextConjunction parses a conjunction.
func (p *Parser) nextConjunction() (conjunction ast.Node, err error) {
	p.enterNext()

	conjunction, err = p.nextComparison()

	for p.tok.Type == token.LAND {
		p.next() // skip '&&'
		conjunction, err = p.nextComparison()
	}

	p.exitNext()
	return conjunction, nil
}
