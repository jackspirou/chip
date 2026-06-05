package parser_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/parser"
)

// The recursive-descent parser descends one Go stack frame per level of nested
// syntax. Without a bound, deeply nested input — a few million nested parens,
// unary operators, array brackets, or if/else-if clauses — overflows the
// goroutine stack, which Go reports as a fatal error that recover cannot catch
// and that also kills any embedding host. No execution cap can stop it: scanning
// and parsing run ahead of the step loop that enforces --max-steps / --timeout,
// so the crash happens before a single step is taken. chip accepts streamed,
// untrusted source, so this was a denial-of-service vector. The maxDepth guard
// turns it into one ordinary, recoverable parse error; these tests pin that.
//
// Each spine drives a distinct recursion cycle, and the guard sits at a different
// choke point for each:
//   - nested parens / nested unary  -> parseUnaryExpr (the whole expression cycle)
//   - nested array type             -> parseArrayType ([][]...Elem)
//   - nested if blocks              -> parseStmt (block-nested control flow)
//   - else-if chain                 -> parseIf (recurses past parseStmt)
//
// A regression that drops any one guard lets that spine parse without error, so
// the depth assertion below fails for it specifically.

// depthSpines builds nested source of a requested depth for each recursion cycle.
// Every form is otherwise well-formed, so the only error a correct parser can
// report on the deep inputs is the depth bailout — never an incidental syntax
// error. `1` is used where a condition is needed (a valid operand; the parser
// does not type-check), and `!` is the unary operator because repeated `-` would
// scan as the `--` (DEC) token rather than nested unary minus.
var depthSpines = []struct {
	name  string
	build func(n int) string
}{
	{"nested parens", func(n int) string {
		return strings.Repeat("(", n) + "1" + strings.Repeat(")", n)
	}},
	{"nested unary", func(n int) string {
		return strings.Repeat("!", n) + "x"
	}},
	{"nested array type", func(n int) string {
		return "func f(x " + strings.Repeat("[]", n) + "int) {}"
	}},
	{"nested if blocks", func(n int) string {
		return strings.Repeat("if 1 {", n) + strings.Repeat("}", n)
	}},
	{"else-if chain", func(n int) string {
		return "if 1 {}" + strings.Repeat(" else if 1 {}", n)
	}},
}

// deepNesting is comfortably past maxDepth (10000) so the guard trips on every
// spine, yet far below the ~2,000,000 levels that overflow the goroutine stack —
// so the descent to the bailout costs only a few megabytes and finishes in
// milliseconds even when the guard fires.
const deepNesting = 20000

// TestParseRejectsExcessiveNesting checks the batch driver (Parse): each spine,
// nested past the limit, yields exactly one parse error — "nesting too deep" —
// instead of crashing. Exactly one because Parse stops after the bailout rather
// than re-erroring on every leftover token.
func TestParseRejectsExcessiveNesting(t *testing.T) {
	for _, sp := range depthSpines {
		t.Run(sp.name, func(t *testing.T) {
			p, err := parser.New(strings.NewReader(sp.build(deepNesting)))
			if err != nil {
				t.Fatalf("parser.New: %v", err)
			}

			_, parseErr := p.Parse()
			if parseErr == nil {
				t.Fatalf("deep %s parsed without error; the depth guard for this spine has regressed (a stack-overflow DoS)", sp.name)
			}

			list, ok := parseErr.(parser.ErrorList)
			if !ok {
				t.Fatalf("Parse error is %T, want parser.ErrorList: %v", parseErr, parseErr)
			}
			if len(list) != 1 {
				t.Fatalf("got %d parse errors, want exactly 1 (the bailout must stop, not flood): %v", len(list), parseErr)
			}
			if list[0].Msg != "nesting too deep" {
				t.Fatalf("got error %q, want %q", list[0].Msg, "nesting too deep")
			}
		})
	}
}

// TestItemsRejectsExcessiveNesting checks the streaming driver (Items): the same
// over-nested input ends the stream with the same recorded parse error. Items is
// a separate loop from Parse, so it gets its own coverage.
func TestItemsRejectsExcessiveNesting(t *testing.T) {
	for _, sp := range depthSpines {
		t.Run(sp.name, func(t *testing.T) {
			p, err := parser.New(strings.NewReader(sp.build(deepNesting)))
			if err != nil {
				t.Fatalf("parser.New: %v", err)
			}

			var streamErr error
			for _, itemErr := range p.Items() {
				if itemErr != nil {
					streamErr = itemErr
					break
				}
			}
			if streamErr == nil {
				t.Fatalf("deep %s streamed without error; the depth guard for this spine has regressed (a stack-overflow DoS)", sp.name)
			}
			if !strings.Contains(streamErr.Error(), "nesting too deep") {
				t.Fatalf("stream error = %q, want it to contain %q", streamErr.Error(), "nesting too deep")
			}
		})
	}
}

// TestParseAllowsRealisticNesting is the negative case: nesting an order of
// magnitude deeper than any real program still parses cleanly, so the guard
// never rejects legitimate code. Each spine is valid chip syntax (names need not
// resolve — parsing does not type-check), so a clean parse is the correct result.
func TestParseAllowsRealisticNesting(t *testing.T) {
	const realistic = 100 // far below maxDepth; deeper than any hand-written file
	for _, sp := range depthSpines {
		t.Run(sp.name, func(t *testing.T) {
			p, err := parser.New(strings.NewReader(sp.build(realistic)))
			if err != nil {
				t.Fatalf("parser.New: %v", err)
			}
			if _, parseErr := p.Parse(); parseErr != nil {
				t.Fatalf("%d-deep %s should parse cleanly, got: %v", realistic, sp.name, parseErr)
			}
		})
	}
}
