// Package format pretty-prints chip syntax trees as canonical source.
package format

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/parser"
)

// Source parses chip source and returns it in canonical form.
func Source(src []byte) ([]byte, error) {
	p, err := parser.New(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	file, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return File(file), nil
}

// File renders a syntax tree as canonical chip source.
func File(f *ast.File) []byte {
	p := &printer{comments: f.Comments}
	p.file(f)
	if p.ci < len(p.comments) {
		p.blank()
		p.flushRemaining()
	}
	return p.buf.Bytes()
}

type printer struct {
	buf      bytes.Buffer
	indent   int
	comments []*ast.Comment
	ci       int // index of the next un-emitted comment
}

func (p *printer) writeIndent() {
	for i := 0; i < p.indent; i++ {
		p.buf.WriteString("    ")
	}
}

// line writes s as a full source line at the current indentation.
func (p *printer) line(s string) {
	p.writeIndent()
	p.buf.WriteString(s)
	p.buf.WriteByte('\n')
}

// blank writes a blank line, unless at the very start of the output.
func (p *printer) blank() {
	if p.buf.Len() > 0 {
		p.buf.WriteByte('\n')
	}
}

// flush emits any pending comments positioned before the given source line.
func (p *printer) flush(beforeLine int) {
	for p.ci < len(p.comments) && p.comments[p.ci].Slash.Line < beforeLine {
		p.comment(p.comments[p.ci].Text)
		p.ci++
	}
}

func (p *printer) flushRemaining() {
	for p.ci < len(p.comments) {
		p.comment(p.comments[p.ci].Text)
		p.ci++
	}
}

func (p *printer) comment(text string) {
	for _, ln := range strings.Split(text, "\n") {
		p.line("// " + strings.TrimRight(ln, " \t\r"))
	}
}

func (p *printer) file(f *ast.File) {
	if f.Package != nil {
		p.flush(f.Package.Pos().Line)
		p.line("package " + f.Package.Name)
	}
	p.imports(f.Imports)
	// Definitions are hoisted into dependency order for streamability; top-level
	// statements follow, in their original order (never reordered).
	for _, d := range orderDecls(f.Decls) {
		p.blank()
		p.flush(d.Pos().Line)
		p.decl(d)
	}
	if len(f.Stmts) > 0 {
		p.blank()
		for _, s := range f.Stmts {
			p.flush(s.Pos().Line)
			p.stmt(s)
		}
	}
}

func (p *printer) imports(specs []*ast.ImportSpec) {
	specs = canonicalImports(specs)
	if len(specs) == 0 {
		return
	}
	p.blank()
	if len(specs) == 1 {
		p.line("import " + importSpecString(specs[0]))
		return
	}
	p.line("import (")
	p.indent++
	for _, s := range specs {
		p.line(importSpecString(s))
	}
	p.indent--
	p.line(")")
}

// canonicalImports returns specs sorted by import path (ties broken by the
// optional alias) with exact duplicates — same path and same alias — removed.
// Sorting and dedup are deterministic, so re-formatting canonical output is a
// no-op, which keeps `chip fmt` idempotent on imports.
func canonicalImports(specs []*ast.ImportSpec) []*ast.ImportSpec {
	sorted := make([]*ast.ImportSpec, len(specs))
	copy(sorted, specs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if pi, pj := specPath(sorted[i]), specPath(sorted[j]); pi != pj {
			return pi < pj
		}
		return specAlias(sorted[i]) < specAlias(sorted[j])
	})
	var out []*ast.ImportSpec
	for _, s := range sorted {
		if n := len(out); n > 0 && specPath(out[n-1]) == specPath(s) && specAlias(out[n-1]) == specAlias(s) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func specPath(s *ast.ImportSpec) string {
	if s.Path != nil {
		return s.Path.Value
	}
	return ""
}

func specAlias(s *ast.ImportSpec) string {
	if s.Name != nil {
		return s.Name.Name
	}
	return ""
}

func importSpecString(s *ast.ImportSpec) string {
	path := ""
	if s.Path != nil {
		path = strconv.Quote(s.Path.Value)
	}
	if s.Name != nil {
		return s.Name.Name + " " + path
	}
	return path
}

func (p *printer) decl(d ast.Decl) {
	if fn, ok := d.(*ast.FuncDecl); ok {
		p.funcDecl(fn)
	}
}

func (p *printer) funcDecl(fn *ast.FuncDecl) {
	var b strings.Builder
	b.WriteString("func " + fn.Name.Name + "(")
	for i, f := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fieldString(f))
	}
	b.WriteString(")")
	for i, f := range fn.Results {
		if i == 0 {
			b.WriteString(" ")
		} else {
			b.WriteString(", ")
		}
		b.WriteString(exprString(f.Type))
	}
	b.WriteString(" {")

	p.line(b.String())
	p.indent++
	p.stmts(fn.Body.List)
	p.indent--
	p.line("}")
}

func fieldString(f *ast.Field) string {
	if f.Name != nil {
		return f.Name.Name + " " + exprString(f.Type)
	}
	return exprString(f.Type)
}

func (p *printer) stmts(list []ast.Stmt) {
	for _, s := range list {
		p.flush(s.Pos().Line)
		p.stmt(s)
	}
}

func (p *printer) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.DeclStmt:
		p.line(s.Name.Name + " := " + exprString(s.Value))
	case *ast.AssignStmt:
		p.line(exprString(s.Lhs) + " " + s.Op.String() + " " + exprString(s.Rhs))
	case *ast.ExprStmt:
		p.line(exprString(s.X))
	case *ast.ReturnStmt:
		if s.Result == nil {
			p.line("return")
		} else {
			p.line("return " + exprString(s.Result))
		}
	case *ast.IfStmt:
		p.ifStmt(s)
	case *ast.ForStmt:
		if s.Cond == nil {
			p.line("for {")
		} else {
			p.line("for " + exprString(s.Cond) + " {")
		}
		p.indent++
		p.stmts(s.Body.List)
		p.indent--
		p.line("}")
	case *ast.Block:
		p.line("{")
		p.indent++
		p.stmts(s.List)
		p.indent--
		p.line("}")
	case *ast.BadStmt:
		p.line("/* bad statement */")
	}
}

func (p *printer) ifStmt(s *ast.IfStmt) {
	p.line("if " + exprString(s.Cond) + " {")
	for {
		p.indent++
		p.stmts(s.Body.List)
		p.indent--
		switch e := s.Else.(type) {
		case *ast.IfStmt:
			p.line("} else if " + exprString(e.Cond) + " {")
			s = e
		case *ast.Block:
			p.line("} else {")
			p.indent++
			p.stmts(e.List)
			p.indent--
			p.line("}")
			return
		default: // nil
			p.line("}")
			return
		}
	}
}
