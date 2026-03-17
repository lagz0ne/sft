# SFT

Go CLI for UI behavioral specs — screens, regions, events, state machines, flows in SQLite.

## Build & Test

```bash
go build ./cmd/sft               # build
go test ./...                     # test all
go test ./internal/store          # test store only
```

Cross-platform binaries: `bash scripts/build.sh`
npm packages: `bash scripts/build-npm.sh`

## Architecture

C3 docs in `.c3/`. Use `/c3` for architecture questions, changes, audits, impact analysis.

**1 container** (cli) · **11 components** · **4 refs**

| Component | Package | Role |
|-----------|---------|------|
| model (c3-101) | `internal/model` | Domain types — no logic |
| store (c3-102) | `internal/store` | SQLite CRUD, resolve, impact, migrations |
| format (c3-103) | `internal/format` | JSON/table/ANSI output |
| loader (c3-110) | `internal/loader` | YAML import/export |
| show (c3-111) | `internal/show` | Spec tree assembly + text render |
| query (c3-112) | `internal/query` | Named queries + raw SQL |
| validator (c3-113) | `internal/validator` | Rule-based spec validation |
| diff (c3-114) | `internal/diff` | Spec comparison |
| render (c3-115) | `internal/render` | json-render output |
| flow (c3-116) | `internal/flow` | Flow sequence parsing |
| entrypoint (c3-117) | `cmd/sft` | Command dispatch |

**Refs:** sqlite-persistence, yaml-format, entity-resolution, event-model

File lookup: `c3x lookup <file-or-glob>` maps files to components + refs.
