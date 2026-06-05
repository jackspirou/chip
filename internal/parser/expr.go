package parser

import (
	"strconv"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// parseExpr parses an expression.
func (p *Parser) parseExpr() ast.Expr {
	return p.parseBinaryExpr(token.LowestPrec + 1)
}

// parseBinaryExpr parses a binary expression using precedence climbing.
// Operators are left-associative.
func (p *Parser) parseBinaryExpr(minPrec int) ast.Expr {
	x := p.parseUnaryExpr()
	for {
		op := p.tok.Type
		prec := op.Precedence()
		if prec < minPrec {
			return x
		}
		opPos := p.pos()
		p.next()
		y := p.parseBinaryExpr(prec + 1)
		x = &ast.BinaryExpr{Left: x, OpPos: opPos, Op: op, Right: y}
	}
}

// parseUnaryExpr parses a unary expression. It is the choke point for the whole
// expression recursion cycle — parens, calls, indexes, composite-literal
// elements, and array lengths all reach an operand through here — so the depth
// guard here bounds every form of expression nesting (see maxDepth).
func (p *Parser) parseUnaryExpr() ast.Expr {
	p.enter()
	defer p.leave()
	switch p.tok.Type {
	case token.ADD, token.SUB, token.NOT:
		opPos := p.pos()
		op := p.tok.Type
		p.next()
		return &ast.UnaryExpr{OpPos: opPos, Op: op, X: p.parseUnaryExpr()}
	}
	return p.parsePrimaryExpr()
}

// parsePrimaryExpr parses an operand followed by any calls, indexes, or
// selectors applied to it.
func (p *Parser) parsePrimaryExpr() ast.Expr {
	x := p.parseOperand()
	for {
		switch p.tok.Type {
		case token.LPAREN:
			x = p.parseCall(x)
		case token.LBRACK:
			x = p.parseIndex(x)
		case token.PERIOD:
			p.next()
			x = &ast.SelectorExpr{X: x, Sel: p.parseIdent()}
		default:
			return x
		}
	}
}

// parseOperand parses a literal, identifier, composite literal, or
// parenthesized expression.
func (p *Parser) parseOperand() ast.Expr {
	switch p.tok.Type {
	case token.IDENT:
		return p.parseIdent()

	case token.INT:
		pos, lit := p.pos(), p.tok.String()
		v, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			p.errorf(pos, "invalid integer literal %q", lit)
		}
		p.next()
		return &ast.IntLit{ValuePos: pos, Value: v, Lit: lit}

	case token.FLOAT:
		pos, lit := p.pos(), p.tok.String()
		v, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			p.errorf(pos, "invalid float literal %q", lit)
		}
		p.next()
		return &ast.FloatLit{ValuePos: pos, Value: v, Lit: lit}

	case token.STRING:
		pos, lit := p.pos(), p.tok.String()
		p.next()
		return &ast.StringLit{ValuePos: pos, Value: lit}

	case token.LBRACK:
		// A leading '[' begins an array type, optionally followed by a
		// composite literal ([]int{...}).
		typ := p.parseArrayType()
		if p.tok.Type == token.LBRACE {
			return p.parseCompositeLit(typ)
		}
		return typ

	case token.LPAREN:
		p.next()
		x := p.parseExpr()
		p.expect(token.RPAREN)
		return x

	default:
		pos := p.pos()
		p.errorf(pos, "expected expression, found %s", p.tok.Type)
		p.next() // guarantee progress
		return &ast.BadExpr{From: pos}
	}
}

// parseIdent parses an identifier.
func (p *Parser) parseIdent() *ast.Ident {
	pos := p.pos()
	if p.tok.Type != token.IDENT {
		p.errorf(pos, "expected identifier, found %s", p.tok.Type)
		return &ast.Ident{NamePos: pos}
	}
	name := p.tok.String()
	p.next()
	return &ast.Ident{NamePos: pos, Name: name}
}

// parseCall parses a call expression given the already-parsed function operand.
func (p *Parser) parseCall(fn ast.Expr) ast.Expr {
	lparen := p.expect(token.LPAREN)
	var args []ast.Expr
	for p.tok.Type != token.RPAREN && p.tok.Type != token.EOF {
		args = append(args, p.parseExpr())
		if !p.accept(token.COMMA) {
			break
		}
	}
	rparen := p.expect(token.RPAREN)
	return &ast.CallExpr{Fn: fn, Lparen: lparen, Args: args, Rparen: rparen}
}

// parseIndex parses an index expression given the already-parsed operand.
func (p *Parser) parseIndex(x ast.Expr) ast.Expr {
	lbrack := p.expect(token.LBRACK)
	index := p.parseExpr()
	rbrack := p.expect(token.RBRACK)
	return &ast.IndexExpr{X: x, Lbrack: lbrack, Index: index, Rbrack: rbrack}
}

// parseArrayType parses []Elem or [Len]Elem. The element type recurses through
// parseType back into parseArrayType for [][]...Elem, a nesting cycle that does
// not pass through the expression choke point, so it carries its own depth guard.
func (p *Parser) parseArrayType() ast.Expr {
	p.enter()
	defer p.leave()
	lbrack := p.expect(token.LBRACK)
	var length ast.Expr
	if p.tok.Type != token.RBRACK {
		length = p.parseExpr()
	}
	p.expect(token.RBRACK)
	return &ast.ArrayType{Lbrack: lbrack, Len: length, Elem: p.parseType()}
}

// parseCompositeLit parses a composite literal {e1, e2, ...} for the given type.
func (p *Parser) parseCompositeLit(typ ast.Expr) ast.Expr {
	lbrace := p.expect(token.LBRACE)
	var elems []ast.Expr
	for p.tok.Type != token.RBRACE && p.tok.Type != token.EOF {
		elems = append(elems, p.parseExpr())
		if !p.accept(token.COMMA) {
			break
		}
	}
	rbrace := p.expect(token.RBRACE)
	return &ast.CompositeLit{Type: typ, Lbrace: lbrace, Elems: elems, Rbrace: rbrace}
}
