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
//
// Packages (packages plan §3): symbols live in one table per package, not a
// single flat namespace. The entry package streams (its body is pulled item by
// item); an imported package loads eagerly and as a unit — every file parsed,
// all signatures registered, then all bodies checked — bounded to that package,
// never a whole-program gate. Resolution is always relative to a "current"
// package (engine.cur); unqualified names resolve current → flat prelude →
// builtins, and a qualified name pkg.Member resolves in the named import.
package stream

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
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
// but defines main, main runs at end of input. Imports resolve against the
// current directory; use RunFile to root them at a program's own directory.
func Run(src io.Reader, out io.Writer) error {
	return RunWithLoader(src, out, DirLoader("."))
}

// RunFile reads and runs the program at path, resolving its imports relative to
// the file's directory (the project root, plan §3.5).
func RunFile(path string, out io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return RunWithLoader(bytes.NewReader(src), out, DirLoader(filepath.Dir(path)))
}

// RunWithLoader streams chip source from src like Run, resolving imports through
// loader. It is the seam used by tests (with an in-memory loader) and by the
// path-based entry points.
func RunWithLoader(src io.Reader, out io.Writer, loader Loader) error {
	e := newEngine(out, loader)
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
// forward references across lines do not resolve: define before use. Imports
// resolve against the current directory.
func REPL(in io.Reader, out io.Writer) error {
	e := newEngine(out, DirLoader("."))
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

// pkg is one package's symbol tables and scopes. The engine holds one per
// loaded package, plus the flat prelude and the streaming entry package.
// Resolution is always relative to a current package (engine.cur).
type pkg struct {
	name        string                   // declared package name (or alias target)
	funcs       map[string]*ast.FuncDecl // top-level function definitions
	global      *env                     // top-level variable bindings
	globalTypes *typeEnv                 // top-level variable types (for checking)
	imports     map[string]*pkg          // reference name (alias or package name) → package
}

func newPkg(name string) *pkg {
	return &pkg{
		name:        name,
		funcs:       make(map[string]*ast.FuncDecl),
		global:      newEnv(nil),
		globalTypes: newTypeEnv(nil),
		imports:     make(map[string]*pkg),
	}
}

// engine is a demand-driven streaming evaluator. Its package tables and the
// per-function signature/checked memos persist across feed calls, which is what
// lets the REPL share state.
type engine struct {
	out     io.Writer
	loader  Loader
	sigs    map[*ast.FuncDecl]*types.Signature // memoized function signatures
	checked map[*ast.FuncDecl]bool             // bodies already type-checked (D9)

	entry   *pkg // the streaming entry package (an implicit "main")
	prelude *pkg // the flat stdlib prelude (unqualified fallback, P10)
	cur     *pkg // the package whose tables resolution currently uses

	loaded    map[string]*pkg // import path → loaded package (load-once cache, P9)
	loading   map[string]bool // import paths currently loading (cycle guard)
	loadStack []string        // import paths on the loading stack (for cycle errors)
	loadDepth int             // >0 while eagerly loading an imported package

	// per-source pull state, (re)set by feed
	next       func() (ast.Node, error, bool)
	pending    []ast.Stmt      // top-level statements awaiting execution (FIFO)
	feedFuncs  map[string]bool // functions defined in the current source (duplicate check)
	ranStmt    bool            // whether any top-level statement ran (gates main-at-EOF)
	depth      int             // current call depth (recursion guard)
	streamDone bool            // the current stream reached EOF (no more read-ahead)
	printBuf   []byte          // reused scratch for building a print line (callPrint)
}

func newEngine(out io.Writer, loader Loader) *engine {
	e := &engine{
		out: out,
		// Resolve chip's built-in stdlib packages ahead of the given loader, so
		// import "math" works on every run path (files, embedding, REPL, tests).
		loader:  stdLoader{user: loader},
		sigs:    make(map[*ast.FuncDecl]*types.Signature),
		checked: make(map[*ast.FuncDecl]bool),
		entry:   newPkg("main"),
		prelude: newPkg("std"),
		loaded:  make(map[string]*pkg),
		loading: make(map[string]bool),
	}
	e.cur = e.entry
	return e
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
	e.streamDone = false
	return e.drain()
}

// loadStd feeds the standard library prelude into the prelude package so its
// functions are available, unqualified, to every program (P10). It runs before
// user code on the streaming run path.
func (e *engine) loadStd() error {
	prev := e.cur
	e.cur = e.prelude
	err := e.feed(strings.NewReader(std.Prelude))
	e.cur = prev
	return err
}

// drain is the pull loop: drain a pending statement (type-checked first, then
// trusting the checker per D10), otherwise pull the next item — registering
// definitions and queuing statements — until the source ends.
func (e *engine) drain() error {
	for {
		if len(e.pending) == 0 {
			item, perr, ok := e.next()
			if !ok {
				e.streamDone = true
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
		if _, _, err := e.execStmt(stmt, e.cur.global); err != nil {
			// The checker may have bound a top-level name whose value never
			// landed because execution failed; drop it so a later REPL line
			// stays consistent.
			if d, ok := stmt.(*ast.DeclStmt); ok {
				delete(e.cur.globalTypes.vars, d.Name.Name)
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
	main, ok := e.entry.funcs["main"]
	if !ok {
		return nil
	}
	e.cur = e.entry
	_, err := e.callFunc(main, nil, main.Pos())
	return err
}

// register records a function definition into the current package, queues a
// top-level statement, or loads an import. A function's signature is recorded
// first, then its body is type-checked (deferred per §3.1) — pulling in any
// referenced signatures and memoizing the result (D9) — so a function with a
// type error is rejected even if nothing calls it.
func (e *engine) register(item ast.Node) error {
	switch n := item.(type) {
	case *ast.ImportSpec:
		return e.registerImport(n)
	case *ast.FuncDecl:
		if n.Name != nil {
			// A source may not define the same function twice. This is
			// per-source, so the REPL (and the prelude) can redefine across feeds.
			if e.feedFuncs[n.Name.Name] {
				return TypeError{Pos: n.Name.Pos(), Msg: fmt.Sprintf("%s redeclared", n.Name.Name)}
			}
			e.feedFuncs[n.Name.Name] = true
			e.cur.funcs[n.Name.Name] = n
		}
		return e.checkFuncBody(n)
	case ast.Stmt:
		e.pending = append(e.pending, n)
	}
	return nil
}

// registerImport eagerly loads the package named by spec and binds it in the
// current package's import table under its reference name — the alias if one was
// given, otherwise the imported package's declared name (plan §3.1).
func (e *engine) registerImport(spec *ast.ImportSpec) error {
	if spec.Path == nil {
		return nil
	}
	p, err := e.load(spec.Path.Value)
	if err != nil {
		return err
	}
	name := p.name
	if spec.Name != nil {
		name = spec.Name.Name
	}
	e.cur.imports[name] = p
	return nil
}

// load eagerly loads the package at importPath as a unit and returns it, caching
// the result (load-once, P9) and guarding against import cycles. Unlike the
// entry package (which streams), an imported package is read in full: every file
// is parsed and all top-level signatures registered (collect), then every body
// is checked (check) — so cross-file and intra-package forward references
// resolve (plan §3.2). It runs no code. An imported package may contain only
// function declarations and imports; a top-level statement in one is rejected
// (there is no defined point at which it would run).
func (e *engine) load(importPath string) (*pkg, error) {
	if p, ok := e.loaded[importPath]; ok {
		return p, nil
	}
	if e.loading[importPath] {
		return nil, e.cycleError(importPath)
	}
	srcs, err := e.loader.Load(importPath)
	if err != nil {
		return nil, fmt.Errorf("cannot import %q: %w", importPath, err)
	}

	e.loading[importPath] = true
	e.loadStack = append(e.loadStack, importPath)
	e.loadDepth++
	prev := e.cur
	p := newPkg(pkgBaseName(importPath))
	e.cur = p
	defer func() {
		e.cur = prev
		e.loadDepth--
		e.loadStack = e.loadStack[:len(e.loadStack)-1]
		delete(e.loading, importPath)
	}()

	// Collect: parse every file, set the package name, load its imports, and
	// register all top-level signatures before any body is checked. Every file
	// that names a package must agree (a multi-file package is one package).
	var order []*ast.FuncDecl
	declared := ""
	for _, s := range srcs {
		f, perr := parseSource(s.Data)
		if perr != nil {
			return nil, perr
		}
		if f.Package != nil {
			if declared != "" && f.Package.Name != declared {
				return nil, TypeError{Pos: f.Package.Pos(), Msg: fmt.Sprintf("package name mismatch: %s declares package %s, want %s", s.Name, f.Package.Name, declared)}
			}
			declared = f.Package.Name
			p.name = declared
		}
		if len(f.Stmts) > 0 {
			return nil, TypeError{Pos: f.Stmts[0].Pos(), Msg: fmt.Sprintf("top-level statements are not allowed in imported package %q", p.name)}
		}
		for _, spec := range f.Imports {
			if err := e.registerImport(spec); err != nil {
				return nil, err
			}
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			if _, dup := p.funcs[fn.Name.Name]; dup {
				return nil, TypeError{Pos: fn.Name.Pos(), Msg: fmt.Sprintf("%s redeclared", fn.Name.Name)}
			}
			p.funcs[fn.Name.Name] = fn
			order = append(order, fn)
		}
	}

	// Check: now that all signatures are registered, check every body.
	for _, fn := range order {
		if err := e.checkFuncBody(fn); err != nil {
			return nil, err
		}
	}

	e.loaded[importPath] = p
	return p, nil
}

// cycleError reports an import cycle, naming the path back to the offending
// package (e.g. "import cycle: a -> b -> a").
func (e *engine) cycleError(importPath string) error {
	return fmt.Errorf("import cycle: %s", strings.Join(append(e.loadStack, importPath), " -> "))
}

// parseSource parses one package file in full (not streamed) — imported
// packages load as a unit.
func parseSource(data []byte) (*ast.File, error) {
	p, err := parser.New(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

// resolve returns the function named name and the package that owns it (its
// home, which becomes the current package while the call runs). It looks in the
// current package, then the flat prelude (P10), then — only while a source is
// streaming — reads ahead so an intra-package forward function reference
// resolves (D5). Reading ahead registers definitions and queues statements (it
// never runs them); it is skipped while eagerly loading an imported package
// (there is no stream to pull) and once the stream has ended. A nil result with
// a nil error means name is undefined; a non-nil error is a parse or load error
// encountered while reading ahead.
func (e *engine) resolve(name string) (*ast.FuncDecl, *pkg, error) {
	if fn, ok := e.cur.funcs[name]; ok {
		return fn, e.cur, nil
	}
	if fn, ok := e.prelude.funcs[name]; ok {
		return fn, e.prelude, nil
	}
	if e.loadDepth != 0 || e.streamDone {
		return nil, nil, nil
	}
	for {
		item, perr, ok := e.next()
		if perr != nil {
			return nil, nil, perr
		}
		if !ok {
			e.streamDone = true
			return nil, nil, nil // EOF: not found
		}
		if err := e.register(item); err != nil {
			return nil, nil, err
		}
		if fn, ok := e.cur.funcs[name]; ok {
			return fn, e.cur, nil
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
