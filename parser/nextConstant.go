package parser

import "github.com/jackspirou/chip/token"

// nextConstant parses a constant.
func (p *Parser) nextConstant() {
	p.enter()
	p.nextExpected(token.CONST)
	if p.tok.Type == token.LPAREN {
		p.next() // skip '('
		for p.tok.Type != token.RPAREN {
			//p.tok.Type == // var name
			p.nextExpected(token.IDENT)
			if p.tok.Type == token.LBRACK {
				p.next() // skip '['
				p.nextExpression()
				p.nextExpected(token.RBRACK)
			}
			if p.tok.Type == token.DEFINE {
				p.nextDeclaration()
			} else {
				//p.tok.Type == // type name
				p.nextExpected(token.IDENT)
				p.nextExpected(token.ASSIGN)
				p.nextExpected(token.IOTA)
				for p.tok.Type != token.RPAREN {
					//p.tok.Type == // var name
					p.nextExpected(token.IDENT)
				}
			}
		}
		p.next() // skip ')'
	} else {
		//p.tok.Type == // var name
		p.nextExpected(token.IDENT)
		p.nextDeclaration()
	}
	p.exit()
}
