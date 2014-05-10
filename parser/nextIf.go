package parser

import "github.com/jackspirou/chip/token"

// Next If. Parse an if statement.
func (p *Parser) nextIf() {
	p.enter()
	for true {
		p.nextExpected(token.IF)
		p.nextExpression()
		p.nextExpected(token.LBRACE)
		p.nextStatement()
		for p.tok != token.RBRACE {
			p.nextStatement()
		}
		p.nextExpected(token.RBRACE)
		if p.tok != token.ELSE {
			break
		}
		p.next() // skip 'else'
		if p.tok != token.IF {
			p.nextExpected(token.LBRACE)
			p.nextStatement()
			for p.tok != token.RBRACE {
				p.nextStatement()
			}
			p.nextExpected(token.RBRACE)
			break
		}
	}
	p.exit()
}
