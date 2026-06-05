package diag_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/stream"
	"github.com/jackspirou/chip/internal/token"
)

const undefSrc = "print(undef)\n"

func undefDiag() diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.SeverityError,
		Phase:    diag.PhaseType,
		Message:  "undefined: undef",
		Primary:  diag.Primary{IsPrimary: true, Line: 1, Column: 7, Offset: 6},
	}
}

func render(t *testing.T, r diag.Renderer, src diag.Source, ds []diag.Diagnostic) string {
	t.Helper()
	var b bytes.Buffer
	if err := r.Render(&b, src, ds); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// The human renderer reproduces the CLI's historical caret layout exactly: this
// is the byte contract A7 locks.
func TestHumanByteIdentical(t *testing.T) {
	got := render(t, diag.Human{}, diag.Source{Name: "<arg>", Bytes: []byte(undefSrc)}, []diag.Diagnostic{undefDiag()})
	want := "<arg>:1:7: undefined: undef\n  print(undef)\n        ^\n"
	if got != want {
		t.Fatalf("human render mismatch:\n got %q\nwant %q", got, want)
	}
}

// A CRLF-terminated source renders snippets without the trailing carriage return:
// the human caret line and the enriched Snippet both stop at the visible text. For
// LF sources (every A7 example) sourceLines is exactly bytes.Split, so this only
// governs the "\r\n" case — a Windows-authored file an editor or agent feeds in.
func TestCRLFSnippetStripsCarriageReturn(t *testing.T) {
	src := diag.Source{Name: "f", Bytes: []byte("x := bad\r\nprint(x)\r\n")}
	d := diag.Diagnostic{
		Severity: diag.SeverityError,
		Message:  "boom",
		Primary:  diag.Primary{IsPrimary: true, Line: 1, Column: 6},
	}

	// Human caret: the source line must not carry a trailing '\r' before its '\n'.
	got := render(t, diag.Human{}, src, []diag.Diagnostic{d})
	want := "f:1:6: boom\n" +
		"  x := bad\n" +
		"  " + strings.Repeat(" ", 5) + "^\n" // caret under column 6
	if got != want {
		t.Fatalf("human render with CRLF:\n got %q\nwant %q", got, want)
	}

	// Enrich: the derived Snippet must be the bare line, no '\r'.
	enr := diag.Enrich([]diag.Diagnostic{d}, src)
	if s := enr[0].Primary.Snippet; s != "x := bad" {
		t.Fatalf("enriched snippet = %q, want %q (trailing CR not stripped)", s, "x := bad")
	}
}

// The human renderer preserves the given order (source order) and never sorts.
func TestHumanPreservesOrder(t *testing.T) {
	ds := []diag.Diagnostic{
		{Message: "second", Primary: diag.Primary{Line: 2, Column: 1}},
		{Message: "first", Primary: diag.Primary{Line: 1, Column: 1}},
	}
	got := render(t, diag.Human{}, diag.Source{Name: "f", Bytes: []byte("a\nb\n")}, ds)
	if !strings.HasPrefix(got, "f:2:1: second") {
		t.Fatalf("human reordered diagnostics: %q", got)
	}
}

// The agent renderer is terse and greppable: one located line per diagnostic
// with a stable severity prefix, sorted by position, and a count summary emitted
// last so truncation from the top can't drop it.
func TestAgentTerseSortedSummaryLast(t *testing.T) {
	ds := []diag.Diagnostic{
		{Severity: diag.SeverityError, Message: "two", Primary: diag.Primary{Line: 2, Column: 3}},
		{Severity: diag.SeverityError, Message: "one", Primary: diag.Primary{Line: 1, Column: 5}},
		{Severity: diag.SeverityWarning, Message: "warn", Primary: diag.Primary{Line: 3, Column: 1}},
	}
	got := render(t, diag.Agent{}, diag.Source{Name: "<stdin>", Bytes: []byte("a\nb\nc\n")}, ds)
	want := "error: <stdin>:1:5: one\n" +
		"error: <stdin>:2:3: two\n" +
		"warning: <stdin>:3:1: warn\n" +
		"found 2 errors, 1 warning\n"
	if got != want {
		t.Fatalf("agent render mismatch:\n got %q\nwant %q", got, want)
	}
}

// Each agent line leads with the severity-and-code prefix so a harness can
// grep '^error\[' to count and route by code (A8). With a numbered code assigned
// (a later slice) the bracket carries it; clean input emits nothing.
func TestAgentCodePrefixAndClean(t *testing.T) {
	withCode := []diag.Diagnostic{{Severity: diag.SeverityError, Code: "TYP003", Message: "m", Primary: diag.Primary{Line: 1, Column: 1}}}
	got := render(t, diag.Agent{}, diag.Source{Name: "f", Bytes: []byte("x\n")}, withCode)
	if !strings.HasPrefix(got, "error[TYP003]: f:1:1: m\n") {
		t.Fatalf("agent code prefix missing: %q", got)
	}
	if clean := render(t, diag.Agent{}, diag.Source{Name: "f", Bytes: []byte("x\n")}, nil); clean != "" {
		t.Fatalf("agent on clean input emitted %q", clean)
	}
}

// Without an explicit code the agent prefix falls back to the phase's catalog
// prefix (PAR/TYP/RUN/…), so every line is greppable as error[PREFIX]: before
// numbered codes exist (A8 holds on the run path's parse/type/runtime faults).
func TestAgentPhasePrefixFallback(t *testing.T) {
	cases := []struct {
		phase  diag.Phase
		prefix string
	}{
		{diag.PhaseParse, "PAR"},
		{diag.PhaseType, "TYP"},
		{diag.PhaseName, "NAM"},
		{diag.PhaseRuntime, "RUN"},
		{diag.PhaseLint, "LNT"},
	}
	for _, c := range cases {
		sev := diag.SeverityError
		if c.phase == diag.PhaseLint {
			sev = diag.SeverityWarning
		}
		ds := []diag.Diagnostic{{Severity: sev, Phase: c.phase, Message: "m", Primary: diag.Primary{Line: 1, Column: 1}}}
		got := render(t, diag.Agent{}, diag.Source{Name: "f", Bytes: []byte("x\n")}, ds)
		want := string(sev) + "[" + c.prefix + "]: f:1:1: m\n"
		if !strings.HasPrefix(got, want) {
			t.Fatalf("phase %s: got %q, want prefix %q", c.phase, got, want)
		}
	}
}

// The LLM renderer is a strict superset of the agent renderer: the agent bytes
// appear verbatim as a prefix, then a grounded "how to fix:" appendix carries the
// catalog's repair guidance for the code present — the Title and Fix on one line and
// the Explanation on an indented "why:" line below it. This is the per-model "llm:*"
// profile (§5.5) — terse diagnostic plus the grounding a model needs to repair
// without guessing by analogy to another language.
func TestLLMSupersetGroundedAppendix(t *testing.T) {
	src := diag.Source{Name: "<arg>", Bytes: []byte(undefSrc)}
	ds := []diag.Diagnostic{{Severity: diag.SeverityError, Code: "NAM001", Message: "undefined: undef", Primary: diag.Primary{Line: 1, Column: 7}}}

	agent := render(t, diag.Agent{}, src, ds)
	llm := render(t, diag.LLM{}, src, ds)

	if !strings.HasPrefix(llm, agent) {
		t.Fatalf("llm render is not a superset of agent:\nagent:\n%q\nllm:\n%q", agent, llm)
	}
	if !strings.Contains(llm, "how to fix:\n") {
		t.Fatalf("llm render missing grounded appendix:\n%s", llm)
	}
	entry, ok := diag.Explain("NAM001")
	if !ok {
		t.Fatal("catalog has no NAM001 entry")
	}
	want := "  NAM001 (" + entry.Title + "): " + entry.Fix + "\n"
	if !strings.Contains(llm, want) {
		t.Fatalf("llm appendix missing grounded fix line %q:\n%s", want, llm)
	}
	// The grounding rides on an indented "why:" line carrying the catalog Explanation.
	why := "    why: " + entry.Explanation + "\n"
	if !strings.Contains(llm, why) {
		t.Fatalf("llm appendix missing grounded explanation %q:\n%s", why, llm)
	}
	// A8: the appendix adds no line a harness grepping '^error\[' would miscount.
	if got, want := countPrefix(llm, "error["), countPrefix(agent, "error["); got != want {
		t.Fatalf("llm changed the coded-error line count: agent=%d llm=%d", want, got)
	}
}

// The appendix lists each code once even when several diagnostics share it.
func TestLLMDedupesCodes(t *testing.T) {
	src := diag.Source{Name: "f", Bytes: []byte("a\nb\n")}
	ds := []diag.Diagnostic{
		{Severity: diag.SeverityError, Code: "TYP001", Message: "one", Primary: diag.Primary{Line: 1, Column: 1}},
		{Severity: diag.SeverityError, Code: "TYP001", Message: "two", Primary: diag.Primary{Line: 2, Column: 1}},
	}
	llm := render(t, diag.LLM{}, src, ds)
	entry, _ := diag.Explain("TYP001")
	marker := "  TYP001 (" + entry.Title + "):"
	if n := strings.Count(llm, marker); n != 1 {
		t.Fatalf("TYP001 appendix line appears %d times, want 1:\n%s", n, llm)
	}
}

// A diagnostic whose code is not in the catalog (here a bare phase prefix, before a
// numbered code is assigned) adds no appendix: the llm render equals the agent
// render byte for byte.
func TestLLMNoAppendixWhenUncatalogued(t *testing.T) {
	src := diag.Source{Name: "f", Bytes: []byte("x\n")}
	ds := []diag.Diagnostic{{Severity: diag.SeverityError, Phase: diag.PhaseType, Message: "m", Primary: diag.Primary{Line: 1, Column: 1}}}
	agent := render(t, diag.Agent{}, src, ds)
	llm := render(t, diag.LLM{}, src, ds)
	if llm != agent {
		t.Fatalf("uncatalogued code produced an appendix:\nagent:\n%q\nllm:\n%q", agent, llm)
	}
	if strings.Contains(llm, "how to fix:") {
		t.Fatalf("llm emitted appendix for uncatalogued code:\n%s", llm)
	}
}

// countPrefix counts the lines of s that begin with prefix.
func countPrefix(s, prefix string) int {
	n := 0
	for line := range strings.SplitSeq(s, "\n") {
		if strings.HasPrefix(line, prefix) {
			n++
		}
	}
	return n
}

// The JSON renderer emits the Result envelope, enriching each diagnostic's
// primary with the source snippet and a rendered caret block so a JSON consumer
// never re-implements formatting.
func TestJSONEnvelope(t *testing.T) {
	out := render(t, diag.JSON{}, diag.Source{Name: "<arg>", Bytes: []byte(undefSrc)}, []diag.Diagnostic{undefDiag()})
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("json output not newline-terminated: %q", out)
	}
	var res diag.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out)
	}
	if res.OK {
		t.Fatalf("ok should be false with an error diagnostic")
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(res.Diagnostics))
	}
	p := res.Diagnostics[0].Primary
	if p.File != "<arg>" || p.Snippet != "print(undef)" {
		t.Fatalf("primary not enriched from source: %+v", p)
	}
	if p.Offset != 6 || p.EndOffset != 6 || p.EndLine != 1 || p.EndColumn != 7 {
		t.Fatalf("point span not collapsed correctly: %+v", p)
	}
	if got, want := res.Diagnostics[0].Rendered, "<arg>:1:7: undefined: undef\n  print(undef)\n        ^"; got != want {
		t.Fatalf("rendered mismatch:\n got %q\nwant %q", got, want)
	}
}

// A clean program serializes to ok:true with an empty (not null) diagnostics
// list — a stable shape for parsers.
func TestJSONClean(t *testing.T) {
	out := render(t, diag.JSON{}, diag.Source{Name: "f", Bytes: []byte("print(1)\n")}, nil)
	var res diag.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if !res.OK {
		t.Fatalf("clean program should be ok")
	}
	if !strings.Contains(out, `"diagnostics":[]`) {
		t.Fatalf("diagnostics should serialize as [] not null: %s", out)
	}
}

// Every front-end error type describes itself as []diag.Diagnostic via the
// Diagnoser interface, tagged with the right phase and severity, with byte
// offsets carried through. diag.From routes any of them without a type switch.
func TestFrontEndsEmitDiagnostics(t *testing.T) {
	pos := token.Pos{Line: 3, Column: 4, Offset: 11}
	cases := []struct {
		name     string
		err      error
		phase    diag.Phase
		severity diag.Severity
	}{
		{"parser", parser.ErrorList{{Pos: pos, Msg: "p"}}, diag.PhaseParse, diag.SeverityError},
		{"check", check.ErrorList{{Pos: pos, Msg: "c"}}, diag.PhaseType, diag.SeverityError},
		{"lint", lint.IssueList{{Pos: pos, Msg: "l"}}, diag.PhaseLint, diag.SeverityWarning},
		{"streamType", stream.TypeError{Pos: pos, Msg: "t"}, diag.PhaseType, diag.SeverityError},
		{"streamRuntime", stream.RuntimeError{Pos: pos, Msg: "r"}, diag.PhaseRuntime, diag.SeverityError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := diag.From(c.err)
			if len(ds) != 1 {
				t.Fatalf("From(%s) = %d diagnostics, want 1", c.name, len(ds))
			}
			d := ds[0]
			if d.Phase != c.phase {
				t.Fatalf("phase = %q, want %q", d.Phase, c.phase)
			}
			if d.Severity != c.severity {
				t.Fatalf("severity = %q, want %q", d.Severity, c.severity)
			}
			if d.Primary.Line != 3 || d.Primary.Column != 4 || d.Primary.Offset != 11 {
				t.Fatalf("position not carried: %+v", d.Primary)
			}
			if !d.Primary.IsPrimary {
				t.Fatalf("primary should be marked isPrimary")
			}
		})
	}
}

// diag.From returns nil for an error that carries no structured diagnostics, so
// the CLI can fall back to a plain message.
func TestFromUnknownErrorIsNil(t *testing.T) {
	if ds := diag.From(errString("boom")); ds != nil {
		t.Fatalf("From(plain error) = %v, want nil", ds)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
