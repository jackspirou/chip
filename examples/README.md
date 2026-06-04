# chip examples

Runnable chip programs. Run one with:

```sh
chip run examples/hello.chp
# or the shorthand
chip examples/hello.chp
```

Every single-file example here is exercised by a test (see `examples_test.go` at
the repo root), so its output is verified.

| File | Shows | Output |
|------|-------|--------|
| [hello.chp](hello.chp) | a minimal `main` | `Hello, chip` |
| [gcd.chp](gcd.chp) | recursion (Euclid's algorithm) | `21` |
| [fibonacci.chp](fibonacci.chp) | recursion, define-before-use | `55` |
| [mutual_recursion.chp](mutual_recursion.chp) | mutual recursion via a forward reference | `1` |
| [forward_reference.chp](forward_reference.chp) | streaming: a statement calls a function defined later | `streamed!` |
| [arrays.chp](arrays.chp) | slices, `len`, indexing, `for` | `4` then `20` |

## Packages

[packages/](packages/) is a multi-file program. Its entry file
[packages/main.chp](packages/main.chp) imports a local two-file package
([packages/geometry/](packages/geometry/)) and the built-in `math` package, then
calls an exported function from each:

```sh
chip run examples/packages/main.chp   # prints 12 then 21
```

Imports resolve relative to the entry file's directory, so `import "geometry"`
finds `packages/geometry/`; `import "math"` resolves to the built-in package
(bundled in the binary). This example is verified by a test in `cmd/chip`.

See [ARCHITECTURE.md](../ARCHITECTURE.md) for how chip streams these programs and
loads packages.
