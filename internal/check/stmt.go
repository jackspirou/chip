package check

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/types"
)

func (c *Checker) checkStmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.DeclStmt:
		c.checkDecl(s)
	case *ast.AssignStmt:
		c.checkAssign(s)
	case *ast.ExprStmt:
		c.checkExpr(s.X)
	case *ast.ReturnStmt:
		c.checkReturn(s)
	case *ast.IfStmt:
		c.checkIf(s)
	case *ast.ForStmt:
		c.checkFor(s)
	case *ast.Block:
		c.checkBlock(s)
	case *ast.BadStmt:
		// nothing to check
	}
}

// checkBlock checks a brace-enclosed block in its own nested scope.
func (c *Checker) checkBlock(b *ast.Block) {
	if b == nil {
		return
	}
	outer := c.scope
	c.scope = scope.New(outer)
	for _, s := range b.List {
		c.checkStmt(s)
	}
	c.scope = outer
}

func (c *Checker) checkDecl(s *ast.DeclStmt) {
	t := c.checkExpr(s.Value)
	if isVoid(t) {
		c.errorf(s.Value.Pos(), "cannot use a void value as an initializer")
		t = types.Invalid
	}
	sym := &scope.Symbol{Name: s.Name.Name, Kind: scope.Var, Type: t, DeclPos: s.Name.Pos()}
	if s.Name.Name != "" {
		if prev := c.scope.Insert(sym); prev != nil {
			c.errorf(s.Name.Pos(), "%s redeclared in this block", s.Name.Name)
		}
	}
	c.info.Defs[s.Name] = sym
}

func (c *Checker) checkAssign(s *ast.AssignStmt) {
	lt := c.checkAssignTarget(s.Lhs)
	rt := c.checkExpr(s.Rhs)
	if !isInvalid(lt) && !isInvalid(rt) && !isVoid(rt) && !types.Identical(lt, rt) {
		c.errorf(s.OpPos, "cannot assign %s to %s", rt, lt)
	}
}

func (c *Checker) checkAssignTarget(e ast.Expr) types.Type {
	switch e := e.(type) {
	case *ast.Ident:
		sym := c.lookup(e.Name)
		if sym == nil {
			c.errorf(e.Pos(), "undefined: %s", e.Name)
			return types.Invalid
		}
		c.info.Uses[e] = sym
		if sym.Kind == scope.Func || sym.Kind == scope.Builtin {
			c.errorf(e.Pos(), "cannot assign to %s", e.Name)
			return types.Invalid
		}
		return sym.Type
	case *ast.IndexExpr:
		return c.checkIndex(e)
	default:
		c.errorf(e.Pos(), "cannot assign to this expression")
		c.checkExpr(e)
		return types.Invalid
	}
}

func (c *Checker) checkReturn(s *ast.ReturnStmt) {
	if c.sig == nil {
		return
	}
	if s.Result == nil {
		if c.sig.Result != nil {
			c.errorf(s.Return, "missing return value (%s expected)", c.sig.Result)
		}
		return
	}

	rt := c.checkExpr(s.Result)
	if isInvalid(rt) {
		return
	}
	switch {
	case c.sig.Result == nil:
		c.errorf(s.Result.Pos(), "too many return values")
	case isVoid(rt):
		c.errorf(s.Result.Pos(), "cannot return a void value")
	case !types.Identical(rt, c.sig.Result):
		c.errorf(s.Result.Pos(), "cannot return %s as %s", rt, c.sig.Result)
	}
}

func (c *Checker) checkIf(s *ast.IfStmt) {
	c.checkCond(s.Cond)
	c.checkBlock(s.Body)
	if s.Else != nil {
		c.checkStmt(s.Else)
	}
}

func (c *Checker) checkFor(s *ast.ForStmt) {
	if s.Cond != nil {
		c.checkCond(s.Cond)
	}
	c.checkBlock(s.Body)
}

func (c *Checker) checkCond(cond ast.Expr) {
	t := c.checkExpr(cond)
	if !isInvalid(t) && !isBool(t) {
		c.errorf(cond.Pos(), "condition must be bool, got %s", t)
	}
}
