package compiler

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

func (c *Compiler) compileExpr(e ast.Expr) {
	switch e := e.(type) {
	case *ast.IntLit:
		c.emit(code.OpConst, c.chunk.AddConstant(value.Int(e.Value)), e.Pos())
	case *ast.FloatLit:
		c.emit(code.OpConst, c.chunk.AddConstant(value.Float(e.Value)), e.Pos())
	case *ast.StringLit:
		c.emit(code.OpConst, c.chunk.AddConstant(value.Str(e.Value)), e.Pos())
	case *ast.Ident:
		c.compileIdent(e)
	case *ast.UnaryExpr:
		c.compileUnary(e)
	case *ast.BinaryExpr:
		c.compileBinary(e)
	case *ast.CallExpr:
		c.compileCall(e)
	case *ast.CompositeLit:
		c.compileCompositeLit(e)
	case *ast.IndexExpr:
		c.compileIndex(e)
	case *ast.BadExpr:
		c.errorf(e.Pos(), "cannot compile a malformed expression")
	default:
		c.errorf(e.Pos(), "cannot compile %T", e)
	}
}

func (c *Compiler) compileIdent(e *ast.Ident) {
	slot, ok := c.slots[c.info.Uses[e]]
	if !ok {
		c.errorf(e.Pos(), "%s is not a variable", e.Name)
		return
	}
	c.emit(code.OpGetLocal, slot, e.Pos())
}

func (c *Compiler) compileUnary(e *ast.UnaryExpr) {
	c.compileExpr(e.X)
	switch e.Op {
	case token.SUB:
		c.emit(code.OpNeg, 0, e.OpPos)
	case token.NOT:
		c.emit(code.OpNot, 0, e.OpPos)
	case token.ADD:
		// unary plus is a no-op
	}
}

// binaryOps maps non-short-circuiting binary operators to their opcodes.
var binaryOps = map[token.Type]code.Opcode{
	token.ADD: code.OpAdd,
	token.SUB: code.OpSub,
	token.MUL: code.OpMul,
	token.QUO: code.OpDiv,
	token.REM: code.OpRem,
	token.EQL: code.OpEqual,
	token.NEQ: code.OpNotEqual,
	token.LSS: code.OpLess,
	token.LEQ: code.OpLessEqual,
	token.GTR: code.OpGreater,
	token.GEQ: code.OpGreaterEqual,
}

func (c *Compiler) compileBinary(e *ast.BinaryExpr) {
	switch e.Op {
	case token.LAND:
		c.compileAnd(e)
		return
	case token.LOR:
		c.compileOr(e)
		return
	}

	c.compileExpr(e.Left)
	c.compileExpr(e.Right)
	op, ok := binaryOps[e.Op]
	if !ok {
		c.errorf(e.OpPos, "operator %s is not supported", e.Op)
		return
	}
	c.emit(op, 0, e.OpPos)
}

// compileAnd compiles a && b with short-circuit evaluation.
func (c *Compiler) compileAnd(e *ast.BinaryExpr) {
	c.compileExpr(e.Left)
	jmpFalse := c.emit(code.OpJumpIfFalse, 0, e.OpPos)
	c.compileExpr(e.Right)
	jmpEnd := c.emit(code.OpJump, 0, e.OpPos)
	c.patch(jmpFalse, len(c.chunk.Code))
	c.emit(code.OpFalse, 0, e.OpPos)
	c.patch(jmpEnd, len(c.chunk.Code))
}

// compileOr compiles a || b with short-circuit evaluation.
func (c *Compiler) compileOr(e *ast.BinaryExpr) {
	c.compileExpr(e.Left)
	jmpFalse := c.emit(code.OpJumpIfFalse, 0, e.OpPos)
	c.emit(code.OpTrue, 0, e.OpPos)
	jmpEnd := c.emit(code.OpJump, 0, e.OpPos)
	c.patch(jmpFalse, len(c.chunk.Code))
	c.compileExpr(e.Right)
	c.patch(jmpEnd, len(c.chunk.Code))
}

func (c *Compiler) compileCompositeLit(e *ast.CompositeLit) {
	for _, el := range e.Elems {
		c.compileExpr(el)
	}
	c.emit(code.OpMakeArray, len(e.Elems), e.Lbrace)
}

func (c *Compiler) compileIndex(e *ast.IndexExpr) {
	c.compileExpr(e.X)
	c.compileExpr(e.Index)
	c.emit(code.OpIndex, 0, e.Lbrack)
}

func (c *Compiler) compileCall(e *ast.CallExpr) {
	id, ok := e.Fn.(*ast.Ident)
	if !ok {
		c.errorf(e.Fn.Pos(), "only direct function calls are supported")
		return
	}
	sym := c.info.Uses[id]
	if sym == nil {
		c.errorf(id.Pos(), "unresolved call to %s", id.Name)
		return
	}

	if sym.Kind == scope.Builtin {
		c.compileBuiltin(id.Name, e)
		return
	}

	idx, ok := c.funcIdx[sym]
	if !ok {
		c.errorf(id.Pos(), "cannot call %s", id.Name)
		return
	}
	for _, a := range e.Args {
		c.compileExpr(a)
	}
	c.emit(code.OpCall, idx, e.Lparen)
}

func (c *Compiler) compileBuiltin(name string, e *ast.CallExpr) {
	for _, a := range e.Args {
		c.compileExpr(a)
	}
	switch name {
	case "print":
		c.emit(code.OpPrint, len(e.Args), e.Lparen)
	case "len":
		c.emit(code.OpLen, 0, e.Lparen)
	default:
		c.errorf(e.Fn.Pos(), "unknown builtin %s", name)
	}
}
