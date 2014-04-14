package parser

import (
	"github.com/jackspirou/chip/scanner"
	"github.com/jackspirou/chip/tokens"
	"fmt"
)

// The parser structure holds the parser's internal state.
type Parser struct {
	src   io.Reader
	scanr *scanner.Scanner
  token *tokens.Token
	tok   tokens.Tokint
	toks  chan *tokens.Token
	tacs  chan *tacs.TAC
	err   error
}

func NewParser(src io.Reader) *Parser {
	p := &Parser{
		src:   src,
		scanr: scanner.NewScanner(src),
    token: *tokens.Token,
		tok:   tokens.EOF, // current Char
		toks:  make(chan *tokens.Token),
		tacs:  make(chan *tacs.TAC),
		err:   nil, // no errors yet
	}
	p.toks = p.scanr.GoScan()
	return p
}

func (p *Parser) GoParse() {
	go p.run()
	// return p.toks
}

func (p *Parser) run() {
	p.next()
	// tac := p.parse()
	p.parse()
	/*
	for tac != tacs.EOP {
		p.tacs <- tacs.NewTac(tok, lit, s.pos, s.err)
		if p.err == nil {
			tac = p.parse()
		} else {
			tac = tacs.EOP
		}
	}
	p.tacs <- tacs.NewTak(tok, lit, s.pos, s.err)
	*/
	close(p.tacs)
}

func (p *Parser) next() tokens.Tokint {
	token, ok := <-p.toks
	if !ok {
		panic("parser.next(): error with channel.")
	}
	if token.Error() != nil {
		panic("parser.next(): scanner sent parser this error: " + token.Error().Error())
	}
	p.tok = token.Int()
	return p.tok
}

func (p *Parser) parse() {
	fmt.Prinln("did it.");
}
