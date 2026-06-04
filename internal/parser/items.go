package parser

import (
	"iter"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// Items returns an iterator over the file's top-level items — function
// declarations and statements — in source order. It is the streaming
// counterpart to Parse: the demand-driven executor pulls items from it (with
// iter.Pull2) and runs them as they arrive, instead of waiting for the whole
// file to parse.
//
// Each well-formed item is yielded with a nil error. The first parse error is
// yielded as (nil, err) and ends the stream. A Parser is single use — call
// either Parse or Items on a given parser, not both.
func (p *Parser) Items() iter.Seq2[ast.Node, error] {
	return func(yield func(ast.Node, error) bool) {
		p.parsePackage()
		p.parseImports()
		if err := p.errs.Err(); err != nil {
			yield(nil, err)
			return
		}

		for p.tok.Type != token.EOF {
			start := p.nadv
			item := p.parseTopLevel()
			if p.nadv == start {
				p.next() // guarantee progress on a malformed item
			}
			if err := p.errs.Err(); err != nil {
				yield(nil, err)
				return
			}
			if !yield(item, nil) {
				return
			}
		}
	}
}
