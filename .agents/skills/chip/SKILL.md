---
name: chip
description: Run, validate, and repair chip programs (a small streaming scripting language) from the command line. Use when writing or fixing .chp source — stream code into `chip run` or `chip check`, read machine-readable diagnostics (stable error codes, a JSON envelope, sysexits exit codes), and repair in a few rounds.
---

# chip

`chip` is a streaming scripting language whose CLI is built for automated use:
stream a program in, get structured diagnostics out, branch on the exit code,
repair, repeat. The three machine signals are always the same — program output
goes to **stdout**, every diagnostic goes to **stderr**, and the **exit code**
is the primary thing to branch on.

## Get the binary

```sh
go build -o chip ./cmd/chip                                  # from a chip checkout
go install github.com/jackspirou/chip/cmd/chip@latest        # or via the module path
chip version --json                                          # confirm it runs
```

The examples below assume `chip` is on `PATH`.

## Run a program (run = execute, fail-fast)

`chip run` executes a program. Source comes from stdin (`-`), inline (`-e`), or a
file path. Execution **halts at the first fault**, so a harness can abort a
still-streaming generation the moment a program goes wrong.

```sh
printf 'print(6*7)\n' | chip run -        # -> 42   (stdout, exit 0)
chip run -e 'print(6 * 7)'                # -> 42   (stdout, exit 0)
chip run program.chp
```

## Validate everything (check = collect-all, never executes)

`chip check` reports **every** parse and type error at once, cascade-pruned, and
never runs the program. Use it to validate generated code before (or instead of)
executing it.

```sh
printf 'func f(n int) string { return n }\nprint(undef)\n' | chip check -
# error[TYP001]: <stdin>:1:31: cannot return int as string
# error[NAM001]: <stdin>:2:7: undefined: undef
# found 2 errors
# exit 2
```

`chip check --strict` also fails on lint warnings; `chip lint` reports unused
names, dead code, and missing returns on their own.

**run vs check:** `run` stops executing at the first fault (fast feedback while a
program streams in); `check` is the whole-program validator (every static error
in one pass). Drive a repair loop with `check`, then `run` once it is clean.

## Machine-readable output

Choose the renderer with `--format auto|human|agent|json|llm` (or the `--json`
shorthand). Off a TTY the default is **agent**; on a TTY it is **human** (the
caret view). Set `CHIP_FORMAT` to change the default.

**agent** — one greppable line per diagnostic, then a count. Grep `^error\[` to
count and route faults:

```
error[CODE]: file:line:col: message
```

**llm** (`--format=llm`, or `llm:<model>` to tag a profile) — the agent lines, then
a `how to fix:` appendix carrying the catalog's one-line fix for each distinct code
present. The diagnostic lines are unchanged (still grep `^error\[`); the grounded
hint rides underneath:

```
error[CODE]: file:line:col: message
found 1 error
how to fix:
  CODE (title): how to fix it
```

**json** — a `Result` envelope on **stderr** (stdout stays pure program output):

```json
{"ok":false,"ranToCompletion":false,"diagnostics":[
  {"code":"NAM001","severity":"error","phase":"type",
   "message":"undefined: undef",
   "primary":{"file":"<stdin>","line":2,"column":7,"endLine":2,"endColumn":7,
              "offset":40,"endOffset":40,"snippet":"print(undef)","highlight":[7,7]},
   "rendered":"<stdin>:2:7: undefined: undef\n  print(undef)\n        ^"}],
 "truncatedDiagnostics":0,"suppressedCascades":0}
```

- `ok` is false when any error-severity diagnostic is present.
- `ranToCompletion` reports whether a `run` finished executing (false for
  `check`, which never runs, and for a run that faulted).
- `offset`/`endOffset` are **byte** offsets into the source — a precise span to
  apply an edit against.
- `suppressedCascades` counts diagnostics hidden as downstream noise.
- A clean `chip run --json` prints **nothing** on stderr; only `check` always
  emits the envelope.

## Exit codes (the primary signal)

| code | meaning |
|---|---|
| 0   | clean — ran or validated with no errors |
| 1   | ran, then threw — a runtime fault (`RUN…`) |
| 2   | static error — parse / type / name / lint (the program never ran) |
| 64  | usage — a bad flag or argument |
| 66  | no input — source missing or unreadable |
| 73  | cannot save — a `--save` target exists (pass `--force`) |
| 124 | timeout — `--timeout` exceeded (`RES001`) |
| 125 | resource cap — `--max-steps` / `--max-output` (`RES002` / `RES003`) |

A repair loop branches on these: **2** → fix the source and retry; **1** → the
program is valid but faulted at runtime; **0** → done.

## Diagnostic codes

Stable, phase-prefixed codes. `chip explain` lists them all; `chip explain CODE`
prints a description, a minimal repro, and a fix.

```
PAR001 syntax error           NAM001 undefined name          TYP001 type mismatch
NAM002 undefined type         TYP002 missing return          TYP003 wrong number of arguments
NAM003 redeclared name        TYP004 not a value
RUN001 division by zero       RUN002 index out of range      RUN003 call stack too deep
RUN004 runtime fault          RES001 timeout                 RES002 step budget exceeded
RES003 output limit exceeded  USE001 usage error
LNT001 unused variable        LNT002 unused function         LNT003 unused import
LNT004 unreachable code       LNT005 used before definition  LNT006 missing return
```

## Safe execution

Run untrusted or runaway code under bounds. Every cap and seal is opt-in and
none of them change a default run.

```sh
chip run - --max-steps=100000      # deterministic step budget  -> exit 125 (RES002)
chip run - --max-output=4096       # cap on printed bytes        -> exit 125 (RES003)
chip run - --timeout=2s            # wall-clock limit            -> exit 124 (RES001)
chip run - --max-depth=8192        # call-depth limit (a breach is a runtime fault, exit 1)
chip run - --no-imports            # refuse host imports (bundled stdlib still resolves)
chip run - --no-prelude            # drop the unqualified prelude (engine builtins remain)
```

The step budget is deterministic: the same input trips the cap at the same point
every time.

## Capture the source

```sh
chip run - --save                  # save the source to a fresh $TMPDIR file (path echoed to stderr)
chip run - --save=out.chp          # save to a path (exit 73 if it exists; --force to overwrite)
chip run - --save-on-error=err.chp # save only when the run faults
```

The captured file always holds the **whole** program, including lines past the
fault.

## Repair loop discipline

1. Generate `.chp` source.
2. Validate with `chip check -` (add `--json` to parse the result). If exit 0,
   run it.
3. On exit 2, read the diagnostics — and `chip explain CODE` for any unfamiliar
   one — fix the cited spans, and retry.
4. Cap at ~3 rounds. If it is not converging, re-generate from scratch rather
   than grind on edits.
5. While streaming a long program to `chip run -`, treat a non-zero exit as the
   cue to abort generation early instead of finishing a doomed program.

## Other verbs

`chip fmt` (canonical formatting: `-w` rewrites, `-l` lists, `-d` diffs),
`chip lint`, `chip ast` (syntax tree), `chip dump` (bytecode disassembly),
`chip repl` (line-by-line, for humans), `chip version`, `chip help`.
