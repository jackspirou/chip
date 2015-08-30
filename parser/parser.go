// Package parser performs token analysis on a UTF-8 io.Reader.
package parser

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jackspirou/chip/parser/scanner"
	"github.com/jackspirou/chip/parser/token"
	"github.com/jackspirou/chip/scope"
	"github.com/jackspirou/chip/ssa"
)

// Parser describes a parser for reading chip source files.
type Parser struct {
	Tracing bool
	level   int // recursion level
	scan    *scanner.Scanner
	tok     token.Token
	alloc   *ssa.Alloc
	scope   *scope.Scope
	tbv     *scope.TBV
}

// New returns a new Parser object.
func New(src io.Reader) (*Parser, error) {
	scan, err := scanner.New(src)
	if err != nil {
		return nil, err
	}

	p := &Parser{
		scan:  scan,
		tok:   token.NewEOF(),
		alloc: ssa.NewAlloc(),
		scope: scope.NewScope(),
		tbv:   scope.NewTBV(),
	}

	p.next()
	p.scope.Open()

	return p, nil
}

// Parse starts Parser parsing.
func (p *Parser) Parse() {
	start := time.Now()
	p.parse()
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
	fmt.Println(p.scope)
}

// parse start parsing the next source file.
func (p *Parser) parse() { p.nextFile() }

// next advances the parser to the next token, skipping comment tokens.
func (p *Parser) next() {
	tok := p.scan.Scan()
	for tok.Type == token.COMMENT {
		tok = p.scan.Scan()
	}
	if tok.Type == token.ERROR {
		log.Fatal(tok)
	}
	p.tok = tok
}
