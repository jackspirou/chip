package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/parser"
)

// Slice 3.0b gives every AST node a byte-exact span: src[Pos().Offset:End().Offset]
// is the node's verbatim source text. These tests parse real and crafted
// programs and check that span across the whole tree — it stays in range, has no
// whitespace at its edges, sits inside its parent's span, and equals the exact
// source for representative composite nodes.

func parseFile(t *testing.T, src string) *ast.File {
	t.Helper()
	p, err := parser.New(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parser.New: %v", err)
	}
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return file
}

// nodeRel pairs a node with its parent so containment can be checked.
type nodeRel struct{ node, parent ast.Node }

// collect appends n and, transitively, every child node, recording each child's
// parent. Children are visited in source (pre)order. Nil children are skipped so
// a typed-nil never enters the result.
func collect(n, parent ast.Node, out *[]nodeRel) {
	*out = append(*out, nodeRel{n, parent})
	switch n := n.(type) {
	case *ast.File:
		if n.Package != nil {
			collect(n.Package, n, out)
		}
		for _, im := range n.Imports {
			collect(im, n, out)
		}
		for _, d := range n.Decls {
			collect(d, n, out)
		}
		for _, s := range n.Stmts {
			collect(s, n, out)
		}
	case *ast.FuncDecl:
		if n.Name != nil {
			collect(n.Name, n, out)
		}
		for _, f := range n.Params {
			collect(f, n, out)
		}
		for _, f := range n.Results {
			collect(f, n, out)
		}
		if n.Body != nil {
			collect(n.Body, n, out)
		}
	case *ast.Field:
		if n.Name != nil {
			collect(n.Name, n, out)
		}
		if n.Type != nil {
			collect(n.Type, n, out)
		}
	case *ast.ImportSpec:
		if n.Name != nil {
			collect(n.Name, n, out)
		}
		if n.Path != nil {
			collect(n.Path, n, out)
		}
	case *ast.Block:
		for _, s := range n.List {
			collect(s, n, out)
		}
	case *ast.DeclStmt:
		if n.Name != nil {
			collect(n.Name, n, out)
		}
		if n.Value != nil {
			collect(n.Value, n, out)
		}
	case *ast.AssignStmt:
		if n.Lhs != nil {
			collect(n.Lhs, n, out)
		}
		if n.Rhs != nil {
			collect(n.Rhs, n, out)
		}
	case *ast.ExprStmt:
		if n.X != nil {
			collect(n.X, n, out)
		}
	case *ast.ReturnStmt:
		if n.Result != nil {
			collect(n.Result, n, out)
		}
	case *ast.IfStmt:
		if n.Cond != nil {
			collect(n.Cond, n, out)
		}
		if n.Body != nil {
			collect(n.Body, n, out)
		}
		if n.Else != nil {
			collect(n.Else, n, out)
		}
	case *ast.ForStmt:
		if n.Cond != nil {
			collect(n.Cond, n, out)
		}
		if n.Body != nil {
			collect(n.Body, n, out)
		}
	case *ast.UnaryExpr:
		if n.X != nil {
			collect(n.X, n, out)
		}
	case *ast.BinaryExpr:
		if n.Left != nil {
			collect(n.Left, n, out)
		}
		if n.Right != nil {
			collect(n.Right, n, out)
		}
	case *ast.CallExpr:
		if n.Fn != nil {
			collect(n.Fn, n, out)
		}
		for _, a := range n.Args {
			collect(a, n, out)
		}
	case *ast.IndexExpr:
		if n.X != nil {
			collect(n.X, n, out)
		}
		if n.Index != nil {
			collect(n.Index, n, out)
		}
	case *ast.SelectorExpr:
		if n.X != nil {
			collect(n.X, n, out)
		}
		if n.Sel != nil {
			collect(n.Sel, n, out)
		}
	case *ast.ArrayType:
		if n.Len != nil {
			collect(n.Len, n, out)
		}
		if n.Elem != nil {
			collect(n.Elem, n, out)
		}
	case *ast.CompositeLit:
		if n.Type != nil {
			collect(n.Type, n, out)
		}
		for _, el := range n.Elems {
			collect(el, n, out)
		}
		// Ident, IntLit, FloatLit, StringLit, TypeName, BadExpr, BadStmt: leaves.
	}
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// TestNodeSpansWellFormed checks, for every node in every example, that its span
// is in range, sits within its parent's span, and (for non-empty nodes) neither
// starts nor ends on whitespace — the property that would break first on an
// off-by-one End.
func TestNodeSpansWellFormed(t *testing.T) {
	for _, path := range parserExampleFiles(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(raw)
		file := parseFile(t, src)
		var rels []nodeRel
		collect(file, nil, &rels)
		for _, r := range rels {
			off, end := r.node.Pos().Offset, r.node.End().Offset
			if off < 0 || end < off || end > len(src) {
				t.Errorf("%s: %T span [%d,%d) out of range (len %d)", path, r.node, off, end, len(src))
				continue
			}
			if r.parent != nil {
				pe := r.parent.End().Offset
				if end > pe {
					t.Errorf("%s: %T span [%d,%d) ends past parent %T (end %d)",
						path, r.node, off, end, r.parent, pe)
				}
				// File.Pos() is a representative start (package clause or first
				// decl), not the minimum child offset — a top-level statement or
				// import can precede it — so the lower bound only binds for
				// genuine bounding-box parents.
				if _, isFile := r.parent.(*ast.File); !isFile {
					if po := r.parent.Pos().Offset; off < po {
						t.Errorf("%s: %T span [%d,%d) starts before parent %T (pos %d)",
							path, r.node, off, end, r.parent, po)
					}
				}
			}
			if end == off {
				continue // zero-width (e.g. a Bad node); no text to inspect
			}
			if isSpace(src[off]) || isSpace(src[end-1]) {
				t.Errorf("%s: %T span %q has a whitespace edge", path, r.node, src[off:end])
			}
		}
		t.Logf("%s: %d node spans checked", filepath.Base(path), len(rels))
	}
}

// TestNodeSpansExact pins the span of representative composite nodes to their
// exact source text — the part an auto-edit would replace.
func TestNodeSpansExact(t *testing.T) {
	cases := []struct {
		name, src string
		match     func(ast.Node) bool
		want      string
	}{
		{
			name:  "call expression",
			src:   "package main\nfunc main() { print(1, 2) }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.CallExpr); return ok },
			want:  "print(1, 2)",
		},
		{
			name:  "qualified call",
			src:   "package main\nimport \"m\"\nfunc main() { m.Do(7) }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.CallExpr); return ok },
			want:  "m.Do(7)",
		},
		{
			name:  "binary precedence",
			src:   "package main\nfunc main() { print(1 + 2 * 3) }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.BinaryExpr); return ok },
			want:  "1 + 2 * 3",
		},
		{
			name:  "unary",
			src:   "package main\nfunc main() { print(-5) }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.UnaryExpr); return ok },
			want:  "-5",
		},
		{
			name:  "index",
			src:   "package main\nfunc main() { xs := [2]int{1, 2}\n print(xs[0]) }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.IndexExpr); return ok },
			want:  "xs[0]",
		},
		{
			name:  "composite literal",
			src:   "package main\nfunc main() { xs := [2]int{1, 2} }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.CompositeLit); return ok },
			want:  "[2]int{1, 2}",
		},
		{
			name:  "string literal keeps quotes",
			src:   "package main\nfunc main() { print(\"hi there\") }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.StringLit); return ok },
			want:  `"hi there"`,
		},
		{
			name:  "block",
			src:   "package main\nfunc f() { return }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.Block); return ok },
			want:  "{ return }",
		},
		{
			name:  "if with else spans to else block",
			src:   "package main\nfunc main() { if 1 == 1 { print(1) } else { print(2) } }\n",
			match: func(n ast.Node) bool { _, ok := n.(*ast.IfStmt); return ok },
			want:  "if 1 == 1 { print(1) } else { print(2) }",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := parseFile(t, tc.src)
			var rels []nodeRel
			collect(file, nil, &rels)
			var found ast.Node
			for _, r := range rels {
				if tc.match(r.node) {
					found = r.node
					break
				}
			}
			if found == nil {
				t.Fatalf("no matching node in %q", tc.src)
			}
			off, end := found.Pos().Offset, found.End().Offset
			got := tc.src[off:end]
			if got != tc.want {
				t.Fatalf("%T span = %q, want %q", found, got, tc.want)
			}
		})
	}
}

// parserExampleFiles returns every .chp program under examples/.
func parserExampleFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".chp" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no example .chp files found")
	}
	return files
}
