package parser

import (
	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/token"
	"github.com/jackspirou/chip/tacs"
	"io"
	"fmt"
)

// The parser structure holds the parser's internal state.
type Parser struct {
	src   io.Reader
	scanr *scanner.Scanner
	tok   *token.Tok
	toks  chan *token.Tok
	tacs  chan *tacs.Tac
	err   error
}

func NewParser(src io.Reader) *Parser {
	p := &Parser{
		src:   src,
		scanr: scanner.NewScanner(src),
    tok:   token.NewEndTok(),
		toks:  make(chan *token.Tok),
		tacs:  make(chan *tacs.Tac),
		err:   nil, // no errors yet
	}
	p.toks = p.scanr.GoScan()
	return p
}

func (p *Parser) GoParse() {
	p.parse()
	go p.run()
	// return p.toks
}

func (p *Parser) run() {
	tok := p.next()
	for (tok.Typ() != token.EOF) {
		fmt.Println(tok)
		tok = p.next()
	}
	fmt.Println(tok)
	close(p.tacs)
}

func (p *Parser) next() *token.Tok {
	tok, ok := <-p.toks
	if !ok {
		panic("parser.next(): error with channel.")
	}
	if tok.Typ() == token.ERROR {
		panic("parser.next(): scanner sent parser this error: " + tok.String())
	}
	p.tok = tok
	return p.tok
}

func (p *Parser) parse() {
	fmt.Println("did it.");
}
