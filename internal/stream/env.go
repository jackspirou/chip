package stream

import "github.com/jackspirou/chip/internal/value"

// env is an activation record: a set of variable bindings plus a parent scope.
// A function call gets its own env (parent = the global env) and each block
// scope gets a child. When the executor finishes a scope and nothing captured
// its env, the env becomes unreachable and Go's GC reclaims it (plan §3.5) —
// there is no manual free and no shared value stack.
type env struct {
	vars   map[string]value.Value
	parent *env
}

func newEnv(parent *env) *env {
	return &env{vars: make(map[string]value.Value), parent: parent}
}

// child returns a fresh scope nested in e.
func (e *env) child() *env { return newEnv(e) }

// lookup returns the value bound to name, searching outward through parents.
func (e *env) lookup(name string) (value.Value, bool) {
	for s := e; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v, true
		}
	}
	return value.Value{}, false
}

// define binds name in this scope (a := declaration).
func (e *env) define(name string, v value.Value) { e.vars[name] = v }

// assign updates an existing binding (an = assignment), searching outward. It
// reports whether a binding was found.
func (e *env) assign(name string, v value.Value) bool {
	for s := e; s != nil; s = s.parent {
		if _, ok := s.vars[name]; ok {
			s.vars[name] = v
			return true
		}
	}
	return false
}
