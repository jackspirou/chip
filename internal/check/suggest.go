package check

import (
	"fmt"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/diag"
)

// This file implements "did you mean" suggestions for undefined names. When a
// name fails to resolve, the nearest in-scope name within a bounded edit distance
// is offered as a grounded hint — and only when one is close enough. A vague or
// far-fetched guess scores worse than saying nothing (plan: hints are grounded,
// never guessed), so beyond the bound no suggestion is attached.

// undefinedIdent records an "undefined: <name>" error for e, attaching a "did you
// mean <name>?" suggestion when a sufficiently similar name is in scope. The
// suggestion's edit replaces the identifier's exact byte span and is tagged
// maybe — a plausible fix a harness may offer but must not auto-apply blindly.
func (c *Checker) undefinedIdent(e *ast.Ident) {
	err := Error{Pos: e.Pos(), Msg: fmt.Sprintf("undefined: %s", e.Name)}
	if cand, ok := nearest(e.Name, c.scopeNames()); ok {
		err.Suggestions = []diag.Suggestion{{
			Message:       "did you mean " + cand + "?",
			Applicability: diag.Maybe,
			Edit:          []diag.Edit{{Offset: e.Pos().Offset, EndOffset: e.End().Offset, NewText: cand}},
		}}
	}
	c.errs = append(c.errs, err)
}

// scopeNames returns every name visible from the current scope — the current
// scope and all its ancestors up to the universe (so builtins, functions,
// parameters, and locals are all candidates) — deduplicated, inner shadowing
// outer.
func (c *Checker) scopeNames() []string {
	seen := make(map[string]bool)
	var names []string
	for s := c.scope; s != nil; s = s.Parent() {
		for _, n := range s.Names() {
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	return names
}

// nearest returns the candidate closest to target by Levenshtein distance, within
// a length-scaled bound, or ok=false when none is close enough to be worth
// suggesting. Ties break to the lexicographically smaller name so the suggestion
// is deterministic (a precondition for reproducible repair loops).
func nearest(target string, candidates []string) (string, bool) {
	bound := max((len(target)+2)/3, 1) // ~⅓ of the name may differ; at least 1
	best := ""
	bestDist := bound + 1
	for _, c := range candidates {
		if c == target {
			continue
		}
		d := levenshtein(target, c)
		if d > bound {
			continue
		}
		if d < bestDist || (d == bestDist && c < best) {
			best, bestDist = c, d
		}
	}
	return best, best != ""
}

// levenshtein returns the edit distance between a and b (single-character
// insertions, deletions, and substitutions), comparing by rune so multi-byte
// names are measured by characters, not bytes.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
