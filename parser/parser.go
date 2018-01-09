// Package parser performs token analysis on a UTF-8 io.Reader.
package parser

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/scope"
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
)

// Parser describes a parser for reading chip source files.
type Parser struct {
	level int // recursion level
	scan  *scanner.Scanner
	tok   token.Token
	alloc *ssa.Alloc
	scope *scope.Scope
	opts  options
}

// New returns a new configurable Parser with a variety of options.
func New(src io.Reader, opts ...Option) (*Parser, error) {

	// new scanner
	scan, err := scanner.New(src)
	if err != nil {
		return nil, err
	}

	// new parser
	p := &Parser{
		scan:  scan,
		tok:   token.NewEOF(),
		alloc: ssa.NewAlloc(),
		scope: scope.NewScope(),
	}

	// set options
	for _, opt := range opts {
		opt(&p.opts) // for now we don't check option errs
	}

	p.next()       // advance the parser to the first token
	p.scope.Open() // open the first scope

	return p, nil
}

// Execute starts the parser.
func (p Parser) Execute() error {
	start := time.Now()
	p.parse()
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
	fmt.Println(p.scope)
	return nil
}

// parse start parsing the next source file.
func (p *Parser) parse() { p.nextFile() }

// next advances the parser to the next token, skipping comment tokens.
func (p *Parser) next() {
	tok := p.scan.Next()
	for tok.Type == token.COMMENT {
		tok = p.scan.Next()
	}
	if tok.Type == token.ERROR {
		log.Fatal(tok)
	}
	p.tok = tok
}
