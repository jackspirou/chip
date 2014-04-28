package parser

import (
	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/token"
	"io"
	"fmt"
)

// The parser structure holds the parser's internal state.
type Parser struct {
	src   io.Reader
	scanr *scanner.Scanner
	token *token.Tok
	tok   token.Tokint
	lit	 string
	toks  chan *token.Tok
	tacs  chan *tacs.Tac
	err   error
	level int //  Parser recursion level.
}

func NewParser(src io.Reader) *Parser {
	p := &Parser{
		src:   src,
		scanr: scanner.NewScanner(src),
		token: token.NewEndTok(),
    tok:   0,
		toks:  make(chan *token.Tok),
		tacs:  make(chan *tacs.Tac),
		err:   nil, // no errors yet
		level: 0,
	}
	p.toks = p.scanr.GoScan()
	return p
}

func (p *Parser) GoParse() {
	go p.run()
	return p.toks
}

func (p *Parser) run() {
	p.next()
	p.parse()
	close(s.toks)
}

func (p *Parser) parse() {
	p.nextFile();
}

func (p *Parser) next() token.Tokint {
	token, ok := <-p.toks
	if !ok {
		panic("parser.next(): error with channel.")
	}
	p.token = token
	p.tok = token.Typ()
	p.lit = token.String()
	return p.tok
}
