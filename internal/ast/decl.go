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
	if d.Package != nil {
		return d.Package.Pos()
	}
	if len(d.Decls) > 0 {
		return d.Decls[0].Pos()
	}
	return token.Pos{}
}

func (*Field) node()      {}
func (*FuncDecl) node()   {}
func (*ImportSpec) node() {}
func (*File) node()       {}

func (*FuncDecl) decl() {}
