package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Unit. Parse a unit.
func (p *Parser) nextUnit() {
	p.enter()
	switch p.tok.Type {
	case token.IDENT:
		//p.tok.Type == // var or proc name
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
		//p.tok.Type ==
		p.next()
	case token.FLOAT:
		//p.tok.Type ==
		p.next()
	case token.CHAR:
		//p.tok.Type ==
		p.next()
	case token.STRING:
		//p.tok.Type ==
		p.next()
	default:
		panic("Term expected.")
	}
	p.exit()
}
