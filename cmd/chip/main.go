// Command chip runs, formats, and inspects chip source files.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/code"
	"github.com/jackspirou/chip/internal/compiler"
	"github.com/jackspirou/chip/internal/format"
	"github.com/jackspirou/chip/internal/lint"
	"github.com/jackspirou/chip/internal/parser"
	"github.com/jackspirou/chip/internal/stream"
	"github.com/jackspirou/chip/internal/token"
)

// errReported marks an error that has already been printed with full context,
// so main exits non-zero without printing it again.
var errReported = errors.New("reported")

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errReported) {
			fmt.Fprintln(os.Stderr, "chip:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return errReported
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:], stdout, stderr)
	case "repl":
		return cmdRepl(stdin, stdout)
	case "fmt":
		return cmdFmt(args[1:], stdout, stderr)
	case "lint":
		return cmdLint(args[1:], stderr)
	case "dump":
		return cmdDump(args[1:], stdout, stderr)
	case "ast":
		return cmdAst(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	default:
		return cmdRun(args, stdout, stderr) // `chip file.chp` is shorthand for `chip run file.chp`
	}
}

// cmdRun streams and executes a program.
func cmdRun(args []string, stdout, stderr io.Writer) error {
	path, src, err := readSource(args)
	if err != nil {
		return err
	}
	if err := stream.Run(bytes.NewReader(src), stdout); err != nil {
		report(stderr, path, src, err)
		return errReported
	}
	return nil
}

// cmdRepl evaluates source read line by line, carrying definitions over.
func cmdRepl(stdin io.Reader, stdout io.Writer) error {
	return stream.REPL(stdin, stdout)
}

// cmdFmt prints a program in canonical form (or rewrites it in place with -w).
func cmdFmt(args []string, stdout, stderr io.Writer) error {
	write := false
	if len(args) > 0 && args[0] == "-w" {
		write, args = true, args[1:]
	}
	path, src, err := readSource(args)
	if err != nil {
		return err
	}
	out, err := format.Source(src)
	if err != nil {
		report(stderr, path, src, err)
		return errReported
	}
	if write {
		return os.WriteFile(path, out, 0o644)
	}
	_, err = stdout.Write(out)
	return err
}

// cmdLint reports style and correctness issues.
func cmdLint(args []string, stderr io.Writer) error {
	path, src, err := readSource(args)
	if err != nil {
		return err
	}
	f, perr := parse(src)
	if perr != nil {
		report(stderr, path, src, perr)
		return errReported
	}
	info, cerr := check.Check(f)
	if cerr != nil {
		report(stderr, path, src, cerr)
		return errReported
	}
	if issues := lint.Lint(f, info); len(issues) > 0 {
		report(stderr, path, src, issues)
		return errReported
	}
	return nil
}

// cmdDump disassembles a program's bytecode (the batch compile path).
func cmdDump(args []string, stdout, stderr io.Writer) error {
	path, src, err := readSource(args)
	if err != nil {
		return err
	}
	prog, err := build(src)
	if err != nil {
		report(stderr, path, src, err)
		return errReported
	}
	fmt.Fprint(stdout, prog.String())
	return nil
}

// cmdAst prints a program's syntax tree.
func cmdAst(args []string, stdout, stderr io.Writer) error {
	path, src, err := readSource(args)
	if err != nil {
		return err
	}
	f, perr := parse(src)
	if perr != nil {
		report(stderr, path, src, perr)
		return errReported
	}
	fmt.Fprintln(stdout, ast.Sprint(f))
	return nil
}

func readSource(args []string) (string, []byte, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("no source file given")
	}
	path := args[0]
	src, err := os.ReadFile(path)
	return path, src, err
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

// report prints err with source context (filename:line:col, the offending
// line, and a caret) for each diagnostic it can locate.
func report(w io.Writer, filename string, src []byte, err error) {
	lines := bytes.Split(src, []byte("\n"))
	emit := func(pos token.Pos, msg string) {
		fmt.Fprintf(w, "%s:%d:%d: %s\n", filename, pos.Line, pos.Column, msg)
		if pos.Line >= 1 && pos.Line <= len(lines) {
			col := pos.Column
			if col < 1 {
				col = 1
			}
			fmt.Fprintf(w, "  %s\n", lines[pos.Line-1])
			fmt.Fprintf(w, "  %s^\n", strings.Repeat(" ", col-1))
		}
	}
	switch e := err.(type) {
	case parser.ErrorList:
		for _, d := range e {
			emit(d.Pos, d.Msg)
		}
	case check.ErrorList:
		for _, d := range e {
			emit(d.Pos, d.Msg)
		}
	case lint.IssueList:
		for _, d := range e {
			emit(d.Pos, d.Msg)
		}
	case stream.TypeError:
		emit(e.Pos, e.Msg)
	case stream.RuntimeError:
		emit(e.Pos, e.Msg)
	default:
		fmt.Fprintf(w, "%s: %s\n", filename, err)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `chip is a toy streaming scripting language.

usage:
    chip run  <file.chp>    stream and run a program
    chip repl               evaluate source line by line, carrying state over
    chip fmt  <file.chp>    print canonical formatting (-w rewrites in place)
    chip lint <file.chp>    report unused names, missing returns, dead code
    chip dump <file.chp>    disassemble a program's bytecode
    chip ast  <file.chp>    print a program's syntax tree
    chip help               show this help

    chip <file.chp>         shorthand for "chip run <file.chp>"
`)
}
