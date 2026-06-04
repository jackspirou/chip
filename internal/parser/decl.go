package parser

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// parseFile parses a whole source file: an optional package clause, optional
// imports, and a sequence of top-level declarations.
func (p *Parser) parseFile() *ast.File {
	file := &ast.File{
		Package: p.parsePackage(),
		Imports: p.parseImports(),
	}

	for p.tok.Type != token.EOF {
		start := p.nadv
		switch p.tok.Type {
		case token.FUNC:
			file.Decls = append(file.Decls, p.parseFuncDecl())
		default:
			p.errorf(p.pos(), "expected declaration, found %s", p.tok.Type)
		}
		if p.nadv == start {
			p.next() // skip an unexpected token to make progress
		}
	}
	file.Comments = p.comments
	return file
}

// parsePackage parses an optional package clause.
func (p *Parser) parsePackage() *ast.Ident {
	if !p.accept(token.PACKAGE) {
		return nil
	}
	return p.parseIdent()
}

// parseImports parses an optional import declaration (single or grouped).
func (p *Parser) parseImports() []*ast.ImportSpec {
	if !p.accept(token.IMPORT) {
		return nil
	}

	var specs []*ast.ImportSpec
	if p.accept(token.LPAREN) {
		for p.tok.Type != token.RPAREN && p.tok.Type != token.EOF {
			start := p.nadv
			specs = append(specs, p.parseImportSpec())
			if p.nadv == start {
				p.next()
			}
		}
		p.expect(token.RPAREN)
		return specs
	}
	return append(specs, p.parseImportSpec())
}

// parseImportSpec parses a single import: an optional alias and a path string.
func (p *Parser) parseImportSpec() *ast.ImportSpec {
	spec := &ast.ImportSpec{}
	if p.tok.Type == token.IDENT {
		spec.Name = p.parseIdent()
	}

	pos := p.pos()
	if p.tok.Type != token.STRING {
		p.errorf(pos, "expected import path string, found %s", p.tok.Type)
		return spec
	}
	spec.Path = &ast.StringLit{ValuePos: pos, Value: p.tok.String()}
	p.next()
	return spec
}

// parseFuncDecl parses a function declaration.
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	funcPos := p.expect(token.FUNC)
	return &ast.FuncDecl{
		Func:    funcPos,
		Name:    p.parseIdent(),
		Params:  p.parseParams(),
		Results: p.parseResults(),
		Body:    p.parseBlock(),
	}
}

// parseParams parses a parenthesized list of "name type" parameters.
func (p *Parser) parseParams() []*ast.Field {
	p.expect(token.LPAREN)
	var fields []*ast.Field
	for p.tok.Type != token.RPAREN && p.tok.Type != token.EOF {
		start := p.nadv
		name := p.parseIdent()
		fields = append(fields, &ast.Field{Name: name, Type: p.parseType()})
		if !p.accept(token.COMMA) {
			break
		}
		if p.nadv == start {
			p.next()
		}
	}
	p.expect(token.RPAREN)
	return fields
}

// parseResults parses zero or more comma-separated result types appearing
// between the parameter list and the function body.
func (p *Parser) parseResults() []*ast.Field {
	if p.tok.Type == token.LBRACE || p.tok.Type == token.EOF {
		return nil
	}
	fields := []*ast.Field{{Type: p.parseType()}}
	for p.accept(token.COMMA) {
		fields = append(fields, &ast.Field{Type: p.parseType()})
	}
	return fields
}

// parseType parses a syntactic type reference: a named type or an array type.
func (p *Parser) parseType() ast.Expr {
	switch p.tok.Type {
	case token.IDENT:
		pos := p.pos()
		name := p.tok.String()
		p.next()
		return &ast.TypeName{NamePos: pos, Name: name}
	case token.LBRACK:
		return p.parseArrayType()
	default:
		pos := p.pos()
		p.errorf(pos, "expected type, found %s", p.tok.Type)
		return &ast.BadExpr{From: pos}
	}
}
