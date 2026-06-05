// Package vm executes compiled chip bytecode on a stack machine.
package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

// maxFrames bounds recursion depth so runaway recursion fails cleanly.
const maxFrames = 1 << 16

// RuntimeError is an error raised while executing a program, carrying the
// source position of the offending instruction.
type RuntimeError struct {
	Pos token.Pos
	Msg string
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

type frame struct {
	proto  *code.FuncProto
	ip     int
	locals []value.Value
}

type vm struct {
	prog   *code.Program
	out    io.Writer
	stack  []value.Value
	frames []*frame
}

// Run executes prog's entry point (main), writing program output to out.
func Run(prog *code.Program, out io.Writer) error {
	if prog.Entry < 0 {
		return fmt.Errorf("no main function to run")
	}
	if prog.Functions[prog.Entry].NumParams != 0 {
		return fmt.Errorf("main must have no parameters")
	}
	m := &vm{prog: prog, out: out, stack: make([]value.Value, 0, 256)}
	m.call(prog.Entry)
	return m.run()
}

func (m *vm) push(v value.Value) { m.stack = append(m.stack, v) }

func (m *vm) pop() value.Value {
	n := len(m.stack) - 1
	v := m.stack[n]
	m.stack = m.stack[:n]
	return v
}

// call pushes a new frame for Functions[idx], moving arguments off the operand
// stack into the new frame's local slots.
func (m *vm) call(idx int) {
	proto := m.prog.Functions[idx]
	locals := make([]value.Value, proto.NumLocals)
	for i := proto.NumParams - 1; i >= 0; i-- {
		locals[i] = m.pop()
	}
	m.frames = append(m.frames, &frame{proto: proto, locals: locals})
}

func (m *vm) popFrame() { m.frames = m.frames[:len(m.frames)-1] }

func (m *vm) run() error {
	for len(m.frames) > 0 {
		fr := m.frames[len(m.frames)-1]
		if fr.ip >= len(fr.proto.Chunk.Code) {
			m.popFrame() // fell off the end: return void
			continue
		}
		inst := fr.proto.Chunk.Code[fr.ip]
		fr.ip++

		switch inst.Op {
		case code.OpConst:
			m.push(fr.proto.Chunk.Constants[inst.Operand])
		case code.OpTrue:
			m.push(value.Bool(true))
		case code.OpFalse:
			m.push(value.Bool(false))
		case code.OpPop:
			m.pop()
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpRem:
			if err := m.arith(inst.Op, fr); err != nil {
				return err
			}
		case code.OpNeg:
			a := m.pop()
			if a.Kind == value.KindFloat {
				m.push(value.Float(-a.AsFloat()))
			} else {
				m.push(value.Int(-a.AsInt()))
			}
		case code.OpNot:
			m.push(value.Bool(!m.pop().AsBool()))
		case code.OpEqual, code.OpNotEqual, code.OpLess, code.OpLessEqual, code.OpGreater, code.OpGreaterEqual:
			if err := m.compare(inst.Op, fr); err != nil {
				return err
			}
		case code.OpGetLocal:
			m.push(fr.locals[inst.Operand])
		case code.OpSetLocal:
			fr.locals[inst.Operand] = m.pop()
		case code.OpJump:
			fr.ip = inst.Operand
		case code.OpJumpIfFalse:
			if !m.pop().AsBool() {
				fr.ip = inst.Operand
			}
		case code.OpCall:
			if len(m.frames) >= maxFrames {
				return m.runtimeErr(fr, "stack overflow")
			}
			m.call(inst.Operand)
		case code.OpReturn:
			result := m.pop()
			m.popFrame()
			m.push(result)
		case code.OpReturnVoid:
			m.popFrame()
		case code.OpMissingReturn:
			// A function with a declared result reached the end of its body
			// without returning. Reporting a clean runtime error here keeps the
			// caller from popping a return value that was never pushed, which
			// would underflow the stack and panic the embedding host.
			return m.runtimeErr(fr, "missing return at end of function %s", fr.proto.Name)
		case code.OpPrint:
			m.doPrint(inst.Operand)
		case code.OpMakeArray:
			n := inst.Operand
			elems := make([]value.Value, n)
			for i := n - 1; i >= 0; i-- {
				elems[i] = m.pop()
			}
			m.push(value.Slice(elems))
		case code.OpIndex:
			idx := m.pop()
			arr := m.pop()
			s := arr.AsSlice()
			i := idx.AsInt()
			if i < 0 || i >= int64(len(s)) {
				return m.runtimeErr(fr, "index out of range: %d (length %d)", i, len(s))
			}
			m.push(s[i])
		case code.OpSetIndex:
			val := m.pop()
			idx := m.pop()
			arr := m.pop()
			s := arr.AsSlice()
			i := idx.AsInt()
			if i < 0 || i >= int64(len(s)) {
				return m.runtimeErr(fr, "index out of range: %d (length %d)", i, len(s))
			}
			s[i] = val
		case code.OpLen:
			v := m.pop()
			if v.Kind == value.KindString {
				m.push(value.Int(int64(len(v.AsStr()))))
			} else {
				m.push(value.Int(int64(len(v.AsSlice()))))
			}
		default:
			return m.runtimeErr(fr, "unknown opcode %s", inst.Op)
		}
	}
	return nil
}

func (m *vm) arith(op code.Opcode, fr *frame) error {
	b := m.pop()
	a := m.pop()

	switch {
	case a.Kind == value.KindString || b.Kind == value.KindString:
		// The checker allows only + on strings; guard the rest so a lenient
		// embedding host cannot turn string subtraction into silent concatenation.
		if op != code.OpAdd {
			return m.runtimeErr(fr, "operator %s is not defined for string", op)
		}
		m.push(value.Str(a.AsStr() + b.AsStr()))
	case a.Kind == value.KindFloat || b.Kind == value.KindFloat:
		x, y := a.AsFloat(), b.AsFloat()
		switch op {
		case code.OpAdd:
			m.push(value.Float(x + y))
		case code.OpSub:
			m.push(value.Float(x - y))
		case code.OpMul:
			m.push(value.Float(x * y))
		case code.OpDiv:
			if y == 0 {
				return m.runtimeErr(fr, "division by zero")
			}
			m.push(value.Float(x / y))
		default:
			// % is the one arithmetic operator the checker forbids on floats.
			return m.runtimeErr(fr, "operator %s is not defined for float", op)
		}
	default:
		x, y := a.AsInt(), b.AsInt()
		switch op {
		case code.OpAdd:
			m.push(value.Int(x + y))
		case code.OpSub:
			m.push(value.Int(x - y))
		case code.OpMul:
			m.push(value.Int(x * y))
		case code.OpDiv:
			if y == 0 {
				return m.runtimeErr(fr, "division by zero")
			}
			m.push(value.Int(x / y))
		case code.OpRem:
			if y == 0 {
				return m.runtimeErr(fr, "division by zero")
			}
			m.push(value.Int(x % y))
		default:
			return m.runtimeErr(fr, "operator %s is not defined for int", op)
		}
	}
	return nil
}

func (m *vm) compare(op code.Opcode, fr *frame) error {
	b := m.pop()
	a := m.pop()
	var res bool
	switch {
	case a.Kind == value.KindString || b.Kind == value.KindString:
		res = ordered(op, a.AsStr(), b.AsStr())
	case a.Kind == value.KindFloat || b.Kind == value.KindFloat:
		res = ordered(op, a.AsFloat(), b.AsFloat())
	case a.Kind == value.KindBool || b.Kind == value.KindBool:
		// Booleans support only == and !=; the checker rejects ordering them, so
		// fail cleanly rather than push a default-false result if one slips through.
		switch op {
		case code.OpEqual:
			res = a.AsBool() == b.AsBool()
		case code.OpNotEqual:
			res = a.AsBool() != b.AsBool()
		default:
			return m.runtimeErr(fr, "operator %s is not defined for bool", op)
		}
	default:
		res = ordered(op, a.AsInt(), b.AsInt())
	}
	m.push(value.Bool(res))
	return nil
}

// ordered evaluates a comparison opcode over any ordered operand type.
func ordered[T int64 | float64 | string](op code.Opcode, x, y T) bool {
	switch op {
	case code.OpEqual:
		return x == y
	case code.OpNotEqual:
		return x != y
	case code.OpLess:
		return x < y
	case code.OpLessEqual:
		return x <= y
	case code.OpGreater:
		return x > y
	case code.OpGreaterEqual:
		return x >= y
	}
	return false
}

func (m *vm) doPrint(n int) {
	args := make([]value.Value, n)
	for i := n - 1; i >= 0; i-- {
		args[i] = m.pop()
	}
	parts := make([]string, n)
	for i, a := range args {
		parts[i] = a.String()
	}
	fmt.Fprintln(m.out, strings.Join(parts, " "))
}

func (m *vm) runtimeErr(fr *frame, format string, args ...any) error {
	return RuntimeError{
		Pos: fr.proto.Chunk.Lines[fr.ip-1],
		Msg: fmt.Sprintf(format, args...),
	}
}
