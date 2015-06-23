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

	// input source
	src io.Reader

	// file scanner
	scanr *scanner.Scanner

	//
	tokn  *token.Tok
	tok   token.Tokint
	lit   string
	alloc *ssa.Allocator
	// tacs  chan *tacs.Tac
	err   error
	level int //  Parser recursion level.
	scope *scope.Scope
}

func NewParser(src io.Reader) *Parser {
	p := &Parser{
		src:   src,
		scanr: scanner.NewScanner(src),
		tokn:  token.NewEndTok(),
		tok:   0,
		alloc: ssa.NewAllocator(),
		// tacs:  make(chan *tacs.Tac),
		err:   nil, // no errors yet
		level: 0,
		scope: scope.NewScope(),
	}
	p.scope.Open()
	return p
}

func (p *Parser) GoParse() {
	go p.run()
	time.Sleep(3 * time.Second)
	fmt.Println(p.scope.String())
	// return p.toks
}

func (p *Parser) run() {
	start := time.Now()
	p.next()
	p.parse()
	// close(p.toks)
	end := time.Now()
	duration := end.Sub(start)
	fmt.Println(duration)
}

func (p *Parser) parse() {
	p.nextFile()
}

func (p *Parser) next() token.Tokint {
	tokn, ok := <-p.toks
	if !ok {
		panic("parser.next(): error with channel.")
	}
	for tokn.Typ() == token.COMMENT {
		tokn, ok = <-p.toks
		if !ok {
			panic("parser.next(): error with channel.")
		}
	}
	p.tokn = tokn
	p.tok = tokn.Typ()
	p.lit = tokn.String()
	return p.tok
}
