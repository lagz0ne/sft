---
id: c3-117
c3-version: 4
title: entrypoint
type: component
category: feature
parent: c3-1
goal: Command dispatch, flag parsing, and CLI UX
summary: main.go routes CLI verbs to handler functions, manages --json flag, provides usage text
uses: []
---

# entrypoint

## Goal

Command dispatch, flag parsing, and CLI UX — the single entry point that wires all components together.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `cmd/sft/main.go` — the binary entry point.
**Depends on:** All other components (imports every internal package)

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | All internal packages | c3-101, c3-102, c3-103, c3-110, c3-111, c3-112, c3-113, c3-114, c3-115, c3-116 |

## Key Aspects

### Command Routing
Switch on `os.Args[1]`: show, query, validate, import, export, diff, add, set, rename, rm, mv, reorder, impact, component, render, attach, detach, list, cat.

### Flag Parsing
Custom `flagVal`/`flagIndex` helpers. `--json` extracted globally. Per-command: `--in`, `--on`, `--from`, `--to`, `--action`, `--description`, `--props`, `--props-file`, `--as`, `--rm`.

### Aliases
q=query, check=validate, ls=list, comp=component, diag=diagram.

### Dispatch Mechanism

No flag library. `--json` is stripped globally from `os.Args` before dispatch and sets `format.JSONMode`. The remaining `os.Args[1]` is matched via a `switch`/`case` in `main()`, routing to `runX(s *store.Store, rest []string)` handler functions. Custom helpers:

| Helper | Signature | Purpose |
|--------|-----------|---------|
| `flagVal` | `(args, "--flag") string` | Extract value after a flag, or `""` |
| `flagIndex` | `(args, "--flag") int` | Index of flag in args, or `-1` |
| `need` | `(args, n, usage)` | Die if `len(args) < n` |
| `must` | `(err)` | Die if err non-nil |
| `die` | `(msg, args...)` | Fprintf to stderr + exit 1 |
| `ok` | `(msg, args...)` | Fprintf to stderr (success message) |

`format.JSONMode` branches output between structured JSON and human-readable tables/text.

### Extension Pattern

To add a CLI command: (1) add a `case "mycmd":` to the switch in `main()` → (2) write a `runMycmd(s *store.Store, rest []string)` function → (3) use `need`/`flagVal`/`must`/`die`/`ok` helpers for arg parsing and error handling → (4) use `format` package for output.
