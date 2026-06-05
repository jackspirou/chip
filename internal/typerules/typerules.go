// Package typerules is the single source of truth for chip's leaf type rules:
// the operand/result rules of every operator, the type predicates those rules
// rest on, the control-flow "does this body always return" analysis, and the
// main-signature rule.
//
// Two checkers consume these rules. The batch checker (internal/check, behind
// chip check / chip lint / the embedding compiler) collects every error and
// resolves names through a scope tree. The streaming checker (internal/stream,
// behind chip run) fails fast and resolves names on demand. Those differences —
// error strategy and name resolution — are theirs to keep. The rules themselves
// must not differ: if chip check accepted an operator or a fall-through that
// chip run rejects, check would be lying about what run will do. Keeping the
// rules here, called from both, makes that drift impossible by construction
// rather than by vigilance, so check is never more lenient than run.
//
// Functions here are pure: they read only the AST and the type system and never
// emit errors. A rule that fails returns a complete, human-readable message; the
// caller decides whether to collect it or stop. The messages match what both
// checkers historically produced, so wiring them through is behavior-preserving
// for every program except the ones a checker was wrongly accepting.
package typerules

import (
	"fmt"
	"slices"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/types"
)

//
// type predicates
//

// BasicKind returns t's basic kind, or KindInvalid if t is not a basic type.
func BasicKind(t types.Type) types.Kind {
	if b, ok := t.(types.Basic); ok {
		return b.Kind
	}
	return types.KindInvalid
}

// IsInvalid reports whether t is the invalid type. A slice is never invalid
// (its element type might be, but the slice itself is a real type), so the
// slice check guards against a slice-of-invalid reading as invalid.
func IsInvalid(t types.Type) bool { return BasicKind(t) == types.KindInvalid && !IsSlice(t) }

// IsVoid reports whether t is the void type (the absence of a value).
func IsVoid(t types.Type) bool { return BasicKind(t) == types.KindVoid }

// IsBool reports whether t is the bool type.
func IsBool(t types.Type) bool { return BasicKind(t) == types.KindBool }

// IsInt reports whether t is the int type.
func IsInt(t types.Type) bool { return BasicKind(t) == types.KindInt }

// IsString reports whether t is the string type.
func IsString(t types.Type) bool { return BasicKind(t) == types.KindString }

// IsSlice reports whether t is a slice type.
func IsSlice(t types.Type) bool {
	_, ok := t.(types.Slice)
	return ok
}

// IsNumeric reports whether t is a numeric type (int or float).
func IsNumeric(t types.Type) bool {
	k := BasicKind(t)
	return k == types.KindInt || k == types.KindFloat
}

// IsOrdered reports whether t supports the ordered comparisons (< <= > >=):
// int, float, and string.
func IsOrdered(t types.Type) bool {
	k := BasicKind(t)
	return k == types.KindInt || k == types.KindFloat || k == types.KindString
}

//
// operator rules
//

// Binary returns the type of applying binary operator op to operands of types
// lt and rt. The second result is "" when the operation is well-typed and a
// complete error message otherwise; on error the returned type is the one the
// expression should still take (Bool for comparisons and logical operators, so
// a broken condition still reads as a condition; Invalid for arithmetic), which
// lets the collect-all checker keep going without cascading.
//
// Callers must rule out invalid operands first (an operand that already errored
// should not provoke a second, derived error); this function assumes lt and rt
// are real types.
func Binary(op token.Type, lt, rt types.Type) (types.Type, string) {
	switch op {
	case token.LAND, token.LOR:
		if !IsBool(lt) || !IsBool(rt) {
			return types.Bool, fmt.Sprintf("operator %s requires bool operands, got %s and %s", op, lt, rt)
		}
		return types.Bool, ""

	case token.EQL, token.NEQ:
		// Slices are not comparable. Identical slice types are types.Identical,
		// so this guard must precede the Identical check; without it slice ==
		// slice slips through and the executor compares raw value words, silently
		// yielding a wrong result.
		if IsSlice(lt) || IsSlice(rt) {
			return types.Bool, "slices are not comparable"
		}
		if !types.Identical(lt, rt) {
			return types.Bool, fmt.Sprintf("cannot compare %s and %s", lt, rt)
		}
		return types.Bool, ""

	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		if !types.Identical(lt, rt) || !IsOrdered(lt) {
			return types.Bool, fmt.Sprintf("operator %s is not defined for %s and %s", op, lt, rt)
		}
		return types.Bool, ""

	case token.ADD:
		if !types.Identical(lt, rt) || (!IsNumeric(lt) && !IsString(lt)) {
			return types.Invalid, fmt.Sprintf("operator + is not defined for %s and %s", lt, rt)
		}
		return lt, ""

	case token.SUB, token.MUL, token.QUO:
		if !types.Identical(lt, rt) || !IsNumeric(lt) {
			return types.Invalid, fmt.Sprintf("operator %s is not defined for %s and %s", op, lt, rt)
		}
		return lt, ""

	case token.REM:
		if !IsInt(lt) || !IsInt(rt) {
			return types.Invalid, fmt.Sprintf("operator %% requires int operands, got %s and %s", lt, rt)
		}
		return types.Int, ""

	default:
		// Bitwise and shift operators (& | ^ << >> &^) parse but are not part of
		// the language chip executes. The executor and the streaming checker
		// reject them; the batch checker once treated them as int operations,
		// which was the leniency this rule removes.
		return types.Invalid, fmt.Sprintf("operator %s is not supported", op)
	}
}

// Unary returns the type of applying unary operator op to an operand of type t,
// with the same (type, message) contract as Binary. Callers must rule out an
// invalid operand first.
func Unary(op token.Type, t types.Type) (types.Type, string) {
	switch op {
	case token.ADD, token.SUB:
		if !IsNumeric(t) {
			return types.Invalid, fmt.Sprintf("operator %s requires a numeric operand, got %s", op, t)
		}
		return t, ""
	case token.NOT:
		if !IsBool(t) {
			return types.Bool, fmt.Sprintf("operator ! requires a bool operand, got %s", t)
		}
		return types.Bool, ""
	default:
		return types.Invalid, fmt.Sprintf("operator %s is not supported", op)
	}
}

//
// control flow
//

// Terminates reports whether a statement list always transfers control away —
// every path returns, or the list ends in a loop that never falls through — so
// control cannot reach the end of the list. A function with a declared result
// must terminate, or it would fall off the end and produce a type-confused zero
// value; both checkers and the linter use this to enforce that.
func Terminates(list []ast.Stmt) bool {
	return slices.ContainsFunc(list, TerminatesStmt)
}

// TerminatesStmt reports whether a single statement always transfers control
// away. The linter uses it to find the first dead statement after a terminator;
// Terminates uses it to decide whether a whole body falls through.
func TerminatesStmt(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		return s.Else != nil && Terminates(s.Body.List) && TerminatesStmt(s.Else)
	case *ast.Block:
		return Terminates(s.List)
	case *ast.ForStmt:
		return s.Cond == nil // an infinite loop never falls through (chip has no break)
	default:
		return false
	}
}

//
// declarations
//

// MainTakesArgs reports whether fn is the entry point main declared with
// parameters, which chip forbids: main must take no arguments.
func MainTakesArgs(fn *ast.FuncDecl) bool {
	return fn.Name != nil && fn.Name.Name == "main" && len(fn.Params) > 0
}
