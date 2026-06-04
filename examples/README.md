# chip examples

Runnable chip programs. Run one with:

```sh
chip run examples/hello.chp
# or the shorthand
chip examples/hello.chp
```

Every example here is exercised by a test (see `examples_test.go` at the repo
root), so its output is verified.

| File | Shows | Output |
|------|-------|--------|
| [hello.chp](hello.chp) | a minimal `main` | `Hello, chip` |
| [gcd.chp](gcd.chp) | recursion (Euclid's algorithm) | `21` |
| [fibonacci.chp](fibonacci.chp) | recursion, define-before-use | `55` |
| [mutual_recursion.chp](mutual_recursion.chp) | mutual recursion via a forward reference | `1` |
| [forward_reference.chp](forward_reference.chp) | streaming: a statement calls a function defined later | `streamed!` |
| [arrays.chp](arrays.chp) | slices, `len`, indexing, `for` | `4` then `20` |

See [ARCHITECTURE.md](../ARCHITECTURE.md) for how chip streams these programs.
