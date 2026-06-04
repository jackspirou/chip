// Package scope implements lexical scopes that map names to symbols. Scopes
// form a tree: each scope has a parent, and name resolution walks outward.
package scope

// Scope maps names to symbols within a single lexical region.
type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

// New returns a new scope nested inside parent (nil for the outermost scope).
func New(parent *Scope) *Scope {
	return &Scope{parent: parent, symbols: make(map[string]*Symbol)}
}

// Parent returns the enclosing scope, or nil for the outermost scope.
func (s *Scope) Parent() *Scope { return s.parent }

// Insert declares sym in s. If a symbol with the same name already exists in s,
// Insert leaves s unchanged and returns that existing symbol; otherwise it adds
// sym and returns nil.
func (s *Scope) Insert(sym *Symbol) *Symbol {
	if prev, ok := s.symbols[sym.Name]; ok {
		return prev
	}
	s.symbols[sym.Name] = sym
	return nil
}

// Lookup returns the symbol named name declared directly in s, or nil.
func (s *Scope) Lookup(name string) *Symbol { return s.symbols[name] }

// LookupParent searches s and its ancestors, returning the first symbol named
// name or nil if none is found.
func (s *Scope) LookupParent(name string) *Symbol {
	for cur := s; cur != nil; cur = cur.parent {
		if sym, ok := cur.symbols[name]; ok {
			return sym
		}
	}
	return nil
}
