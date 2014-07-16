package parser

import (
	"fmt"
	"github.com/JackSpirou/chip/scanner"
	"github.com/JackSpirou/chip/scope"
	"github.com/JackSpirou/chip/ssa"
	"github.com/JackSpirou/chip/token"
	"io"
	"time"
)

// The parser structure holds the parser's internal state.
type Parser struct {
	src   io.Reader
	scanr *scanner.Scanner
	tokn  *token.Tok
	tok   token.Tokint
	lit   string
	toks  chan *token.Tok
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
		toks:  make(chan *token.Tok),
		alloc: ssa.NewAllocator(),
		// tacs:  make(chan *tacs.Tac),
		err:   nil, // no errors yet
		level: 0,
		scope: scope.NewScope(),
	}
	p.toks = p.scanr.GoScan()
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
