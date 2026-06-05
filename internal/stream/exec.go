package stream

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

// flow signals how a statement (or block) completed.
type flow uint8

const (
	flowNormal flow = iota // fell through to the next statement
	flowReturn             // a return statement fired; carry its value up
)

// execStmts runs a list of statements in env, stopping early on a return.
func (e *engine) execStmts(stmts []ast.Stmt, scope *env) (flow, value.Value, error) {
	for _, s := range stmts {
		fl, v, err := e.execStmt(s, scope)
		if err != nil {
			return flowNormal, value.Value{}, err
		}
		if fl == flowReturn {
			return fl, v, nil
		}
	}
	return flowNormal, value.Value{}, nil
}

// execStmt executes a single statement.
func (e *engine) execStmt(s ast.Stmt, scope *env) (flow, value.Value, error) {
	// Statement boundary: account one step and enforce the wall-clock/step caps
	// (a no-op unless a cap is set). This is where a runaway run is stopped.
	if err := e.step(s.Pos()); err != nil {
		return flowNormal, value.Value{}, err
	}
	switch s := s.(type) {
	case *ast.DeclStmt:
		v, err := e.eval(s.Value, scope)
		if err != nil {
			return flowNormal, value.Value{}, err
		}
		scope.define(s.Name.Name, v)
		return flowNormal, value.Value{}, nil

	case *ast.AssignStmt:
		return flowNormal, value.Value{}, e.execAssign(s, scope)

	case *ast.ExprStmt:
		_, err := e.eval(s.X, scope)
		return flowNormal, value.Value{}, err

	case *ast.ReturnStmt:
		if s.Result == nil {
			return flowReturn, value.Value{}, nil
		}
		v, err := e.eval(s.Result, scope)
		if err != nil {
			return flowNormal, value.Value{}, err
		}
		return flowReturn, v, nil

	case *ast.IfStmt:
		return e.execIf(s, scope)

	case *ast.ForStmt:
		return e.execFor(s, scope)

	case *ast.Block:
		inner := scope
		if blockDeclares(s.List) {
			inner = scope.child()
		}
		return e.execStmts(s.List, inner)

	case *ast.BadStmt:
		return flowNormal, value.Value{}, e.errorf(s.Pos(), "malformed statement")

	default:
		return flowNormal, value.Value{}, e.errorf(s.Pos(), "cannot execute %T", s)
	}
}

// execAssign evaluates `lhs = rhs` for a variable or an indexed element.
func (e *engine) execAssign(s *ast.AssignStmt, scope *env) error {
	switch lhs := s.Lhs.(type) {
	case *ast.Ident:
		rhs, err := e.eval(s.Rhs, scope)
		if err != nil {
			return err
		}
		if !scope.assign(lhs.Name, rhs) {
			return e.errorf(lhs.Pos(), "undefined: %s", lhs.Name)
		}
		return nil

	case *ast.IndexExpr:
		arr, err := e.eval(lhs.X, scope)
		if err != nil {
			return err
		}
		idx, err := e.eval(lhs.Index, scope)
		if err != nil {
			return err
		}
		rhs, err := e.eval(s.Rhs, scope)
		if err != nil {
			return err
		}
		elems := arr.AsSlice()
		i := idx.AsInt()
		if i < 0 || i >= int64(len(elems)) {
			return e.errorf(lhs.Lbrack, "index out of range: %d (length %d)", i, len(elems))
		}
		elems[i] = rhs
		return nil

	default:
		return e.errorf(s.Lhs.Pos(), "cannot assign to %T", s.Lhs)
	}
}

// execIf evaluates an if/else; the taken branch runs in a child scope.
func (e *engine) execIf(s *ast.IfStmt, scope *env) (flow, value.Value, error) {
	cond, err := e.eval(s.Cond, scope)
	if err != nil {
		return flowNormal, value.Value{}, err
	}
	if cond.AsBool() {
		inner := scope
		if blockDeclares(s.Body.List) {
			inner = scope.child()
		}
		return e.execStmts(s.Body.List, inner)
	}
	if s.Else != nil {
		return e.execStmt(s.Else, scope)
	}
	return flowNormal, value.Value{}, nil
}

// execFor runs a for loop; each iteration's body runs in its own child scope. A
// nil condition is an infinite loop (chip has no break yet, so it exits only via
// return).
func (e *engine) execFor(s *ast.ForStmt, scope *env) (flow, value.Value, error) {
	// Whether the body needs a fresh scope is fixed across iterations, so decide
	// it once: a body that declares nothing runs directly in scope.
	declares := blockDeclares(s.Body.List)
	for {
		// Tick once per iteration too, so an empty-bodied or condition-only loop
		// (which runs no inner statement to tick) is still capped and cancellable.
		if err := e.step(s.Pos()); err != nil {
			return flowNormal, value.Value{}, err
		}
		if s.Cond != nil {
			cond, err := e.eval(s.Cond, scope)
			if err != nil {
				return flowNormal, value.Value{}, err
			}
			if !cond.AsBool() {
				return flowNormal, value.Value{}, nil
			}
		}
		inner := scope
		if declares {
			inner = scope.child()
		}
		fl, v, err := e.execStmts(s.Body.List, inner)
		if err != nil {
			return flowNormal, value.Value{}, err
		}
		if fl == flowReturn {
			return fl, v, nil
		}
	}
}

// callFunc invokes fn with args in a fresh per-call scope (parent = global) and
// returns its result. A function that falls off the end yields the zero value.
func (e *engine) callFunc(fn *ast.FuncDecl, args []value.Value, pos token.Pos) (value.Value, error) {
	if e.depth >= e.maxDepth {
		return value.Value{}, e.errorf(pos, "call stack too deep")
	}
	if len(args) != len(fn.Params) {
		return value.Value{}, e.errorf(pos, "%s expects %d argument(s), got %d", fn.Name.Name, len(fn.Params), len(args))
	}

	scope := newEnv(e.cur.global)
	for i, p := range fn.Params {
		if p.Name != nil {
			scope.define(p.Name.Name, args[i])
		}
	}

	e.depth++
	_, v, err := e.execStmts(fn.Body.List, scope)
	e.depth--
	if err != nil {
		return value.Value{}, err
	}
	return v, nil
}

// blockDeclares reports whether stmts binds a new variable directly in its own
// scope — that is, contains a := declaration at this level. A nested if/for/block
// gets its own child scope, so declarations inside one land there, not here;
// only a direct DeclStmt forces this block to have a scope of its own. A body
// that declares nothing can run in its enclosing scope with identical semantics
// (assignments search outward either way), which lets execFor/execIf skip a
// per-iteration child *env. The scan is shallow and cheaper than the scope it
// avoids allocating.
func blockDeclares(stmts []ast.Stmt) bool {
	for _, s := range stmts {
		if _, ok := s.(*ast.DeclStmt); ok {
			return true
		}
	}
	return false
}
