package parser

import (
	"fmt"
	"math"

	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextProduct parses a product.
func (p *Parser) nextProduct() (product ast.Node, err error) {
	p.enterNext()

	left, err := p.nextTerm()
	if err != nil {
		return nil, err
	}
	product = left

	for p.tok.Type == token.MUL || p.tok.Type == token.QUO || p.tok.Type == token.REM {

		// '*' or '/'
		operator := ast.NewNode(ast.OPERATOR, p.tok, ast.String(p.tok.String()))

		p.next() // skip '*' or '/'
		right, err := p.nextTerm()
		if err != nil {
			return nil, err
		}

		switch operator.Token().Type {
		case token.MUL:
			fmt.Println(left.IntegerValue() * right.IntegerValue())
		case token.QUO:
			fmt.Println(left.IntegerValue() / right.IntegerValue())
		case token.REM:
			fmt.Println(right.IntegerValue())
			fmt.Println(math.Mod(float64(left.IntegerValue()), float64(right.IntegerValue())))
		}

		product = right
	}

	p.exitNext()
	return product, err
}
