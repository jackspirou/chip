package scope

import (
	"errors"

	"github.com/jackspirou/chip/src/chip/node"
)

// SYMBOLTABLE. SymbolTable which stores descriptor nodes.
type SymTab struct {
	table map[string]node.Node
}

func NewSymTab() *SymTab {
	return &SymTab{
		table: make(map[string]node.Node),
	}
}

func (s *SymTab) Contains(name string) bool {
	_, ok := s.table[name]
	return ok
}

func (s *SymTab) Set(name string, node node.Node) error {
	if s.Contains(name) {
		return errors.New("'" + name + "' cannot be declared twice.")
	}
	s.table[name] = node
	return nil
}

func (s *SymTab) Get(name string) (node.Node, error) {
	if s.Contains(name) {
		node, _ := s.table[name]
		return node, nil
	}
	return nil, errors.New("'" + name + "' has not yet been declared twice.")
}
