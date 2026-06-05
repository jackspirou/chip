# AGENTS.md

chip is a small streaming scripting language implemented in Go (standard library
only, single binary). This file orients an agent working **in this repository**.
For the contract on **using the `chip` CLI** to run, validate, and repair
programs, read the bundled skill: `.agents/skills/chip/SKILL.md` (also mirrored
at `.claude/skills/chip/SKILL.md`).

## Build, test, run

```sh
go build -o chip ./cmd/chip     # build the CLI
go test ./...                   # full test suite
go vet ./...                    # static checks
gofmt -l cmd internal           # formatting — empty output means clean
./chip run examples/hello.chp   # run an example
```

## Layout

- `cmd/chip/` — the CLI: `run`, `check`, `fmt`, `lint`, `dump`, `ast`, `explain`,
  `version`, `repl`.
- `internal/` — the implementation: `scanner` / `parser` / `ast` (front end),
  `check` / `types` / `scope` (type checker), `stream` (the demand-driven
  tree-walking engine behind `run` and `check`), `vm` / `compiler` / `code` (the
  bytecode VM behind `dump`), `diag` (the diagnostic spine: codes, renderers,
  JSON envelope), `lint`, `format`, `analyze` (shared batch analysis), `std`
  (bundled standard library and prelude).
- `chip.go` and the root `chip` package — the embedding API (`chip.Run`,
  `chip.Check`, `chip.Format`, `chip.Lint`, and the streaming/limit variants).
- `examples/` — runnable `.chp` programs.

## Conventions

- The CLI is the agent surface: program output → stdout, diagnostics → stderr,
  and the exit code is the signal (0 ok, 1 runtime fault, 2 static error, 64
  usage, 66 no input, 73 cannot save, 124 timeout, 125 resource cap).
- `chip run` is fail-fast — it halts execution at the first fault. `chip check`
  is the whole-program collect-all validator and never executes.
- Diagnostics carry stable, phase-prefixed codes (`PAR` / `NAM` / `TYP` / `RUN`
  / `RES` / `LNT` / `USE`); `chip explain CODE` documents each one.
- Format Go with `gofmt`. Keep `chip run <file>.chp` output byte-compatible with
  the current CLI unless you are deliberately changing the contract.
