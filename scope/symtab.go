package scope

import (
	"fmt"

	"github.com/jackspirou/chip/ssa"
)

// SymTab describes a symboltable that stores describtor nodes.
type SymTab struct {
	table map[string]ssa.Node
}

// newSymTab returns a new symTab object.
func newSymTab() SymTab {
	return SymTab{make(map[string]ssa.Node)}
}

// contains checks if the symboltable contains a specific node name.
func (s SymTab) contains(name string) bool {
	_, ok := s.table[name]
	return ok
}

// set sets a node and its name in the symboltable.
func (s *SymTab) set(name string, node ssa.Node) error {
	if s.contains(name) {
		return fmt.Errorf("'%s' already declared", name)
	}
	s.table[name] = node
	return nil
}

// get gets a node by its name.
func (s SymTab) get(name string) (ssa.Node, error) {
	if node, ok := s.table[name]; ok {
		return node, nil
	}
	return nil, fmt.Errorf("'%s' not yet declared", name)
}
