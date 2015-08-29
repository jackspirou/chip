package parser

import (
	"fmt"
	"log"

	"github.com/jackspirou/chip/parser/token"
)

// nextStatement parse a statement.
func (p *Parser) nextStatement() {
	p.enter()
	switch p.tok.Type {
	case token.IDENT:
		nameTok := p.tok
		fmt.Println(nameTok)
		p.next()
		for p.tok.Type == token.PERIOD {
			p.next() // '.'
			p.nextExpected(token.IDENT)
		}
		if p.tok.Type.Assignment() {
			p.nextAssignment()
		} else {
			switch p.tok.Type {
			case token.DEFINE:
				p.nextDeclaration()
			case token.LBRACK:
				p.next() // '['
				p.nextExpression()
				p.nextExpected(token.RBRACK)
				p.nextAssignment()
			case token.LPAREN:
				p.next() // '('
				if p.tok.Type != token.RPAREN {
					paramNode := p.nextExpression()
					fmt.Println(paramNode)
					for p.tok.Type == token.COMMA {
						p.next() // ','
						paramTok := p.tok
						fmt.Println(paramTok)
						p.nextExpression()
					}
				}
				p.nextExpected(token.RPAREN)
			default:
				log.Fatalf("assignment or declaration statement expected, got '%s'", p.tok)
			}
		}
	case token.IF:
		p.nextIf()
	/*
		case token.FOR:
			p.nextFor()
	*/
	case token.RETURN:
		p.nextReturn()
	/*
		case token.SWITCH:
			p.nextSwitch()
	*/
	case token.CONST:
		p.nextConstant()
	default:
		log.Fatalf("statement expected, got '%s'", p.tok)
	}
	p.exit()
}
