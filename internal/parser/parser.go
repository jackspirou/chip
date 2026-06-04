// Package parser builds an abstract syntax tree from chip source.
package parser

import (
	"fmt"
	"io"
	"os"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/scanner"
	"github.com/jackspirou/chip/internal/token"
)

// Parser parses chip source into an *ast.File.
type Parser struct {
	scan     *scanner.Scanner
	tok      token.Token // current token
	nadv     int         // number of tokens advanced (progress counter)
	comments []*ast.Comment
	errs     ErrorList
	opts     options
}

// New returns a Parser that reads chip source from src.
func New(src io.Reader, opts ...Option) (*Parser, error) {
	scan, err := scanner.New(src)
	if err != nil {
		return nil, err
	}

	p := &Parser{scan: scan, tok: token.NewEOF()}
	for _, opt := range opts {
		opt(&p.opts)
	}

	p.next() // load the first token
	return p, nil
}

// Parse parses the source and returns the file syntax tree. A non-nil error is
// an ErrorList holding every parse error found.
func (p *Parser) Parse() (*ast.File, error) {
	file := p.parseFile()
	return file, p.errs.Err()
}

// next advances to the next non-comment token. A scanner error is recorded and
// treated as end of input so parsing terminates cleanly.
func (p *Parser) next() {
	tok := p.scan.Next()
	for tok.Type == token.COMMENT {
		p.comments = append(p.comments, &ast.Comment{Slash: tok.Pos(), Text: tok.String()})
		tok = p.scan.Next()
	}
	if tok.Type == token.ERROR {
		p.error(tok.Pos(), tok.String())
		tok = token.New(token.EOF, "EOF", tok.Pos())
	}
	if p.opts.trace {
		fmt.Fprintf(os.Stderr, "%d:%d\t%s\t%q\n", tok.Line(), tok.Column(), tok.Type, tok.String())
	}
	p.tok = tok
	p.nadv++
}

// pos returns the position of the current token.
func (p *Parser) pos() token.Pos { return p.tok.Pos() }

// error records a parse error at pos.
func (p *Parser) error(pos token.Pos, msg string) {
	p.errs = append(p.errs, Error{Pos: pos, Msg: msg})
}

// errorf records a formatted parse error at pos.
func (p *Parser) errorf(pos token.Pos, format string, args ...any) {
	p.error(pos, fmt.Sprintf(format, args...))
}

// expect consumes the current token if it matches t, otherwise records an
// error without consuming. It returns the position of the expected token.
func (p *Parser) expect(t token.Type) token.Pos {
	pos := p.pos()
	if p.tok.Type != t {
		p.errorf(pos, "expected %s, found %s", t, p.tok.Type)
		return pos
	}
	p.next()
	return pos
}

// accept consumes the current token if it matches t and reports whether it did.
func (p *Parser) accept(t token.Type) bool {
	if p.tok.Type == t {
		p.next()
		return true
	}
	return false
}
