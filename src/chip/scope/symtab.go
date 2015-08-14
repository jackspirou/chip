package scope

import (
	"fmt"

	"github.com/jackspirou/chip/src/chip/node"
)

// SymTab describes a symboltable that stores describtor nodes.
type SymTab struct {
	table map[string]node.Node
}

// newSymTab returns a new symTab object.
func newSymTab() SymTab {
	return SymTab{
		table: make(map[string]node.Node),
	}
}

// contains checks if the symboltable contains a specific node name.
func (s SymTab) contains(name string) bool {
	_, ok := s.table[name]
	return ok
}

// set sets a node and its name in the symboltable.
func (s *SymTab) set(name string, node node.Node) error {
	if s.contains(name) {
		return fmt.Errorf("'%s' cannot be declared twice", name)
	}
	s.table[name] = node
	return nil
}

// get gets a node by its name.
func (s SymTab) get(name string) (node.Node, error) {
	if node, ok := s.table[name]; ok {
		return node, nil
	}
	return nil, fmt.Errorf("'%s' has not yet been declared", name)
}
