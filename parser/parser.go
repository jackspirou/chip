package parser

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/scope"
	"github.com/jackspirou/chip/ssa"
	"github.com/jackspirou/chip/token"
)

// Parser represents a parser for reading chip source files.
type Parser struct {
	scan    *scanner.Scanner
	tok     token.Token
	alloc   *ssa.Allocator
	level   int // recursion level
	scope   *scope.Scope
	Tracing bool
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
		alloc: ssa.NewAllocator(),
		scope: scope.NewScope(),
	}

	p.next()
	p.scope.Open()

	return p, nil
}

// Parse starts Parser parsing.
func (p *Parser) Parse() error {
	start := time.Now()
	if err := p.parse(); err != nil {
		return err
	}
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
	return nil
}

func (p *Parser) parse() error { return p.nextFile() }

func (p *Parser) next() error {
	tok := p.scan.Scan()
	for tok.Type == token.COMMENT {
		tok = p.scan.Scan()
	}
	if tok.Type == token.ERROR {
		return errors.New(tok.String())
	}
	p.tok = tok
	return nil
}
