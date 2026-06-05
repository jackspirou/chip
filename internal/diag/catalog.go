package diag

// This file is the error-code catalog and the classifier that assigns a code to
// every diagnostic. A code is phase-prefixed (PAR/TYP/NAM/RUN/RES/LNT/USE) so it
// is routable on its own (plan: "stable, explainable codes"), and every code that
// Classify can return has a CatalogEntry here — so `chip explain <code>` resolves
// for any code chip ever prints (the 3.5 verify bar).
//
// Detection is unchanged: codes are derived from a diagnostic's phase and message
// by Classify, never stored in the front-ends. Presentation derives the code; the
// detector stays frozen (D6).

import (
	"sort"
	"strings"
)

// CatalogEntry is the long-form explanation behind a code: a one-line title, a
// paragraph of explanation, a minimal repro that triggers it, and how to fix it.
// `chip explain <code>` prints these.
type CatalogEntry struct {
	Code        string
	Phase       Phase
	Title       string
	Explanation string
	Repro       string
	Fix         string
}

// catalog is the assigned codes in display order (by prefix, then number). Codes
// are assigned once and never reused; new diagnostics get new codes (plan).
var catalog = []CatalogEntry{
	{
		Code: "USE001", Phase: PhaseUsage,
		Title:       "usage error",
		Explanation: "The command was invoked incorrectly: an unknown flag, a missing flag argument, or a bad value (for example an unrecognized --format).",
		Repro:       "chip run --format=bogus -e 'print(1)'",
		Fix:         "Check the flag spelling and value; run `chip help` for the accepted flags.",
	},
	{
		Code: "PAR001", Phase: PhaseParse,
		Title:       "syntax error",
		Explanation: "The source does not parse: a token was unexpected, or a bracket, parenthesis, or literal was left unclosed. Parsing reports the first problem in each construct and resynchronizes, so independent syntax errors are all reported at once.",
		Repro:       "chip run -e 'print(1 +'",
		Fix:         "Close the construct or remove the stray token at the reported position.",
	},
	{
		Code: "NAM001", Phase: PhaseType,
		Title:       "undefined name",
		Explanation: "A name was used that is not declared in any enclosing scope and is not a builtin. If a name with a similar spelling is in scope, the diagnostic suggests it (\"did you mean\"); a grounded suggestion is offered only when one is close enough, never guessed.",
		Repro:       "chip run -e 'print(undefinedName)'",
		Fix:         "Declare the name before use, or correct the spelling to a name that is in scope.",
	},
	{
		Code: "NAM002", Phase: PhaseType,
		Title:       "undefined type",
		Explanation: "A type name was used that chip does not define. chip's types are int, float, string, bool, and slices of those ([]int and so on).",
		Repro:       "chip run -e 'func f(x foo) { }'",
		Fix:         "Use one of chip's built-in type names, or a slice of one.",
	},
	{
		Code: "NAM003", Phase: PhaseType,
		Title:       "redeclared name",
		Explanation: "A name was declared twice where redefinition is not allowed: a duplicate function, a duplicate parameter, or a second := for a name already bound in the same block. (Top-level := may rebind, matching the REPL and the streaming runtime.)",
		Repro:       "chip run -e 'func f(x int, x int) { }'",
		Fix:         "Rename one of the declarations, or remove the duplicate.",
	},
	{
		Code: "TYP001", Phase: PhaseType,
		Title:       "type mismatch",
		Explanation: "An expression's type does not fit where it is used: the two sides of an assignment or comparison differ, an operator is applied to the wrong type, a condition is not bool, an index is not int, or a returned or argument value has the wrong type.",
		Repro:       "chip run -e 'print(1 + \"x\")'",
		Fix:         "Make the types agree — convert one side, or use a value of the expected type.",
	},
	{
		Code: "TYP002", Phase: PhaseType,
		Title:       "missing return",
		Explanation: "A function that declares a result must return a value on every path: it either falls off the end without returning, or a `return` with no value appears where a value is expected.",
		Repro:       "chip run -e 'func f() int { }\nprint(f())'",
		Fix:         "Add a `return <value>` so every path returns the declared type.",
	},
	{
		Code: "TYP003", Phase: PhaseType,
		Title:       "wrong number of arguments",
		Explanation: "A call passes a different number of arguments than the function or builtin accepts, or a return statement yields a different number of values than declared. chip does not support variadic calls or multiple return values.",
		Repro:       "chip run -e 'func f(x int) { }\nf(1, 2)'",
		Fix:         "Pass exactly the declared number of arguments.",
	},
	{
		Code: "TYP004", Phase: PhaseType,
		Title:       "not a value",
		Explanation: "Something that is not a value was used as one: a builtin or type name used without calling it, a non-function called, a package name used as a value, or a qualified name (pkg.Member) used outside a direct call.",
		Repro:       "chip run -e 'x := print'",
		Fix:         "Call the function or builtin, or use an actual value in this position.",
	},
	{
		Code: "RUN001", Phase: PhaseRuntime,
		Title:       "division by zero",
		Explanation: "A `/` or `%` was evaluated with a zero divisor at run time. chip's checker cannot prove a divisor non-zero in general, so this surfaces as a runtime fault: the program ran up to this point and then threw.",
		Repro:       "chip run -e 'print(1 / 0)'",
		Fix:         "Guard the divisor (`if d != 0 { ... }`) before dividing.",
	},
	{
		Code: "RUN002", Phase: PhaseRuntime,
		Title:       "index out of range",
		Explanation: "A slice or string was indexed with an offset outside [0, len). This is a runtime fault: the program ran up to this point and then threw.",
		Repro:       "chip run -e 'xs := []int{1}\nprint(xs[5])'",
		Fix:         "Check the index against len before indexing.",
	},
	{
		Code: "RUN003", Phase: PhaseRuntime,
		Title:       "call stack too deep",
		Explanation: "Calls nested deeper than the interpreter's call-depth limit — almost always unbounded recursion with no base case. chip recurses on the host stack, so this is the event a native runtime would kill as a stack overflow.",
		Repro:       "chip run -e 'func loop() { loop() }\nloop()'",
		Fix:         "Add or fix the base case so the recursion terminates; if the depth is genuinely needed, raise --max-depth.",
	},
	{
		Code: "RUN004", Phase: PhaseRuntime,
		Title:       "runtime fault",
		Explanation: "The program ran and then threw for a reason other than the specific runtime codes above (for example an unsupported operation reached at run time).",
		Repro:       "",
		Fix:         "Read the message and position; the fault occurred while executing that statement.",
	},
	{
		Code: "RES001", Phase: PhaseResource,
		Title:       "timeout",
		Explanation: "The run exceeded its wall-clock budget (--timeout) and was cancelled between steps.",
		Repro:       "",
		Fix:         "Reduce the work, or raise --timeout if the run legitimately needs longer.",
	},
	{
		Code: "RES002", Phase: PhaseResource,
		Title:       "step budget exceeded",
		Explanation: "The run exceeded its interpreter-step budget (--max-steps). The step count is deterministic, so the same input hits the cap at the same point every time.",
		Repro:       "",
		Fix:         "Reduce the work, or raise --max-steps if the run legitimately needs more steps.",
	},
	{
		Code: "RES003", Phase: PhaseResource,
		Title:       "output limit exceeded",
		Explanation: "The program produced more output than the cap allows (--max-output); output was truncated so a runaway print cannot flood the consumer.",
		Repro:       "",
		Fix:         "Print less, or raise --max-output.",
	},
	{
		Code: "LNT001", Phase: PhaseLint,
		Title:       "unused variable",
		Explanation: "A variable is declared but never read. An unread binding is usually a leftover or a typo at the use site.",
		Repro:       "chip lint -e 'x := 1\nprint(2)'",
		Fix:         "Use the variable, or remove its declaration.",
	},
	{
		Code: "LNT002", Phase: PhaseLint,
		Title:       "unused function",
		Explanation: "A function other than main is declared but never called.",
		Repro:       "chip lint -e 'func f() { }\nprint(1)'",
		Fix:         "Call the function, or remove it.",
	},
	{
		Code: "LNT003", Phase: PhaseLint,
		Title:       "unused import",
		Explanation: "An imported package is never referenced as a qualifier (pkg.Member).",
		Repro:       "",
		Fix:         "Use the package, or remove the import.",
	},
	{
		Code: "LNT004", Phase: PhaseLint,
		Title:       "unreachable code",
		Explanation: "A statement follows one that always transfers control away (a return, or an infinite loop), so it can never run.",
		Repro:       "chip lint -e 'func f() int { return 1\nprint(2) }\nprint(f())'",
		Fix:         "Remove the unreachable statement, or fix the control flow above it.",
	},
	{
		Code: "LNT005", Phase: PhaseLint,
		Title:       "used before definition",
		Explanation: "A function is called before its own definition in source order. The forward reference still resolves (the stream reads ahead), but it makes the stream wait; `chip fmt` reorders definitions to remove it.",
		Repro:       "",
		Fix:         "Move the definition above its first use, or run `chip fmt`.",
	},
	{
		Code: "LNT006", Phase: PhaseLint,
		Title:       "missing return",
		Explanation: "A function that declares a result can fall off its end without returning. Reported as a warning by `chip lint`; the streaming runtime treats the same condition as a hard error before the function runs.",
		Repro:       "chip lint -e 'func f() int { print(1) }\nprint(f())'",
		Fix:         "Add a `return <value>` so every path returns the declared type.",
	},
}

// byCode indexes the catalog for O(1) lookup.
var byCode = func() map[string]CatalogEntry {
	m := make(map[string]CatalogEntry, len(catalog))
	for _, e := range catalog {
		m[e.Code] = e
	}
	return m
}()

// Explain returns the catalog entry for a code (case-insensitive), or false if no
// such code exists.
func Explain(code string) (CatalogEntry, bool) {
	e, ok := byCode[strings.ToUpper(strings.TrimSpace(code))]
	return e, ok
}

// Codes returns every assigned code in display order, for `chip explain` with no
// argument (a discoverable index of the catalog).
func Codes() []CatalogEntry {
	out := make([]CatalogEntry, len(catalog))
	copy(out, catalog)
	return out
}

// Classify returns the stable code for a diagnostic. A code already set on the
// diagnostic wins; otherwise the code is derived from the phase and message. The
// result is always a code that Explain resolves, so a classified diagnostic can
// always be explained.
func Classify(d Diagnostic) string {
	if d.Code != "" {
		return d.Code
	}
	switch d.Phase {
	case PhaseUsage:
		return "USE001"
	case PhaseParse:
		return "PAR001"
	case PhaseRuntime:
		return classifyRuntime(d.Message)
	case PhaseResource:
		return classifyResource(d.Message)
	case PhaseLint:
		return classifyLint(d.Message)
	default: // type, name, or unset — name resolution rides the type phase
		return classifyType(d.Message)
	}
}

// classifyType routes a type/name-resolution message. Name-resolution faults
// (undefined, redeclared) get NAM codes even though they ride the type phase;
// everything else is a TYP code, defaulting to the type-mismatch bucket.
func classifyType(msg string) string {
	switch {
	case strings.HasPrefix(msg, "undefined type:"):
		return "NAM002"
	case strings.HasPrefix(msg, "undefined:"):
		return "NAM001"
	case strings.Contains(msg, "redeclared"), strings.HasPrefix(msg, "duplicate parameter "):
		return "NAM003"
	case strings.HasPrefix(msg, "missing return"):
		return "TYP002"
	case strings.HasPrefix(msg, "wrong number of arguments"),
		strings.HasPrefix(msg, "too many return values"),
		strings.HasPrefix(msg, "multiple return values"),
		strings.Contains(msg, "expects") && strings.Contains(msg, "argument"):
		return "TYP003"
	case strings.Contains(msg, "is not a value"),
		strings.HasPrefix(msg, "cannot call non-function"),
		strings.HasPrefix(msg, "only direct function calls"),
		strings.HasPrefix(msg, "qualified name is only valid"),
		strings.Contains(msg, "is a package, not a value"),
		strings.Contains(msg, "is not exported by"),
		msg == "type used as a value":
		return "TYP004"
	default:
		return "TYP001"
	}
}

func classifyRuntime(msg string) string {
	switch {
	case msg == "division by zero":
		return "RUN001"
	case strings.HasPrefix(msg, "index out of range"):
		return "RUN002"
	case msg == "call stack too deep":
		return "RUN003"
	default:
		return "RUN004"
	}
}

func classifyResource(msg string) string {
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"):
		return "RES001"
	case strings.Contains(msg, "output"):
		return "RES003"
	default:
		return "RES002"
	}
}

func classifyLint(msg string) string {
	switch {
	case strings.Contains(msg, "imported and not used"):
		return "LNT003"
	case strings.Contains(msg, "declared and not used"):
		return "LNT001"
	case strings.Contains(msg, "is never used"):
		return "LNT002"
	case strings.HasPrefix(msg, "unreachable"):
		return "LNT004"
	case strings.Contains(msg, "used before its definition"):
		return "LNT005"
	case strings.HasPrefix(msg, "missing return"):
		return "LNT006"
	default:
		return "LNT001"
	}
}

// WithCodes returns a copy of ds with each diagnostic's Code filled in by
// Classify where it was empty, so the renderers and JSON envelope carry a stable
// code without the front-ends having to assign one. Diagnostics that already
// carry a code are left unchanged.
func WithCodes(ds []Diagnostic) []Diagnostic {
	if len(ds) == 0 {
		return ds
	}
	out := make([]Diagnostic, len(ds))
	for i, d := range ds {
		if d.Code == "" {
			d.Code = Classify(d)
		}
		out[i] = d
	}
	return out
}

// sortCatalog keeps the catalog in code order at init in case entries are added
// out of order; display and explain rely on a stable order.
func init() {
	sort.SliceStable(catalog, func(i, j int) bool { return catalog[i].Code < catalog[j].Code })
}
