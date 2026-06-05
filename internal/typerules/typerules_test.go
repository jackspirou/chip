package typerules_test

import (
	"strings"
	"testing"

	"github.com/jackspirou/chip/internal/ast"
	"github.com/jackspirou/chip/internal/token"
	"github.com/jackspirou/chip/internal/typerules"
	"github.com/jackspirou/chip/internal/types"
)

// These tests pin the shared rule core that both checkers call. The batch
// checker (chip check) and the streaming checker (chip run) historically each
// owned a copy of these rules; the copies drifted, and chip check ended up more
// lenient than chip run. The rules now live here and are called from both, so a
// test that fails here is a rule both checkers would get wrong together — which
// is the point: drift is impossible, but a wrong shared rule is now a single,
// testable thing.

//
// predicates
//

func TestPredicates(t *testing.T) {
	sliceOfInt := types.Slice{Elem: types.Int}
	sliceOfInvalid := types.Slice{Elem: types.Invalid}

	tests := []struct {
		name string
		pred func(types.Type) bool
		yes  []types.Type // types the predicate must accept
		no   []types.Type // types it must reject
	}{
		{
			name: "IsInvalid",
			pred: typerules.IsInvalid,
			yes:  []types.Type{types.Invalid},
			// A slice whose element is invalid is still a real slice type, not the
			// invalid type — the guard in IsInvalid exists precisely for this.
			no: []types.Type{types.Int, types.Bool, types.Void, sliceOfInt, sliceOfInvalid},
		},
		{
			name: "IsVoid",
			pred: typerules.IsVoid,
			yes:  []types.Type{types.Void},
			no:   []types.Type{types.Invalid, types.Int, types.Bool, sliceOfInt},
		},
		{
			name: "IsBool",
			pred: typerules.IsBool,
			yes:  []types.Type{types.Bool},
			no:   []types.Type{types.Int, types.String, types.Void, sliceOfInt},
		},
		{
			name: "IsInt",
			pred: typerules.IsInt,
			yes:  []types.Type{types.Int},
			no:   []types.Type{types.Float, types.Bool, types.String, sliceOfInt},
		},
		{
			name: "IsString",
			pred: typerules.IsString,
			yes:  []types.Type{types.String},
			no:   []types.Type{types.Int, types.Bool, sliceOfInt},
		},
		{
			name: "IsSlice",
			pred: typerules.IsSlice,
			yes:  []types.Type{sliceOfInt, sliceOfInvalid},
			no:   []types.Type{types.Int, types.String, types.Invalid},
		},
		{
			name: "IsNumeric",
			pred: typerules.IsNumeric,
			yes:  []types.Type{types.Int, types.Float},
			no:   []types.Type{types.Bool, types.String, types.Void, sliceOfInt},
		},
		{
			name: "IsOrdered",
			pred: typerules.IsOrdered,
			yes:  []types.Type{types.Int, types.Float, types.String},
			no:   []types.Type{types.Bool, types.Void, sliceOfInt},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, ty := range tt.yes {
				if !tt.pred(ty) {
					t.Errorf("%s(%s) = false, want true", tt.name, ty)
				}
			}
			for _, ty := range tt.no {
				if tt.pred(ty) {
					t.Errorf("%s(%s) = true, want false", tt.name, ty)
				}
			}
		})
	}
}

func TestBasicKind(t *testing.T) {
	if k := typerules.BasicKind(types.Int); k != types.KindInt {
		t.Errorf("BasicKind(int) = %v, want KindInt", k)
	}
	// A non-basic type (a slice) has no basic kind and reads as invalid.
	if k := typerules.BasicKind(types.Slice{Elem: types.Int}); k != types.KindInvalid {
		t.Errorf("BasicKind([]int) = %v, want KindInvalid", k)
	}
}

//
// binary operators
//

func TestBinaryWellTyped(t *testing.T) {
	tests := []struct {
		op   token.Type
		lt   types.Type
		rt   types.Type
		want types.Type
	}{
		// logical
		{token.LAND, types.Bool, types.Bool, types.Bool},
		{token.LOR, types.Bool, types.Bool, types.Bool},
		// equality (comparisons yield bool)
		{token.EQL, types.Int, types.Int, types.Bool},
		{token.NEQ, types.String, types.String, types.Bool},
		{token.EQL, types.Bool, types.Bool, types.Bool},
		// ordered
		{token.LSS, types.Int, types.Int, types.Bool},
		{token.LEQ, types.Float, types.Float, types.Bool},
		{token.GTR, types.String, types.String, types.Bool},
		{token.GEQ, types.Int, types.Int, types.Bool},
		// arithmetic yields the operand type
		{token.ADD, types.Int, types.Int, types.Int},
		{token.ADD, types.Float, types.Float, types.Float},
		{token.ADD, types.String, types.String, types.String}, // + concatenates strings
		{token.SUB, types.Int, types.Int, types.Int},
		{token.MUL, types.Float, types.Float, types.Float},
		{token.QUO, types.Int, types.Int, types.Int},
		{token.REM, types.Int, types.Int, types.Int},
	}
	for _, tt := range tests {
		got, msg := typerules.Binary(tt.op, tt.lt, tt.rt)
		if msg != "" {
			t.Errorf("Binary(%s, %s, %s) error %q, want well-typed", tt.op, tt.lt, tt.rt, msg)
		}
		if got != tt.want {
			t.Errorf("Binary(%s, %s, %s) = %s, want %s", tt.op, tt.lt, tt.rt, got, tt.want)
		}
	}
}

func TestBinaryIllTyped(t *testing.T) {
	tests := []struct {
		name   string
		op     token.Type
		lt     types.Type
		rt     types.Type
		want   types.Type // the cascade type the rule returns on error
		substr string     // a fragment the message must contain
	}{
		// A broken logical/comparison operand still reads as bool so a condition
		// keeps its shape and the collect-all checker does not cascade.
		{"logical non-bool", token.LAND, types.Int, types.Bool, types.Bool, "requires bool operands"},
		{"compare mismatch", token.EQL, types.Int, types.String, types.Bool, "cannot compare"},
		{"ordered non-ordered", token.LSS, types.Bool, types.Bool, types.Bool, "is not defined"},
		// A broken arithmetic operand reads as invalid.
		{"add mismatch", token.ADD, types.Int, types.String, types.Invalid, "operator + is not defined"},
		{"add bool", token.ADD, types.Bool, types.Bool, types.Invalid, "operator + is not defined"},
		{"sub non-numeric", token.SUB, types.String, types.String, types.Invalid, "is not defined"},
		{"rem non-int", token.REM, types.Float, types.Float, types.Invalid, "requires int operands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := typerules.Binary(tt.op, tt.lt, tt.rt)
			if msg == "" {
				t.Fatalf("Binary(%s, %s, %s) accepted, want error", tt.op, tt.lt, tt.rt)
			}
			if !strings.Contains(msg, tt.substr) {
				t.Errorf("message = %q, want substring %q", msg, tt.substr)
			}
			if got != tt.want {
				t.Errorf("cascade type = %s, want %s", got, tt.want)
			}
		})
	}
}

// Slices are not comparable. This guard must precede the identical-type check,
// or two identical slice types would slip through == and the executor would
// compare raw value words, silently yielding a wrong result.
func TestBinarySlicesNotComparable(t *testing.T) {
	s := types.Slice{Elem: types.Int}
	for _, op := range []token.Type{token.EQL, token.NEQ} {
		got, msg := typerules.Binary(op, s, s)
		if msg == "" {
			t.Errorf("Binary(%s, []int, []int) accepted, want error", op)
		}
		if !strings.Contains(msg, "slices are not comparable") {
			t.Errorf("message = %q, want %q", msg, "slices are not comparable")
		}
		if got != types.Bool {
			t.Errorf("cascade type = %s, want bool", got)
		}
	}
}

// Bitwise and shift operators parse (the precedence table gives them a level)
// but chip does not execute them: the streaming checker and the executor reject
// them, so the shared rule must too. This was the batch checker's headline
// leniency — its default arm returned int and silently accepted them.
func TestBinaryBitwiseAndShiftRejected(t *testing.T) {
	ops := []token.Type{token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AndNot}
	for _, op := range ops {
		got, msg := typerules.Binary(op, types.Int, types.Int)
		if msg == "" {
			t.Errorf("Binary(%s, int, int) accepted, want %q", op, "not supported")
		}
		if !strings.Contains(msg, "is not supported") {
			t.Errorf("Binary(%s) message = %q, want substring %q", op, msg, "is not supported")
		}
		if got != types.Invalid {
			t.Errorf("Binary(%s) type = %s, want invalid", op, got)
		}
	}
}

//
// unary operators
//

func TestUnary(t *testing.T) {
	tests := []struct {
		name    string
		op      token.Type
		t       types.Type
		want    types.Type
		wantErr bool
		substr  string
	}{
		{"negate int", token.SUB, types.Int, types.Int, false, ""},
		{"plus float", token.ADD, types.Float, types.Float, false, ""},
		{"negate non-numeric", token.SUB, types.Bool, types.Invalid, true, "requires a numeric operand"},
		{"not bool", token.NOT, types.Bool, types.Bool, false, ""},
		// ! on a non-bool still reads as bool so a broken condition keeps its shape.
		{"not non-bool", token.NOT, types.Int, types.Bool, true, "requires a bool operand"},
		// Any other unary operator is not part of the language.
		{"unsupported", token.XOR, types.Int, types.Invalid, true, "is not supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := typerules.Unary(tt.op, tt.t)
			if tt.wantErr && msg == "" {
				t.Fatalf("Unary(%s, %s) accepted, want error", tt.op, tt.t)
			}
			if !tt.wantErr && msg != "" {
				t.Fatalf("Unary(%s, %s) error %q, want well-typed", tt.op, tt.t, msg)
			}
			if tt.substr != "" && !strings.Contains(msg, tt.substr) {
				t.Errorf("message = %q, want substring %q", msg, tt.substr)
			}
			if got != tt.want {
				t.Errorf("Unary(%s, %s) = %s, want %s", tt.op, tt.t, got, tt.want)
			}
		})
	}
}

//
// control flow: Terminates / TerminatesStmt
//

// small AST builders so the termination tests read as control-flow shapes, not
// struct literals.
func ret() ast.Stmt                   { return &ast.ReturnStmt{} }
func expr() ast.Stmt                  { return &ast.ExprStmt{X: &ast.Ident{Name: "x"}} }
func block(ss ...ast.Stmt) *ast.Block { return &ast.Block{List: ss} }

// ifElse builds `if cond { body } else { els }`; els may be another *IfStmt to
// model an else-if chain, or a *Block.
func ifElse(body *ast.Block, els ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{Cond: &ast.Ident{Name: "c"}, Body: body, Else: els}
}

func TestTerminatesStmt(t *testing.T) {
	tests := []struct {
		name string
		stmt ast.Stmt
		want bool
	}{
		{"return", ret(), true},
		{"bare expr", expr(), false},
		{"block that returns", block(ret()), true},
		{"block that falls through", block(expr()), false},
		{"empty block", block(), false},
		// if with no else can always fall through the missing branch.
		{"if without else", &ast.IfStmt{Cond: &ast.Ident{Name: "c"}, Body: block(ret())}, false},
		// if/else where both arms return terminates.
		{"if/else both return", ifElse(block(ret()), block(ret())), true},
		// if/else where one arm falls through does not.
		{"if/else one falls through", ifElse(block(ret()), block(expr())), false},
		{"if/else body falls through", ifElse(block(expr()), block(ret())), false},
		// else-if chains recurse through the Else *IfStmt.
		{"else-if all return", ifElse(block(ret()), ifElse(block(ret()), block(ret()))), true},
		{"else-if tail falls through", ifElse(block(ret()), ifElse(block(ret()), block(expr()))), false},
		// an infinite loop (nil Cond) never falls through — chip has no break.
		{"infinite for", &ast.ForStmt{Body: block(expr())}, true},
		// a conditional loop may run zero times and fall through.
		{"conditional for", &ast.ForStmt{Cond: &ast.Ident{Name: "c"}, Body: block(ret())}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typerules.TerminatesStmt(tt.stmt); got != tt.want {
				t.Errorf("TerminatesStmt(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestTerminates(t *testing.T) {
	tests := []struct {
		name string
		list []ast.Stmt
		want bool
	}{
		{"empty", nil, false},
		{"falls through", []ast.Stmt{expr(), expr()}, false},
		{"ends in return", []ast.Stmt{expr(), ret()}, true},
		// a terminator anywhere in the list means control cannot reach the end;
		// dead code after it is the linter's concern, not termination's.
		{"return then dead code", []ast.Stmt{ret(), expr()}, true},
		{"ends in infinite for", []ast.Stmt{expr(), &ast.ForStmt{Body: block(expr())}}, true},
		{"ends in terminating if", []ast.Stmt{ifElse(block(ret()), block(ret()))}, true},
		{"ends in fall-through if", []ast.Stmt{ifElse(block(ret()), block(expr()))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typerules.Terminates(tt.list); got != tt.want {
				t.Errorf("Terminates(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

//
// main signature
//

func TestMainTakesArgs(t *testing.T) {
	param := &ast.Field{Name: &ast.Ident{Name: "x"}, Type: &ast.Ident{Name: "int"}}
	tests := []struct {
		name string
		fn   *ast.FuncDecl
		want bool
	}{
		{"main with a param", &ast.FuncDecl{Name: &ast.Ident{Name: "main"}, Params: []*ast.Field{param}}, true},
		{"main with no params", &ast.FuncDecl{Name: &ast.Ident{Name: "main"}}, false},
		// only the entry point is constrained — an ordinary function may take args.
		{"non-main with a param", &ast.FuncDecl{Name: &ast.Ident{Name: "f"}, Params: []*ast.Field{param}}, false},
		{"anonymous", &ast.FuncDecl{Params: []*ast.Field{param}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typerules.MainTakesArgs(tt.fn); got != tt.want {
				t.Errorf("MainTakesArgs(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
