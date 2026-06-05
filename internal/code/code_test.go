package code

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

// The code package had no tests; the bytecode it defines is the contract between
// the compiler and the VM, and `chip dump` prints it. These tests pin the chunk
// builders' return indices, the opcode mnemonics (including the out-of-range
// fallback), and the disassembly format.

func TestChunkEmit(t *testing.T) {
	var c Chunk
	pos := token.Pos{Line: 1, Column: 1}
	if i := c.Emit(OpTrue, 0, pos); i != 0 {
		t.Errorf("first Emit index = %d, want 0", i)
	}
	if i := c.Emit(OpPop, 0, pos); i != 1 {
		t.Errorf("second Emit index = %d, want 1", i)
	}
	if len(c.Code) != 2 || len(c.Lines) != 2 {
		t.Fatalf("Code/Lines lengths = %d/%d, want 2/2", len(c.Code), len(c.Lines))
	}
	if c.Code[0].Op != OpTrue || c.Code[1].Op != OpPop {
		t.Errorf("opcodes = %v/%v, want True/Pop", c.Code[0].Op, c.Code[1].Op)
	}
}

func TestChunkAddConstant(t *testing.T) {
	var c Chunk
	if i := c.AddConstant(value.Int(7)); i != 0 {
		t.Errorf("first constant index = %d, want 0", i)
	}
	if i := c.AddConstant(value.Str("hi")); i != 1 {
		t.Errorf("second constant index = %d, want 1", i)
	}
	if got := c.Constants[0].AsInt(); got != 7 {
		t.Errorf("Constants[0] = %d, want 7", got)
	}
	if got := c.Constants[1].AsStr(); got != "hi" {
		t.Errorf("Constants[1] = %q, want %q", got, "hi")
	}
}

func TestOpcodeString(t *testing.T) {
	tests := []struct {
		op   Opcode
		want string
	}{
		{OpConst, "Const"},
		{OpAdd, "Add"},
		{OpMissingReturn, "MissingReturn"},
		{OpLen, "Len"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("Opcode(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
	// An opcode past the end of the name table renders as the unknown sentinel
	// rather than panicking on an out-of-range index.
	if got := Opcode(250).String(); got != "Op(?)" {
		t.Errorf("unknown opcode String() = %q, want %q", got, "Op(?)")
	}
}

func TestHasOperand(t *testing.T) {
	// Opcodes that carry an operand the disassembler must print.
	withOperand := []Opcode{OpConst, OpGetLocal, OpSetLocal, OpJump, OpJumpIfFalse, OpCall, OpPrint, OpMakeArray}
	for _, op := range withOperand {
		if !op.hasOperand() {
			t.Errorf("%s.hasOperand() = false, want true", op)
		}
	}
	// Opcodes that take no operand.
	without := []Opcode{OpTrue, OpFalse, OpPop, OpAdd, OpReturn, OpReturnVoid, OpMissingReturn, OpIndex, OpLen}
	for _, op := range without {
		if op.hasOperand() {
			t.Errorf("%s.hasOperand() = true, want false", op)
		}
	}
}

// TestProgramString disassembles a small hand-built program and checks the
// format `chip dump` emits: a function header, then one line per instruction,
// with operands printed only for operand-carrying opcodes and the constant value
// shown beside OpConst.
func TestProgramString(t *testing.T) {
	pos := token.Pos{Line: 1, Column: 1}
	chunk := &Chunk{}
	chunk.Emit(OpConst, chunk.AddConstant(value.Int(42)), pos)
	chunk.Emit(OpReturn, 0, pos)
	prog := &Program{
		Functions: []*FuncProto{{Name: "main", NumParams: 0, NumLocals: 0, Chunk: chunk}},
		Entry:     0,
	}

	got := prog.String()
	for _, want := range []string{
		"fn 0 main/0 (locals 0)",
		"Const",
		"(42)", // the constant value rendered beside OpConst
		"Return",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("disassembly missing %q:\n%s", want, got)
		}
	}
	// OpReturn takes no operand, so its line must not print one. The constant
	// index 0 on the OpConst line is fine; check the Return line specifically.
	for line := range strings.SplitSeq(got, "\n") {
		if strings.Contains(line, "Return") && strings.Contains(line, " 0 (") {
			t.Errorf("Return line should not print an operand: %q", line)
		}
	}
}
