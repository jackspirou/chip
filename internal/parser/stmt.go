package parser

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// parseBlock parses a brace-enclosed list of statements.
func (p *Parser) parseBlock() *ast.Block {
	lbrace := p.expect(token.LBRACE)
	var list []ast.Stmt
	for p.tok.Type != token.RBRACE && p.tok.Type != token.EOF {
		start := p.nadv
		list = append(list, p.parseStmt())
		if p.nadv == start {
			p.next() // guarantee progress on a malformed statement
		}
	}
	rbrace := p.expect(token.RBRACE)
	return &ast.Block{Lbrace: lbrace, List: list, Rbrace: rbrace}
}

// parseStmt parses a single statement. Every statement in a block reaches here,
// so the depth guard bounds block-nested control flow (if/for bodies). Else-if
// chains recurse through parseIf without re-entering parseStmt, so parseIf
// guards itself separately.
func (p *Parser) parseStmt() ast.Stmt {
	p.enter()
	defer p.leave()
	switch p.tok.Type {
	case token.IF:
		return p.parseIf()
	case token.FOR:
		return p.parseFor()
	case token.RETURN:
		return p.parseReturn()
	case token.IDENT, token.INT, token.FLOAT, token.STRING,
		token.LPAREN, token.ADD, token.SUB, token.NOT:
		return p.parseSimpleStmt()
	default:
		pos := p.pos()
		p.errorf(pos, "expected statement, found %s", p.tok.Type)
		return &ast.BadStmt{From: pos}
	}
}

// parseSimpleStmt parses a declaration (:=), an assignment (=), or a bare
// expression statement.
func (p *Parser) parseSimpleStmt() ast.Stmt {
	x := p.parseExpr()
	switch p.tok.Type {
	case token.DEFINE:
		defPos := p.pos()
		p.next()
		rhs := p.parseExpr()
		name, ok := x.(*ast.Ident)
		if !ok {
			p.error(x.Pos(), "expected identifier on left side of :=")
			name = &ast.Ident{NamePos: x.Pos()}
		}
		return &ast.DeclStmt{Name: name, DefPos: defPos, Value: rhs}

	case token.ASSIGN:
		opPos := p.pos()
		p.next()
		return &ast.AssignStmt{Lhs: x, OpPos: opPos, Op: token.ASSIGN, Rhs: p.parseExpr()}

	default:
		return &ast.ExprStmt{X: x}
	}
}

// parseIf parses an if statement with an optional else / else-if. An else-if
// chain recurses here directly (els = parseIf) rather than through parseStmt, and
// each iteration's block is fully parsed and left before the tail recurses, so
// the parseStmt and parseBlock guards do not see the chain depth — parseIf must
// guard itself to bound a long else-if chain.
func (p *Parser) parseIf() ast.Stmt {
	p.enter()
	defer p.leave()
	ifPos := p.expect(token.IF)
	cond := p.parseExpr()
	body := p.parseBlock()

	var els ast.Stmt
	if p.accept(token.ELSE) {
		if p.tok.Type == token.IF {
			els = p.parseIf()
		} else {
			els = p.parseBlock()
		}
	}
	return &ast.IfStmt{If: ifPos, Cond: cond, Body: body, Else: els}
}

// parseFor parses a for loop. A missing condition (for { ... }) is an infinite
// loop.
func (p *Parser) parseFor() ast.Stmt {
	forPos := p.expect(token.FOR)
	var cond ast.Expr
	if p.tok.Type != token.LBRACE {
		cond = p.parseExpr()
	}
	return &ast.ForStmt{For: forPos, Cond: cond, Body: p.parseBlock()}
}

// parseReturn parses a return statement with an optional result.
func (p *Parser) parseReturn() ast.Stmt {
	retPos := p.expect(token.RETURN)
	var result ast.Expr
	if p.tok.Type != token.RBRACE && p.tok.Type != token.EOF {
		result = p.parseExpr()
	}
	return &ast.ReturnStmt{Return: retPos, Result: result}
}
