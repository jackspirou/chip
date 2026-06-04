package stream

import (
	"fmt"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/types"
)

// This file is the streaming type checker: incremental, demand-driven, and
// fail-fast. It mirrors the rules of the batch internal/check package but runs
// per function body (at registration) and per top-level statement (right before
// it executes), resolving forward function references on demand via
// engine.resolve (D9). The batch checker stays whole-program tooling (D1); the
// executor then trusts these results — it carries no runtime type guards (D10).

// TypeError is a static type error surfaced during a streaming run.
type TypeError struct {
	Pos token.Pos
	Msg string
}

func (e TypeError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// typeEnv mirrors env but binds names to their static types for checking.
type typeEnv struct {
	vars   map[string]types.Type
	parent *typeEnv
}

func newTypeEnv(parent *typeEnv) *typeEnv {
	return &typeEnv{vars: make(map[string]types.Type), parent: parent}
}

func (t *typeEnv) child() *typeEnv { return newTypeEnv(t) }

func (t *typeEnv) lookup(name string) (types.Type, bool) {
	for s := t; s != nil; s = s.parent {
		if ty, ok := s.vars[name]; ok {
			return ty, true
		}
	}
	return nil, false
}

func (t *typeEnv) define(name string, ty types.Type) { t.vars[name] = ty }

// sigOf returns fn's signature, resolving its parameter and result type names.
// The result is memoized so recursive references reuse it.
func (e *engine) sigOf(fn *ast.FuncDecl) *types.Signature {
	if sig, ok := e.sigs[fn]; ok {
		return sig
	}
	sig := &types.Signature{}
	for _, f := range fn.Params {
		sig.Params = append(sig.Params, resolveType(f.Type))
	}
	if len(fn.Results) > 0 {
		sig.Result = resolveType(fn.Results[0].Type)
	}
	e.sigs[fn] = sig
	return sig
}

// resolveType maps a syntactic type reference to a types.Type.
func resolveType(e ast.Expr) types.Type {
	switch e := e.(type) {
	case *ast.TypeName:
		if b, ok := types.Lookup(e.Name); ok {
			return b
		}
		return types.Invalid
	case *ast.ArrayType:
		return types.Slice{Elem: resolveType(e.Elem)}
	default:
		return types.Invalid
	}
}

// checkFuncBody type-checks fn's body once (memoized, D9). It is called when a
// function is registered, so a function with a type error is rejected even if
// nothing ever calls it. Referenced functions are pulled in on demand.
func (e *engine) checkFuncBody(fn *ast.FuncDecl) error {
	if fn == nil || e.checked[fn] {
		return nil
	}
	e.checked[fn] = true // mark first so mutual recursion terminates

	tc := &typeChecker{e: e}
	if fn.Name != nil && fn.Name.Name == "main" && len(fn.Params) > 0 {
		return tc.errorf(fn.Name.Pos(), "main must take no arguments")
	}
	// Validate parameter and result types up front: an unknown type must be an
	// error, never silently treated as invalid (which would disable checks).
	for _, p := range fn.Params {
		if err := tc.checkTypeExpr(p.Type); err != nil {
			return err
		}
	}
	for _, r := range fn.Results {
		if err := tc.checkTypeExpr(r.Type); err != nil {
			return err
		}
	}

	sig := e.sigOf(fn)
	tc.result = sig.Result
	// A function with a declared result must return on every path, or it would
	// fall off the end and yield a type-confused zero value.
	if sig.Result != nil && !terminates(fn.Body.List) {
		return tc.errorf(fn.Body.Rbrace, "missing return at end of function %s", fn.Name.Name)
	}

	scope := newTypeEnv(e.cur.globalTypes)
	for i, p := range fn.Params {
		if p.Name != nil {
			scope.define(p.Name.Name, sig.Params[i])
		}
	}
	return tc.checkStmts(fn.Body.List, scope)
}

// checkTopStmt type-checks a top-level statement in the global type scope, just
// before it runs.
func (e *engine) checkTopStmt(s ast.Stmt) error {
	tc := &typeChecker{e: e, result: nil}
	return tc.checkStmt(s, e.cur.globalTypes)
}

// typeChecker checks one function body or one top-level statement. result is the
// enclosing function's declared result type (nil at top level or for a void
// function).
type typeChecker struct {
	e      *engine
	result types.Type
}

func (tc *typeChecker) errorf(pos token.Pos, format string, args ...any) error {
	return TypeError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (tc *typeChecker) checkStmts(stmts []ast.Stmt, scope *typeEnv) error {
	for _, s := range stmts {
		if err := tc.checkStmt(s, scope); err != nil {
			return err
		}
	}
	return nil
}

func (tc *typeChecker) checkStmt(s ast.Stmt, scope *typeEnv) error {
	switch s := s.(type) {
	case *ast.DeclStmt:
		t, err := tc.checkExpr(s.Value, scope)
		if err != nil {
			return err
		}
		if isVoid(t) {
			return tc.errorf(s.Value.Pos(), "cannot use a void value as an initializer")
		}
		// A := may not redeclare a name already bound in the same block. The
		// global scope is exempt, so a top-level name (and a REPL binding) can be
		// redefined.
		if scope != tc.e.cur.globalTypes {
			if _, exists := scope.vars[s.Name.Name]; exists {
				return tc.errorf(s.Name.Pos(), "%s redeclared in this block", s.Name.Name)
			}
		}
		scope.define(s.Name.Name, t)
		return nil

	case *ast.AssignStmt:
		return tc.checkAssign(s, scope)

	case *ast.ExprStmt:
		_, err := tc.checkExpr(s.X, scope)
		return err

	case *ast.ReturnStmt:
		return tc.checkReturn(s, scope)

	case *ast.IfStmt:
		if err := tc.checkCond(s.Cond, scope); err != nil {
			return err
		}
		if err := tc.checkStmts(s.Body.List, scope.child()); err != nil {
			return err
		}
		if s.Else != nil {
			return tc.checkStmt(s.Else, scope)
		}
		return nil

	case *ast.ForStmt:
		if s.Cond != nil {
			if err := tc.checkCond(s.Cond, scope); err != nil {
				return err
			}
		}
		return tc.checkStmts(s.Body.List, scope.child())

	case *ast.Block:
		return tc.checkStmts(s.List, scope.child())

	case *ast.BadStmt:
		return tc.errorf(s.Pos(), "malformed statement")

	default:
		return nil
	}
}

func (tc *typeChecker) checkAssign(s *ast.AssignStmt, scope *typeEnv) error {
	var lt types.Type
	switch lhs := s.Lhs.(type) {
	case *ast.Ident:
		ty, ok := scope.lookup(lhs.Name)
		if !ok {
			return tc.errorf(lhs.Pos(), "undefined: %s", lhs.Name)
		}
		lt = ty
	case *ast.IndexExpr:
		ty, err := tc.checkIndex(lhs, scope)
		if err != nil {
			return err
		}
		lt = ty
	default:
		return tc.errorf(s.Lhs.Pos(), "cannot assign to this expression")
	}

	rt, err := tc.checkExpr(s.Rhs, scope)
	if err != nil {
		return err
	}
	if !isInvalid(lt) && !isVoid(rt) && !types.Identical(lt, rt) {
		return tc.errorf(s.OpPos, "cannot assign %s to %s", rt, lt)
	}
	return nil
}

func (tc *typeChecker) checkReturn(s *ast.ReturnStmt, scope *typeEnv) error {
	if s.Result == nil {
		if tc.result != nil {
			return tc.errorf(s.Return, "missing return value (%s expected)", tc.result)
		}
		return nil
	}
	rt, err := tc.checkExpr(s.Result, scope)
	if err != nil {
		return err
	}
	switch {
	case tc.result == nil:
		return tc.errorf(s.Result.Pos(), "too many return values")
	case isVoid(rt):
		return tc.errorf(s.Result.Pos(), "cannot return a void value")
	case !isInvalid(tc.result) && !types.Identical(rt, tc.result):
		return tc.errorf(s.Result.Pos(), "cannot return %s as %s", rt, tc.result)
	}
	return nil
}

func (tc *typeChecker) checkCond(cond ast.Expr, scope *typeEnv) error {
	t, err := tc.checkExpr(cond, scope)
	if err != nil {
		return err
	}
	if !isBool(t) {
		return tc.errorf(cond.Pos(), "condition must be bool, got %s", t)
	}
	return nil
}

// checkTypeExpr reports an error if e does not denote a known type, so an
// unknown type name cannot slip through as "invalid" and disable checking.
func (tc *typeChecker) checkTypeExpr(e ast.Expr) error {
	switch e := e.(type) {
	case *ast.TypeName:
		if _, ok := types.Lookup(e.Name); !ok {
			return tc.errorf(e.Pos(), "undefined type: %s", e.Name)
		}
		return nil
	case *ast.ArrayType:
		return tc.checkTypeExpr(e.Elem)
	default:
		return tc.errorf(e.Pos(), "invalid type")
	}
}

func (tc *typeChecker) checkExpr(x ast.Expr, scope *typeEnv) (types.Type, error) {
	switch x := x.(type) {
	case *ast.IntLit:
		return types.Int, nil
	case *ast.FloatLit:
		return types.Float, nil
	case *ast.StringLit:
		return types.String, nil
	case *ast.Ident:
		if ty, ok := scope.lookup(x.Name); ok {
			return ty, nil
		}
		return nil, tc.errorf(x.Pos(), "undefined: %s", x.Name)
	case *ast.UnaryExpr:
		return tc.checkUnary(x, scope)
	case *ast.BinaryExpr:
		return tc.checkBinary(x, scope)
	case *ast.CallExpr:
		return tc.checkCall(x, scope)
	case *ast.IndexExpr:
		return tc.checkIndex(x, scope)
	case *ast.CompositeLit:
		return tc.checkCompositeLit(x, scope)
	case *ast.SelectorExpr:
		// A qualified name is only valid as a call's callee (functions only, P7).
		return nil, tc.errorf(x.Pos(), "qualified name is only valid as a function call")
	case *ast.BadExpr:
		return nil, tc.errorf(x.Pos(), "malformed expression")
	default:
		return nil, tc.errorf(x.Pos(), "cannot type-check %T", x)
	}
}

func (tc *typeChecker) checkUnary(x *ast.UnaryExpr, scope *typeEnv) (types.Type, error) {
	t, err := tc.checkExpr(x.X, scope)
	if err != nil {
		return nil, err
	}
	switch x.Op {
	case token.ADD, token.SUB:
		if !isNumeric(t) {
			return nil, tc.errorf(x.OpPos, "operator %s requires a numeric operand, got %s", x.Op, t)
		}
		return t, nil
	case token.NOT:
		if !isBool(t) {
			return nil, tc.errorf(x.OpPos, "operator ! requires a bool operand, got %s", t)
		}
		return types.Bool, nil
	default:
		return nil, tc.errorf(x.OpPos, "operator %s is not supported", x.Op)
	}
}

func (tc *typeChecker) checkBinary(x *ast.BinaryExpr, scope *typeEnv) (types.Type, error) {
	lt, err := tc.checkExpr(x.Left, scope)
	if err != nil {
		return nil, err
	}
	rt, err := tc.checkExpr(x.Right, scope)
	if err != nil {
		return nil, err
	}

	switch x.Op {
	case token.LAND, token.LOR:
		if !isBool(lt) || !isBool(rt) {
			return nil, tc.errorf(x.OpPos, "operator %s requires bool operands, got %s and %s", x.Op, lt, rt)
		}
		return types.Bool, nil
	case token.EQL, token.NEQ:
		if isSlice(lt) || isSlice(rt) {
			return nil, tc.errorf(x.OpPos, "slices are not comparable")
		}
		if !types.Identical(lt, rt) {
			return nil, tc.errorf(x.OpPos, "cannot compare %s and %s", lt, rt)
		}
		return types.Bool, nil
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		if !types.Identical(lt, rt) || !isOrdered(lt) {
			return nil, tc.errorf(x.OpPos, "operator %s is not defined for %s and %s", x.Op, lt, rt)
		}
		return types.Bool, nil
	case token.ADD:
		if !types.Identical(lt, rt) || (!isNumeric(lt) && !isString(lt)) {
			return nil, tc.errorf(x.OpPos, "operator + is not defined for %s and %s", lt, rt)
		}
		return lt, nil
	case token.SUB, token.MUL, token.QUO:
		if !types.Identical(lt, rt) || !isNumeric(lt) {
			return nil, tc.errorf(x.OpPos, "operator %s is not defined for %s and %s", x.Op, lt, rt)
		}
		return lt, nil
	case token.REM:
		if !isInt(lt) || !isInt(rt) {
			return nil, tc.errorf(x.OpPos, "operator %% requires int operands, got %s and %s", lt, rt)
		}
		return types.Int, nil
	default:
		return nil, tc.errorf(x.OpPos, "operator %s is not supported", x.Op)
	}
}

// checkCall dispatches by the shape of the callee — a bare name (a builtin or a
// function in scope) or a qualified pkg.Member — mirroring the executor.
func (tc *typeChecker) checkCall(x *ast.CallExpr, scope *typeEnv) (types.Type, error) {
	switch fn := x.Fn.(type) {
	case *ast.Ident:
		return tc.checkIdentCall(x, fn, scope)
	case *ast.SelectorExpr:
		return tc.checkQualifiedCall(x, fn, scope)
	default:
		return nil, tc.errorf(x.Fn.Pos(), "only direct function calls are supported")
	}
}

func (tc *typeChecker) checkIdentCall(x *ast.CallExpr, id *ast.Ident, scope *typeEnv) (types.Type, error) {
	switch id.Name {
	case "print":
		for _, a := range x.Args {
			if _, err := tc.checkExpr(a, scope); err != nil {
				return nil, err
			}
		}
		return types.Void, nil
	case "len":
		if len(x.Args) != 1 {
			return nil, tc.errorf(x.Lparen, "len expects 1 argument, got %d", len(x.Args))
		}
		t, err := tc.checkExpr(x.Args[0], scope)
		if err != nil {
			return nil, err
		}
		if !isSlice(t) && !isString(t) {
			return nil, tc.errorf(x.Args[0].Pos(), "len expects a slice or string, got %s", t)
		}
		return types.Int, nil
	}

	fn, _, err := tc.e.resolve(id.Name)
	if err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, tc.errorf(id.Pos(), "undefined: %s", id.Name)
	}
	return tc.checkCallSig(x, tc.e.sigOf(fn), scope)
}

// checkQualifiedCall checks pkg.Member(...): the named import must be in scope
// and must define the member.
func (tc *typeChecker) checkQualifiedCall(x *ast.CallExpr, sel *ast.SelectorExpr, scope *typeEnv) (types.Type, error) {
	pkgID, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, tc.errorf(sel.X.Pos(), "only direct function calls are supported")
	}
	p, ok := tc.e.cur.imports[pkgID.Name]
	if !ok {
		return nil, tc.errorf(pkgID.Pos(), "undefined: %s", pkgID.Name)
	}
	fn, ok := p.funcs[sel.Sel.Name]
	if !ok {
		return nil, tc.errorf(sel.Sel.Pos(), "undefined: %s.%s", pkgID.Name, sel.Sel.Name)
	}
	return tc.checkCallSig(x, tc.e.sigOf(fn), scope)
}

// checkCallSig checks a call's argument count and types against a resolved
// signature and returns the call's result type.
func (tc *typeChecker) checkCallSig(x *ast.CallExpr, sig *types.Signature, scope *typeEnv) (types.Type, error) {
	if len(x.Args) != len(sig.Params) {
		return nil, tc.errorf(x.Lparen, "wrong number of arguments: got %d, want %d", len(x.Args), len(sig.Params))
	}
	for i, a := range x.Args {
		at, err := tc.checkExpr(a, scope)
		if err != nil {
			return nil, err
		}
		if !isInvalid(sig.Params[i]) && !types.Identical(at, sig.Params[i]) {
			return nil, tc.errorf(a.Pos(), "argument %d: cannot use %s as %s", i+1, at, sig.Params[i])
		}
	}

	if sig.Result == nil {
		return types.Void, nil
	}
	return sig.Result, nil
}

func (tc *typeChecker) checkIndex(x *ast.IndexExpr, scope *typeEnv) (types.Type, error) {
	xt, err := tc.checkExpr(x.X, scope)
	if err != nil {
		return nil, err
	}
	it, err := tc.checkExpr(x.Index, scope)
	if err != nil {
		return nil, err
	}
	if !isInt(it) {
		return nil, tc.errorf(x.Index.Pos(), "index must be int, got %s", it)
	}
	st, ok := xt.(types.Slice)
	if !ok {
		return nil, tc.errorf(x.X.Pos(), "cannot index %s", xt)
	}
	return st.Elem, nil
}

func (tc *typeChecker) checkCompositeLit(x *ast.CompositeLit, scope *typeEnv) (types.Type, error) {
	st, ok := resolveType(x.Type).(types.Slice)
	if !ok {
		return nil, tc.errorf(x.Pos(), "composite literal requires a slice type")
	}
	for _, el := range x.Elems {
		et, err := tc.checkExpr(el, scope)
		if err != nil {
			return nil, err
		}
		if !isInvalid(st.Elem) && !types.Identical(et, st.Elem) {
			return nil, tc.errorf(el.Pos(), "cannot use %s as %s in array literal", et, st.Elem)
		}
	}
	return st, nil
}

//
// type predicates (mirroring internal/check)
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

// terminates reports whether a statement list always transfers control away
// (returns, or loops forever), so control never falls off the end. Ported from
// the batch linter to give the streaming checker a missing-return rule.
func terminates(list []ast.Stmt) bool {
	for _, s := range list {
		if terminatesStmt(s) {
			return true
		}
	}
	return false
}

func terminatesStmt(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		return s.Else != nil && terminates(s.Body.List) && terminatesStmt(s.Else)
	case *ast.Block:
		return terminates(s.List)
	case *ast.ForStmt:
		return s.Cond == nil // an infinite loop never falls through (chip has no break)
	default:
		return false
	}
}
