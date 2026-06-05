package stream

import (
	"io"
	"os"
	"strings"
	"testing"
)

// These benchmarks measure the streaming run path end to end (Run parses,
// checks, and executes a whole program, feeding the stdlib prelude first). They
// exist to track allocation on the hot path — each program below is tuned to be
// CPU/allocation-bound and deterministic, and named for the path it stresses:
//
//   - fib              tree recursion: function-call machinery + 1-arg evalArgs
//   - call_overhead    a trivial function called in a tight loop: pure call cost
//   - loop_scope       a no-declaration loop body: per-iteration scope churn
//   - loop_decl        a loop body that declares a scalar: with-child-scope cost
//   - nested_loops     nested loops: compounded scope churn
//   - string_build     repeated string concatenation: string value + arith
//   - array_build      a composite literal built each iteration: slice values
//   - mutual_recursion even/odd with a forward reference, called repeatedly
//   - import_math      a qualified call into an imported package (math.Gcd)
//
// loadStd runs on every Run, a constant cost across before/after, so the
// per-optimization deltas stay valid. The heavy programs dominate that fixed
// cost; the light example programs in BenchmarkExamples do not (they are a
// regression guard on real syntax, not a hot-path measurement).
var benchPrograms = []struct {
	name string
	src  string
}{
	{
		// Tree recursion: 392,835 calls, each a fresh per-call scope with one
		// bound parameter and a single-argument evalArgs. The canonical
		// function-call-overhead workload.
		name: "fib",
		src: `func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n - 1) + fib(n - 2)
}
print(fib(27))
`,
	},
	{
		// Pure call cost: a one-line function returning its argument, called
		// 50,000 times. Isolates newEnv + param binding + execStmts + evalArgs
		// from any real work inside the callee.
		name: "call_overhead",
		src: `func id(n int) int {
	return n
}
func run() int {
	total := 0
	i := 0
	for i < 50000 {
		total = total + id(i)
		i = i + 1
	}
	return total
}
print(run())
`,
	},
	{
		// Loop scope churn with a body that declares nothing (only assignments).
		// Today every iteration still gets a child scope; this is the program
		// that should benefit most from skipping it.
		name: "loop_scope",
		src: `func run() int {
	total := 0
	i := 0
	for i < 200000 {
		total = total + i
		i = i + 1
	}
	return total
}
print(run())
`,
	},
	{
		// Loop whose body declares a scalar, so it genuinely needs a child
		// scope. The control case against loop_scope: this one keeps its scope.
		name: "loop_decl",
		src: `func run() int {
	total := 0
	i := 0
	for i < 50000 {
		x := i * 2
		total = total + x
		i = i + 1
	}
	return total
}
print(run())
`,
	},
	{
		// Nested loops: 400 x 400. The outer body declares the inner counter (a
		// child scope); the inner body declares nothing (160,000 iterations of
		// scope churn).
		name: "nested_loops",
		src: `func run() int {
	total := 0
	i := 0
	for i < 400 {
		j := 0
		for j < 400 {
			total = total + 1
			j = j + 1
		}
		i = i + 1
	}
	return total
}
print(run())
`,
	},
	{
		// String building: 2,000 concatenations. Each `s + "x"` allocates a new
		// string (inherent); the loop body declares nothing.
		name: "string_build",
		src: `func build(n int) string {
	s := ""
	i := 0
	for i < n {
		s = s + "x"
		i = i + 1
	}
	return s
}
print(len(build(2000)))
`,
	},
	{
		// Slice construction: a fresh []int composite literal every iteration,
		// indexed once. Exercises evalCompositeLit and the slice-backed Value.
		name: "array_build",
		src: `func build() int {
	total := 0
	i := 0
	for i < 2000 {
		xs := []int{1, 2, 3, 4, 5, 6, 7, 8}
		total = total + xs[i % 8]
		i = i + 1
	}
	return total
}
print(build())
`,
	},
	{
		// print in a tight loop: 20,000 lines to the sink. Isolates callPrint's
		// per-call cost (formatting + the write), which is otherwise a
		// once-per-program event the other benchmarks barely touch.
		name: "print_loop",
		src: `func run() {
	i := 0
	for i < 20000 {
		print(i)
		i = i + 1
	}
}
run()
`,
	},
	{
		// Mutual recursion through a forward reference (even calls odd, defined
		// later). even(20) is depth-safe; calling it 5,000 times is 100,000
		// calls without tripping the recursion guard.
		name: "mutual_recursion",
		src: `func even(n int) int {
	if n == 0 {
		return 1
	}
	return odd(n - 1)
}
func odd(n int) int {
	if n == 0 {
		return 0
	}
	return even(n - 1)
}
func run() int {
	total := 0
	i := 0
	for i < 5000 {
		total = total + even(20)
		i = i + 1
	}
	return total
}
print(run())
`,
	},
	{
		// Qualified call into an imported built-in package, exercising the
		// package-resolution path (loader + evalQualifiedCall) 5,000 times on
		// top of Gcd's own recursion.
		name: "import_math",
		src: `import "math"

func run() int {
	total := 0
	i := 1
	for i < 5000 {
		total = total + math.Gcd(i, 252)
		i = i + 1
	}
	return total
}
print(run())
`,
	},
}

// BenchmarkStreamRun runs each tuned program through the public Run path,
// reporting allocations. Compare runs with:
//
//	go test ./internal/stream/ -bench BenchmarkStreamRun -benchmem -count=6
func BenchmarkStreamRun(b *testing.B) {
	for _, p := range benchPrograms {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := Run(strings.NewReader(p.src), io.Discard); err != nil {
					b.Fatalf("Run(%s): %v", p.name, err)
				}
			}
		})
	}
}

// BenchmarkExamples runs the real example programs from examples/ (read once,
// up front). These are small and dominated by parse + loadStd, so they are a
// regression guard on real syntax rather than a hot-path measurement.
func BenchmarkExamples(b *testing.B) {
	examples := []string{
		"fibonacci.chp",
		"mutual_recursion.chp",
		"arrays.chp",
		"gcd.chp",
	}
	for _, name := range examples {
		src, err := os.ReadFile("../../examples/" + name)
		if err != nil {
			b.Fatalf("read %s: %v", name, err)
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := Run(strings.NewReader(string(src)), io.Discard); err != nil {
					b.Fatalf("Run(%s): %v", name, err)
				}
			}
		})
	}
}
