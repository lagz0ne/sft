# SFT

Go CLI for UI behavioral specs — screens, regions, events, state machines, flows in SQLite.

## Build & Test

```bash
go build ./cmd/sft               # build
go test ./...                     # test all
```

Tests use in-memory SQLite with `t.Cleanup()`. Example specs in `examples/*.sft.yaml`.

## `/c3` is the authority

**All work starts with `/c3`.** It is the canonical source for how things work, why they work that way, component boundaries, dependencies, schema, extension patterns, refs, and ADRs. No guessing — `/c3` first, then act.

**2 containers** (cli, view) · **13 components** · **4 refs**

### Data Flow

```
YAML ──import──→ loader ──→ store (SQLite .sft/db)
                               ↕
                  ┌────────────┼────────────┐
                  ↓            ↓            ↓
                show         query      validator
                ↕  ↘           ↓
              diff  render   format ──→ stdout
```

### Dependency Direction

```
entrypoint → store, loader, show, query, validator, diff, render, format, diagram, view
loader → store, show, model       diff → show, store, loader
show → store                      render → show, store
query → store                     validator → store (SQL only)
store → model, flow               flow → model
```

`model` and `format` are leaves.

## Domain Vocabulary

- **Regions** nest recursively (region → region → screen → app). Scoped naming: unique per parent, not globally
- **Events** belong to regions, optionally typed: `select_email(email)` → annotation
- **Transitions** fire on events: `{on_event, from_state?, to_state?, action?}`. Actions: `navigate(screen)`, `emit(event, target:[...])`, or freeform
- **State machines** on screens/regions: first state = initial. `.` = self-transition. Null to_state = terminal
- **Flows** are token sequences: `Screen → Region → event → [Back] → Screen(H)` — parsed into typed FlowStep rows
- **Fixtures** bind sample data to states. Support `extends:` inheritance
- **State regions** — which child regions are visible per state
- **Ambient refs** — `data(source, .query)` links region to screen context
- **Type system** — scalars (string, number, boolean, date, datetime) + data type refs + `[]` arrays + `?` optionals. Enums standalone
- **Components** — json-render type + props bound to regions via `sft component`. Bridge between wireframe preview and production renderer
- **Tags** — layout only: position (`sidebar`, `header`, `split:wide`), composition (`mobile:bottomnav`), visual (`elevated`). Tailwind-like colon syntax
- **Component sets** — named rendering implementations (wireframe, styled, compact). Switchable in playground dock
