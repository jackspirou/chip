// Package stream implements chip's demand-driven streaming evaluator: source
// flows reader → scanner → parser → executor with no whole-program gate.
// Symbols (functions) resolve on demand as the stream is pulled; the instant
// one arrives, execution proceeds, and the end of the stream is the resolution
// deadline (undefined otherwise).
//
// The engine consumes parser.Items() with iter.Pull2 on a single goroutine —
// the Go call stack is the suspension when a symbol must be read ahead for (see
// the plan's §3). It is a checked tree-walker, not the bytecode VM (D8): each
// function body is type-checked when it registers and each top-level statement
// just before it runs, so there is no compile pass and no whole-program gate.
// The engine's tables persist across sources, so a whole file (Run) and a
// line-by-line session (REPL) share one implementation.
package stream

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/std"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/types"
)

// maxCallDepth bounds recursion so runaway recursion fails cleanly instead of
// exhausting the Go stack.
const maxCallDepth = 1 << 14

// Run streams chip source from src, executing top-level items as they arrive
// and writing program output to out. If the source has no top-level statements
// but defines main, main runs at end of input.
func Run(src io.Reader, out io.Writer) error {
	e := newEngine(out)
	if err := e.loadStd(); err != nil {
		return err
	}
	if err := e.feed(src); err != nil {
		return err
	}
	return e.runMain()
}

// REPL evaluates chip source read line by line from in against a persistent
// engine — definitions and top-level variables carry over between lines — and
// writes program output and errors to out. Each line is its own source, so
// forward references across lines do not resolve: define before use.
func REPL(in io.Reader, out io.Writer) error {
	e := newEngine(out)
	if err := e.loadStd(); err != nil {
		return err
	}
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		if err := e.feed(strings.NewReader(sc.Text())); err != nil {
			fmt.Fprintln(out, err)
		}
	}
	return sc.Err()
}

// engine is a demand-driven streaming evaluator. Its symbol tables and global
// scope persist across feed calls, which is what lets the REPL share state.
type engine struct {
	out         io.Writer
	funcs       map[string]*ast.FuncDecl           // registered function definitions
	sigs        map[*ast.FuncDecl]*types.Signature // memoized function signatures
	checked     map[*ast.FuncDecl]bool             // bodies already type-checked (D9)
	global      *env                               // top-level variable bindings
	globalTypes *typeEnv                           // top-level variable types (for checking)

	// per-source pull state, (re)set by feed
	next      func() (ast.Node, error, bool)
	pending   []ast.Stmt      // top-level statements awaiting execution (FIFO)
	feedFuncs map[string]bool // functions defined in the current source (duplicate check)
	ranStmt   bool            // whether any top-level statement ran (gates main-at-EOF)
	depth     int             // current call depth (recursion guard)
}

func newEngine(out io.Writer) *engine {
	return &engine{
		out:         out,
		funcs:       make(map[string]*ast.FuncDecl),
		sigs:        make(map[*ast.FuncDecl]*types.Signature),
		checked:     make(map[*ast.FuncDecl]bool),
		global:      newEnv(nil),
		globalTypes: newTypeEnv(nil),
	}
}

// feed streams one source through the engine, sharing accumulated state. It
// processes every item — registering definitions, running top-level statements
// in order — until the source ends. It does not run main; Run does that once,
// after the whole file is consumed.
func (e *engine) feed(src io.Reader) error {
	p, err := parser.New(src)
	if err != nil {
		return err
	}
	next, stop := iter.Pull2(p.Items())
	defer stop()
	e.next = next
	e.pending = nil
	e.feedFuncs = map[string]bool{}
	e.ranStmt = false
	return e.drain()
}

// loadStd feeds the standard library prelude so its functions are available to
// every program. It runs before user code on the streaming run path.
func (e *engine) loadStd() error {
	return e.feed(strings.NewReader(std.Prelude))
}

// drain is the pull loop: drain a pending statement (type-checked first, then
// trusting the checker per D10), otherwise pull the next item — registering
// definitions and queuing statements — until the source ends.
func (e *engine) drain() error {
	for {
		if len(e.pending) == 0 {
			item, perr, ok := e.next()
			if !ok {
				return nil // source exhausted
			}
			if perr != nil {
				return perr
			}
			if err := e.register(item); err != nil {
				return err
			}
			continue
		}
		stmt := e.pending[0]
		e.pending = e.pending[1:]
		e.ranStmt = true
		if err := e.checkTopStmt(stmt); err != nil {
			return err
		}
		if _, _, err := e.execStmt(stmt, e.global); err != nil {
			// The checker may have bound a top-level name whose value never
			// landed because execution failed; drop it so a later REPL line
			// stays consistent.
			if d, ok := stmt.(*ast.DeclStmt); ok {
				delete(e.globalTypes.vars, d.Name.Name)
			}
			return err
		}
	}
}

// runMain calls main if it is defined and no top-level statement ran, so a
// batch-style program (only declarations) still executes.
func (e *engine) runMain() error {
	if e.ranStmt {
		return nil
	}
	main, ok := e.funcs["main"]
	if !ok {
		return nil
	}
	_, err := e.callFunc(main, nil, main.Pos())
	return err
}

// register records a function definition or queues a top-level statement. A
// function's signature is recorded first, then its body is type-checked
// (deferred per §3.1) — pulling in any referenced signatures and memoizing the
// result (D9) — so a function with a type error is rejected even if nothing
// calls it.
func (e *engine) register(item ast.Node) error {
	switch n := item.(type) {
	case *ast.FuncDecl:
		if n.Name != nil {
			// A source may not define the same function twice. This is
			// per-source, so the REPL (and the prelude) can redefine across feeds.
			if e.feedFuncs[n.Name.Name] {
				return TypeError{Pos: n.Name.Pos(), Msg: fmt.Sprintf("%s redeclared", n.Name.Name)}
			}
			e.feedFuncs[n.Name.Name] = true
			e.funcs[n.Name.Name] = n
		}
		return e.checkFuncBody(n)
	case ast.Stmt:
		e.pending = append(e.pending, n)
	}
	return nil
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
		if err := e.register(item); err != nil {
			return nil, err
		}
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
