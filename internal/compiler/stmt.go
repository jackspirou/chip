package compiler

import (
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/code"
)

func (c *Compiler) compileStmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.DeclStmt:
		c.compileExpr(s.Value)
		if sym := c.info.Defs[s.Name]; sym != nil {
			c.emit(code.OpSetLocal, c.declareLocal(sym), s.Name.Pos())
		}
	case *ast.AssignStmt:
		c.compileAssign(s)
	case *ast.ExprStmt:
		c.compileExpr(s.X)
		if !c.isVoid(s.X) {
			c.emit(code.OpPop, 0, s.X.Pos())
		}
	case *ast.ReturnStmt:
		if s.Result == nil {
			c.emit(code.OpReturnVoid, 0, s.Return)
			return
		}
		c.compileExpr(s.Result)
		c.emit(code.OpReturn, 0, s.Return)
	case *ast.IfStmt:
		c.compileIf(s)
	case *ast.ForStmt:
		c.compileFor(s)
	case *ast.Block:
		c.compileBlock(s)
	case *ast.BadStmt:
		// nothing to compile
	}
}

func (c *Compiler) compileBlock(b *ast.Block) {
	for _, s := range b.List {
		c.compileStmt(s)
	}
}

// compileAssign compiles `lhs = rhs` for a variable or an indexed element.
func (c *Compiler) compileAssign(s *ast.AssignStmt) {
	switch lhs := s.Lhs.(type) {
	case *ast.Ident:
		c.compileExpr(s.Rhs)
		slot, ok := c.slots[c.info.Uses[lhs]]
		if !ok {
			c.errorf(lhs.Pos(), "cannot assign to %s", lhs.Name)
			return
		}
		c.emit(code.OpSetLocal, slot, s.OpPos)
	case *ast.IndexExpr:
		c.compileExpr(lhs.X)
		c.compileExpr(lhs.Index)
		c.compileExpr(s.Rhs)
		c.emit(code.OpSetIndex, 0, s.OpPos)
	default:
		c.errorf(s.Lhs.Pos(), "unsupported assignment target")
	}
}

func (c *Compiler) compileIf(s *ast.IfStmt) {
	c.compileExpr(s.Cond)
	jmpFalse := c.emit(code.OpJumpIfFalse, 0, s.If)
	c.compileBlock(s.Body)

	if s.Else == nil {
		c.patch(jmpFalse, len(c.chunk.Code))
		return
	}

	jmpEnd := c.emit(code.OpJump, 0, s.If)
	c.patch(jmpFalse, len(c.chunk.Code))
	c.compileStmt(s.Else)
	c.patch(jmpEnd, len(c.chunk.Code))
}

func (c *Compiler) compileFor(s *ast.ForStmt) {
	start := len(c.chunk.Code)
	exit := -1
	if s.Cond != nil {
		c.compileExpr(s.Cond)
		exit = c.emit(code.OpJumpIfFalse, 0, s.For)
	}
	c.compileBlock(s.Body)
	c.emit(code.OpJump, start, s.For)
	if exit >= 0 {
		c.patch(exit, len(c.chunk.Code))
	}
}
