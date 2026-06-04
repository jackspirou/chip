// Package stream implements chip's demand-driven streaming evaluator: source
// flows reader → scanner → parser → executor with no whole-program gate.
// Symbols resolve on demand as the stream is pulled; the instant one arrives,
// execution proceeds, and the end of the stream is the resolution deadline.
//
// The engine consumes parser.Items() with iter.Pull2 on a single goroutine —
// the Go call stack is the suspension when a symbol must be awaited (see the
// plan's §3.1). This file is the Slice 0 scaffolding: it wires up the pull
// driver. The symbol tables, the pending FIFO, the resolve seam, and the
// tree-walking executor arrive in later slices.
package stream

import (
	"io"
	"iter"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/parser"
)

// engine drives a demand-driven stream of top-level items pulled from a parser.
type engine struct {
	next func() (ast.Node, error, bool)
	stop func()
}

// newEngine builds an engine that pulls the top-level items of src on demand.
// The caller must call (*engine).close to release the pull iterator.
func newEngine(src io.Reader) (*engine, error) {
	p, err := parser.New(src)
	if err != nil {
		return nil, err
	}
	next, stop := iter.Pull2(p.Items())
	return &engine{next: next, stop: stop}, nil
}

// close releases the engine's pull iterator.
func (e *engine) close() { e.stop() }

// drain pulls every remaining top-level item, in source order, stopping at the
// first parse error. It exercises the pull driver end to end; the streaming
// executor (Slice 1) instead pulls items one at a time as execution demands
// them.
func (e *engine) drain() ([]ast.Node, error) {
	var items []ast.Node
	for {
		item, err, ok := e.next()
		if !ok {
			return items, nil
		}
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
}
