package parser

import (
	"iter"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// Items returns an iterator over the file's top-level items — import specs,
// function declarations, and statements — in source order. It is the streaming
// counterpart to Parse: the demand-driven executor pulls items from it (with
// iter.Pull2) and runs them as they arrive, instead of waiting for the whole
// file to parse.
//
// The optional package clause is consumed but not yielded (the entry file's
// own name is implicit). Each import spec is yielded ahead of the code that
// might use it, so the engine can load imported packages before they are
// referenced. Each well-formed item is yielded with a nil error. The first
// parse error is yielded as (nil, err) and ends the stream. A Parser is single
// use — call either Parse or Items on a given parser, not both.
func (p *Parser) Items() iter.Seq2[ast.Node, error] {
	return func(yield func(ast.Node, error) bool) {
		p.parsePackage()
		imports := p.parseImports()
		if err := p.errs.Err(); err != nil {
			yield(nil, err)
			return
		}
		for _, spec := range imports {
			if !yield(spec, nil) {
				return
			}
		}

		for p.tok.Type != token.EOF {
			start := p.nadv
			item, _ := p.parseTopLevelSafe()
			if p.nadv == start {
				p.next() // guarantee progress on a malformed item
			}
			// A depth bailout records a parse error, so the errs.Err check below
			// ends the stream just as any other parse error would — no separate
			// handling of the bailed flag is needed here.
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
