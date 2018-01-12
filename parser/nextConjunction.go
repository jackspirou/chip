package parser

import (
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
)

// nextConjunction parses a conjunction.
func (p *Parser) nextConjunction() ssa.Node {
	p.enter()

	p.nextComparison()

	for p.tok.Type == token.LAND {
		p.next() // skip '&&'
		p.nextComparison()
	}

	p.exit()
	return nil
}
