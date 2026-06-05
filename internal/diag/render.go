package diag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Source is the context a renderer needs: the file name as it appears in
// messages (a path, or <stdin>/<arg> for synthetic input) and the raw bytes, for
// snippets and carets.
type Source struct {
	Name  string
	Bytes []byte
}

// Renderer turns structured diagnostics into bytes. The four implementations —
// Human, Agent, LLM, JSON — are the presentation layer; they never change what was
// detected, only how it reads (D6).
type Renderer interface {
	Render(w io.Writer, src Source, ds []Diagnostic) error
}

// Caret writes one diagnostic in the canonical human layout: "file:line:col:
// msg", then the offending source line and a caret under the column. This is the
// exact byte layout the CLI has always produced (locked by A7); it is the single
// shared primitive so the renderers and the CLI never drift.
func Caret(w io.Writer, filename string, lines [][]byte, line, col int, msg string) {
	fmt.Fprintf(w, "%s:%d:%d: %s\n", filename, line, col, msg)
	if line >= 1 && line <= len(lines) {
		if col < 1 {
			col = 1
		}
		fmt.Fprintf(w, "  %s\n", lines[line-1])
		fmt.Fprintf(w, "  %s^\n", strings.Repeat(" ", col-1))
	}
}

// sourceLines splits src into the display lines snippets and carets index by
// 1-based line number. It breaks on '\n' and trims a trailing '\r' from each
// line, so a "\r\n"-terminated source renders without a stray carriage return;
// for "\n"-only sources (every A7 example) it is exactly bytes.Split and changes
// nothing. The resulting line count and indices match the scanner's line numbers
// for "\n" and "\r\n" sources alike (see offset.NewFile).
func sourceLines(src []byte) [][]byte {
	lines := bytes.Split(src, []byte("\n"))
	for i, ln := range lines {
		lines[i] = bytes.TrimSuffix(ln, []byte("\r"))
	}
	return lines
}

// Human renders the caret view: file:line:col, the source line, and a caret —
// colorless and byte-identical to the CLI's historical output. It preserves the
// given order (source order, as the front-ends emit) rather than re-sorting.
type Human struct{}

func (Human) Render(w io.Writer, src Source, ds []Diagnostic) error {
	lines := sourceLines(src.Bytes)
	for _, d := range ds {
		Caret(w, src.Name, lines, d.Primary.Line, d.Primary.Column, d.Message)
	}
	return nil
}

// Agent renders terse, located, greppable text for non-tty consumers: one line
// per diagnostic, "severity[CODE]: file:line:col: message", in deterministic
// position order, with the count summary emitted last so truncation from the top
// can't drop it. The severity-and-code prefix leads each line so a harness can
// grep '^error\[' to count and route by code (A8); the located "file:line:col:"
// follows so existing position scrapers still match. No color.
type Agent struct{}

func (Agent) Render(w io.Writer, src Source, ds []Diagnostic) error {
	sorted := sortByPosition(ds)

	var errs, warns int
	for _, d := range sorted {
		sev := d.Severity
		if sev == "" {
			sev = SeverityError
		}
		if sev == SeverityWarning {
			warns++
		} else {
			errs++
		}
		bracket := ""
		if code := agentCode(d); code != "" {
			bracket = "[" + code + "]"
		}
		loc := src.Name
		if d.Primary.Line > 0 {
			loc = fmt.Sprintf("%s:%d:%d", src.Name, d.Primary.Line, d.Primary.Column)
		}
		fmt.Fprintf(w, "%s%s: %s: %s\n", sev, bracket, loc, d.Message)
		// A grounded suggestion ("did you mean …") rides under its diagnostic as an
		// indented help line. It starts with spaces, not "error[", so a harness
		// grepping '^error\[' still counts one line per diagnostic (A8).
		for _, s := range d.Suggestions {
			fmt.Fprintf(w, "  help: %s\n", s.Message)
		}
	}

	if errs+warns > 0 {
		parts := make([]string, 0, 2)
		if errs > 0 {
			parts = append(parts, plural(errs, "error"))
		}
		if warns > 0 {
			parts = append(parts, plural(warns, "warning"))
		}
		fmt.Fprintf(w, "found %s\n", strings.Join(parts, ", "))
	}
	return nil
}

// LLM renders the agent view, then appends a "how to fix:" section: a grounded
// repair block per distinct code present, drawn from the catalog (Explain). It is a
// strict superset of Agent — the same located, greppable diagnostic lines and the
// same trailing count, untouched — followed, for each code in the order it first
// appears by position, by the catalog's Title and Fix and then an indented "why:"
// line carrying the catalog Explanation. The Explanation is where the chip-specific
// facts live (the set of built-in types, that there are no variadics or multiple
// return values, and so on); a terse code alone leaves a model to guess, and it
// guesses by analogy to other languages (an "undefined type: int64" becomes "i64",
// still undefined), so grounding the code with the explanation is what actually
// shortens the repair loop (§5.5). A harness that only greps the diagnostic lines
// ('^error\[') sees exactly what Agent emits — every appendix line is indented or
// prose, so none of them match.
type LLM struct{}

func (LLM) Render(w io.Writer, src Source, ds []Diagnostic) error {
	if err := (Agent{}).Render(w, src, ds); err != nil {
		return err
	}
	var b strings.Builder
	seen := make(map[string]bool)
	for _, code := range codesByPosition(ds) {
		if seen[code] {
			continue
		}
		seen[code] = true
		entry, ok := Explain(code)
		if !ok || entry.Fix == "" {
			continue
		}
		fmt.Fprintf(&b, "  %s (%s): %s\n", code, entry.Title, entry.Fix)
		// The Fix is a one-line directive; the Explanation grounds it in chip's
		// actual rules. It rides on an indented "why:" line so a '^error\[' grep is
		// unaffected (A8) and the Fix line above stays byte-stable for consumers.
		if entry.Explanation != "" {
			fmt.Fprintf(&b, "    why: %s\n", entry.Explanation)
		}
	}
	if b.Len() == 0 {
		return nil // no catalogued code present: nothing grounded to add
	}
	if _, err := io.WriteString(w, "how to fix:\n"); err != nil {
		return err
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// JSON renders the Result envelope. It enriches each diagnostic's primary with
// the source snippet and a rendered caret block (so a JSON consumer never has to
// re-implement formatting), and sets ok from whether any error-severity
// diagnostic is present. Run/check callers that have counts and stats build a
// richer Result directly and marshal it with MarshalResult.
type JSON struct{}

func (JSON) Render(w io.Writer, src Source, ds []Diagnostic) error {
	res := Result{
		OK:          !hasErrors(ds),
		Diagnostics: Enrich(ds, src),
	}
	return MarshalResult(w, &res, src)
}

// MarshalResult enriches the result's diagnostics from src (snippet, rendered)
// and writes the envelope as one compact JSON line. It is the entry point for
// callers (the run/check paths) that assemble a full Result with stats and
// suppression counts.
func MarshalResult(w io.Writer, res *Result, src Source) error {
	res.Diagnostics = Enrich(res.Diagnostics, src)
	if res.Diagnostics == nil {
		res.Diagnostics = []Diagnostic{} // stable [] for parsers, never null
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

// Enrich fills the source-derived presentation fields the detector leaves empty:
// the file name, the line snippet, point-collapsed End* spans, the highlight
// column range, and the rendered caret block. It copies, leaving the input
// diagnostics untouched.
func Enrich(ds []Diagnostic, src Source) []Diagnostic {
	if len(ds) == 0 {
		return ds
	}
	lines := sourceLines(src.Bytes)
	out := make([]Diagnostic, len(ds))
	for i, d := range ds {
		p := d.Primary
		if p.File == "" {
			p.File = src.Name
		}
		if p.Line >= 1 && p.Line <= len(lines) {
			p.Snippet = string(lines[p.Line-1])
		}
		if p.EndLine == 0 {
			p.EndLine = p.Line
		}
		if p.EndColumn == 0 {
			p.EndColumn = p.Column
		}
		if p.EndOffset == 0 {
			p.EndOffset = p.Offset
		}
		if p.Highlight == nil && p.Column > 0 {
			p.Highlight = []int{p.Column, p.EndColumn}
		}
		d.Primary = p

		var buf bytes.Buffer
		Caret(&buf, p.File, lines, p.Line, p.Column, d.Message)
		d.Rendered = strings.TrimRight(buf.String(), "\n")
		out[i] = d
	}
	return out
}

// sortByPosition returns a copy of ds ordered by primary position (line, then
// column) — the deterministic display order the agent and llm renderers share, so
// the llm appendix reads in the same order as the diagnostics above it.
func sortByPosition(ds []Diagnostic) []Diagnostic {
	sorted := append([]Diagnostic(nil), ds...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].Primary, sorted[j].Primary
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	return sorted
}

// codesByPosition returns the agent codes of ds in display order (by position),
// repeated as the diagnostics repeat. The llm renderer dedupes them to build its
// one-line-per-code appendix.
func codesByPosition(ds []Diagnostic) []string {
	sorted := sortByPosition(ds)
	codes := make([]string, 0, len(sorted))
	for _, d := range sorted {
		if c := agentCode(d); c != "" {
			codes = append(codes, c)
		}
	}
	return codes
}

// agentCode is the bracketed code the agent renderer leads with: the assigned
// Code when a later slice has set one, otherwise the diagnostic's phase prefix
// (PAR/TYP/…), so the prefix is greppable and routable before numbered codes
// exist. Empty when neither is known (the prefix is then omitted).
func agentCode(d Diagnostic) string {
	if d.Code != "" {
		return d.Code
	}
	return prefixForPhase(d.Phase)
}

// prefixForPhase maps a phase to its stable error-code prefix (the catalog:
// USE/PAR/TYP/NAM/RUN/RES/LNT). Numbered codes (TYP003) extend a prefix in a
// later slice; until then the prefix alone names the category.
func prefixForPhase(p Phase) string {
	switch p {
	case PhaseUsage:
		return "USE"
	case PhaseParse:
		return "PAR"
	case PhaseType:
		return "TYP"
	case PhaseName:
		return "NAM"
	case PhaseRuntime:
		return "RUN"
	case PhaseResource:
		return "RES"
	case PhaseLint:
		return "LNT"
	}
	return ""
}

// plural renders "1 error" / "2 errors".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
