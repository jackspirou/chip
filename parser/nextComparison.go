package parser

import (
	"github.com/jackspirou/chip/token"
)

// nextComparison parses a comparison.
func (p *Parser) nextComparison() {
	p.enter()
	p.nextSum()
	switch p.tok.Type {
	case token.EQL:
		p.next() // skip '=='
		p.nextSum()
	case token.LSS:
		p.next() // skip '<'
		p.nextSum()
	case token.GTR:
		p.next() // skip '>'
		p.nextSum()
	case token.NEQ:
		p.next() // skip '!='
		p.nextSum()
	case token.LEQ:
		p.next() // skip '<='
		p.nextSum()
	case token.GEQ:
		p.next() // skip '>='
		p.nextSum()
	}
	p.exit()
}
