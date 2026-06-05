package check

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/typerules"
	"github.com/jackspirou/chip/internal/types"
)

// checkExpr type-checks e, records its type in Info, and returns it.
func (c *Checker) checkExpr(e ast.Expr) types.Type {
	t := c.checkExpr0(e)
	c.info.Types[e] = t
	return t
}

func (c *Checker) checkExpr0(e ast.Expr) types.Type {
	switch e := e.(type) {
	case *ast.IntLit:
		return types.Int
	case *ast.FloatLit:
		return types.Float
	case *ast.StringLit:
		return types.String
	case *ast.Ident:
		return c.checkIdent(e)
	case *ast.UnaryExpr:
		return c.checkUnary(e)
	case *ast.BinaryExpr:
		return c.checkBinary(e)
	case *ast.CallExpr:
		return c.checkCall(e)
	case *ast.IndexExpr:
		return c.checkIndex(e)
	case *ast.CompositeLit:
		return c.checkCompositeLit(e)
	case *ast.ArrayType:
		c.errorf(e.Pos(), "type used as a value")
		return types.Invalid
	case *ast.SelectorExpr:
		c.checkExpr(e.X)
		return types.Invalid // packages and structs are not supported yet
	case *ast.TypeName:
		c.errorf(e.Pos(), "type used as a value")
		return types.Invalid
	default: // *ast.BadExpr and anything unexpected
		return types.Invalid
	}
}

func (c *Checker) checkIdent(e *ast.Ident) types.Type {
	sym := c.lookup(e.Name)
	if sym == nil {
		c.undefinedIdent(e)
		return types.Invalid
	}
	c.info.Uses[e] = sym
	// A name reaching here is in value position: a call resolves its callee in
	// checkCall before this point, so a builtin or function seen here is being
	// used as a value, which chip does not allow (it has neither first-class
	// builtins nor first-class functions). The streaming checker (chip run)
	// rejects the same misuse; rejecting it here keeps chip check from being more
	// lenient than chip run. Without this, a bare function name read as its
	// signature, which isInvalid then quietly swallowed.
	switch sym.Kind {
	case scope.Builtin, scope.Func:
		c.errorf(e.Pos(), "%s is not a value", e.Name)
		return types.Invalid
	}
	return sym.Type
}

func (c *Checker) checkUnary(e *ast.UnaryExpr) types.Type {
	t := c.checkExpr(e.X)
	if isInvalid(t) {
		return types.Invalid
	}
	// The operator rule is shared with the streaming checker via typerules, so an
	// operand the streaming checker rejects cannot pass here. Cascades from an
	// already-invalid operand are suppressed above.
	res, msg := typerules.Unary(e.Op, t)
	if msg != "" {
		c.errorf(e.OpPos, "%s", msg)
	}
	return res
}

func (c *Checker) checkBinary(e *ast.BinaryExpr) types.Type {
	lt := c.checkExpr(e.Left)
	rt := c.checkExpr(e.Right)

	// Suppress cascades: an operand that already errored must not provoke a
	// second, derived error. A comparison or logical expression still reads as a
	// bool so an enclosing condition stays well-formed.
	if isInvalid(lt) || isInvalid(rt) {
		switch e.Op {
		case token.LAND, token.LOR, token.EQL, token.NEQ,
			token.LSS, token.LEQ, token.GTR, token.GEQ:
			return types.Bool
		}
		return types.Invalid
	}

	// The operator rules are shared with the streaming checker via typerules, so
	// chip check and chip run agree on exactly which operators and operand types
	// are allowed (the bitwise and shift operators, for one, are rejected by
	// both rather than silently accepted here).
	res, msg := typerules.Binary(e.Op, lt, rt)
	if msg != "" {
		c.errorf(e.OpPos, "%s", msg)
	}
	return res
}

func (c *Checker) checkIndex(e *ast.IndexExpr) types.Type {
	xt := c.checkExpr(e.X)
	it := c.checkExpr(e.Index)
	if !isInvalid(it) && !isInt(it) {
		c.errorf(e.Index.Pos(), "index must be int, got %s", it)
	}
	st, ok := xt.(types.Slice)
	if !ok {
		if !isInvalid(xt) {
			c.errorf(e.X.Pos(), "cannot index %s", xt)
		}
		return types.Invalid
	}
	return st.Elem
}

func (c *Checker) checkCompositeLit(e *ast.CompositeLit) types.Type {
	t := c.resolveType(e.Type)
	st, ok := t.(types.Slice)
	if !ok {
		if !isInvalid(t) {
			c.errorf(e.Pos(), "composite literal requires a slice type, got %s", t)
		}
		for _, el := range e.Elems {
			c.checkExpr(el)
		}
		return types.Invalid
	}
	for _, el := range e.Elems {
		et := c.checkExpr(el)
		if !isInvalid(et) && !types.Identical(et, st.Elem) {
			c.errorf(el.Pos(), "cannot use %s as %s in array literal", et, st.Elem)
		}
	}
	return st
}

func (c *Checker) checkCall(e *ast.CallExpr) types.Type {
	// A callee that names a builtin or a function is resolved here, in call
	// position. Routing it directly — rather than through checkExpr/checkIdent —
	// lets checkIdent treat a bare function name as the value-position error it
	// is, and records the use the compiler needs to find the callee.
	if id, ok := e.Fn.(*ast.Ident); ok {
		if sym := c.lookup(id.Name); sym != nil {
			switch sym.Kind {
			case scope.Builtin:
				c.info.Uses[id] = sym
				return c.checkBuiltin(id.Name, e)
			case scope.Func:
				c.info.Uses[id] = sym
				if sig, ok := sym.Type.(*types.Signature); ok {
					return c.checkCallSig(e, sig)
				}
				return types.Invalid // a Func symbol always carries a signature
			}
		}
	}

	ft := c.checkExpr(e.Fn)
	sig, ok := ft.(*types.Signature)
	if !ok {
		if !isInvalid(ft) {
			c.errorf(e.Fn.Pos(), "cannot call non-function (type %s)", ft)
		}
		for _, a := range e.Args {
			c.checkExpr(a)
		}
		return types.Invalid
	}
	return c.checkCallSig(e, sig)
}

// checkCallSig checks a call's argument count and types against a resolved
// signature and returns the call's result type. It is shared by the two callee
// shapes — a name that resolves to a function, and any other expression of
// function type — so both agree on argument checking.
func (c *Checker) checkCallSig(e *ast.CallExpr, sig *types.Signature) types.Type {
	if len(e.Args) != len(sig.Params) {
		c.errorf(e.Lparen, "wrong number of arguments: got %d, want %d", len(e.Args), len(sig.Params))
	}
	for i, a := range e.Args {
		at := c.checkExpr(a)
		if i < len(sig.Params) && !isInvalid(at) && !types.Identical(at, sig.Params[i]) {
			c.errorf(a.Pos(), "argument %d: cannot use %s as %s", i+1, at, sig.Params[i])
		}
	}

	if sig.Result == nil {
		return types.Void
	}
	return sig.Result
}

// checkBuiltin checks a call to a predeclared builtin (print, len).
func (c *Checker) checkBuiltin(name string, e *ast.CallExpr) types.Type {
	switch name {
	case "print":
		for _, a := range e.Args {
			c.checkExpr(a)
		}
		return types.Void
	case "len":
		if len(e.Args) != 1 {
			c.errorf(e.Lparen, "len expects 1 argument, got %d", len(e.Args))
		}
		for i, a := range e.Args {
			t := c.checkExpr(a)
			if i == 0 && !isInvalid(t) && !isSlice(t) && !isString(t) {
				c.errorf(a.Pos(), "len expects a slice or string, got %s", t)
			}
		}
		return types.Int
	}
	for _, a := range e.Args {
		c.checkExpr(a)
	}
	return types.Invalid
}
