// Package ast declares the types used to represent syntax trees for chip
// source files. Nodes are purely syntactic; semantic information (resolved
// names and types) is attached by later passes.
package ast

import "github.com/jackspirou/chip/internal/token"

// Node is implemented by all AST nodes.
type Node interface {
	// Pos returns the position of the node's first token.
	Pos() token.Pos
	node()
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
