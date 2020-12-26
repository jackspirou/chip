package parser

import (
	"github.com/jackspirou/chip/ast"
	"github.com/jackspirou/chip/token"
)

// nextComparison parses a comparison.
func (p *Parser) nextComparison() (comparison ast.Node, err error) {
	p.enterNext()

	// no comparison, usually the forever loop case
	if p.tok.Type == token.LBRACE {
		booltok := token.New(token.IDENT, "", token.Pos{
			Line:   p.tok.Line(),
			Column: p.tok.Column() - 1,
		})
		return ast.NewNode(ast.COMPARISON, booltok, ast.Boolean(true)), nil
	}

	left, err := p.nextSum()
	if err != nil {
		return nil, err
	}

	if p.tok.Type.Comparison() {

		comparitor := ast.NewNode(ast.COMPARISON, p.tok, ast.String(p.tok.String()))

		tbool := ast.NewNode(ast.COMPARISON, p.tok, ast.Boolean(true))
		fbool := ast.NewNode(ast.COMPARISON, p.tok, ast.Boolean(false))
		p.next() // skip comparison token

		right, err := p.nextSum()
		if err != nil {
			return nil, err
		}

		switch comparitor.Token().Type {
		case token.EQL: // left == right
			if left.IntegerValue() == right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		case token.LSS: // left < right
			if left.IntegerValue() < right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		case token.GTR: // left > right
			if left.IntegerValue() > right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		case token.NEQ: // left != right
			if left.IntegerValue() != right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		case token.LEQ: // left <= right
			if left.IntegerValue() <= right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		case token.GEQ: // left >= right
			if left.IntegerValue() >= right.IntegerValue() {
				comparison = tbool
			}
			comparison = fbool
		}
	}

	p.exitNext()
	return comparison, nil
}
