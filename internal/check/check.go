// Package check performs name resolution and type checking on a chip syntax
// tree, producing the type information consumed by later compilation.
package check

import (
	"fmt"
	"path"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/typerules"
	"github.com/jackspirou/chip/internal/types"
)

// Info records the results of type checking.
type Info struct {
	Types map[ast.Expr]types.Type      // the type of every checked expression
	Defs  map[*ast.Ident]*scope.Symbol // identifiers that declare a symbol
	Uses  map[*ast.Ident]*scope.Symbol // identifiers that refer to a symbol
}

// Checker holds the state of a single type-checking run.
type Checker struct {
	info     *Info
	universe *scope.Scope     // predeclared builtins
	global   *scope.Scope     // top-level declarations
	file     *scope.Scope     // top-level statement bindings (redefinable, like the REPL)
	scope    *scope.Scope     // current scope while checking a body
	sig      *types.Signature // signature of the function being checked
	errs     ErrorList
}

// Check resolves names and checks types in file. It returns the collected
// information and an error (an ErrorList) that is nil when file is well-typed.
func Check(file *ast.File) (*Info, error) {
	c := &Checker{
		info: &Info{
			Types: make(map[ast.Expr]types.Type),
			Defs:  make(map[*ast.Ident]*scope.Symbol),
			Uses:  make(map[*ast.Ident]*scope.Symbol),
		},
	}
	c.universe = scope.New(nil)
	c.defineBuiltins()
	c.global = scope.New(c.universe)

	c.collect(file)
	c.collectImports(file)
	c.checkFuncs(file)
	c.checkTopStmts(file)

	return c.info, c.errs.Err()
}

// checkTopStmts checks the program's top-level statements in a scope nested in
// the global declarations, so they may reference top-level functions, imports,
// and earlier top-level bindings. The streaming runtime checks these per
// statement just before each runs (and the batch compiler ignores them); this
// pass lets whole-program tooling — chip check, lint — see top-level errors too.
//
// Like the streaming runtime (and the REPL), a top-level := may rebind a name,
// so a redefinition overwrites rather than reporting a redeclaration.
func (c *Checker) checkTopStmts(file *ast.File) {
	if len(file.Stmts) == 0 {
		return
	}
	c.file = scope.New(c.global)
	c.scope, c.sig = c.file, nil
	for _, s := range file.Stmts {
		c.checkStmt(s)
	}
	c.scope = nil
}

// collectImports registers each imported package's reference name (its alias,
// else the last element of the import path) so qualified references resolve.
//
// The batch checker has no Loader, so it does not type cross-package members: a
// package name, and any qualified call through it, is opaque (types.Invalid).
// This is a deliberate boundary — real package resolution and execution happen
// on the streaming path (chip run, internal/stream); the batch tooling only
// needs enough awareness to avoid spurious "undefined" errors so chip lint and
// chip ast work on programs with imports.
func (c *Checker) collectImports(file *ast.File) {
	for _, spec := range file.Imports {
		name := importRefName(spec)
		if name == "" || c.global.Lookup(name) != nil {
			continue
		}
		c.global.Insert(&scope.Symbol{
			Name:    name,
			Kind:    scope.Pkg,
			Type:    types.Invalid,
			DeclPos: spec.Pos(),
		})
	}
}

// importRefName is the name an import binds: its alias if given, else the last
// element of the import path.
func importRefName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	if spec.Path != nil {
		return path.Base(spec.Path.Value)
	}
	return ""
}

func (c *Checker) defineBuiltins() {
	for _, name := range []string{"print", "len"} {
		c.universe.Insert(&scope.Symbol{Name: name, Kind: scope.Builtin, Type: types.Invalid})
	}
}

// collect declares every top-level function up front so that bodies may
// reference functions regardless of declaration order (and recurse).
func (c *Checker) collect(file *ast.File) {
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		sym := &scope.Symbol{
			Name:    fn.Name.Name,
			Kind:    scope.Func,
			Type:    c.signature(fn),
			DeclPos: fn.Name.Pos(),
		}
		if prev := c.global.Insert(sym); prev != nil {
			c.errorf(fn.Name.Pos(), "%s redeclared", fn.Name.Name)
		}
		c.info.Defs[fn.Name] = sym
	}
}

// signature builds the function type for fn, resolving its parameter and result
// type names.
func (c *Checker) signature(fn *ast.FuncDecl) *types.Signature {
	sig := &types.Signature{}
	for _, f := range fn.Params {
		sig.Params = append(sig.Params, c.resolveType(f.Type))
	}
	if len(fn.Results) > 0 {
		if len(fn.Results) > 1 {
			c.errorf(fn.Results[1].Pos(), "multiple return values are not supported yet")
		}
		sig.Result = c.resolveType(fn.Results[0].Type)
	}
	return sig
}

// resolveType maps a syntactic type reference to a types.Type.
func (c *Checker) resolveType(e ast.Expr) types.Type {
	switch e := e.(type) {
	case *ast.TypeName:
		if b, ok := types.Lookup(e.Name); ok {
			return b
		}
		c.errorf(e.Pos(), "undefined type: %s", e.Name)
		return types.Invalid
	case *ast.ArrayType:
		if e.Len != nil {
			c.errorf(e.Lbrack, "fixed-size arrays are not supported yet")
		}
		return types.Slice{Elem: c.resolveType(e.Elem)}
	default:
		return types.Invalid
	}
}

// checkFuncs checks the body of every function.
func (c *Checker) checkFuncs(file *ast.File) {
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			c.checkFunc(fn)
		}
	}
}

func (c *Checker) checkFunc(fn *ast.FuncDecl) {
	sym := c.global.Lookup(fn.Name.Name)
	if sym == nil {
		return
	}
	sig, ok := sym.Type.(*types.Signature)
	if !ok {
		return
	}

	// main must take no arguments, and a function with a declared result must
	// return on every path. The streaming checker (chip run) rejects both; this
	// batch checker (chip check) must reject them too, or it would pass code that
	// run refuses — the rules are shared with run via internal/typerules so they
	// cannot drift. A missing return is reported at the closing brace, where
	// control would fall off the end.
	if typerules.MainTakesArgs(fn) {
		c.errorf(fn.Name.Pos(), "main must take no arguments")
	}
	if sig.Result != nil && !typerules.Terminates(fn.Body.List) {
		c.errorf(fn.Body.Rbrace, "missing return at end of function %s", fn.Name.Name)
	}

	fnScope := scope.New(c.global)
	for i, f := range fn.Params {
		if f.Name == nil {
			continue
		}
		psym := &scope.Symbol{Name: f.Name.Name, Kind: scope.Param, Type: sig.Params[i], DeclPos: f.Name.Pos()}
		if prev := fnScope.Insert(psym); prev != nil {
			c.errorf(f.Name.Pos(), "duplicate parameter %s", f.Name.Name)
		}
		c.info.Defs[f.Name] = psym
	}

	savedScope, savedSig := c.scope, c.sig
	c.scope, c.sig = fnScope, sig
	for _, s := range fn.Body.List {
		c.checkStmt(s)
	}
	c.scope, c.sig = savedScope, savedSig
}

// lookup resolves name in the current scope chain.
func (c *Checker) lookup(name string) *scope.Symbol {
	return c.scope.LookupParent(name)
}

func (c *Checker) errorf(pos token.Pos, format string, args ...any) {
	c.errs = append(c.errs, Error{Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

//
// type predicates
//
// These forward to internal/typerules so the streaming checker (chip run) and
// this batch checker (chip check) share one definition of every rule and cannot
// drift. The local names keep the call sites in this package readable.

func isInvalid(t types.Type) bool { return typerules.IsInvalid(t) }
func isVoid(t types.Type) bool    { return typerules.IsVoid(t) }
func isBool(t types.Type) bool    { return typerules.IsBool(t) }
func isInt(t types.Type) bool     { return typerules.IsInt(t) }
func isString(t types.Type) bool  { return typerules.IsString(t) }
func isSlice(t types.Type) bool   { return typerules.IsSlice(t) }
