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
q=query, check=validate, ls=list, comp=component.
