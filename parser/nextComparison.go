package parser

import (
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
)

// nextComparison parses a comparison.
func (p *Parser) nextComparison() ssa.Node {
	p.enter()

	if p.tok.Type == token.LBRACE {
		return nil
	}

	p.nextSum()

	if p.tok.Type.Comparison() {
		switch p.tok.Type {
		case token.EQL: // left == right
		case token.LSS: // left < right
		case token.GTR: // left > right
		case token.NEQ: // left != right
		case token.LEQ: // left <= right
		case token.GEQ: // left >= right
		}
		p.next() // comparison
		p.nextSum()
	}

	p.exit()
	return nil
}
