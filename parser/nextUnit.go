package parser

import (
	"log"

	"github.com/jackspirou/chip/token"
)

// nextUnit parses a unit.
func (p *Parser) nextUnit() {
	p.enter()
	switch p.tok.Type {
	case token.IDENT:
		// p.tok.String() // var or func name
		p.next()
		if p.tok.Type == token.LPAREN {
			p.next() // skip '('
			if p.tok.Type != token.RPAREN {
				p.nextExpression()
				for p.tok.Type == token.COMMA {
					p.next() // skip ','
					p.nextExpression()
				}
			}
			p.nextExpected(token.RPAREN)
		} else if p.tok.Type == token.LBRACK {
			p.next()
			p.nextExpression()
			p.nextExpected(token.RBRACK)
		}
	case token.INT:
		// p.tok.String()
		p.next()
	case token.FLOAT:
		// p.tok.String()
		p.next()
	case token.CHAR:
		// p.tok.String()
		p.next()
	case token.STRING:
		// p.tok.String()
		p.next()
	default:
		log.Fatalf("term expected, got '%s'", p.tok)
	}
	p.exit()
}
