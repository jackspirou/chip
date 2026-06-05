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

// Replace declares sym in s, overwriting any existing binding of the same name.
// Unlike Insert it never reports a conflict; it is for scopes where a name may
// be redefined, such as the top-level (file and REPL) scope.
func (s *Scope) Replace(sym *Symbol) { s.symbols[sym.Name] = sym }

// Lookup returns the symbol named name declared directly in s, or nil.
func (s *Scope) Lookup(name string) *Symbol { return s.symbols[name] }

// Names returns the names declared directly in s (not its ancestors), in no
// particular order. It lets name resolution enumerate candidate names for a
// "did you mean" suggestion when a lookup fails.
func (s *Scope) Names() []string {
	names := make([]string, 0, len(s.symbols))
	for name := range s.symbols {
		names = append(names, name)
	}
	return names
}

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
