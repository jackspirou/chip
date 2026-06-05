package diag_test

import (
	"testing"

	"github.com/jackspirou/chip/internal/diag"
)

// Every code in the catalog resolves through Explain — the contract that backs
// `chip explain CODE` for every code the classifier can emit.
func TestEveryCatalogCodeResolves(t *testing.T) {
	codes := diag.Codes()
	if len(codes) == 0 {
		t.Fatal("catalog is empty")
	}
	for _, e := range codes {
		got, ok := diag.Explain(e.Code)
		if !ok {
			t.Errorf("Explain(%q) not found", e.Code)
			continue
		}
		if got.Code != e.Code {
			t.Errorf("Explain(%q).Code = %q", e.Code, got.Code)
		}
		if got.Title == "" || got.Explanation == "" {
			t.Errorf("code %s has empty Title or Explanation", e.Code)
		}
	}
}

// Classify routes a representative message for every branch to a code, and that
// code itself resolves — so "every emitted code resolves in chip explain" holds
// for the whole classifier, not only the codes one run happens to hit.
func TestClassifyEmitsOnlyResolvableCodes(t *testing.T) {
	cases := []struct {
		name  string
		phase diag.Phase
		msg   string
		want  string
	}{
		{"usage", diag.PhaseUsage, "unknown flag --nope", "USE001"},
		{"parse", diag.PhaseParse, "expected ')', found ';'", "PAR001"},
		{"undefined name", diag.PhaseType, "undefined: foo", "NAM001"},
		{"undefined type", diag.PhaseType, "undefined type: Foo", "NAM002"},
		{"redeclared", diag.PhaseType, "x redeclared in this block", "NAM003"},
		{"duplicate param", diag.PhaseType, "duplicate parameter x", "NAM003"},
		{"type mismatch", diag.PhaseType, "cannot return int as string", "TYP001"},
		{"missing return", diag.PhaseType, "missing return value (int expected)", "TYP002"},
		{"arity", diag.PhaseType, "wrong number of arguments: got 2, want 1", "TYP003"},
		{"builtin arity", diag.PhaseType, "len expects 1 argument, got 0", "TYP003"},
		{"not a value", diag.PhaseType, "print is not a value", "TYP004"},
		{"non-function", diag.PhaseType, "cannot call non-function (type int)", "TYP004"},
		{"div by zero", diag.PhaseRuntime, "division by zero", "RUN001"},
		{"index oob", diag.PhaseRuntime, "index out of range: 5 (length 2)", "RUN002"},
		{"stack depth", diag.PhaseRuntime, "call stack too deep", "RUN003"},
		{"runtime fallback", diag.PhaseRuntime, "nil dereference", "RUN004"},
		{"timeout", diag.PhaseResource, "execution timed out after 1s", "RES001"},
		{"output budget", diag.PhaseResource, "output limit exceeded", "RES003"},
		{"step budget", diag.PhaseResource, "step budget exhausted", "RES002"},
		{"unused import", diag.PhaseLint, `"math" imported and not used`, "LNT003"},
		{"unused var", diag.PhaseLint, "x declared and not used", "LNT001"},
		{"unused func", diag.PhaseLint, "helper is never used", "LNT002"},
		{"unreachable", diag.PhaseLint, "unreachable code", "LNT004"},
		{"used before def", diag.PhaseLint, "x used before its definition", "LNT005"},
		{"lint missing return", diag.PhaseLint, "missing return at end of function", "LNT006"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diag.Classify(diag.Diagnostic{Phase: tc.phase, Message: tc.msg})
			if got != tc.want {
				t.Errorf("Classify(%s, %q) = %q, want %q", tc.phase, tc.msg, got, tc.want)
			}
			if _, ok := diag.Explain(got); !ok {
				t.Errorf("Classify emitted %q, which Explain cannot resolve", got)
			}
		})
	}
}

// A code already set on a diagnostic wins over classification.
func TestClassifyHonorsExplicitCode(t *testing.T) {
	got := diag.Classify(diag.Diagnostic{Code: "ZZZ999", Phase: diag.PhaseType, Message: "undefined: x"})
	if got != "ZZZ999" {
		t.Errorf("Classify = %q, want the preset %q", got, "ZZZ999")
	}
}

// Explain is case- and space-insensitive, and unknown codes do not resolve.
func TestExplainNormalizesAndRejectsUnknown(t *testing.T) {
	up, ok := diag.Explain("NAM001")
	if !ok {
		t.Fatal("Explain(NAM001) not found")
	}
	for _, variant := range []string{"nam001", "  Nam001 ", "nAm001"} {
		got, ok := diag.Explain(variant)
		if !ok || got.Code != up.Code {
			t.Errorf("Explain(%q) = (%q, %v), want (%q, true)", variant, got.Code, ok, up.Code)
		}
	}
	if _, ok := diag.Explain("BOGUS"); ok {
		t.Error("Explain(BOGUS) resolved, want not found")
	}
}

// WithCodes fills an empty Code by classification and leaves a preset one alone,
// without mutating its input.
func TestWithCodesFillsAndPreserves(t *testing.T) {
	in := []diag.Diagnostic{
		{Phase: diag.PhaseType, Message: "undefined: foo"},    // -> NAM001
		{Code: "TYP001", Phase: diag.PhaseType, Message: "x"}, // preserved
	}
	out := diag.WithCodes(in)
	if out[0].Code != "NAM001" {
		t.Errorf("out[0].Code = %q, want NAM001", out[0].Code)
	}
	if out[1].Code != "TYP001" {
		t.Errorf("out[1].Code = %q, want TYP001 (preserved)", out[1].Code)
	}
	if in[0].Code != "" {
		t.Errorf("WithCodes mutated input: in[0].Code = %q, want empty", in[0].Code)
	}
}

// Codes lists entries in ascending, duplicate-free code order — a stable index.
func TestCodesSortedAndUnique(t *testing.T) {
	codes := diag.Codes()
	seen := make(map[string]bool)
	for i, e := range codes {
		if seen[e.Code] {
			t.Errorf("duplicate code %q", e.Code)
		}
		seen[e.Code] = true
		if i > 0 && codes[i-1].Code >= e.Code {
			t.Errorf("codes out of order: %q before %q", codes[i-1].Code, e.Code)
		}
	}
}
