// Package check performs name resolution and type checking on a chip syntax
// tree, producing the type information consumed by later compilation.
package check

import (
	"fmt"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
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
	c.checkFuncs(file)

	return c.info, c.errs.Err()
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

func basicKind(t types.Type) types.Kind {
	if b, ok := t.(types.Basic); ok {
		return b.Kind
	}
	return types.KindInvalid
}

func isInvalid(t types.Type) bool { return basicKind(t) == types.KindInvalid && !isSlice(t) }
func isVoid(t types.Type) bool    { return basicKind(t) == types.KindVoid }
func isBool(t types.Type) bool    { return basicKind(t) == types.KindBool }
func isInt(t types.Type) bool     { return basicKind(t) == types.KindInt }
func isString(t types.Type) bool  { return basicKind(t) == types.KindString }

func isSlice(t types.Type) bool {
	_, ok := t.(types.Slice)
	return ok
}

func isNumeric(t types.Type) bool {
	k := basicKind(t)
	return k == types.KindInt || k == types.KindFloat
}

func isOrdered(t types.Type) bool {
	k := basicKind(t)
	return k == types.KindInt || k == types.KindFloat || k == types.KindString
}
