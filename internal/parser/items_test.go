package parser

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/ast"
)

// TestItemsYieldsTopLevelItems checks that Items streams a file's top-level
// items in source order, with a function declaration and a statement each
// surfacing as the right node type.
func TestItemsYieldsTopLevelItems(t *testing.T) {
	const src = "func a() {}\nx := 1\n"

	p, err := New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var got []ast.Node
	for item, err := range p.Items() {
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		got = append(got, item)
	}

	if len(got) != 2 {
		t.Fatalf("got %d items, want 2: %v", len(got), got)
	}
	if _, ok := got[0].(*ast.FuncDecl); !ok {
		t.Errorf("items[0] = %T, want *ast.FuncDecl", got[0])
	}
	if _, ok := got[1].(*ast.DeclStmt); !ok {
		t.Errorf("items[1] = %T, want *ast.DeclStmt", got[1])
	}
}

// TestParseAllowsTopLevelStmts checks that the batch Parse path also accepts
// top-level statements, collecting them in File.Stmts (and not as Decls).
func TestParseAllowsTopLevelStmts(t *testing.T) {
	const src = "x := 1\nprint(x)\n"

	f := parseFileStr(t, src)
	if len(f.Decls) != 0 {
		t.Errorf("got %d decls, want 0", len(f.Decls))
	}
	if len(f.Stmts) != 2 {
		t.Fatalf("got %d top-level stmts, want 2", len(f.Stmts))
	}
	if _, ok := f.Stmts[0].(*ast.DeclStmt); !ok {
		t.Errorf("stmts[0] = %T, want *ast.DeclStmt", f.Stmts[0])
	}
	if _, ok := f.Stmts[1].(*ast.ExprStmt); !ok {
		t.Errorf("stmts[1] = %T, want *ast.ExprStmt", f.Stmts[1])
	}
}
