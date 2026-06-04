package stream

import "github.com/jackspirou/chip/internal/value"

// binding is one name → value pair in a scope.
type binding struct {
	name string
	val  value.Value
}

// env is an activation record: a set of variable bindings plus a parent scope.
// A function call gets its own env (parent = the global env) and each block
// scope gets a child. When the executor finishes a scope and nothing captured
// its env, the env becomes unreachable and Go's GC reclaims it (plan §3.5) —
// there is no manual free and no shared value stack.
//
// Bindings are kept in a small slice rather than a map: chip's scopes are tiny
// (a function scope holds only its parameters; most block scopes hold nothing),
// and a linear scan over a handful of entries beats a map's hashing and its
// per-scope header + bucket allocations. chip has no closures over locals — a
// call's parent is the global env, not the caller — so appending to (and thus
// reallocating) the slice is safe: nothing holds a pointer into it.
type env struct {
	binds  []binding
	parent *env
}

func newEnv(parent *env) *env {
	return &env{parent: parent}
}

// child returns a fresh scope nested in e.
func (e *env) child() *env { return newEnv(e) }

// lookup returns the value bound to name, searching outward through parents.
func (e *env) lookup(name string) (value.Value, bool) {
	for s := e; s != nil; s = s.parent {
		for i := range s.binds {
			if s.binds[i].name == name {
				return s.binds[i].val, true
			}
		}
	}
	return value.Value{}, false
}

// define binds name in this scope (a := declaration). If the name is already
// bound here it is overwritten, matching the previous map's semantics (the
// global/REPL scope redefines a name via :=).
func (e *env) define(name string, v value.Value) {
	for i := range e.binds {
		if e.binds[i].name == name {
			e.binds[i].val = v
			return
		}
	}
	e.binds = append(e.binds, binding{name: name, val: v})
}

// assign updates an existing binding (an = assignment), searching outward. It
// reports whether a binding was found.
func (e *env) assign(name string, v value.Value) bool {
	for s := e; s != nil; s = s.parent {
		for i := range s.binds {
			if s.binds[i].name == name {
				s.binds[i].val = v
				return true
			}
		}
	}
	return false
}
