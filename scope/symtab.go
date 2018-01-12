package scope

import (
	"fmt"

	"github.com/jackspirou/chip/node"
)

// SymbolTable describes a symboltable that stores describtor nodes.
type SymbolTable struct {
	table map[string]node.Node
}

// newSymbolTable returns a new symTab object.
func newSymbolTable() SymbolTable {
	return SymbolTable{make(map[string]node.Node)}
}

// contains checks if the symboltable contains a specific node name.
func (s SymbolTable) contains(name string) bool {
	_, ok := s.table[name]
	return ok
}

// set sets a node and its name in the symboltable.
func (s *SymbolTable) set(name string, node node.Node) error {
	if s.contains(name) {
		return fmt.Errorf("'%s' already declared", name)
	}
	s.table[name] = node
	return nil
}

// get gets a node by its name.
func (s SymbolTable) get(name string) (node.Node, error) {
	if node, ok := s.table[name]; ok {
		return node, nil
	}
	return nil, fmt.Errorf("'%s' not yet declared", name)
}
