// Command chip runs, formats, and inspects chip source files.
package main

import (
	"bytes"
	"context"
	_ "embed" // for the //go:embed of the agent contract (chip prompt)
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/jackspirou/chip/internal/analyze"
	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/compiler"
	"github.com/jackspirou/chip/internal/diag"
	"github.com/jackspirou/chip/internal/fix"
	"github.com/jackspirou/chip/internal/format"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/stream"
)

// errReported marks an error that has already been printed with full context,
// so main exits non-zero without printing it again.
var errReported = errors.New("reported")

// Build metadata. Release builds stamp these via -ldflags (-X main.version=…);
// otherwise versionInfo falls back to runtime/debug build info.
var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	os.Exit(exitStatus(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}

// exitStatus maps the error a command returns to a process exit code (D22): the
// canonical, sysexits-aligned table a harness branches on without scraping text
// — 0 clean, 1 ran-then-threw, 2 static error, 64 usage, 66 no input, plus the
// codes commands set directly. Diagnostics with source context are already on
// stderr by now; this only prints a terse "chip: …" line for the driver errors
// (usage/no-input) that have nothing to render.
func exitStatus(err error) int {
	if err == nil {
		return 0 // success / clean
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code // the command already reported and chose its code
	}
	var ue *usageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, "chip:", ue.Error())
		return 64 // EX_USAGE — bad flag / args
	}
	var ne *noInputError
	if errors.As(err, &ne) {
		fmt.Fprintln(os.Stderr, "chip:", ne.Error())
		return 66 // EX_NOINPUT — source missing / unreadable
	}
	if !errors.Is(err, errReported) {
		fmt.Fprintln(os.Stderr, "chip:", err)
	}
	return 1 // generic fault
}

// exitError carries a process exit code up to main. Its diagnostics are assumed
// already printed, so exitStatus returns the code without printing again.
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// usageError is a bad invocation — unknown flag, missing flag argument, or a
// command misused. It maps to exit 64 (EX_USAGE); exitStatus prints its message
// (there is no source context to render).
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// noInputError is a missing or unreadable source: a file that does not exist, or
// stdin requested with nothing to read. It maps to exit 66 (EX_NOINPUT).
type noInputError struct{ err error }

func (e *noInputError) Error() string { return e.err.Error() }
func (e *noInputError) Unwrap() error { return e.err }

// exitCodeFor maps an already-reported front-end error to its process exit code
// by phase: a runtime fault means the program ran and then threw (1); a parse,
// type, name, or lint error means the source needs fixing before it can run (2);
// a resource cap is its own code — 124 for a wall-clock timeout (RES001), 125 for
// a step or output budget (RES002/RES003). An error that carries no structured
// diagnostics is a generic fault (1).
func exitCodeFor(err error) int {
	ds := diag.From(err)
	if len(ds) == 0 {
		return 1
	}
	static := false
	for _, d := range ds {
		switch d.Phase {
		case diag.PhaseRuntime:
			return 1
		case diag.PhaseResource:
			if diag.Classify(d) == "RES001" {
				return 124 // wall-clock timeout
			}
			return 125 // step or output budget
		case diag.PhaseParse, diag.PhaseType, diag.PhaseName, diag.PhaseLint:
			static = true
		}
	}
	if static {
		return 2
	}
	return 1
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return &exitError{code: 64} // EX_USAGE — no command given
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:], stdin, stdout, stderr)
	case "repl":
		return cmdRepl(stdin, stdout)
	case "fmt":
		return cmdFmt(args[1:], stdin, stdout, stderr)
	case "check":
		return cmdCheck(args[1:], stdin, stdout, stderr)
	case "lint":
		return cmdLint(args[1:], stdin, stderr)
	case "fix":
		return cmdFix(args[1:], stdin, stdout, stderr)
	case "dump":
		return cmdDump(args[1:], stdin, stdout, stderr)
	case "ast":
		return cmdAst(args[1:], stdin, stdout, stderr)
	case "explain":
		return cmdExplain(args[1:], stdout)
	case "prompt":
		return cmdPrompt(args[1:], stdout)
	case "skill":
		return cmdSkill(args[1:], stdout, stderr)
	case "lsp":
		return cmdLsp(args[1:], stdin, stdout)
	case "session":
		return cmdSession(args[1:], stdin, stdout)
	case "version", "--version":
		return cmdVersion(args[1:], stdout)
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	default:
		return cmdRun(args, stdin, stdout, stderr) // `chip file.chp` is shorthand for `chip run file.chp`
	}
}

// cmdRun streams and executes a program, resolving its imports relative to the
// source file's directory (or the working directory for stdin/-e source).
func cmdRun(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, renderer, _, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	cfg, args, ferr := parseRunFlags(args)
	if ferr != nil {
		return ferr
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}
	// Look before you clobber (D8): if a capture is bound for an existing explicit
	// path without --force, refuse up front — before running — so a doomed
	// invocation costs nothing and, for --save-on-error, a later fault is
	// guaranteed somewhere to land. A bare --save (no path) picks a fresh $TMPDIR
	// name and so never clobbers.
	if cfg.save != saveNone && cfg.savePath != "" && !cfg.force {
		if _, statErr := os.Stat(cfg.savePath); statErr == nil {
			return clobberRefused(stderr, cfg.savePath)
		}
	}
	ctx := context.Background()
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}
	// --no-imports seals the only host-access vector by swapping the filesystem
	// loader for one that refuses every import; bundled stdlib still resolves.
	loader := stream.DirLoader(sourceDir(path))
	if cfg.noImports {
		loader = stream.DenyLoader()
	}
	rerr := stream.RunSealed(ctx, bytes.NewReader(src), stdout, loader, cfg.lim, cfg.noPrelude)
	if rerr != nil {
		report(stderr, renderer, path, src, rerr)
	}
	// Capture the source if asked — always for --save, only on a fault for
	// --save-on-error. The whole program is already buffered in src, so the saved
	// file always carries every line, including those past the fault (A5). A save
	// failure (clobber race, write error) is itself EX_CANTCREAT and supersedes the
	// run's exit code: the artifact the caller asked for could not be produced.
	if cfg.save == saveAlways || (cfg.save == saveOnError && rerr != nil) {
		if serr := saveSource(stderr, cfg, src); serr != nil {
			return serr
		}
	}
	if rerr != nil {
		return &exitError{code: exitCodeFor(rerr)}
	}
	return nil
}

// saveSource captures the program source to the target chosen by cfg (plan §4.3):
// the explicit --save=PATH, or a fresh non-clobbering name in $TMPDIR when no path
// was given. Without --force an existing explicit path is never overwritten
// (O_EXCL guards the up-front check against a race); a clobber is refused with
// exit 73 (EX_CANTCREAT). The resolved path is echoed to stderr so a caller —
// especially with the generated $TMPDIR name — knows where the source landed.
func saveSource(stderr io.Writer, cfg runConfig, src []byte) error {
	if cfg.savePath == "" {
		// No path given: create a fresh file in $TMPDIR atomically (O_CREATE|O_EXCL
		// under the hood), so the name never collides and there is nothing to clobber.
		f, err := os.CreateTemp("", "chip-*.chp")
		if err != nil {
			return saveFailed(stderr, err)
		}
		return finishSave(stderr, f, f.Name(), src)
	}
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !cfg.force {
		flag = os.O_WRONLY | os.O_CREATE | os.O_EXCL // refuse to clobber
	}
	f, err := os.OpenFile(cfg.savePath, flag, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return clobberRefused(stderr, cfg.savePath)
		}
		return saveFailed(stderr, err)
	}
	return finishSave(stderr, f, cfg.savePath, src)
}

// finishSave writes src to an already-opened capture file, closes it, and echoes
// where the source landed. A write or close failure is EX_CANTCREAT.
func finishSave(stderr io.Writer, f *os.File, path string, src []byte) error {
	if _, err := f.Write(src); err != nil {
		f.Close()
		return saveFailed(stderr, err)
	}
	if err := f.Close(); err != nil {
		return saveFailed(stderr, err)
	}
	fmt.Fprintf(stderr, "chip: saved source to %s\n", path)
	return nil
}

// clobberRefused reports that a capture target already exists and would be
// overwritten, mapping to exit 73 (EX_CANTCREAT); --force opts into overwriting.
func clobberRefused(stderr io.Writer, path string) error {
	fmt.Fprintf(stderr, "chip: refusing to overwrite %s (pass --force to overwrite)\n", path)
	return &exitError{code: 73}
}

// saveFailed reports an I/O failure capturing the source, mapping to exit 73
// (EX_CANTCREAT) — the requested output file could not be created.
func saveFailed(stderr io.Writer, err error) error {
	fmt.Fprintf(stderr, "chip: cannot save source: %s\n", err)
	return &exitError{code: 73}
}

// saveMode selects when `chip run` captures the program source to disk (plan
// §4.3): never (the default), always, or only when the run faults.
type saveMode int

const (
	saveNone    saveMode = iota // no capture (the default)
	saveAlways                  // --save: capture after the run, success or fault
	saveOnError                 // --save-on-error: capture only when the run faults
)

// runConfig is the set of flags `chip run` accepts beyond the source: resource
// caps (plan §4.1), the sealing knobs for hermetic execution (plan §4.2), and
// source capture (plan §4.3). Its zero value is the default run — no caps, host
// imports allowed, prelude loaded, nothing captured.
type runConfig struct {
	lim       stream.Limits // step/output/depth budget
	timeout   time.Duration // wall-clock limit (0 = none)
	noImports bool          // --no-imports: refuse host imports (bundled stdlib still resolves)
	noPrelude bool          // --no-prelude: also drop the unqualified stdlib prelude
	save      saveMode      // --save / --save-on-error: when to capture source
	savePath  string        // explicit capture path; "" → a fresh name in $TMPDIR
	force     bool          // --force: overwrite an existing capture target
}

// parseRunFlags pulls the flags `chip run` accepts (plan §4.1 caps, §4.2 sealing)
// out of args in any position — like the format flags — and returns the parsed
// config and the remaining args for readSource. The cap flags accept --flag=value
// or --flag value; --no-imports and --no-prelude are booleans. The argument to -e
// is source, never a flag, so -e and its value pass through verbatim.
func parseRunFlags(args []string) (cfg runConfig, rest []string, err error) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-e" { // -e <src>: the source is opaque, keep both verbatim
			rest = append(rest, a)
			if i+1 < len(args) {
				rest = append(rest, args[i+1])
				i++
			}
			continue
		}
		name, val, hasVal := strings.Cut(a, "=")
		switch name {
		case "--no-imports":
			cfg.noImports = true
		case "--no-prelude":
			cfg.noPrelude = true
		case "--save", "--save-on-error":
			// The path is optional and attached-only (--save=PATH); a bare --save
			// captures to a fresh $TMPDIR name. A space-separated value is not read
			// — it would be indistinguishable from the source argument.
			if name == "--save" {
				cfg.save = saveAlways
			} else {
				cfg.save = saveOnError
			}
			if hasVal {
				cfg.savePath = val
			}
		case "--force":
			cfg.force = true
		case "--timeout", "--max-steps", "--max-output", "--max-depth":
			if !hasVal {
				if i+1 >= len(args) {
					return cfg, nil, &usageError{msg: fmt.Sprintf("%s requires a value", name)}
				}
				val = args[i+1]
				i++
			}
			if cerr := setCap(name, val, &cfg.lim, &cfg.timeout); cerr != nil {
				return cfg, nil, cerr
			}
		default:
			rest = append(rest, a)
		}
	}
	return cfg, rest, nil
}

// setCap parses one cap flag's value into lim or timeout.
func setCap(name, val string, lim *stream.Limits, timeout *time.Duration) error {
	switch name {
	case "--timeout":
		d, err := time.ParseDuration(val)
		if err != nil {
			return &usageError{msg: fmt.Sprintf("--timeout: invalid duration %q (e.g. 500ms, 2s)", val)}
		}
		if d < 0 {
			return &usageError{msg: "--timeout must not be negative"}
		}
		*timeout = d
	case "--max-steps":
		n, err := capInt(name, val)
		if err != nil {
			return err
		}
		lim.Steps = n
	case "--max-output":
		n, err := capInt(name, val)
		if err != nil {
			return err
		}
		lim.OutBytes = n
	case "--max-depth":
		n, err := capInt(name, val)
		if err != nil {
			return err
		}
		lim.Depth = int(n)
	}
	return nil
}

// capInt parses a non-negative integer cap value; 0 means "no cap" for that
// dimension, matching the engine's zero-value semantics.
func capInt(name, val string) (int64, error) {
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, &usageError{msg: fmt.Sprintf("%s: invalid value %q (want a non-negative integer)", name, val)}
	}
	if n < 0 {
		return 0, &usageError{msg: fmt.Sprintf("%s must not be negative", name)}
	}
	return n, nil
}

// cmdRepl evaluates source read line by line, carrying definitions over.
func cmdRepl(stdin io.Reader, stdout io.Writer) error {
	return stream.REPL(stdin, stdout)
}

// cmdFmt prints a program in canonical form, rewrites it in place (-w), lists it
// when it is not already canonical (-l, like gofmt -l — for CI), or shows the
// reformatting as a unified diff (-d/--diff). -l and -d exit 2 on a non-canonical
// file so the exit bit alone is the signal (D16); with no flag the canonical
// source is written to stdout. (The old --check is dropped — chip is pre-release,
// and "check" now names only the analysis verb.)
func cmdFmt(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, renderer, _, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	var write, list, diff bool
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" && args[0] != "-e" {
		switch args[0] {
		case "-w":
			write = true
		case "-l":
			list = true
		case "-d", "--diff":
			diff = true
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for fmt: %s", args[0])}
		}
		args = args[1:]
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}
	out, err := format.Source(src)
	if err != nil {
		report(stderr, renderer, path, src, err)
		return &exitError{code: exitCodeFor(err)}
	}
	switch {
	case list:
		if !bytes.Equal(out, src) {
			fmt.Fprintln(stdout, path) // the filename is the signal (gofmt -l)
			return &exitError{code: 2} // non-canonical: "source needs work" (D16)
		}
		return nil
	case diff:
		if !bytes.Equal(out, src) {
			writeUnifiedDiff(stdout, path, src, out)
			return &exitError{code: 2}
		}
		return nil
	case write:
		if !isFile(path) {
			return &usageError{msg: fmt.Sprintf("cannot rewrite %s in place; -w needs a file path", path)}
		}
		return os.WriteFile(path, out, 0o644)
	default:
		_, err = stdout.Write(out)
		return err
	}
}

// cmdCheck statically analyzes a program and reports every parse and type error
// at once (collect-all, cascade-pruned) without running it. It exits 2 when the
// program has errors and 0 when it is clean. With --strict, lint warnings on an
// otherwise clean program are treated as failures too. Diagnostics go to stderr
// in the selected format (--format/--json); there is no program output.
func cmdCheck(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, renderer, asJSON, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	strict := false
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" && args[0] != "-e" {
		switch args[0] {
		case "--strict", "-strict":
			strict = true
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for check: %s", args[0])}
		}
		args = args[1:]
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}

	rep := analyze.Source(src)
	ds := diag.WithCodes(rep.Diagnostics)
	// Lint only runs once the program is otherwise clean: a parse/type error
	// would make Lint repeat and fail, and those diagnostics are already in hand.
	if strict && rep.OK() {
		ds = append(ds, diag.WithCodes(lintDiagnostics(src))...)
	}

	source := diag.Source{Name: path, Bytes: src}
	if asJSON {
		res := diag.Result{
			OK:                 len(ds) == 0,
			Diagnostics:        ds,
			SuppressedCascades: rep.SuppressedCascades,
		}
		_ = diag.MarshalResult(stderr, &res, source)
	} else {
		_ = renderer.Render(stderr, source, ds)
		if rep.SuppressedCascades > 0 {
			noun := "errors"
			if rep.SuppressedCascades == 1 {
				noun = "error"
			}
			fmt.Fprintf(stderr, "(%d cascade-derived %s suppressed)\n", rep.SuppressedCascades, noun)
		}
	}
	if len(ds) > 0 {
		return &exitError{code: 2}
	}
	return nil
}

// cmdLint reports style and correctness issues. A finding is "source needs
// work," so a non-empty report exits 2 (like check); a parse/type error along
// the way maps to its own phase code.
func cmdLint(args []string, stdin io.Reader, stderr io.Writer) error {
	args, renderer, _, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}
	f, perr := parse(src)
	if perr != nil {
		report(stderr, renderer, path, src, perr)
		return &exitError{code: exitCodeFor(perr)}
	}
	info, cerr := check.Check(f)
	if cerr != nil {
		report(stderr, renderer, path, src, cerr)
		return &exitError{code: exitCodeFor(cerr)}
	}
	if issues := lint.Lint(f, info); len(issues) > 0 {
		report(stderr, renderer, path, src, issues)
		return &exitError{code: 2}
	}
	return nil
}

// cmdFix applies the machine-applicable repairs a program's diagnostics carry —
// currently only dead-code removal — and writes the fixed source to stdout, or
// back to the file with -w. Repairs that are only a guess (a "did you mean",
// tagged Maybe) are never applied blind; --all <fixId> opts a whole repair class
// in. --plan shows what would change without touching anything (--json for the
// machine shape). This is the D6 apply path: the diagnostic carries the edits,
// one applier realizes the safe ones.
func cmdFix(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, _, asJSON, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	var plan, write bool
	var allFixID string
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" && args[0] != "-e" {
		name, val, hasVal := strings.Cut(args[0], "=")
		switch name {
		case "--plan":
			plan = true
		case "-w":
			write = true
		case "--all":
			// Accept --all=<fixId> and --all <fixId>; the id never starts with
			// "-", so the space form is unambiguous against the next flag.
			if !hasVal {
				if len(args) < 2 {
					return &usageError{msg: "--all requires a fix id (e.g. --all unreachable)"}
				}
				val = args[1]
				args = args[1:]
			}
			allFixID = val
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for fix: %s", args[0])}
		}
		args = args[1:]
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}

	if plan {
		return emitPlan(stdout, fix.BuildPlan(src, allFixID), asJSON)
	}

	out, applied, _, err := fix.Apply(src, allFixID)
	if err != nil {
		// A bad splice (e.g. two fixes whose edits overlap) is refused rather
		// than guessed: report it and fail static (exit 2).
		fmt.Fprintf(stderr, "chip: cannot apply fixes: %s\n", err)
		return &exitError{code: 2}
	}
	if write {
		if !isFile(path) {
			return &usageError{msg: fmt.Sprintf("cannot rewrite %s in place; -w needs a file path", path)}
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "chip: applied %d fix(es) to %s\n", len(applied), path)
		return nil
	}
	_, err = stdout.Write(out)
	return err
}

// emitPlan writes a fix plan: the machine JSON shape (Zero-style) under --json,
// else a terse one-line-per-fix listing with its disposition (apply or skip) and
// applicability, then the apply/skip tally.
func emitPlan(w io.Writer, plan fix.Plan, asJSON bool) error {
	if asJSON {
		b, err := json.Marshal(plan)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if len(plan.Fixes) == 0 {
		fmt.Fprintln(w, "no fixes available")
		return nil
	}
	for _, f := range plan.Fixes {
		disp := "skip "
		if f.Apply {
			disp = "apply"
		}
		fmt.Fprintf(w, "%s %s:%d:%d %s [%s]\n", disp, f.Code, f.Line, f.Column, f.Message, f.Applicability)
	}
	fmt.Fprintf(w, "%d to apply, %d skipped\n", plan.Applied, plan.Skipped)
	return nil
}

// cmdDump disassembles a program's bytecode (the batch compile path).
func cmdDump(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, renderer, _, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}
	prog, berr := build(src)
	if berr != nil {
		report(stderr, renderer, path, src, berr)
		return &exitError{code: exitCodeFor(berr)}
	}
	fmt.Fprint(stdout, prog.String())
	return nil
}

// cmdAst prints a program's syntax tree.
func cmdAst(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args, renderer, _, ferr := resolveFormat(args, stderr)
	if ferr != nil {
		return ferr
	}
	path, src, err := readSource(args, stdin)
	if err != nil {
		return err
	}
	f, perr := parse(src)
	if perr != nil {
		report(stderr, renderer, path, src, perr)
		return &exitError{code: exitCodeFor(perr)}
	}
	fmt.Fprintln(stdout, ast.Sprint(f))
	return nil
}

// Synthetic source names used as the diagnostic file when a program does not
// come from a path on disk.
const (
	stdinName = "<stdin>" // read from standard input (`-`, or piped with no arg)
	argName   = "<arg>"   // read inline from `-e <src>`
)

// readSource resolves where a command reads its program from, returning a
// display name (used as the diagnostic file), the source bytes, and any error:
//
//   - `-e <src>` → inline source, named "<arg>"
//   - `-`        → standard input, named "<stdin>"
//   - (no args)  → standard input if it is not a terminal, named "<stdin>"
//   - <path>     → the file at <path>, named <path>
func readSource(args []string, stdin io.Reader) (string, []byte, error) {
	if len(args) > 0 && args[0] == "-e" {
		if len(args) < 2 {
			return "", nil, &usageError{msg: "-e requires a source argument"}
		}
		return argName, []byte(args[1]), nil
	}
	if len(args) == 0 || args[0] == "-" {
		if len(args) == 0 && !readableStdin(stdin) {
			return "", nil, &noInputError{err: fmt.Errorf("no source file given")}
		}
		src, err := io.ReadAll(stdin)
		if err != nil {
			return stdinName, nil, &noInputError{err: fmt.Errorf("reading stdin: %w", err)}
		}
		return stdinName, src, nil
	}
	path := args[0]
	src, err := os.ReadFile(path)
	if err != nil {
		return path, nil, &noInputError{err: err}
	}
	return path, src, nil
}

// isFile reports whether name refers to a real path on disk rather than a
// synthetic stdin/-e source, so callers can refuse in-place rewrites of input
// that has nowhere to be written back.
func isFile(name string) bool { return name != stdinName && name != argName }

// sourceDir returns the directory imports resolve against: the file's directory
// for a real path, or the working directory for stdin/-e source.
func sourceDir(name string) string {
	if !isFile(name) {
		return "."
	}
	return filepath.Dir(name)
}

// readableStdin reports whether stdin has data to read without blocking on a
// terminal prompt: true for a pipe or redirect (and for a non-*os.File reader,
// as in tests), false for an interactive terminal.
func readableStdin(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	if !ok {
		return stdin != nil
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func parse(src []byte) (*ast.File, error) {
	p, err := parser.New(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

func build(src []byte) (*code.Program, error) {
	f, err := parse(src)
	if err != nil {
		return nil, err
	}
	info, err := check.Check(f)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(f, info)
}

// lintDiagnostics runs the linter over clean source and returns its findings as
// structured lint-phase diagnostics (warnings, with byte offsets). It returns nil
// when the source does not parse or type-check — those errors are the caller's to
// report from its own analysis pass, not the linter's to repeat.
func lintDiagnostics(src []byte) []diag.Diagnostic {
	f, err := parse(src)
	if err != nil {
		return nil
	}
	info, err := check.Check(f)
	if err != nil {
		return nil
	}
	return lint.Lint(f, info).Diagnostics()
}

// report prints err in the selected format (human caret block, terse agent
// lines, or the JSON envelope) — every front-end error type describes itself as
// structured diagnostics (diag.Diagnoser). An error that carries none (for
// example a plain I/O error) falls back to "filename: err".
func report(w io.Writer, r diag.Renderer, filename string, src []byte, err error) {
	if ds := diag.From(err); len(ds) > 0 {
		_ = r.Render(w, diag.Source{Name: filename, Bytes: src}, diag.WithCodes(ds))
		return
	}
	fmt.Fprintf(w, "%s: %s\n", filename, err)
}

// resolveFormat pulls the output-selection flags every diagnostic-emitting
// command shares — --format auto|human|agent|json, its --json sugar, and
// --no-color — out of args from any position, resolves the renderer (env
// CHIP_FORMAT is the fallback, then auto by whether stderr is a terminal), and
// returns the remaining args for the command's own flag parser. The argument to
// -e is opaque (it is source, never a flag), so it is kept verbatim.
func resolveFormat(args []string, stderr io.Writer) ([]string, diag.Renderer, bool, error) {
	spec := os.Getenv("CHIP_FORMAT")
	kept := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-e":
			kept = append(kept, a)
			if i+1 < len(args) {
				kept = append(kept, args[i+1])
				i++
			}
		case a == "--json" || a == "-json":
			spec = "json"
		case a == "--no-color":
			// No color is emitted yet (the human renderer is colorless), so
			// --no-color and NO_COLOR are accepted and need no action.
		case a == "--format" || a == "-format":
			if i+1 >= len(args) {
				return nil, nil, false, &usageError{msg: "--format requires a value (auto|human|agent|json|llm[:model])"}
			}
			spec = args[i+1]
			i++
		case strings.HasPrefix(a, "--format="):
			spec = strings.TrimPrefix(a, "--format=")
		case strings.HasPrefix(a, "-format="):
			spec = strings.TrimPrefix(a, "-format=")
		default:
			kept = append(kept, a)
		}
	}
	r, isJSON, err := rendererFor(spec, stderr)
	if err != nil {
		return nil, nil, false, err
	}
	return kept, r, isJSON, nil
}

// rendererFor turns a format spec into a renderer. The empty spec and "auto"
// resolve by tty: a terminal gets the human caret view, a pipe or file gets the
// terse agent text (D21/D12) — so a harness scraping a pipe gets greppable
// output by default, while a person at a prompt still sees carets. The bool
// reports whether the JSON envelope was selected (callers with counts build a
// richer Result). "llm" and "llm:<model>" render the grounded LLM view — agent
// diagnostics plus the catalog's repair hints (§5.5).
func rendererFor(spec string, stderr io.Writer) (diag.Renderer, bool, error) {
	switch spec {
	case "", "auto":
		if isTerminal(stderr) {
			return diag.Human{}, false, nil
		}
		return diag.Agent{}, false, nil
	case "human":
		return diag.Human{}, false, nil
	case "agent":
		return diag.Agent{}, false, nil
	case "json":
		return diag.JSON{}, true, nil
	case "llm":
		return diag.LLM{}, false, nil
	}
	if strings.HasPrefix(spec, "llm:") {
		return diag.LLM{}, false, nil
	}
	return nil, false, &usageError{msg: fmt.Sprintf("unknown format %q (want auto|human|agent|json|llm[:model])", spec)}
}

// isTerminal reports whether w is an interactive terminal (a character device),
// so auto can pick the human renderer for a person and the agent renderer for a
// pipe, redirect, or test buffer.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// writeUnifiedDiff writes the line-oriented unified diff turning src into its
// canonical form (the formatter only ever rewrites whole lines), labelled with
// the path. One hunk covering the file is enough to see what -w would do.
func writeUnifiedDiff(w io.Writer, path string, src, formatted []byte) {
	a, b := splitLines(src), splitLines(formatted)
	fmt.Fprintf(w, "--- %s\n+++ %s (formatted)\n", path, path)
	fmt.Fprintf(w, "@@ -1,%d +1,%d @@\n", len(a), len(b))
	for _, op := range lcsDiff(a, b) {
		switch op.tag {
		case opEqual:
			fmt.Fprintf(w, " %s\n", a[op.ai])
		case opDel:
			fmt.Fprintf(w, "-%s\n", a[op.ai])
		case opIns:
			fmt.Fprintf(w, "+%s\n", b[op.bi])
		}
	}
}

// splitLines splits source into logical lines, dropping the single trailing
// newline so a file's lines aren't followed by a spurious empty one.
func splitLines(b []byte) []string {
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// diff op tags for lcsDiff.
const (
	opEqual = iota
	opDel
	opIns
)

type diffOp struct {
	tag    int
	ai, bi int
}

// lcsDiff returns the edit script transforming a into b via a longest-common-
// subsequence dynamic program: equal lines are shared context, the rest are
// deletions from a and insertions from b, in order.
func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{opEqual, i, j})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{opDel, i, j})
			i++
		default:
			ops = append(ops, diffOp{opIns, i, j})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{opDel, i, j})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{opIns, i, j})
	}
	return ops
}

// cmdExplain prints the long-form explanation behind a diagnostic code — what it
// means, a minimal repro, and how to fix it — the crisp expert form that a code
// alone can't carry (the `--explain` model). With no argument it lists every
// code as a discoverable index. Explanations are reference output the user asked
// for, so they go to stdout; an unknown code is a usage error (exit 64).
func cmdExplain(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		for _, e := range diag.Codes() {
			fmt.Fprintf(stdout, "%-6s  %s\n", e.Code, e.Title)
		}
		return nil
	}
	if len(args) > 1 {
		return &usageError{msg: "explain takes a single code (e.g. chip explain TYP001)"}
	}
	entry, ok := diag.Explain(args[0])
	if !ok {
		return &usageError{msg: fmt.Sprintf("unknown code %q (run `chip explain` to list all codes)", args[0])}
	}
	fmt.Fprintf(stdout, "%s [%s]: %s\n\n%s\n", entry.Code, entry.Phase, entry.Title, entry.Explanation)
	if entry.Repro != "" {
		fmt.Fprintf(stdout, "\nrepro:\n    %s\n", entry.Repro)
	}
	if entry.Fix != "" {
		fmt.Fprintf(stdout, "\nfix: %s\n", entry.Fix)
	}
	return nil
}

// promptText is the agent-facing contract — the same body the bundled SKILL.md
// carries — embedded from prompt.md so `chip prompt` can never drift from the
// CLI that ships it (the Zero model, plan §5.2).
//
//go:embed prompt.md
var promptText string

// skillFrontmatter is the Agent Skill's YAML frontmatter (name + description),
// embedded so the binary can assemble a complete SKILL.md from one source. The
// shipped .agents/.claude SKILL.md files are exactly this frontmatter followed by
// promptText; TestSkillContentMatchesBundled locks that equality.
//
//go:embed skill_frontmatter.md
var skillFrontmatter string

// skillContent assembles the full SKILL.md the binary ships: the frontmatter
// (name + description) then the agent contract. `chip skill print` writes these
// bytes and `chip skill install` places them, so the binary is the single source
// of truth for the skill (D24, plan §5.8) — no copy on disk can silently drift.
func skillContent() string { return skillFrontmatter + promptText }

// skillName reads the skill's name out of the embedded frontmatter so `chip skill
// list` can report it without a second hard-coded copy to drift.
func skillName() string {
	for line := range strings.SplitSeq(skillFrontmatter, "\n") {
		if rest, ok := strings.CutPrefix(line, "name:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "chip"
}

// cmdPrompt writes the agent-facing contract to stdout: how to stream programs
// in (`run -`/`-e`), validate them all at once (`check`), read the machine
// diagnostics, branch on the exit code, and repair. A harness curls it into
// context as the single source of truth for driving the CLI — the same text the
// bundled SKILL.md embeds verbatim. It takes no arguments.
func cmdPrompt(args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return &usageError{msg: "prompt takes no arguments"}
	}
	_, err := io.WriteString(stdout, promptText)
	return err
}

// cmdSkill emits or installs the embedded Agent Skill (D24, plan §5.8). The skill
// is built into the binary from one //go:embed source, so it is always
// version-matched to the CLI it documents and can be served two ways: printed to
// stdout (the Zero model — nothing on disk is touched) or installed as files (the
// vercel-labs/skills model — explicit, idempotent, non-clobbering). A binary
// install never installs a skill; placement is always this explicit command.
func cmdSkill(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{msg: "skill needs a subcommand: print, install, or list"}
	}
	switch args[0] {
	case "print":
		return skillPrint(args[1:], stdout)
	case "install":
		return skillInstall(args[1:], stdout, stderr)
	case "list":
		return skillList(args[1:], stdout)
	default:
		return &usageError{msg: fmt.Sprintf("unknown skill subcommand %q (want print, install, or list)", args[0])}
	}
}

// skillPrint writes the complete SKILL.md to stdout and touches nothing on disk —
// the Zero model: a harness pipes it wherever it wants.
func skillPrint(args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return &usageError{msg: "skill print takes no arguments"}
	}
	_, err := io.WriteString(stdout, skillContent())
	return err
}

// skillTarget is one harness location chip skill install knows how to write.
type skillTarget struct {
	agent string // the --agent selector
	dir   string // the skill directory under the base (cwd, or $HOME for --global)
}

// skillTargets are the harnesses install writes by default: the Agent Skills
// standard directory and Claude Code's mirror. The two SKILL.md files are
// byte-identical copies (copies, not symlinks).
var skillTargets = []skillTarget{
	{"agents", filepath.Join(".agents", "skills", "chip")},
	{"claude", filepath.Join(".claude", "skills", "chip")},
}

// skillInstall places the embedded skill as files. By default it writes both
// SKILL.md copies and maintains a managed block in AGENTS.md; --agent NAME narrows
// to one harness's SKILL.md and skips AGENTS.md (a surgical install). Installs are
// idempotent (an unchanged file is a no-op) and non-clobbering (a SKILL.md that
// differs from what the binary ships — a user-edited or stale copy — is refused
// without --force). --global writes the user-level directories under $HOME;
// --print dry-runs the skill to stdout without touching disk.
func skillInstall(args []string, stdout, stderr io.Writer) error {
	var agent string
	var global, force, printOnly bool
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--print":
			printOnly = true
		case a == "--global":
			global = true
		case a == "--force":
			force = true
		case a == "--agent":
			if i+1 >= len(args) {
				return &usageError{msg: "--agent needs a value (agents or claude)"}
			}
			i++
			agent = args[i]
		case strings.HasPrefix(a, "--agent="):
			agent = strings.TrimPrefix(a, "--agent=")
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for skill install: %s", a)}
		}
	}
	if printOnly {
		_, err := io.WriteString(stdout, skillContent())
		return err
	}
	targets, err := selectSkillTargets(agent)
	if err != nil {
		return err
	}
	base, err := skillBaseDir(global)
	if err != nil {
		return err
	}
	content := skillContent()
	for _, t := range targets {
		path := filepath.Join(base, t.dir, "SKILL.md")
		status, err := installSkillFile(stderr, path, content, force)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%-9s %s\n", status, path)
	}
	// A full install (no --agent) also maintains the cross-harness AGENTS.md
	// pointer block; a targeted --agent install stays surgical.
	if agent == "" {
		path := filepath.Join(base, "AGENTS.md")
		status, err := upsertAgentsBlock(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%-9s %s\n", status, path)
	}
	return nil
}

// selectSkillTargets resolves --agent to the targets to write: all of them when
// empty, or the single named harness. An unknown name is a usage error.
func selectSkillTargets(agent string) ([]skillTarget, error) {
	if agent == "" {
		return skillTargets, nil
	}
	for _, t := range skillTargets {
		if t.agent == agent {
			return []skillTarget{t}, nil
		}
	}
	return nil, &usageError{msg: fmt.Sprintf("unknown --agent %q (want agents or claude)", agent)}
}

// skillBaseDir is where install and list operate: the working directory by
// default, or the user's home directory for --global (the user-level skill dirs
// live under $HOME).
func skillBaseDir(global bool) (string, error) {
	if global {
		return os.UserHomeDir()
	}
	return os.Getwd()
}

// installSkillFile writes content to a SKILL.md path, reporting what it did:
// "created" when the file was absent, "unchanged" when it already holds exactly
// these bytes (the idempotent re-install), or "updated" when --force overwrote a
// differing copy. A copy that differs without --force is refused (exit 73) so a
// user-edited or stale skill is never silently clobbered.
func installSkillFile(stderr io.Writer, path, content string, force bool) (string, error) {
	existing, rerr := os.ReadFile(path)
	exists := rerr == nil
	if rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		return "", rerr
	}
	if exists {
		if string(existing) == content {
			return "unchanged", nil
		}
		if !force {
			return "", clobberRefused(stderr, path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	if exists {
		return "updated", nil
	}
	return "created", nil
}

const (
	agentsBlockBegin = "<!-- chip:begin -->"
	agentsBlockEnd   = "<!-- chip:end -->"
)

// agentsManagedBlock is the region chip maintains in AGENTS.md: a short pointer to
// the installed skill, fenced by markers. chip owns only the text between the
// markers; anything outside is the user's and is preserved untouched. Refreshing
// the block is by design (the marker says so), so it needs no --force.
func agentsManagedBlock() string {
	return agentsBlockBegin + "\n" +
		"## chip\n" +
		"\n" +
		"This repository ships the **chip** Agent Skill. To drive the `chip` CLI —\n" +
		"stream, validate, and repair `.chp` programs — read\n" +
		"`.agents/skills/chip/SKILL.md` (mirrored at `.claude/skills/chip/SKILL.md`),\n" +
		"or run `chip prompt` for the same contract.\n" +
		"\n" +
		"_Managed by `chip skill install`; edits between the chip markers are overwritten._\n" +
		agentsBlockEnd
}

// upsertAgentsBlock maintains chip's managed block in AGENTS.md, preserving every
// line outside the markers. It creates AGENTS.md (with a title) when absent,
// replaces the block in place when present, and reports "unchanged" when the block
// already matches verbatim — so a re-install is a no-op.
func upsertAgentsBlock(path string) (string, error) {
	block := agentsManagedBlock()
	existing, rerr := os.ReadFile(path)
	if errors.Is(rerr, os.ErrNotExist) {
		content := "# AGENTS.md\n\n" + block + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", err
		}
		return "created", nil
	}
	if rerr != nil {
		return "", rerr
	}
	text := string(existing)
	bi := strings.Index(text, agentsBlockBegin)
	ei := strings.Index(text, agentsBlockEnd)
	if bi >= 0 && ei > bi {
		end := ei + len(agentsBlockEnd)
		if text[bi:end] == block {
			return "unchanged", nil
		}
		updated := text[:bi] + block + text[end:]
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return "", err
		}
		return "updated", nil
	}
	// No managed block yet: append one, preserving all existing content.
	sep := "\n\n"
	switch {
	case text == "":
		sep = ""
	case strings.HasSuffix(text, "\n"):
		sep = "\n"
	}
	updated := text + sep + block + "\n"
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return "updated", nil
}

// skillList reports the embedded skill's name and version and, for each harness
// target, whether a SKILL.md is present and whether it matches what the binary
// ships. --global inspects the user-level directories under $HOME.
func skillList(args []string, stdout io.Writer) error {
	global := false
	for _, a := range args {
		switch a {
		case "--global":
			global = true
		default:
			return &usageError{msg: fmt.Sprintf("unknown flag for skill list: %s", a)}
		}
	}
	base, err := skillBaseDir(global)
	if err != nil {
		return err
	}
	v, _, _ := versionInfo()
	fmt.Fprintf(stdout, "skill %s %s\n", skillName(), v)
	content := skillContent()
	for _, t := range skillTargets {
		path := filepath.Join(base, t.dir, "SKILL.md")
		state := "absent"
		if b, rerr := os.ReadFile(path); rerr == nil {
			if string(b) == content {
				state = "installed (current)"
			} else {
				state = "installed (differs)"
			}
		}
		fmt.Fprintf(stdout, "  %-7s %s — %s\n", t.agent, path, state)
	}
	return nil
}

// cmdVersion reports build metadata: the release tag (or the module version when
// installed via `go install`), the commit, the build date, and the Go version.
// `--json` emits the same fields as a single JSON object.
func cmdVersion(args []string, stdout io.Writer) error {
	asJSON := false
	for _, a := range args {
		switch a {
		case "--json", "-json":
			asJSON = true
		default:
			return fmt.Errorf("unknown flag for version: %s", a)
		}
	}
	v, c, d := versionInfo()
	if asJSON {
		b, err := json.Marshal(struct {
			Version string `json:"version"`
			Commit  string `json:"commit,omitempty"`
			Date    string `json:"date,omitempty"`
			Go      string `json:"go"`
		}{v, c, d, runtime.Version()})
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
		return nil
	}
	fmt.Fprintf(stdout, "chip %s\n", v)
	if c != "" {
		fmt.Fprintf(stdout, "commit: %s\n", c)
	}
	if d != "" {
		fmt.Fprintf(stdout, "date:   %s\n", d)
	}
	fmt.Fprintf(stdout, "go:     %s\n", runtime.Version())
	return nil
}

// versionInfo prefers ldflags-stamped build metadata and falls back to
// runtime/debug build info, so `go install …@latest` still reports the module
// tag and the embedded VCS stamp.
func versionInfo() (v, c, d string) {
	v, c, d = version, commit, date
	if v == "" {
		v = "(devel)"
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return v, c, d
	}
	if (v == "" || v == "(devel)") && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		v = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if c == "" {
				c = s.Value
			}
		case "vcs.time":
			if d == "" {
				d = s.Value
			}
		}
	}
	return v, c, d
}

func usage(w io.Writer) {
	fmt.Fprint(w, `chip is a toy streaming scripting language.

usage:
    chip run  <file.chp>    stream and run a program
    chip repl               evaluate source line by line, carrying state over
    chip fmt  <file.chp>    print canonical formatting (-w rewrites, -l lists, -d diffs)
    chip check <file.chp>   report every parse and type error at once (no execution)
    chip lint <file.chp>    report unused names and dead code
    chip fix  <file.chp>    apply safe repairs (-w rewrites, --plan shows, --all <id>)
    chip dump <file.chp>    disassemble a program's bytecode
    chip ast  <file.chp>    print a program's syntax tree
    chip explain <CODE>     explain a diagnostic code (no arg lists all codes)
    chip prompt             print the agent-facing contract for driving the CLI
    chip skill <cmd>        print or install the bundled Agent Skill (print, install, list)
    chip lsp --stdio        run a diagnostics language server over stdio (for editors)
    chip session            stream source-in/event-out as NDJSON, state across frames
    chip version            print version, commit, and build date
    chip help               show this help

    chip <file.chp>         shorthand for "chip run <file.chp>"

source input (run, fmt, check, lint, dump, ast):
    <file.chp>              read from a file
    -                       read from standard input
    -e '<src>'              read inline source (e.g. chip run -e 'print(6*7)')
    (no arg, piped stdin)   read from standard input

resource caps (run):
    --timeout=DUR          wall-clock limit (e.g. 500ms, 2s); exceeding it exits 124
    --max-steps=N          deterministic interpreter-step limit; exceeding it exits 125
    --max-output=BYTES     cap on printed bytes; output truncates and exits 125
    --max-depth=N          call-depth limit (default 16384); a breach is a runtime fault

sealed execution (run):
    --no-imports           refuse host imports (run untrusted code; stdlib still resolves)
    --no-prelude           also drop the unqualified stdlib prelude (only builtins remain)

capture (run):
    --save[=PATH]          save the full source after the run (default: a fresh $TMPDIR file)
    --save-on-error[=PATH] save the source only if the run faults
    --force                overwrite an existing --save/--save-on-error target (else exit 73)

diagnostics (run, check, lint, dump, ast):
    --format human|agent|json|llm   output renderer (auto: human on a tty, agent off-tty)
    --format llm[:model]            agent diagnostics plus grounded fix hints from the catalog
    --json                          shorthand for --format json
    diagnostics print to stderr; the exit code is the signal (0 ok, 2 static error,
    1 runtime fault, 64 usage, 66 no input, 73 cannot save, 124 timeout, 125 resource cap)
`)
}
