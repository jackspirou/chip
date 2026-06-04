package ast

import "github.com/jackspirou/chip/internal/token"

type (
	// Ident is an identifier such as a variable, function, or type name.
	Ident struct {
		NamePos token.Pos
		Name    string
	}

	// IntLit is an integer literal.
	IntLit struct {
		ValuePos token.Pos
		Value    int64
		Lit      string // original literal text
	}

	// FloatLit is a floating-point literal.
	FloatLit struct {
		ValuePos token.Pos
		Value    float64
		Lit      string // original literal text
	}

	// StringLit is a string literal (without surrounding quotes).
	StringLit struct {
		ValuePos token.Pos
		Value    string
	}

	// UnaryExpr is a unary expression: Op X (e.g. -x, !ok).
	UnaryExpr struct {
		OpPos token.Pos
		Op    token.Type
		X     Expr
	}

	// BinaryExpr is a binary expression: Left Op Right (e.g. a + b, x < y).
	BinaryExpr struct {
		Left  Expr
		OpPos token.Pos
		Op    token.Type
		Right Expr
	}

	// CallExpr is a function call: Fn(Args).
	CallExpr struct {
		Fn     Expr
		Lparen token.Pos
		Args   []Expr
		Rparen token.Pos
	}

	// IndexExpr is an index expression: X[Index].
	IndexExpr struct {
		X      Expr
		Lbrack token.Pos
		Index  Expr
		Rbrack token.Pos
	}

	// SelectorExpr is a selector: X.Sel (e.g. fmt.Println).
	SelectorExpr struct {
		X   Expr
		Sel *Ident
	}

	// ArrayType is a slice or array type: []Elem, or [Len]Elem.
	ArrayType struct {
		Lbrack token.Pos
		Len    Expr // nil for a slice type ([]Elem)
		Elem   Expr
	}

	// CompositeLit is a typed composite literal: Type{Elems}.
	CompositeLit struct {
		Type   Expr // *ArrayType
		Lbrace token.Pos
		Elems  []Expr
		Rbrace token.Pos
	}

	// TypeName is a syntactic reference to a named type (e.g. int, string).
	// A later pass resolves it to a types.Type.
	TypeName struct {
		NamePos token.Pos
		Name    string
	}

	// BadExpr stands in for an expression that could not be parsed.
	BadExpr struct {
		From token.Pos
	}
)

func (e *Ident) Pos() token.Pos        { return e.NamePos }
func (e *IntLit) Pos() token.Pos       { return e.ValuePos }
func (e *FloatLit) Pos() token.Pos     { return e.ValuePos }
func (e *StringLit) Pos() token.Pos    { return e.ValuePos }
func (e *UnaryExpr) Pos() token.Pos    { return e.OpPos }
func (e *BinaryExpr) Pos() token.Pos   { return e.Left.Pos() }
func (e *CallExpr) Pos() token.Pos     { return e.Fn.Pos() }
func (e *IndexExpr) Pos() token.Pos    { return e.X.Pos() }
func (e *SelectorExpr) Pos() token.Pos { return e.X.Pos() }
func (e *ArrayType) Pos() token.Pos    { return e.Lbrack }
func (e *CompositeLit) Pos() token.Pos { return e.Type.Pos() }
func (e *TypeName) Pos() token.Pos     { return e.NamePos }
func (e *BadExpr) Pos() token.Pos      { return e.From }

func (*Ident) node()        {}
func (*IntLit) node()       {}
func (*FloatLit) node()     {}
func (*StringLit) node()    {}
func (*UnaryExpr) node()    {}
func (*BinaryExpr) node()   {}
func (*CallExpr) node()     {}
func (*IndexExpr) node()    {}
func (*SelectorExpr) node() {}
func (*ArrayType) node()    {}
func (*CompositeLit) node() {}
func (*TypeName) node()     {}
func (*BadExpr) node()      {}

func (*Ident) expr()        {}
func (*IntLit) expr()       {}
func (*FloatLit) expr()     {}
func (*StringLit) expr()    {}
func (*UnaryExpr) expr()    {}
func (*BinaryExpr) expr()   {}
func (*CallExpr) expr()     {}
func (*IndexExpr) expr()    {}
func (*SelectorExpr) expr() {}
func (*ArrayType) expr()    {}
func (*CompositeLit) expr() {}
func (*TypeName) expr()     {}
func (*BadExpr) expr()      {}
