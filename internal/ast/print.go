package ast

import (
	"strconv"
	"strings"
)

// Sprint returns an S-expression rendering of an AST node, intended for
// debugging and tests. Declarations in a File are placed on their own lines;
// everything else renders on a single line.
func Sprint(n Node) string {
	switch n := n.(type) {
	case nil:
		return "nil"

	case *File:
		pkg := "_"
		if n.Package != nil {
			pkg = n.Package.Name
		}
		var b strings.Builder
		b.WriteString("(file pkg=" + pkg)
		for _, im := range n.Imports {
			b.WriteString("\n  " + Sprint(im))
		}
		for _, d := range n.Decls {
			b.WriteString("\n  " + Sprint(d))
		}
		for _, s := range n.Stmts {
			b.WriteString("\n  " + Sprint(s))
		}
		b.WriteString(")")
		return b.String()

	case *ImportSpec:
		s := "(import "
		if n.Name != nil {
			s += n.Name.Name + " "
		}
		if n.Path != nil {
			s += strconv.Quote(n.Path.Value)
		}
		return s + ")"

	case *FuncDecl:
		var b strings.Builder
		b.WriteString("(func " + n.Name.Name + " (params")
		for _, f := range n.Params {
			b.WriteString(" " + Sprint(f))
		}
		b.WriteString(") (results")
		for _, f := range n.Results {
			b.WriteString(" " + Sprint(f))
		}
		b.WriteString(") " + Sprint(n.Body) + ")")
		return b.String()

	case *Field:
		name := "_"
		if n.Name != nil {
			name = n.Name.Name
		}
		return "(" + name + " " + Sprint(n.Type) + ")"

	case *Block:
		var b strings.Builder
		b.WriteString("(block")
		for _, s := range n.List {
			b.WriteString(" " + Sprint(s))
		}
		b.WriteString(")")
		return b.String()

	case *DeclStmt:
		return "(:= " + n.Name.Name + " " + Sprint(n.Value) + ")"

	case *AssignStmt:
		return "(" + n.Op.String() + " " + Sprint(n.Lhs) + " " + Sprint(n.Rhs) + ")"

	case *ExprStmt:
		return "(expr " + Sprint(n.X) + ")"

	case *ReturnStmt:
		if n.Result == nil {
			return "(return)"
		}
		return "(return " + Sprint(n.Result) + ")"

	case *IfStmt:
		s := "(if " + Sprint(n.Cond) + " " + Sprint(n.Body)
		if n.Else != nil {
			s += " " + Sprint(n.Else)
		}
		return s + ")"

	case *ForStmt:
		s := "(for "
		if n.Cond != nil {
			s += Sprint(n.Cond) + " "
		}
		return s + Sprint(n.Body) + ")"

	case *BadStmt:
		return "(badstmt)"

	case *Ident:
		return n.Name

	case *IntLit:
		return n.Lit

	case *FloatLit:
		return n.Lit

	case *StringLit:
		return strconv.Quote(n.Value)

	case *UnaryExpr:
		return "(" + n.Op.String() + " " + Sprint(n.X) + ")"

	case *BinaryExpr:
		return "(" + n.Op.String() + " " + Sprint(n.Left) + " " + Sprint(n.Right) + ")"

	case *CallExpr:
		var b strings.Builder
		b.WriteString("(call " + Sprint(n.Fn))
		for _, a := range n.Args {
			b.WriteString(" " + Sprint(a))
		}
		b.WriteString(")")
		return b.String()

	case *IndexExpr:
		return "(index " + Sprint(n.X) + " " + Sprint(n.Index) + ")"

	case *SelectorExpr:
		return "(sel " + Sprint(n.X) + " " + n.Sel.Name + ")"

	case *ArrayType:
		if n.Len != nil {
			return "[" + Sprint(n.Len) + "]" + Sprint(n.Elem)
		}
		return "[]" + Sprint(n.Elem)

	case *CompositeLit:
		var b strings.Builder
		b.WriteString("(lit " + Sprint(n.Type))
		for _, el := range n.Elems {
			b.WriteString(" " + Sprint(el))
		}
		b.WriteString(")")
		return b.String()

	case *TypeName:
		return n.Name

	case *BadExpr:
		return "(badexpr)"

	default:
		return "?"
	}
}
