// Package lint reports style and correctness issues that the type checker does
// not treat as hard errors: unused variables and functions, missing returns,
// and unreachable code.
package lint

import (
	"fmt"
	"path"
	"sort"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/check"
	"github.com/jackspirou/chip/internal/scope"
	"github.com/jackspirou/chip/internal/token"
)

// Issue is a single lint finding.
type Issue struct {
	Pos token.Pos
	Msg string
}

func (i Issue) Error() string {
	return fmt.Sprintf("%d:%d: %s", i.Pos.Line, i.Pos.Column, i.Msg)
}

// IssueList is a collection of lint findings.
type IssueList []Issue

func (l IssueList) Error() string {
	switch len(l) {
	case 0:
		return "no issues"
	case 1:
		return l[0].Error()
	default:
		return fmt.Sprintf("%s (and %d more issues)", l[0], len(l)-1)
	}
}

// Err returns an error equivalent to the list, or nil if it is empty.
func (l IssueList) Err() error {
	if len(l) == 0 {
		return nil
	}
	return l
}

// Lint analyzes a type-checked file and returns any issues, sorted by position.
func Lint(file *ast.File, info *check.Info) IssueList {
	l := &linter{info: info}
	l.checkUnused(file)
	l.checkUnusedImports(file)
	l.checkFlow(file)
	l.checkOrder()
	sort.Slice(l.issues, func(i, j int) bool {
		a, b := l.issues[i].Pos, l.issues[j].Pos
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	return l.issues
}

type linter struct {
	info   *check.Info
	issues IssueList
}

func (l *linter) add(pos token.Pos, format string, args ...any) {
	l.issues = append(l.issues, Issue{Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

//
// unused variables and functions
//

func (l *linter) checkUnused(file *ast.File) {
	// Identifiers that are pure write targets (assignment LHS) are not reads.
	writes := make(map[*ast.Ident]bool)
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			collectWrites(fn.Body, writes)
		}
	}

	read := make(map[*scope.Symbol]bool)
	for id, sym := range l.info.Uses {
		if !writes[id] {
			read[sym] = true
		}
	}

	for id, sym := range l.info.Defs {
		switch sym.Kind {
		case scope.Var:
			if !read[sym] {
				l.add(id.Pos(), "%s declared and not used", sym.Name)
			}
		case scope.Func:
			if sym.Name != "main" && !read[sym] {
				l.add(id.Pos(), "function %s is never used", sym.Name)
			}
		}
	}
}

func collectWrites(b *ast.Block, writes map[*ast.Ident]bool) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		collectWritesStmt(s, writes)
	}
}

func collectWritesStmt(s ast.Stmt, writes map[*ast.Ident]bool) {
	switch s := s.(type) {
	case *ast.AssignStmt:
		if id, ok := s.Lhs.(*ast.Ident); ok {
			writes[id] = true
		}
	case *ast.IfStmt:
		collectWrites(s.Body, writes)
		collectWritesStmt(s.Else, writes)
	case *ast.ForStmt:
		collectWrites(s.Body, writes)
	case *ast.Block:
		collectWrites(s, writes)
	}
}

//
// unused imports
//

// checkUnusedImports flags an import whose package is never referenced as a
// qualifier (pkg.Fn — the only place a package name may appear, P7). The
// reference name is the alias when present, else the last element of the import
// path (P6). This is a purely syntactic check over the tree, independent of the
// type checker (which does not yet resolve qualified calls — Slice 5).
func (l *linter) checkUnusedImports(file *ast.File) {
	if len(file.Imports) == 0 {
		return
	}
	used := map[string]bool{}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			collectQualifiers(fn.Body, used)
		}
	}
	for _, s := range file.Stmts {
		collectQualifiersStmt(s, used)
	}
	for _, spec := range file.Imports {
		if name := importRefName(spec); name == "" || used[name] {
			continue
		}
		if spec.Path != nil {
			l.add(spec.Pos(), "%q imported and not used", spec.Path.Value)
		}
	}
}

// importRefName is the name an import binds: its alias if given, else the last
// element of the import path.
func importRefName(s *ast.ImportSpec) string {
	if s.Name != nil {
		return s.Name.Name
	}
	if s.Path != nil {
		return path.Base(s.Path.Value)
	}
	return ""
}

// collectQualifiers records the base identifier of every selector (the pkg in
// pkg.Fn) reachable from a block, so unused imports can be told from used ones.
func collectQualifiers(b *ast.Block, used map[string]bool) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		collectQualifiersStmt(s, used)
	}
}

func collectQualifiersStmt(s ast.Stmt, used map[string]bool) {
	switch s := s.(type) {
	case *ast.DeclStmt:
		collectQualifiersExpr(s.Value, used)
	case *ast.AssignStmt:
		collectQualifiersExpr(s.Lhs, used)
		collectQualifiersExpr(s.Rhs, used)
	case *ast.ExprStmt:
		collectQualifiersExpr(s.X, used)
	case *ast.ReturnStmt:
		collectQualifiersExpr(s.Result, used)
	case *ast.IfStmt:
		collectQualifiersExpr(s.Cond, used)
		collectQualifiers(s.Body, used)
		collectQualifiersStmt(s.Else, used)
	case *ast.ForStmt:
		collectQualifiersExpr(s.Cond, used)
		collectQualifiers(s.Body, used)
	case *ast.Block:
		collectQualifiers(s, used)
	}
}

func collectQualifiersExpr(e ast.Expr, used map[string]bool) {
	switch e := e.(type) {
	case *ast.SelectorExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			used[id.Name] = true
		} else {
			collectQualifiersExpr(e.X, used)
		}
	case *ast.CallExpr:
		collectQualifiersExpr(e.Fn, used)
		for _, a := range e.Args {
			collectQualifiersExpr(a, used)
		}
	case *ast.UnaryExpr:
		collectQualifiersExpr(e.X, used)
	case *ast.BinaryExpr:
		collectQualifiersExpr(e.Left, used)
		collectQualifiersExpr(e.Right, used)
	case *ast.IndexExpr:
		collectQualifiersExpr(e.X, used)
		collectQualifiersExpr(e.Index, used)
	case *ast.CompositeLit:
		for _, el := range e.Elems {
			collectQualifiersExpr(el, used)
		}
	}
}

//
// streamability: functions used before they are defined
//

// checkOrder flags a function called before its own definition in source order.
// The forward reference still works (functions resolve by reading ahead) but
// makes the stream wait; `chip fmt` reorders definitions to remove it.
func (l *linter) checkOrder() {
	for id, sym := range l.info.Uses {
		if sym.Kind == scope.Func && posBefore(id.Pos(), sym.DeclPos) {
			l.add(id.Pos(), "%s used before its definition", sym.Name)
		}
	}
}

func posBefore(a, b token.Pos) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}

//
// control flow: missing return and unreachable code
//

func (l *linter) checkFlow(file *ast.File) {
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if len(fn.Results) > 0 && !terminates(fn.Body.List) {
			l.add(fn.Body.Rbrace, "missing return at end of function %s", fn.Name.Name)
		}
		l.checkUnreachable(fn.Body)
	}
}

func (l *linter) checkUnreachable(b *ast.Block) {
	if b == nil {
		return
	}
	flagged := false
	for i, s := range b.List {
		if !flagged && i+1 < len(b.List) && terminatesStmt(s) {
			l.add(b.List[i+1].Pos(), "unreachable code")
			flagged = true
		}
		l.unreachableInStmt(s)
	}
}

// unreachableInStmt recurses into a statement's nested blocks.
func (l *linter) unreachableInStmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.IfStmt:
		l.checkUnreachable(s.Body)
		l.unreachableInStmt(s.Else)
	case *ast.ForStmt:
		l.checkUnreachable(s.Body)
	case *ast.Block:
		l.checkUnreachable(s)
	}
}

// terminates reports whether a statement list always transfers control away
// (returns, or loops forever) so control never falls off the end.
func terminates(list []ast.Stmt) bool {
	for _, s := range list {
		if terminatesStmt(s) {
			return true
		}
	}
	return false
}

func terminatesStmt(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		return s.Else != nil && terminates(s.Body.List) && terminatesStmt(s.Else)
	case *ast.Block:
		return terminates(s.List)
	case *ast.ForStmt:
		return s.Cond == nil // an infinite loop never falls through (chip has no break)
	default:
		return false
	}
}
