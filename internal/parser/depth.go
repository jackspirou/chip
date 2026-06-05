package parser

import "github.com/jackspirou/chip/internal/ast"

// maxDepth bounds how deeply the recursive-descent parser will nest. It guards
// against a parse-time stack overflow on pathological input — a few million
// nested parens, unary operators, array brackets, or if/else-if clauses — which
// is a fatal, unrecoverable crash (Go aborts the process on goroutine stack
// exhaustion; recover cannot catch it) that also takes down any embedding host.
//
// No execution cap can stop it: --max-steps, --timeout, and --max-output are
// enforced inside the execution step loop, but scanning and parsing run ahead of
// that loop, so a malformed program can exhaust the stack before a single step
// is taken. chip accepts streamed, untrusted source, so the front end has to
// defend itself.
//
// The limit is far beyond any real program (genuine nesting is tens of levels,
// not thousands) and far below the depth that overflows the goroutine stack
// (empirically ~2,000,000 nested parens at the default 1 GB limit; 10,000 levels
// costs single-digit megabytes), so it rejects only abuse and never a real file.
const maxDepth = 10000

// bailout is the value panicked when the parser exceeds maxDepth. It is recovered
// in parseTopLevelSafe, turning a would-be stack overflow into a single recorded
// parse error. A distinct unexported type lets the recover tell our controlled
// unwind apart from a genuine bug, which it re-panics.
type bailout struct{}

// enter records one level of recursive-descent nesting and trips the depth guard
// when the nesting passes maxDepth. It is paired with leave via
//
//	p.enter()
//	defer p.leave()
//
// at the head of every function on a recursion cycle that input length can drive
// without bound: parseUnaryExpr (expression nesting), parseArrayType (array-type
// nesting), parseStmt (block-nested statements), and parseIf (else-if chains).
// Tripping records a parse error at the current position and panics bailout,
// which unwinds to parseTopLevelSafe.
func (p *Parser) enter() {
	p.depth++
	if p.depth > maxDepth {
		p.errorf(p.pos(), "nesting too deep")
		panic(bailout{})
	}
}

// leave undoes one enter. Deferred, so it also runs as the stack unwinds during a
// bailout, leaving the depth counter consistent.
func (p *Parser) leave() { p.depth-- }

// parseTopLevelSafe parses one top-level item, recovering a depth bailout so an
// over-nested construct becomes one recorded error instead of a crashed process.
// It is the single recover site for both parse drivers — Parse (batch) and Items
// (streaming) — since every item, in either, is parsed through here.
//
// bailed reports whether the depth guard tripped. The batch driver uses it to
// stop after the offending item rather than re-erroring on every leftover token;
// the streaming driver stops on the recorded error either way. A panic that is
// not a bailout is a real bug and is re-panicked unchanged.
func (p *Parser) parseTopLevelSafe() (node ast.Node, bailed bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r) // not our controlled unwind — a genuine bug
			}
			node, bailed = &ast.BadStmt{From: p.pos()}, true
		}
	}()
	return p.parseTopLevel(), false
}
