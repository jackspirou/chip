package diag

// This file defines the diagnostic spine: the one structured shape every chip
// front-end (parser, type checker, streaming checker, runtime, lint) produces
// and the renderers consume. Detection produces data; renderers produce
// presentation (plan decision D6), so per-model tuning never moves the detector
// and retries reproduce.
//
// Front-ends fill position, message, phase, and severity; the renderers present
// those. Code and suggestions (with their applicability) are filled too: Code is
// derived from phase and message by the catalog's Classify, and a grounded "did
// you mean" rides as a Suggestion. The remaining schema fields (expected/found,
// related, repair, help) are reserved for later slices.

// Severity classifies a diagnostic's impact. An error makes a program invalid; a
// warning is advisory (lint) and only blocks under --strict.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Phase names the stage that produced a diagnostic, so the diagnostic is
// routable on its own; the error-code prefix mirrors it (PAR/TYP/NAM/RUN/...).
type Phase string

const (
	PhaseUsage    Phase = "usage"    // driver/usage: bad flag, missing source
	PhaseParse    Phase = "parse"    // syntax
	PhaseType     Phase = "type"     // type checking
	PhaseName     Phase = "name"     // name resolution
	PhaseRuntime  Phase = "runtime"  // execution fault
	PhaseResource Phase = "resource" // cap hit: timeout, step/output budget
	PhaseLint     Phase = "lint"     // style/quality warning
)

// Applicability says how safely a suggested edit may be applied (rustfix's
// model). A harness auto-applies only Machine edits, lets the model choose among
// Maybe, and never blindly applies Placeholder — the guardrail that makes
// auto-repair safe. The "did you mean" suggestion is tagged Maybe, so a harness
// surfaces it without auto-applying.
type Applicability string

const (
	Machine     Applicability = "machine"
	Maybe       Applicability = "maybe"
	Placeholder Applicability = "placeholder"
	Unspecified Applicability = "unspecified"
)

// Primary locates a diagnostic's main span. It carries line/column (for humans)
// and byte offsets (half-open [Offset, EndOffset); agents apply edits on bytes).
// Snippet and Highlight are presentation the renderers fill from the source; the
// detector leaves them empty. For a point diagnostic the End fields equal the
// start.
type Primary struct {
	File      string `json:"file,omitempty"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
	Offset    int    `json:"offset"`
	EndOffset int    `json:"endOffset,omitempty"`
	Snippet   string `json:"snippet,omitempty"`
	Highlight []int  `json:"highlight,omitempty"` // [startColumn, endColumn] on the primary line
}

// Related is a secondary location relevant to a diagnostic (rustc children),
// flat and homogeneous with Primary — one level, same shape.
type Related struct {
	File      string `json:"file,omitempty"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Offset    int    `json:"offset"`
	EndOffset int    `json:"endOffset,omitempty"`
	Message   string `json:"message"`
}

// Edit is a byte-range replacement: insertion is an empty range. Edits within a
// suggestion must not overlap; order decides same-point insertion.
type Edit struct {
	Offset    int    `json:"offset"`
	EndOffset int    `json:"endOffset"`
	NewText   string `json:"newText"`
}

// Suggestion is a proposed fix: a message plus an applicability tag and the
// concrete byte edits. The type checker populates these for "did you mean".
type Suggestion struct {
	Message       string        `json:"message"`
	Applicability Applicability `json:"applicability,omitempty"`
	Edit          []Edit        `json:"edit,omitempty"`
}

// Repair routes a diagnostic to a fix: ID is a stable typed category (telemetry/
// routing); FixID groups instances so one repair class applies across a stream.
type Repair struct {
	ID    string `json:"id,omitempty"`
	FixID string `json:"fixId,omitempty"`
}

// Diagnostic is the single structured shape every front-end produces. Code and
// Suggestions are filled (by Classify and the type checker); Expected/Found,
// Related, Repair, and Help remain for later slices; Rendered is filled by the
// JSON renderer.
type Diagnostic struct {
	Code        string       `json:"code,omitempty"`
	Severity    Severity     `json:"severity"`
	Phase       Phase        `json:"phase,omitempty"`
	Message     string       `json:"message"`
	Expected    string       `json:"expected,omitempty"`
	Found       string       `json:"found,omitempty"`
	Primary     Primary      `json:"primary"`
	Related     []Related    `json:"related,omitempty"`
	Repair      *Repair      `json:"repair,omitempty"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
	Help        string       `json:"help,omitempty"`
	Rendered    string       `json:"rendered,omitempty"`
	RootCause   bool         `json:"rootCause,omitempty"`
}

// Stats summarizes a run for the JSON envelope. Fields are filled by callers that
// have the numbers (the run path); zero values are omitted.
type Stats struct {
	ItemsParsed int   `json:"itemsParsed,omitempty"`
	StmtsRun    int   `json:"stmtsRun,omitempty"`
	Steps       int   `json:"steps,omitempty"`
	DurationMs  int64 `json:"durationMs,omitempty"`
	BytesOut    int   `json:"bytesOut,omitempty"`
}

// Saved records a captured source buffer (the --save family). Filled by the run
// path when capture is on.
type Saved struct {
	Path     string `json:"path"`
	Bytes    int    `json:"bytes"`
	Complete bool   `json:"complete"`
}

// Result is the JSON envelope (--format=json). SuppressedCascades makes cascade
// pruning auditable; TruncatedDiagnostics makes any cap explicit (no silent
// caps). Stats and Saved are filled only by callers that have them.
type Result struct {
	OK                   bool         `json:"ok"`
	Phase                Phase        `json:"phase,omitempty"`
	ChipVersion          string       `json:"chipVersion,omitempty"`
	RanToCompletion      bool         `json:"ranToCompletion"`
	Diagnostics          []Diagnostic `json:"diagnostics"`
	TruncatedDiagnostics int          `json:"truncatedDiagnostics"`
	SuppressedCascades   int          `json:"suppressedCascades"`
	Stats                *Stats       `json:"stats,omitempty"`
	Saved                *Saved       `json:"saved,omitempty"`
}

// Diagnoser is implemented by front-end error types that describe themselves as
// structured diagnostics. The CLI's reporter and the library converter use it
// instead of type-switching over every front-end error type (D6).
type Diagnoser interface {
	Diagnostics() []Diagnostic
}

// From converts a known front-end error into structured diagnostics, or returns
// nil when err carries none (e.g. a plain I/O error), so callers can fall back.
func From(err error) []Diagnostic {
	if d, ok := err.(Diagnoser); ok {
		return d.Diagnostics()
	}
	return nil
}

// hasErrors reports whether any diagnostic is error severity (an empty severity
// counts as error). It backs the envelope's ok bit.
func hasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity != SeverityWarning {
			return true
		}
	}
	return false
}
