package ast

import "github.com/jackspirou/chip/internal/token"

type (
	// Field is a name/type pair in a function signature. Name may be nil for
	// an unnamed result.
	Field struct {
		Name *Ident
		Type Expr // *TypeName
	}

	// FuncDecl is a function declaration.
	FuncDecl struct {
		Func    token.Pos
		Name    *Ident
		Params  []*Field
		Results []*Field
		Body    *Block
	}

	// ImportSpec is a single import; Name (alias) may be nil.
	ImportSpec struct {
		Name *Ident
		Path *StringLit
	}

	// File is a parsed chip source file.
	File struct {
		Package  *Ident
		Imports  []*ImportSpec
		Decls    []Decl
		Stmts    []Stmt     // top-level statements, in source order
		Comments []*Comment // all comments, in source order
	}
)

func (d *Field) Pos() token.Pos {
	if d.Name != nil {
		return d.Name.Pos()
	}
	if d.Type != nil {
		return d.Type.Pos()
	}
	return token.Pos{}
}

func (d *FuncDecl) Pos() token.Pos { return d.Func }

func (d *ImportSpec) Pos() token.Pos {
	if d.Name != nil {
		return d.Name.Pos()
	}
	if d.Path != nil {
		return d.Path.Pos()
	}
	return token.Pos{}
}

func (d *File) Pos() token.Pos {
	switch {
	case d.Package != nil:
		return d.Package.Pos()
	case len(d.Decls) > 0:
		return d.Decls[0].Pos()
	case len(d.Stmts) > 0:
		return d.Stmts[0].Pos()
	default:
		return token.Pos{}
	}
}

func (d *Field) End() token.Pos {
	if d.Type != nil {
		return d.Type.End()
	}
	if d.Name != nil {
		return d.Name.End()
	}
	return token.Pos{}
}

func (d *FuncDecl) End() token.Pos {
	if d.Body != nil {
		return d.Body.End()
	}
	if n := len(d.Results); n > 0 {
		return d.Results[n-1].End()
	}
	if d.Name != nil {
		return d.Name.End()
	}
	return endOf(d.Func, "func")
}

func (d *ImportSpec) End() token.Pos {
	if d.Path != nil {
		return d.Path.End()
	}
	if d.Name != nil {
		return d.Name.End()
	}
	return token.Pos{}
}

// End returns the end of the file's last top-level element. Decls and Stmts are
// stored separately but interleave in source, so the furthest end by byte offset
// wins.
func (d *File) End() token.Pos {
	end := token.Pos{}
	if d.Package != nil {
		end = maxPos(end, d.Package.End())
	}
	for _, im := range d.Imports {
		end = maxPos(end, im.End())
	}
	for _, dc := range d.Decls {
		end = maxPos(end, dc.End())
	}
	for _, st := range d.Stmts {
		end = maxPos(end, st.End())
	}
	return end
}

func (*Field) node()      {}
func (*FuncDecl) node()   {}
func (*ImportSpec) node() {}
func (*File) node()       {}

func (*FuncDecl) decl() {}
