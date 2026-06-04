package parser

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
)

// parseFileStr parses src and fails the test on any parse error.
func parseFileStr(t *testing.T, src string) *ast.File {
	t.Helper()
	p, err := New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f
}

// parseExprStr parses src as a single expression.
func parseExprStr(t *testing.T, src string) ast.Expr {
	t.Helper()
	p, err := New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := p.parseExpr()
	if err := p.errs.Err(); err != nil {
		t.Fatalf("parseExpr(%q): %v", src, err)
	}
	return e
}

func TestParseSimpleAdd(t *testing.T) {
	const src = `package main

func main() {
    three := 1+10
    return
}

func add(x int, y int) int {
    return x + y
}`

	f := parseFileStr(t, src)

	if f.Package == nil || f.Package.Name != "main" {
		t.Fatalf("package = %v, want main", f.Package)
	}
	if len(f.Decls) != 2 {
		t.Fatalf("got %d decls, want 2", len(f.Decls))
	}

	main, ok := f.Decls[0].(*ast.FuncDecl)
	if !ok || main.Name.Name != "main" {
		t.Fatalf("decls[0] = %v, want func main", f.Decls[0])
	}
	if len(main.Body.List) != 2 {
		t.Fatalf("main body = %d stmts, want 2", len(main.Body.List))
	}
	decl, ok := main.Body.List[0].(*ast.DeclStmt)
	if !ok || decl.Name.Name != "three" {
		t.Fatalf("main stmt[0] = %v, want three := ...", main.Body.List[0])
	}
	if bin, ok := decl.Value.(*ast.BinaryExpr); !ok || bin.Op != token.ADD {
		t.Fatalf("three value = %v, want a binary +", decl.Value)
	}
	if _, ok := main.Body.List[1].(*ast.ReturnStmt); !ok {
		t.Fatalf("main stmt[1] = %v, want return", main.Body.List[1])
	}

	add, ok := f.Decls[1].(*ast.FuncDecl)
	if !ok || add.Name.Name != "add" {
		t.Fatalf("decls[1] = %v, want func add", f.Decls[1])
	}
	if len(add.Params) != 2 {
		t.Fatalf("add params = %d, want 2", len(add.Params))
	}
	if add.Params[0].Name.Name != "x" || ast.Sprint(add.Params[0].Type) != "int" {
		t.Fatalf("add param[0] = %s, want (x int)", ast.Sprint(add.Params[0]))
	}
	if len(add.Results) != 1 || ast.Sprint(add.Results[0].Type) != "int" {
		t.Fatalf("add results = %v, want one int", add.Results)
	}
}

func TestParseCallStmt(t *testing.T) {
	const src = `package add

func main() {
    print(a+b)
}`

	f := parseFileStr(t, src)
	main := f.Decls[0].(*ast.FuncDecl)
	es, ok := main.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("stmt[0] = %v, want expression statement", main.Body.List[0])
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expr = %v, want call", es.X)
	}
	if got := ast.Sprint(call); got != "(call print (+ a b))" {
		t.Fatalf("call = %s, want (call print (+ a b))", got)
	}
}

func TestExprPrecedence(t *testing.T) {
	cases := map[string]string{
		"1 + 2 * 3":   "(+ 1 (* 2 3))",
		"a + b + c":   "(+ (+ a b) c)",
		"x == y + 1":  "(== x (+ y 1))",
		"-a * b":      "(* (- a) b)",
		"a % b == 0":  "(== (% a b) 0)",
		"f(x) + g(y)": "(+ (call f x) (call g y))",
	}
	for src, want := range cases {
		if got := ast.Sprint(parseExprStr(t, src)); got != want {
			t.Errorf("%q => %s, want %s", src, got, want)
		}
	}
}

func TestParseErrorsAreReported(t *testing.T) {
	// A malformed statement (a := with no right-hand side) should be reported
	// as an error rather than panic or hang.
	p, err := New(strings.NewReader("func main() {\n  x :=\n}"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := p.Parse(); err == nil {
		t.Fatal("expected a parse error, got nil")
	}
}
