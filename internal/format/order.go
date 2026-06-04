package format

import "github.com/jackspirou/chip/internal/ast"

// orderDecls returns the top-level function declarations in dependency order —
// each function placed after the functions it calls — so a streamed program
// resolves forward references with less waiting ("optimize for streamability").
// Mutually recursive functions (a cycle) stay grouped, and ties keep source
// order, which makes the ordering stable and idempotent.
//
// It reorders definitions only; the caller emits statements separately and
// never reorders them.
func orderDecls(decls []ast.Decl) []ast.Decl {
	funcs := make([]*ast.FuncDecl, 0, len(decls))
	for _, d := range decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil {
			funcs = append(funcs, fn)
		}
	}

	index := make(map[string]int, len(funcs))
	for i, fn := range funcs {
		index[fn.Name.Name] = i
	}

	// callees[i] = distinct indices of functions i calls (self-calls excluded);
	// callers[j] = functions that call j; outdeg[i] = unemitted callees of i.
	callees := make([]map[int]bool, len(funcs))
	callers := make([]map[int]bool, len(funcs))
	for i := range funcs {
		callees[i] = map[int]bool{}
		callers[i] = map[int]bool{}
	}
	for i, fn := range funcs {
		for name := range calledNames(fn.Body) {
			if j, ok := index[name]; ok && j != i {
				callees[i][j] = true
			}
		}
	}
	for i := range funcs {
		for j := range callees[i] {
			callers[j][i] = true
		}
	}
	outdeg := make([]int, len(funcs))
	for i := range funcs {
		outdeg[i] = len(callees[i])
	}

	// Kahn's algorithm, emitting callees before callers. Pick the lowest-index
	// function whose callees are all emitted; if a cycle leaves none ready,
	// break it with the lowest-index remaining function (keeps the order
	// deterministic, so formatting is idempotent).
	emitted := make([]bool, len(funcs))
	order := make([]ast.Decl, 0, len(funcs))
	for len(order) < len(funcs) {
		pick := -1
		for i := range funcs {
			if !emitted[i] && outdeg[i] == 0 {
				pick = i
				break
			}
		}
		if pick == -1 {
			for i := range funcs {
				if !emitted[i] {
					pick = i
					break
				}
			}
		}
		emitted[pick] = true
		order = append(order, funcs[pick])
		for caller := range callers[pick] {
			if !emitted[caller] {
				outdeg[caller]--
			}
		}
	}
	return order
}

// calledNames collects the names that appear as the function of a call anywhere
// in a block (builtins included; the caller ignores names that are not
// top-level functions).
func calledNames(b *ast.Block) map[string]bool {
	names := map[string]bool{}
	if b == nil {
		return names
	}

	var walkExpr func(ast.Expr)
	walkExpr = func(e ast.Expr) {
		switch e := e.(type) {
		case *ast.CallExpr:
			if id, ok := e.Fn.(*ast.Ident); ok {
				names[id.Name] = true
			} else {
				walkExpr(e.Fn)
			}
			for _, a := range e.Args {
				walkExpr(a)
			}
		case *ast.UnaryExpr:
			walkExpr(e.X)
		case *ast.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *ast.IndexExpr:
			walkExpr(e.X)
			walkExpr(e.Index)
		case *ast.CompositeLit:
			for _, el := range e.Elems {
				walkExpr(el)
			}
		}
	}

	var walkStmt func(ast.Stmt)
	walkStmt = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.DeclStmt:
			walkExpr(s.Value)
		case *ast.AssignStmt:
			walkExpr(s.Lhs)
			walkExpr(s.Rhs)
		case *ast.ExprStmt:
			walkExpr(s.X)
		case *ast.ReturnStmt:
			if s.Result != nil {
				walkExpr(s.Result)
			}
		case *ast.IfStmt:
			walkExpr(s.Cond)
			for _, st := range s.Body.List {
				walkStmt(st)
			}
			if s.Else != nil {
				walkStmt(s.Else)
			}
		case *ast.ForStmt:
			if s.Cond != nil {
				walkExpr(s.Cond)
			}
			for _, st := range s.Body.List {
				walkStmt(st)
			}
		case *ast.Block:
			for _, st := range s.List {
				walkStmt(st)
			}
		}
	}

	for _, s := range b.List {
		walkStmt(s)
	}
	return names
}
