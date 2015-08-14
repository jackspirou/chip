package parser

import (
	"fmt"
	"io"
	"time"

	"github.com/jackspirou/chip/src/chip/scanner"
	"github.com/jackspirou/chip/src/chip/scope"
	"github.com/jackspirou/chip/src/chip/ssa"
	"github.com/jackspirou/chip/src/chip/token"
)

// Parser represents a parser for reading chip source files.
type Parser struct {
	scan  *scanner.Scanner
	tok   token.Token
	lit   string // literal value of a token
	alloc *ssa.Allocator
	level int // recursion level
	scope *scope.Scope
}

func NewParser(src io.Reader) *Parser {
	p := &Parser{
		scan:  scanner.NewScanner(src),
		tok:   token.NewEOF(),
		alloc: ssa.NewAllocator(),
		scope: scope.NewScope(),
	}
	p.scope.Open()
	return p
}

// Parse starts Parser parsing.
func (p *Parser) Parse() {
	p.run()
	fmt.Println(p.scope.String())
}

// run is does all the real work for the Parse method.
func (p *Parser) run() {
	start := time.Now()
	p.next()
	p.parse()
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
}

func (p *Parser) parse() { p.nextFile() }

func (p *Parser) next() token.Token {
	for tok.Type() == token.COMMENT {
		//
	}
	p.tokn = tokn
	p.tok = tokn.Typ()
	p.lit = tokn.String()
	return p.tok
}
