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

_(none yet — baseline only)_
