package parser

import "github.com/jackspirou/chip/token"

// nextIf parses an if statement.
func (p *Parser) nextIf() {
	p.enter()
	for {
		// the nextIf loop expects an 'if' token
		p.nextExpected(token.IF)
		p.nextExpression()
		p.nextExpected(token.LBRACE)
		p.nextStatement()
		for p.tok.Type != token.RBRACE {
			p.nextStatement()
		}
		p.nextExpected(token.RBRACE)
		if p.tok.Type != token.ELSE {
			break
		}
		p.next() // skip 'else'
		if p.tok.Type != token.IF {
			p.nextExpected(token.LBRACE)
			p.nextStatement()
			for p.tok.Type != token.RBRACE {
				p.nextStatement()
			}
			p.nextExpected(token.RBRACE)
			break
		}
	}
	p.exit()
}
