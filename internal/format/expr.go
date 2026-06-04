package format

import (
	"strconv"
	"strings"

	"github.com/jackspirou/chip/internal/ast"
)

// exprString renders an expression as canonical chip source on a single line,
// inserting parentheses only where operator precedence requires them.
func exprString(e ast.Expr) string {
	switch e := e.(type) {
	case nil:
		return ""
	case *ast.Ident:
		return e.Name
	case *ast.IntLit:
		return e.Lit
	case *ast.FloatLit:
		return e.Lit
	case *ast.StringLit:
		return strconv.Quote(e.Value)
	case *ast.UnaryExpr:
		return e.Op.String() + unaryOperand(e.X)
	case *ast.BinaryExpr:
		prec := e.Op.Precedence()
		return binOperand(e.Left, prec, false) + " " + e.Op.String() + " " + binOperand(e.Right, prec, true)
	case *ast.CallExpr:
		return exprString(e.Fn) + "(" + exprList(e.Args) + ")"
	case *ast.IndexExpr:
		return exprString(e.X) + "[" + exprString(e.Index) + "]"
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len != nil {
			return "[" + exprString(e.Len) + "]" + exprString(e.Elem)
		}
		return "[]" + exprString(e.Elem)
	case *ast.CompositeLit:
		return exprString(e.Type) + "{" + exprList(e.Elems) + "}"
	case *ast.TypeName:
		return e.Name
	case *ast.BadExpr:
		return "/* bad */"
	default:
		return "?"
	}
}

func exprList(es []ast.Expr) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = exprString(e)
	}
	return strings.Join(parts, ", ")
}

// unaryOperand parenthesizes a unary operand when it is a binary expression
// (which binds more loosely than any unary operator).
func unaryOperand(e ast.Expr) string {
	if _, ok := e.(*ast.BinaryExpr); ok {
		return "(" + exprString(e) + ")"
	}
	return exprString(e)
}

// binOperand parenthesizes a binary operand to preserve precedence and
// left-associativity: a left operand needs parens when it binds more loosely,
// a right operand also when it binds equally.
func binOperand(e ast.Expr, parentPrec int, isRight bool) string {
	s := exprString(e)
	if b, ok := e.(*ast.BinaryExpr); ok {
		cp := b.Op.Precedence()
		if cp < parentPrec || (cp == parentPrec && isRight) {
			return "(" + s + ")"
		}
	}
	return s
}
