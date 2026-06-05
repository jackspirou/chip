package ast

import "github.com/jackspirou/chip/internal/token"

type (
	// Block is a brace-enclosed list of statements.
	Block struct {
		Lbrace token.Pos
		List   []Stmt
		Rbrace token.Pos
	}

	// DeclStmt is a short variable declaration: Name := Value.
	DeclStmt struct {
		Name   *Ident
		DefPos token.Pos
		Value  Expr
	}

	// AssignStmt is an assignment: Lhs Op Rhs (Op is currently only =).
	AssignStmt struct {
		Lhs   Expr
		OpPos token.Pos
		Op    token.Type
		Rhs   Expr
	}

	// ExprStmt is an expression used as a statement (e.g. a function call).
	ExprStmt struct {
		X Expr
	}

	// ReturnStmt is a return statement; Result may be nil.
	ReturnStmt struct {
		Return token.Pos
		Result Expr
	}

	// IfStmt is an if statement; Else may be nil, an *IfStmt, or a *Block.
	IfStmt struct {
		If   token.Pos
		Cond Expr
		Body *Block
		Else Stmt
	}

	// ForStmt is a for loop; a nil Cond means an infinite loop.
	ForStmt struct {
		For  token.Pos
		Cond Expr
		Body *Block
	}

	// BadStmt stands in for a statement that could not be parsed.
	BadStmt struct {
		From token.Pos
	}
)

func (s *Block) Pos() token.Pos      { return s.Lbrace }
func (s *DeclStmt) Pos() token.Pos   { return s.Name.Pos() }
func (s *AssignStmt) Pos() token.Pos { return s.Lhs.Pos() }
func (s *ExprStmt) Pos() token.Pos   { return s.X.Pos() }
func (s *ReturnStmt) Pos() token.Pos { return s.Return }
func (s *IfStmt) Pos() token.Pos     { return s.If }
func (s *ForStmt) Pos() token.Pos    { return s.For }
func (s *BadStmt) Pos() token.Pos    { return s.From }

func (s *Block) End() token.Pos { return after(s.Rbrace) }

func (s *DeclStmt) End() token.Pos {
	if s.Value != nil {
		return s.Value.End()
	}
	return endOf(s.DefPos, token.DEFINE.String())
}

func (s *AssignStmt) End() token.Pos {
	if s.Rhs != nil {
		return s.Rhs.End()
	}
	return endOf(s.OpPos, s.Op.String())
}

func (s *ExprStmt) End() token.Pos { return s.X.End() }

func (s *ReturnStmt) End() token.Pos {
	if s.Result != nil {
		return s.Result.End()
	}
	return endOf(s.Return, "return")
}

func (s *IfStmt) End() token.Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}

func (s *ForStmt) End() token.Pos { return s.Body.End() }
func (s *BadStmt) End() token.Pos { return s.From }

func (*Block) node()      {}
func (*DeclStmt) node()   {}
func (*AssignStmt) node() {}
func (*ExprStmt) node()   {}
func (*ReturnStmt) node() {}
func (*IfStmt) node()     {}
func (*ForStmt) node()    {}
func (*BadStmt) node()    {}

func (*Block) stmt()      {}
func (*DeclStmt) stmt()   {}
func (*AssignStmt) stmt() {}
func (*ExprStmt) stmt()   {}
func (*ReturnStmt) stmt() {}
func (*IfStmt) stmt()     {}
func (*ForStmt) stmt()    {}
func (*BadStmt) stmt()    {}
