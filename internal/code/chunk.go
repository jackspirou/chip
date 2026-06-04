package code

import (
	"fmt"
	"strings"

	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

// Instruction is a single VM instruction.
type Instruction struct {
	Op      Opcode
	Operand int
}

// Chunk is a sequence of instructions plus the constants they reference.
type Chunk struct {
	Code      []Instruction
	Constants []value.Value
	Lines     []token.Pos // source position of each instruction, parallel to Code
}

// Emit appends an instruction and returns its index.
func (c *Chunk) Emit(op Opcode, operand int, pos token.Pos) int {
	c.Code = append(c.Code, Instruction{Op: op, Operand: operand})
	c.Lines = append(c.Lines, pos)
	return len(c.Code) - 1
}

// AddConstant appends a constant and returns its index.
func (c *Chunk) AddConstant(v value.Value) int {
	c.Constants = append(c.Constants, v)
	return len(c.Constants) - 1
}

// FuncProto is a compiled function.
type FuncProto struct {
	Name      string
	NumParams int
	NumLocals int // total local slots, including parameters
	Chunk     *Chunk
}

// Program is a compiled chip program: its functions and the index of the entry
// point (main), or -1 if there is none.
type Program struct {
	Functions []*FuncProto
	Entry     int
}

// String disassembles the whole program for debugging.
func (p *Program) String() string {
	var b strings.Builder
	for i, fn := range p.Functions {
		fmt.Fprintf(&b, "fn %d %s/%d (locals %d)\n", i, fn.Name, fn.NumParams, fn.NumLocals)
		for ip, inst := range fn.Chunk.Code {
			fmt.Fprintf(&b, "  %4d  %-12s", ip, inst.Op)
			if inst.Op.hasOperand() {
				fmt.Fprintf(&b, " %d", inst.Operand)
				if inst.Op == OpConst {
					fmt.Fprintf(&b, " (%s)", fn.Chunk.Constants[inst.Operand])
				}
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
