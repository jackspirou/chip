# chip

[![CI](https://github.com/jackspirou/chip/actions/workflows/ci.yml/badge.svg)](https://github.com/jackspirou/chip/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/jackspirou/chip.svg)](https://pkg.go.dev/github.com/jackspirou/chip) [![Go Report Card](https://goreportcard.com/badge/github.com/jackspirou/chip)](https://goreportcard.com/report/github.com/jackspirou/chip) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A toy systems scripting language with Go-like syntax that runs as a
demand-driven stream.

```chip
func gcd(a int, b int) int {
    if b == 0 {
        return a
    }
    return gcd(b, a % b)
}

func main() {
    print(gcd(252, 105))   // 21
}
```

## Why chip

- **Streaming execution.** A program runs as it is read — there is no
  whole-program gate. A statement can even call a function defined further down
  the file; chip reads ahead to resolve it
  ([example](examples/forward_reference.chp)).
- **Minimal, Go-like syntax.** `func`, `:=`, `if`, `for`, and little else.
- **Strongly typed, incrementally.** Type checking is woven into the stream,
  not run as a separate pass.
- **Pure Go.** No cgo, no codegen backend — embed it as a library or use the CLI.
- **A standard library written in chip.** Only `print` and `len` are built in;
  the rest grows in chip.

## Install

```sh
go install github.com/jackspirou/chip/cmd/chip@latest
```

## Usage

```sh
chip run examples/hello.chp   # run a program (shorthand: chip examples/hello.chp)
chip repl                     # interactive session
chip fmt  <file.chp>          # canonical formatting
chip lint <file.chp>          # unused names, missing returns, dead code
```

## Learn more

- [ARCHITECTURE.md](ARCHITECTURE.md) — how chip streams, and the package layout.
- [examples/](examples/) — small, runnable programs, each verified by a test.

## Status

A learning project and a labor of love — expect rough edges and breaking
changes. chip reimagines SNARL, a MIPS-targeting teaching compiler, in the
spirit of Go's lexical simplicity: extreme minimal syntax, idiomatic Go
implementation.

## License

[MIT](LICENSE)
