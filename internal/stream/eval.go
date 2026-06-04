package stream

import (
	"fmt"
	"strings"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/value"
)

// eval evaluates an expression to a value. The Go call stack carries the
// intermediate results — there is no operand stack (plan §3.5).
func (e *engine) eval(x ast.Expr, scope *env) (value.Value, error) {
	switch x := x.(type) {
	case *ast.IntLit:
		return value.Int(x.Value), nil
	case *ast.FloatLit:
		return value.Float(x.Value), nil
	case *ast.StringLit:
		return value.Str(x.Value), nil
	case *ast.Ident:
		if v, ok := scope.lookup(x.Name); ok {
			return v, nil
		}
		return value.Value{}, e.errorf(x.Pos(), "undefined: %s", x.Name)
	case *ast.UnaryExpr:
		return e.evalUnary(x, scope)
	case *ast.BinaryExpr:
		return e.evalBinary(x, scope)
	case *ast.CallExpr:
		return e.evalCall(x, scope)
	case *ast.CompositeLit:
		return e.evalCompositeLit(x, scope)
	case *ast.IndexExpr:
		return e.evalIndex(x, scope)
	case *ast.BadExpr:
		return value.Value{}, e.errorf(x.Pos(), "malformed expression")
	default:
		return value.Value{}, e.errorf(x.Pos(), "cannot evaluate %T", x)
	}
}

func (e *engine) evalUnary(x *ast.UnaryExpr, scope *env) (value.Value, error) {
	v, err := e.eval(x.X, scope)
	if err != nil {
		return value.Value{}, err
	}
	switch x.Op {
	case token.SUB:
		if v.Kind == value.KindFloat {
			return value.Float(-v.AsFloat()), nil
		}
		return value.Int(-v.AsInt()), nil
	case token.NOT:
		return value.Bool(!v.AsBool()), nil
	case token.ADD:
		return v, nil // unary plus is a no-op
	default:
		return value.Value{}, e.errorf(x.OpPos, "operator %s is not supported", x.Op)
	}
}

func (e *engine) evalBinary(x *ast.BinaryExpr, scope *env) (value.Value, error) {
	// && and || short-circuit, so they evaluate the right operand lazily.
	switch x.Op {
	case token.LAND:
		left, err := e.eval(x.Left, scope)
		if err != nil {
			return value.Value{}, err
		}
		if !left.AsBool() {
			return value.Bool(false), nil
		}
		right, err := e.eval(x.Right, scope)
		if err != nil {
			return value.Value{}, err
		}
		return value.Bool(right.AsBool()), nil
	case token.LOR:
		left, err := e.eval(x.Left, scope)
		if err != nil {
			return value.Value{}, err
		}
		if left.AsBool() {
			return value.Bool(true), nil
		}
		right, err := e.eval(x.Right, scope)
		if err != nil {
			return value.Value{}, err
		}
		return value.Bool(right.AsBool()), nil
	}

	left, err := e.eval(x.Left, scope)
	if err != nil {
		return value.Value{}, err
	}
	right, err := e.eval(x.Right, scope)
	if err != nil {
		return value.Value{}, err
	}

	switch x.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		return e.arith(x.Op, left, right, x.OpPos)
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return compare(x.Op, left, right), nil
	default:
		return value.Value{}, e.errorf(x.OpPos, "operator %s is not supported", x.Op)
	}
}

// arith applies an arithmetic operator, mirroring the VM: + concatenates
// strings, a float operand promotes to float, otherwise integer math.
func (e *engine) arith(op token.Type, a, b value.Value, pos token.Pos) (value.Value, error) {
	if a.Kind == value.KindString || b.Kind == value.KindString {
		if op == token.ADD {
			return value.Str(a.AsStr() + b.AsStr()), nil
		}
		return value.Value{}, e.errorf(pos, "operator %s is not defined on string", op)
	}
	if a.Kind == value.KindFloat || b.Kind == value.KindFloat {
		x, y := a.AsFloat(), b.AsFloat()
		switch op {
		case token.ADD:
			return value.Float(x + y), nil
		case token.SUB:
			return value.Float(x - y), nil
		case token.MUL:
			return value.Float(x * y), nil
		case token.QUO:
			if y == 0 {
				return value.Value{}, e.errorf(pos, "division by zero")
			}
			return value.Float(x / y), nil
		}
		return value.Value{}, e.errorf(pos, "operator %s is not defined on float", op)
	}
	x, y := a.AsInt(), b.AsInt()
	switch op {
	case token.ADD:
		return value.Int(x + y), nil
	case token.SUB:
		return value.Int(x - y), nil
	case token.MUL:
		return value.Int(x * y), nil
	case token.QUO:
		if y == 0 {
			return value.Value{}, e.errorf(pos, "division by zero")
		}
		return value.Int(x / y), nil
	case token.REM:
		if y == 0 {
			return value.Value{}, e.errorf(pos, "division by zero")
		}
		return value.Int(x % y), nil
	}
	return value.Value{}, e.errorf(pos, "operator %s is not supported", op)
}

// compare applies a comparison operator over any ordered operand type (bool
// supports only == and !=), mirroring the VM.
func compare(op token.Type, a, b value.Value) value.Value {
	switch {
	case a.Kind == value.KindString || b.Kind == value.KindString:
		return value.Bool(ordered(op, a.AsStr(), b.AsStr()))
	case a.Kind == value.KindFloat || b.Kind == value.KindFloat:
		return value.Bool(ordered(op, a.AsFloat(), b.AsFloat()))
	case a.Kind == value.KindBool || b.Kind == value.KindBool:
		switch op {
		case token.NEQ:
			return value.Bool(a.AsBool() != b.AsBool())
		default: // EQL
			return value.Bool(a.AsBool() == b.AsBool())
		}
	default:
		return value.Bool(ordered(op, a.AsInt(), b.AsInt()))
	}
}

func ordered[T int64 | float64 | string](op token.Type, x, y T) bool {
	switch op {
	case token.EQL:
		return x == y
	case token.NEQ:
		return x != y
	case token.LSS:
		return x < y
	case token.LEQ:
		return x <= y
	case token.GTR:
		return x > y
	case token.GEQ:
		return x >= y
	}
	return false
}

func (e *engine) evalCall(x *ast.CallExpr, scope *env) (value.Value, error) {
	id, ok := x.Fn.(*ast.Ident)
	if !ok {
		return value.Value{}, e.errorf(x.Fn.Pos(), "only direct function calls are supported")
	}

	switch id.Name {
	case "print":
		return e.callPrint(x, scope)
	case "len":
		return e.callLen(x, scope)
	}

	// A user function may be referenced before it streams in; resolve reads
	// ahead until it arrives or the stream ends.
	fn, err := e.resolve(id.Name)
	if err != nil {
		return value.Value{}, err
	}
	if fn == nil {
		return value.Value{}, e.errorf(id.Pos(), "undefined: %s", id.Name)
	}

	args := make([]value.Value, len(x.Args))
	for i, a := range x.Args {
		v, err := e.eval(a, scope)
		if err != nil {
			return value.Value{}, err
		}
		args[i] = v
	}
	return e.callFunc(fn, args, x.Lparen)
}

// callPrint evaluates each argument and writes them space-separated, newline
// terminated, to the engine's output.
func (e *engine) callPrint(x *ast.CallExpr, scope *env) (value.Value, error) {
	parts := make([]string, len(x.Args))
	for i, a := range x.Args {
		v, err := e.eval(a, scope)
		if err != nil {
			return value.Value{}, err
		}
		parts[i] = v.String()
	}
	fmt.Fprintln(e.out, strings.Join(parts, " "))
	return value.Value{}, nil
}

// callLen returns the length of a string or slice.
func (e *engine) callLen(x *ast.CallExpr, scope *env) (value.Value, error) {
	if len(x.Args) != 1 {
		return value.Value{}, e.errorf(x.Lparen, "len expects 1 argument, got %d", len(x.Args))
	}
	v, err := e.eval(x.Args[0], scope)
	if err != nil {
		return value.Value{}, err
	}
	if v.Kind == value.KindString {
		return value.Int(int64(len(v.AsStr()))), nil
	}
	return value.Int(int64(len(v.AsSlice()))), nil
}

func (e *engine) evalCompositeLit(x *ast.CompositeLit, scope *env) (value.Value, error) {
	elems := make([]value.Value, len(x.Elems))
	for i, el := range x.Elems {
		v, err := e.eval(el, scope)
		if err != nil {
			return value.Value{}, err
		}
		elems[i] = v
	}
	return value.Slice(elems), nil
}

func (e *engine) evalIndex(x *ast.IndexExpr, scope *env) (value.Value, error) {
	arr, err := e.eval(x.X, scope)
	if err != nil {
		return value.Value{}, err
	}
	idx, err := e.eval(x.Index, scope)
	if err != nil {
		return value.Value{}, err
	}
	elems := arr.AsSlice()
	i := idx.AsInt()
	if i < 0 || i >= int64(len(elems)) {
		return value.Value{}, e.errorf(x.Lbrack, "index out of range: %d (length %d)", i, len(elems))
	}
	return elems[i], nil
}
