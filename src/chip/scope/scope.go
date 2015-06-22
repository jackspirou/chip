package scope

import (
	"errors"
	"fmt"

	"github.com/jackspirou/chip/src/chip/node"
	"github.com/jackspirou/gostack"
)

// Scope. Manage scopes.
type Scope struct {
	stack *gostack.Stack
}

// NewScope.  Create a new instance of Scope.
func NewScope() *Scope {
	return &Scope{
		stack: gostack.NewStack(),
	}
}

// Scope, Empty. Tests if Scope is empty.
func (s *Scope) Empty() bool {
	return s.stack.Empty()
}

// Scope, Open. Push a new symbol table on the Scope stack.
func (s *Scope) Open() {
	s.stack.Push(NewSymTab())
}

// Scope, Close. Pop a symbol table off the scope stack.
func (s *Scope) Close() (*SymTab, error) {
	if s.Empty() {
		return nil, errors.New("Could not pop an empty scope stack.")
	} else {
		i, err := s.stack.Pop()
		return i.Item.(*SymTab), err
	}
}

// Scope, SetGlobal. Set a descriptor in the global scope.
func (s *Scope) Global(name string, node node.Node) (bool, error) {
	result, err := s.stack.Bottom()
	if err != nil {
		return false, err
	}
	gsymtab := result.Item.(*SymTab)
	if gsymtab.Contains(name) {
		return false, errors.New("'" + name + "' cannot be declared globally twice.")
	}
	err = gsymtab.Set(name, node)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Scope, Contains. Tests if name appears in a scope.
func (s *Scope) Contains(name string) bool {
	result, err := s.stack.Peek()
	if err != nil {
		return false
	}
	symtab := result.Item.(*SymTab)
	return symtab.Contains(name)
}

// Scope, GetDes. Returns the first descriptor in scope.
func (s *Scope) Lookup(name string) (node.Node, error) {
	if s.Empty() {
		return nil, errors.New("Could not pop an empty scope stack.")
	}
	listcpy := s.stack.GetList()
	for _, value := range listcpy {
		symtab := value.Item.(*SymTab)
		if symtab.Contains(name) {
			return symtab.Get(name)
		}
	}
	return nil, errors.New("'" + name + "' has not yet been declared.")
}

// Set Descriptor. Adds a name to the topmost scope.
func (s *Scope) Insert(name string, node node.Node) error {
	if s.Contains(name) {
		return errors.New("'" + name + "' cannot be declared twice.")
	}
	result, err := s.stack.Peek()
	if err != nil {
		return err
	}
	symtab := result.Item.(*SymTab)
	err = symtab.Set(name, node)
	if err != nil {
		return err
	}
	return nil
}

func (s *Scope) String() string {
	return s.stack.String()
}

func (s *Scope) Print() {
	fmt.Print(s.String)
}
