# chip architecture

chip is a toy systems scripting language implemented in Go. Its defining trait
is **streaming execution**: a program runs as a *demand-driven stream* — source
flows reader → scanner → parser → executor with **no whole-program gate**.
Execution can begin (and produce output) before the rest of the file has even
been parsed.

This document explains how chip is put together and, in particular, what
"streamability" means and how it is achieved.

## The big idea: streamability

Most compilers and interpreters are *batch*: they parse the whole program, then
check it, then run it. Each stage is a gate — nothing downstream happens until
the stage before it finishes for the entire program.

chip's execution path has no such gate. The parser yields one **top-level item**
at a time (a function definition or a statement), and the engine acts on each as
it arrives:

- a **definition** is registered (and type-checked) immediately;
- a **statement** runs immediately.

When running code needs a symbol that hasn't streamed in yet — a call to a
function defined later in the file — the engine simply **reads ahead** until the
definition arrives. The end of the stream is the deadline: a symbol still
unresolved at EOF is an `undefined` error.

Guiding principle: **wait as long as we can; proceed the moment we can.**

```
        ┌─────────┐   bytes   ┌──────────┐  runes  ┌──────────┐  tokens
 source │ reader  │ ────────► │ scanner  │ ──────► │  parser  │ ─────────┐
        └─────────┘           └──────────┘         └──────────┘          │
                                                                         │ one top-level
                                                  ast.Node items         │ item at a time
                                                                         ▼
                          ┌───────────────────────────┐      ┌────────────────────────┐
                          │  BATCH consumers (tooling) │      │  STREAMING consumer     │
                          │  parser.Parse() -> *File   │      │  parser.Items()         │
                          │   • check  (type check)    │      │   -> iter.Seq2[Node,err]│
                          │   • lint   (style/flow)    │      │   pulled by the engine  │
                          │   • format (canonical src) │      │   (internal/stream)     │
                          │   • compiler+vm (chip dump)│      │   = the default runtime │
                          └───────────────────────────┘      └────────────────────────┘
```

One parser, two consumers. Tooling reads the whole tree at once; execution pulls
it item by item.

## Package map

| Package | Role |
|---------|------|
| `internal/reader` | reads source into runes |
| `internal/scanner` | runes → tokens (positions tracked for diagnostics) |
| `internal/token` | token kinds, positions, precedence |
| `internal/ast` | syntax tree nodes (`File`, `FuncDecl`, statements, expressions) |
| `internal/parser` | recursive-descent parser. `Parse()` → `*ast.File` (batch); `Items()` → `iter.Seq2[ast.Node, error]` (streaming) |
| `internal/types` | the static type system (`Basic`, `Slice`, `Signature`, `Identical`) |
| `internal/scope` | lexical scopes and symbols (used by the batch checker) |
| `internal/check` | **batch** name resolution + type checker → `check.Info` |
| `internal/lint` | batch style/flow checks (unused, missing return, unreachable, use-before-def) |
| `internal/format` | canonical pretty-printer, including fmt-for-streamability |
| `internal/value` | runtime values (tagged union; scalars inline) |
| `internal/code` | bytecode (`Opcode`, `Chunk`, `Program`) + disassembler |
| `internal/compiler` | `*ast.File` + `check.Info` → bytecode (batch path) |
| `internal/vm` | stack VM that executes bytecode (batch path) |
| **`internal/stream`** | **the streaming evaluator — chip's default runtime** |
| `internal/std` | the standard library, written in chip and embedded (the flat prelude + importable packages like `math`) |
| `cmd/chip` | the CLI (`run`, `repl`, `fmt`, `lint`, `dump`, `ast`) |
| `chip` (root) | the embeddable Go API (`Run`, `Compile`, `Format`, `Lint`) |

## Streaming execution (`internal/stream`)

The engine is a **single goroutine** running a pull loop over `parser.Items()`
via Go 1.23's `iter.Pull2`. The key insight that keeps it simple: in a file,
"wait for a symbol" just means "read more of the file" — a sequential *pull*, not
an asynchronous event. So no channels, no locks, no goroutine coordination — the
**Go call stack is the suspension** when the engine reads ahead.

The loop (`drain`):

1. If a statement is pending, type-check it, then execute it.
2. Otherwise pull the next item: register a definition, or queue a statement.
3. At end of input, if no top-level statement ran and `main` is defined, call
   `main` — so a declaration-only program still runs.

Forward references are handled by `resolve(name)`: if a called function isn't
registered yet, it pulls more items (registering definitions, queuing statements
— never running them) until the function arrives or the stream ends. Because
read-ahead never *runs* statements, **functions are forward-referenceable but
top-level variables are define-before-use**.

### A checked tree-walker, not a dynamic VM

Execution is a tree-walk over the AST (`eval` for expressions, `execStmt` for
statements) — there is deliberately no compile step on the run path, so no
whole-program gate can sneak in. But it is a *checked* tree-walker: chip stays
strongly typed because type checking is woven into the stream rather than run as
a batch gate:

- A function body is type-checked **when it registers** (so an error is caught
  even if nothing calls the function). Missing signatures are pulled in on
  demand; results are memoized.
- A top-level statement is checked **just before it runs**.

Because the checker has already validated a node, the evaluator carries **no
runtime type guards** — it only performs checks that are inherently dynamic
(index-out-of-range, division-by-zero) and a recursion-depth guard.

### Runtime state, scope lifetime, and GC

Values live in three places, by role:

- **transient sub-expression values** ride the Go call stack (`eval` returns
  values up the recursion — there is no separate operand stack);
- **function parameters and locals** live in a per-call `*env` (a parent-linked
  map of name → value), and each block scope gets a child `*env`;
- **functions and globals** live in the engine's persistent tables.

Scope lifetime equals Go object lifetime: when the executor finishes a scope and
nothing captured its `*env`, the `*env` becomes unreachable and Go's GC reclaims
it. There is no arena and no manually reused value stack (which would retain dead
scopes via slice aliasing). Scalars (`int`/`float`/`bool`) are stored inline in
`value.Value`, so they never touch the heap; only string/slice data is GC'd.

### One engine, many sources: files, embedding, and the REPL

The engine consumes any `io.Reader`, and its tables **persist across `feed`
calls**. That single property gives three things for free:

- `chip run file.chp` and the embeddable `chip.Run` feed one source.
- `chip repl` feeds standard input line by line; definitions and variables made
  on one line are available on the next.
- the standard library is just another source, fed first (see below).

## Packages

A program can grow from one file into many via a small, Go-style package system.
The guiding rule keeps streaming intact:

> **Stream within a package, load across packages.**

- A **package** is a directory of `.chp` files that all declare the same
  `package NAME`. You `import "path"` and refer to members by name; an alias
  (`import g "path"`) rebinds the reference name. A top-level name is **exported**
  iff its first letter is uppercase (Go-style, no new syntax).
- The **entry package** (the file you run) still *streams* — intra-package
  forward references resolve by reading ahead, exactly as before. A single-file
  program is an implicit `main` package and runs unchanged.
- An **imported package loads eagerly and as a unit**: every file is parsed, all
  top-level signatures are registered, then every body is checked
  (collect-then-check, so cross-file references inside the package resolve). This
  is bounded to one dependency — *not* a whole-program gate. A package loads **at
  most once** (cached by import path) and **import cycles are detected** and
  reported (`import cycle: a -> b -> a`).
- The engine's single flat namespace becomes **one table per package** (`pkg`):
  its functions, globals, and resolved imports. Resolution is always relative to
  a *current* package: an **unqualified** name resolves current package → flat
  prelude → builtins (`print`/`len`); a **qualified** `pkg.Member` resolves in
  the named import and must be exported.
- Imports reach the filesystem through a small **`Loader` seam** (`internal/stream/loader.go`):
  a `DirLoader` rooted at the entry file's directory, plus an in-memory
  `MapLoader` that keeps package tests hermetic. Built-in stdlib packages
  (`import "math"`) are served by a `stdLoader` that resolves them *ahead of* the
  filesystem, so they are never shadowed by a local directory of the same path.

The streaming type checker and the executor each dispatch a call by the shape of
its callee — a bare name (`checkIdentCall`/`evalIdentCall`) or a qualified
`pkg.Member` (`checkQualifiedCall`/`evalQualifiedCall`) — and the qualified path
applies the export check. The batch tooling (`chip lint`/`dump`/`ast`) treats a
package opaquely rather than resolving it: there is no `Loader` on the batch
path, so a `pkg.Fn(...)` type-checks as opaque and `chip dump` reports packages
as unsupported (the bytecode VM is single-package); the streaming run is the
source of truth for cross-package execution.

## Incremental type checking

`internal/stream/check.go` is a fail-fast, incremental checker. It reuses the
type system in `internal/types` and mirrors the rules (and error messages) of the
batch `internal/check` package, but runs per body / per statement and resolves
forward references on demand. The first type error stops the run.

The batch `internal/check` package remains the whole-program checker behind the
tooling (`chip lint`, the compiler). Keeping a separate streaming checker means
the run path never needs the whole tree at once.

## Two backends

The streaming tree-walker is the source of truth and the default. A second,
**batch** backend exists for inspection and (in future) durable artifacts:

```
*ast.File + check.Info ──► internal/compiler ──► internal/code (bytecode) ──► internal/vm
```

`chip dump` uses it to disassemble a program. The two backends share the value
and type packages and agree on results for well-formed programs.

## Batch tooling

Type checking for tools, linting, and formatting all operate on the whole
`*ast.File` from `parser.Parse()` — they are batch by design (you want a tool to
see the entire program). Notably, `internal/format` **optimizes for
streamability**: it topologically sorts top-level definitions so a callee comes
before its caller (mutually recursive functions stay grouped; the ordering is
stable and idempotent), and `internal/lint` emits a *use-before-def* hint that
`chip fmt` clears.

## Standard library

chip keeps only `print` and `len` built into the language. Everything else is
intended to live in `internal/std`, **written in chip itself** and embedded into
the binary. The library has two shapes:

- a **flat prelude** (currently `abs`, `min`, `max`) the engine feeds before user
  code, so its functions are available *unqualified* to every program — a direct
  payoff of the persistent, source-agnostic engine; and
- **importable packages** (currently `math`) you bring in explicitly with
  `import "math"` and call qualified (`math.Gcd`). These resolve ahead of the
  filesystem, so a built-in is never shadowed by a local directory.

The guiding rule: grow the library in chip; add a new built-in only when a
primitive genuinely cannot be expressed in chip.

> Note: the prelude is loaded on the streaming *run* path (`chip run`, `chip
> repl`, `chip.Run`). The batch tooling (`chip lint`, `chip dump`) does not yet
> know about it, so it would report calls to library functions as undefined.
> Teaching the batch checker about the prelude is future work.

## Concurrency

The engine is single-goroutine on purpose. The model is *pull* (demand-driven
read-ahead); a goroutine pipeline is *push* and cannot read ahead on demand.
Channel hand-offs also cost far more than the per-token work, so a fine-grained
pipeline would be slower, not faster — "concurrency is not parallelism." Where
concurrency would earn its place is coarse-grained and deferred: prefetching a
slow/remote source behind the same `resolve` seam, parallel per-function
compilation in a future `chip build`, and chip's own language-level concurrency.

## Errors

Diagnostics carry a source position (`line:column`). Parse errors come from the
parser as an `ErrorList`; the streaming checker raises `stream.TypeError`; the
evaluator raises `stream.RuntimeError`. The CLI prints them with source context
(the offending line and a caret); the embeddable API returns them as a
`chip.Error` carrying `Diagnostic`s. The streaming run fails fast at the first
error (effects already emitted are not rolled back); the batch tooling collects
all of them.

## Where to start reading

- The streaming engine: `internal/stream/stream.go` (the loop), then `eval.go`,
  `exec.go`, `check.go`, `env.go`.
- The parser seam: `internal/parser/items.go`.
- The language by example: `examples/`.
