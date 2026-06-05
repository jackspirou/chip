// Package ast declares the types used to represent syntax trees for chip
// source files. Nodes are purely syntactic; semantic information (resolved
// names and types) is attached by later passes.
package ast

import (
	"unicode/utf8"

	"github.com/jackspirou/chip/internal/token"
)

// Node is implemented by all AST nodes.
type Node interface {
	// Pos returns the position of the node's first token.
	Pos() token.Pos
	// End returns the position one past the node's last character, so the
	// half-open byte range [Pos().Offset, End().Offset) is the node's exact
	// source span (plan slice 3.0b). Positions carry byte offsets recorded by
	// the scanner, so spans are byte-exact, not approximated by token length.
	End() token.Pos
	node()
}

// endOf returns the position one past a single-line token that starts at pos and
// whose source text is text — a leaf node's end (identifiers, literals, named
// types). Column counts runes; Offset counts bytes, so multi-byte runes widen
// the byte span without inflating the column.
func endOf(pos token.Pos, text string) token.Pos {
	return token.Pos{
		Line:   pos.Line,
		Column: pos.Column + utf8.RuneCountInString(text),
		Offset: pos.Offset + len(text),
	}
}

// after returns the position one past a single-byte delimiter at pos — the ')',
// ']', or '}' that closes a bracketed node — i.e. the start of what follows it.
func after(pos token.Pos) token.Pos {
	return token.Pos{Line: pos.Line, Column: pos.Column + 1, Offset: pos.Offset + 1}
}

// maxPos returns whichever position sits later in the source (by byte offset).
func maxPos(a, b token.Pos) token.Pos {
	if b.Offset > a.Offset {
		return b
	}
	return a
}

// Expr is implemented by all expression nodes.
type Expr interface {
	Node
	expr()
}

// Stmt is implemented by all statement nodes.
type Stmt interface {
	Node
	stmt()
}

// Decl is implemented by all top-level declaration nodes.
type Decl interface {
	Node
	decl()
}
