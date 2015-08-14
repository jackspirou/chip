package scope

import (
	"errors"
	"fmt"

	"github.com/jackspirou/chip/src/chip/node"
)

// Scope describes multiple variable scopes.
type Scope struct {
	stack *stack
}

// NewScope creates a new Scope object.
func NewScope() *Scope {
	return &Scope{newStack()}
}

// Empty returns false if no elements are in any scope.
func (s Scope) Empty() bool {
	return s.stack.empty()
}

// Open pushes a new SymTab (symbolTable) on the Scope stack.
func (s *Scope) Open() {
	s.stack.push(newSymTab())
}

// Close pops a SymTab (symbolTable) off the Scope stack.
func (s *Scope) Close() (SymTab, error) {
	if s.Empty() {
		return SymTab{}, errors.New("can't pop an empty scope stack")
	}
	return s.stack.pop()
}

// Global sets a node and its name in the global scope.
func (s *Scope) Global(name string, n node.Node) (bool, error) {
	symtab, err := s.stack.bottom()
	if err != nil {
		return false, err
	}
	if symtab.contains(name) {
		return false, fmt.Errorf("'%s' cannot be declared globally twice", name)
	}
	err = symtab.set(name, n)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Contains checks if a node name appears in the current scope.
func (s Scope) Contains(name string) bool {
	symtab, err := s.stack.peek()
	if err != nil {
		return false
	}
	return symtab.contains(name)
}

// Lookup finds the first node in the current scope by name.
func (s *Scope) Lookup(name string) (node.Node, error) {
	if s.Empty() {
		return nil, fmt.Errorf("can't find a node '%s' in an empty scope stack", name)
	}
	for _, symtab := range s.stack.list {
		if symtab.contains(name) {
			return symtab.get(name)
		}
	}
	return nil, fmt.Errorf("'%s' has not yet been declared", name)
}

// Add adds a name to the topmost scope in the scope stack.
func (s *Scope) Add(name string, n node.Node) error {
	if s.Contains(name) {
		return fmt.Errorf("'%s' cannot be declared twice", name)
	}
	symtab, err := s.stack.peek()
	if err != nil {
		return err
	}
	err = symtab.set(name, n)
	if err != nil {
		return err
	}
	return nil
}

// String impliments the fmt.Stringer interface.
func (s *Scope) String() string {
	return s.stack.String()
}
