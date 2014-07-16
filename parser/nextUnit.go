package parser

import "github.com/JackSpirou/chip/token"

// Next Unit. Parse a unit.
func (p *Parser) nextUnit() {
	p.enter()
	switch p.tok {
	case token.IDENT:
		//p.lit // var or proc name
		p.next()
		if p.tok == token.LPAREN {
			p.next() // skip '('
			if p.tok != token.RPAREN {
				p.nextExpression()
				for p.tok == token.COMMA {
					p.next() // skip ','
					p.nextExpression()
				}
			}
			p.nextExpected(token.RPAREN)
		} else if p.tok == token.LBRACK {
			p.next()
			p.nextExpression()
			p.nextExpected(token.RBRACK)
		}
	case token.INT:
		//p.lit
		p.next()
	case token.FLOAT:
		//p.lit
		p.next()
	case token.CHAR:
		//p.lit
		p.next()
	case token.STRING:
		//p.lit
		p.next()
	default:
		panic("Term expected.")
	}
	p.exit()
}
