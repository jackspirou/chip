package stream

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/ast"
)

// TestEngineDrainsItemsInOrder checks the Slice 0 pull driver: streaming a
// source through iter.Pull2 yields its top-level items in source order, with a
// function declaration and a statement surfacing as the right node types.
func TestEngineDrainsItemsInOrder(t *testing.T) {
	const src = "func a() {}\nx := 1\n"

	e, err := newEngine(strings.NewReader(src))
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	defer e.close()

	items, err := e.drain()
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2: %v", len(items), items)
	}
	if _, ok := items[0].(*ast.FuncDecl); !ok {
		t.Errorf("items[0] = %T, want *ast.FuncDecl", items[0])
	}
	if _, ok := items[1].(*ast.DeclStmt); !ok {
		t.Errorf("items[1] = %T, want *ast.DeclStmt", items[1])
	}
}
