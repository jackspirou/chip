// Package parser performs token analysis on a UTF-8 io.Reader.
package parser

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/scope"
	"github.com/jackspirou/chip/token"
)

// Parser describes a parser for reading chip source files.
type Parser struct {
	level int              // recursion level
	scan  *scanner.Scanner // token scanner
	tok   token.Token      // current token
	scope *scope.Scope     // scope
	opts  options          // option settings
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
		scope: scope.New(),
	}

	// set options
	for _, opt := range opts {
		opt(&p.opts) // for now we don't check option errs
	}

	p.next()       // advance the parser to the first token
	p.scope.Open() // open the first scope

	return p, nil
}

// Parse starts the parser.
func (p Parser) Parse() error {
	start := time.Now()
	p.parse()
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
	// fmt.Println(p.scope)
	return nil
}

// parse starts parsing the next source file.
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

// nextExpected expects the next token to match the token.Type provided.
func (p *Parser) nextExpected(expected token.Type) {
	if p.tok.Type == expected {
		p.next()
		return
	}
	log.Fatalf("expected '%s', got '%s'", expected, p.tok)
}
