// Package compiler translates a type-checked chip syntax tree into bytecode.
package compiler

import (
	"errors"
	"fmt"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/types"
)

// Compiler holds the state of a single compilation.
type Compiler struct {
	info    *check.Info
	funcIdx map[*scope.Symbol]int // function symbol -> index in Program.Functions

	// per-function state
	chunk  *code.Chunk
	slots  map[*scope.Symbol]int // local/param symbol -> frame slot
	nslots int

	errs []error
}

// Compile translates a type-checked file into a Program.
func Compile(file *ast.File, info *check.Info) (*code.Program, error) {
	c := &Compiler{info: info, funcIdx: make(map[*scope.Symbol]int)}

	var fns []*ast.FuncDecl
	prog := &code.Program{Entry: -1}
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		idx := len(fns)
		fns = append(fns, fn)
		if sym := info.Defs[fn.Name]; sym != nil {
			c.funcIdx[sym] = idx
		}
		if fn.Name.Name == "main" {
			prog.Entry = idx
		}
	}

	for _, fn := range fns {
		prog.Functions = append(prog.Functions, c.compileFunc(fn))
	}

	if len(c.errs) > 0 {
		return nil, errors.Join(c.errs...)
	}
	return prog, nil
}

func (c *Compiler) compileFunc(fn *ast.FuncDecl) *code.FuncProto {
	c.chunk = &code.Chunk{}
	c.slots = make(map[*scope.Symbol]int)
	c.nslots = 0

	// Parameters occupy the first slots, in order.
	for _, p := range fn.Params {
		if sym := c.info.Defs[p.Name]; sym != nil {
			c.declareLocal(sym)
		}
	}

	for _, s := range fn.Body.List {
		c.compileStmt(s)
	}
	c.emit(code.OpReturnVoid, 0, fn.Body.Rbrace) // safety net for fall-through

	return &code.FuncProto{
		Name:      fn.Name.Name,
		NumParams: len(fn.Params),
		NumLocals: c.nslots,
		Chunk:     c.chunk,
	}
}

// declareLocal assigns the next frame slot to sym.
func (c *Compiler) declareLocal(sym *scope.Symbol) int {
	slot := c.nslots
	c.slots[sym] = slot
	c.nslots++
	return slot
}

func (c *Compiler) emit(op code.Opcode, operand int, pos token.Pos) int {
	return c.chunk.Emit(op, operand, pos)
}

// patch rewrites the operand of a previously emitted instruction (for jumps).
func (c *Compiler) patch(at, operand int) {
	c.chunk.Code[at].Operand = operand
}

// isVoid reports whether expression e produces no usable value.
func (c *Compiler) isVoid(e ast.Expr) bool {
	b, ok := c.info.Types[e].(types.Basic)
	return ok && (b.Kind == types.KindVoid || b.Kind == types.KindInvalid)
}

func (c *Compiler) errorf(pos token.Pos, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	c.errs = append(c.errs, fmt.Errorf("%d:%d: %s", pos.Line, pos.Column, msg))
}
