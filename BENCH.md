# Streaming engine: allocation benchmarks

These benchmarks track allocation on the streaming run path (`internal/stream`).
The goal of the work they support is to **cut allocations on the hot path
without changing language semantics, output, or the public API**. Every change
is justified by a before/after measured here.

## How to run

```sh
go test ./internal/stream/ -bench . -benchmem -count=6
```

For a statistical before/after, save two runs and compare with benchstat:

```sh
go test ./internal/stream/ -bench . -benchmem -count=6 > old.txt
# ...make a change...
go test ./internal/stream/ -bench . -benchmem -count=6 > new.txt
go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
```

`BenchmarkStreamRun` measures the public `Run` end to end (parse + check +
execute, with the stdlib prelude fed first). `loadStd` is a constant cost in
every measurement, so per-change deltas stay valid; the heavy programs dominate
it. `BenchmarkExamples` runs the real programs in `examples/` — small and
parse-dominated, so they are a regression guard on real syntax rather than a
hot-path signal.

## What each program isolates

| Program | Stresses |
|---|---|
| `fib` | tree recursion: per-call scope + 1-arg `evalArgs`, 392,835 calls |
| `call_overhead` | a trivial `id(n)` in a tight loop: pure call machinery |
| `loop_scope` | a no-declaration loop body: per-iteration child-scope churn |
| `loop_decl` | a loop body that declares a scalar: the genuine child-scope case |
| `nested_loops` | 400×400 nested loops: compounded scope churn |
| `string_build` | repeated `s + "x"`: string `Value` + arithmetic |
| `array_build` | a `[]int{...}` composite literal built each iteration |
| `mutual_recursion` | `even`/`odd` via a forward reference, called repeatedly |
| `import_math` | a qualified call into the imported `math` package (`math.Gcd`) |

## Baseline

Environment: Apple M2 Max (`darwin/arm64`), Go 1.26.2, `-count=6`. Variance was
±0% on `B/op` and `allocs/op` (the run is deterministic) and ≤5% on `sec/op`.

| Benchmark | sec/op | B/op | allocs/op |
|---|--:|--:|--:|
| `StreamRun/fib` | 269.6m | 549,192,000 | 3,178,427 |
| `StreamRun/call_overhead` | 26.02m | 44,815,700 | 300,342 |
| `StreamRun/loop_scope` | 51.59m | 12,814,890 | 400,309 |
| `StreamRun/loop_decl` | 24.92m | 38,415,260 | 150,318 |
| `StreamRun/nested_loops` | 41.73m | 10,562,880 | 321,535 |
| `StreamRun/string_build` | 810.7µ | 2,270,270 | 6,318 |
| `StreamRun/array_build` | 1.251m | 2,575,645 | 8,347 |
| `StreamRun/mutual_recursion` | 41.61m | 88,017,860 | 440,419 |
| `StreamRun/import_math` | 13.74m | 27,484,215 | 140,341 |
| `Examples/fibonacci.chp` | 101.2µ | 166,247 | 1,216 |
| `Examples/mutual_recursion.chp` | 33.06µ | 23,910 | 424 |
| `Examples/arrays.chp` | 30.42µ | 16,546 | 396 |
| `Examples/gcd.chp` | 26.82µ | 16,370 | 343 |

Reading the baseline: `loop_scope` does ~2 allocs per iteration (400,309 over
200,000) even though its body binds nothing — that is the per-iteration child
`*env` (the struct plus its `map` header). The call-heavy programs (`fib`,
`call_overhead`, `mutual_recursion`) carry a per-call scope *with* a populated
map, plus a fresh `[]value.Value` for arguments. Those are the allocations the
optimizations below target.

## Optimizations

One commit per change, each with its benchstat delta. Filled in as they land.

### 1. `env`: slice of bindings instead of a per-scope map

Every scope was a `map[string]value.Value`. chip's scopes are tiny — a function
scope holds only its parameters, and most block scopes hold nothing — so a map
paid a header allocation (and a bucket once populated) per scope for a handful
of entries. Replacing it with a `[]binding` scanned linearly removes the map
header on every scope and the bucket on every populated one; an empty block
scope now costs a single `*env` struct.

vs baseline (`-count=6`, benchstat):

| Benchmark | sec/op | B/op | allocs/op |
|---|--:|--:|--:|
| `StreamRun/fib` | −35.0% | −77.8% | −30.0% |
| `StreamRun/call_overhead` | −41.1% | −76.8% | −33.3% |
| `StreamRun/loop_scope` | −38.9% | −50.0% | −50.0% |
| `StreamRun/loop_decl` | −53.4% | −85.4% | −33.3% |
| `StreamRun/nested_loops` | ~ (noisy) | −51.0% | −49.9% |
| `StreamRun/array_build` | −35.3% | −51.0% | −24.0% |
| `StreamRun/mutual_recursion` | −36.1% | −78.6% | −26.1% |
| `StreamRun/import_math` | −23.6% | −55.2% | −7.1% |
| **geomean** | **−30.0%** | **−57.8%** | **−25.9%** |

All deltas `p=0.002 (n=6)` except `nested_loops` sec/op (`p=0.065`, a one-sample
timing outlier; its allocs/op and B/op both show a clean −50%).

### 2. `exec`: skip the child scope for blocks that declare nothing

A `for`/`if`/bare block was given a fresh child `*env` unconditionally — every
loop iteration, every taken branch. But a child scope is only needed to hold and
isolate `:=` declarations; if a body has no direct `DeclStmt`, running it in the
enclosing scope is identical (assignments search outward either way, and a
nested `if`/`for`/block still makes its own scope for *its* declarations). A
shallow `blockDeclares` scan decides this — once, before the loop in `execFor` —
and it is cheaper than the scope it avoids allocating.

vs optimization 1 (`-count=6`, benchstat):

| Benchmark | sec/op | B/op | allocs/op |
|---|--:|--:|--:|
| `StreamRun/loop_scope` | −8.8% | −99.8% | −99.85% (200,307 → 303) |
| `StreamRun/nested_loops` | −52.0% | −98.9% | −99.30% (161k → 1.1k) |
| `StreamRun/string_build` | −13.7% | −2.9% | −46.3% |
| `StreamRun/call_overhead` | (noise) | −15.4% | −25.0% |
| `StreamRun/fib` | −3.6% | −8.3% | −14.3% |
| `StreamRun/loop_decl` | ~ | ~ | ~ (body declares; opt n/a) |
| **geomean (full suite)** | — | **−57.6%** | **−62.5%** |

`allocs/op` and `B/op` are deterministic (±0%, `p≤0.002`). The `sec/op` figures
are from a back-to-back A/B (`-count=10`, opt1 stashed/restored on the same
machine) because the cross-run comparison drifted thermally — `nested_loops`
read as +9.9% across runs taken minutes apart but is −52% measured back to back,
and `call_overhead`'s time sample was too noisy to call (its allocs are a clean
−25%). The loop benchmarks lose ~all per-iteration allocation: `loop_scope`'s
body binds nothing, so it drops from one child `*env` per iteration to none.

### 3. `value`: 56-byte Value (merge the int/float/bool word)

`Value`'s `int`, `bool`, and `float` payloads each fit in 8 bytes and are never
live at the same time as one another, so they shared one `uint64` (a float is
held as `math.Float64bits`). That drops `Value` from 64 to 56 bytes with no API
or behavior change — the accessors are byte-identical. This does not change
allocation *counts*; it shrinks every `[]Value` and the bytes copied on each of
the millions of by-value returns the tree-walker makes.

vs optimization 2, back-to-back A/B (opt2 stashed/restored on the same machine):

| Benchmark | sec/op | B/op | allocs/op |
|---|--:|--:|--:|
| `StreamRun/array_build` | −9.8% | −10.2% (512→448 B/literal) | ~ |
| `StreamRun/import_math` | −2.1% | −8.0% (Gcd's 2-arg slice 128→112 B) | ~ |
| `StreamRun/string_build` | −14.8% | ~ | ~ |
| `StreamRun/fib` | −12.2% | ~ | ~ |
| `StreamRun/call_overhead` | −8.8% | ~ | ~ |
| `StreamRun/loop_scope` | −8.9% | ~ | ~ |
| `StreamRun/nested_loops` | −6.6% | ~ | ~ |

`B/op` falls only where a slice's element count crosses a Go size class (8-elem
and 2-elem `[]Value`); a 1-element slice is unchanged because 56 B still rounds
up to the 64 B class. The `sec/op` gains come from copying a smaller `Value`.
(`string_build` first read +42.9% in a `-count=10` sample but −14.8% at
`-count=20` — a load spike; its string path never touches the merged word.)

### 4. `callPrint`: format into a reused buffer, no per-call holder slice

`callPrint` built a fresh `[]string` of every argument's `String()`, joined it
with `strings.Join`, and handed that to `fmt.Fprintln` — for the common
single-argument `print(x)` that is two heap allocations per call (the holder
slice and the joined/boxed string) on top of the one `String()` itself makes.
The rewrite formats directly into a byte buffer reused across calls
(`engine.printBuf`) and writes it once. The single-argument case — by far the
most common — takes a fast path that allocates no holder at all; multiple
arguments keep a per-call `[]string` (the same 16 B/elem the original used) but
still skip `Join`/`Fprintln` in favor of the reused buffer. Every argument is
evaluated before the buffer is touched, so a nested `print` (an argument that
calls a function that prints) can't corrupt it and a mid-evaluation error still
writes nothing — output and error behavior are byte-for-byte unchanged.

This adds the `print_loop` program (20,000 single-argument prints to the sink),
which the baseline suite did not isolate. vs optimization 3, back-to-back A/B
(HEAD `callPrint` stashed/restored on the same machine, `-count=20`, benchstat):

| Benchmark | sec/op | B/op | allocs/op |
|---|--:|--:|--:|
| `StreamRun/print_loop` | −25.6% | −75.5% (414.0Ki → 101.3Ki) | −49.8% (40.17k → 20.18k) |

`B/op` and `allocs/op` are deterministic (±0%, `p=0.000 n=20`). Allocations halve
— from two per print to one — because the holder slice and the joined string are
both gone; the single survivor is the value's own `String()` (e.g. `FormatInt`),
which is inherent. The reused `printBuf` grows once per run and is amortized away.
The only programs that call `print` more than once at top level are `print_loop`
and the example files; the rest print a single result, so this change is a wash
for them and a large win wherever printing is on the hot path.
