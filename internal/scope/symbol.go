package scope

import (
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/types"
)

// Kind classifies what a symbol denotes.
type Kind int

const (
	Var     Kind = iota // a local variable (:=)
	Param               // a function parameter
	Func                // a function
	Builtin             // a predeclared builtin (e.g. print)
	Pkg                 // an imported package name
)

// Symbol is a named program entity with a type.
type Symbol struct {
	Name    string
	Kind    Kind
	Type    types.Type
	DeclPos token.Pos
}
