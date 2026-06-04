// Package stream implements chip's demand-driven streaming evaluator: source
// flows reader → scanner → parser → executor with no whole-program gate.
// Symbols (functions) resolve on demand as the stream is pulled; the instant
// one arrives, execution proceeds, and the end of the stream is the resolution
// deadline (undefined otherwise).
//
// The engine consumes parser.Items() with iter.Pull2 on a single goroutine —
// the Go call stack is the suspension when a symbol must be read ahead for (see
// the plan's §3). It is a tree-walker, not the bytecode VM (D8): there is no
// compile pass on the run path, so no whole-program gate can exist.
//
// Slice 1 runs untyped: it trusts its input and operates on runtime value
// kinds, exactly as the VM does. Incremental type checking arrives in Slice 2.
package stream

import (
	"fmt"
	"io"
	"iter"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/token"
)

// maxCallDepth bounds recursion so runaway recursion fails cleanly instead of
// exhausting the Go stack.
const maxCallDepth = 1 << 14

// Run streams chip source from src, executing top-level items as they arrive
// and writing program output to out. It is the demand-driven counterpart to the
// batch compile+VM path.
func Run(src io.Reader, out io.Writer) error {
	e, err := newEngine(src, out)
	if err != nil {
		return err
	}
	defer e.close()
	return e.run()
}

// engine is a demand-driven streaming evaluator over a pull iterator of
// top-level items.
type engine struct {
	next    func() (ast.Node, error, bool)
	stop    func()
	out     io.Writer
	funcs   map[string]*ast.FuncDecl // registered function definitions
	pending []ast.Stmt               // top-level statements awaiting execution (FIFO)
	global  *env                     // top-level variable bindings
	depth   int                      // current call depth (recursion guard)
}

func newEngine(src io.Reader, out io.Writer) (*engine, error) {
	p, err := parser.New(src)
	if err != nil {
		return nil, err
	}
	next, stop := iter.Pull2(p.Items())
	return &engine{
		next:   next,
		stop:   stop,
		out:    out,
		funcs:  make(map[string]*ast.FuncDecl),
		global: newEnv(nil),
	}, nil
}

// close releases the engine's pull iterator.
func (e *engine) close() { e.stop() }

// run is the pull loop: drain a pending statement, otherwise pull the next item
// — registering definitions and queuing statements — until the stream ends. If
// no top-level statement ran and main is defined, call main at EOF, so a
// batch-style program (only declarations) still runs.
func (e *engine) run() error {
	ranStmt := false
	for {
		if len(e.pending) == 0 {
			item, perr, ok := e.next()
			if !ok {
				break // EOF
			}
			if perr != nil {
				return perr
			}
			e.register(item)
			continue
		}
		stmt := e.pending[0]
		e.pending = e.pending[1:]
		ranStmt = true
		if _, _, err := e.execStmt(stmt, e.global); err != nil {
			return err
		}
	}

	if !ranStmt {
		if main, ok := e.funcs["main"]; ok {
			if _, err := e.callFunc(main, nil, main.Pos()); err != nil {
				return err
			}
		}
	}
	return nil
}

// register records a function definition or queues a top-level statement.
func (e *engine) register(item ast.Node) {
	switch n := item.(type) {
	case *ast.FuncDecl:
		if n.Name != nil {
			e.funcs[n.Name.Name] = n
		}
	case ast.Stmt:
		e.pending = append(e.pending, n)
	}
}

// resolve returns the function named name, reading ahead in the stream until it
// is registered or the stream ends. Reading ahead registers definitions and
// queues statements (it never runs them) — so a forward function reference
// resolves (D5), but a forward variable reference cannot. A nil result with a
// nil error means the stream ended without defining name (the caller reports
// "undefined"); a non-nil error is a parse error encountered while reading
// ahead.
func (e *engine) resolve(name string) (*ast.FuncDecl, error) {
	if fn, ok := e.funcs[name]; ok {
		return fn, nil
	}
	for {
		item, perr, ok := e.next()
		if perr != nil {
			return nil, perr
		}
		if !ok {
			return nil, nil // EOF: not found
		}
		e.register(item)
		if fn, ok := e.funcs[name]; ok {
			return fn, nil
		}
	}
}

// RuntimeError is an error raised while streaming-executing a program, carrying
// the source position of the offending node.
type RuntimeError struct {
	Pos token.Pos
	Msg string
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// errorf builds a positioned RuntimeError.
func (e *engine) errorf(pos token.Pos, format string, args ...any) error {
	return RuntimeError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}
