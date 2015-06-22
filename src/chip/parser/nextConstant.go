package parser

import "github.com/jackspirou/chip/src/chip/token"

// Next Constant. Parse a constant.
func (p *Parser) nextConstant() {
	p.enter()
	p.nextExpected(token.CONST)
	if p.tok == token.LPAREN {
		p.next() // skip '('
		for p.tok != token.RPAREN {
			//p.lit // var name
			p.nextExpected(token.IDENT)
			if p.tok == token.LBRACK {
				p.next() // skip '['
				p.nextExpression()
				p.nextExpected(token.RBRACK)
			}
			if p.tok == token.DEFINE {
				p.nextDeclaration()
			} else {
				//p.lit // type name
				p.nextExpected(token.IDENT)
				p.nextExpected(token.ASSIGN)
				p.nextExpected(token.IOTA)
				for p.tok != token.RPAREN {
					//p.lit // var name
					p.nextExpected(token.IDENT)
				}
			}
		}
		p.next() // skip ')'
	} else {
		//p.lit // var name
		p.nextExpected(token.IDENT)
		p.nextDeclaration()
	}
	p.exit()
}
