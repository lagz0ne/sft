# Remove Flows — Focus on Screen State Machines

**Date**: 2026-03-30
**Status**: Approved
**Decision**: Full removal of the flow subsystem. State machines on screens/regions are the sole behavioral model. Flows will be rebuilt from scratch in a future iteration.

## Motivation

Flows describe user journeys as linear token sequences (`inbox → thread_view → [Back] → inbox(H)`). State machines on screens/regions already capture the same behavioral information — transitions, navigation, event handling — but as a declarative graph rather than a rigid sequence. Maintaining both is redundant. The flow abstraction will be redesigned later; for now, remove it entirely.

## Scope

### Removed

| Layer | Location | Artifact |
|-------|----------|----------|
| Model | `internal/model/model.go` | `Flow`, `FlowStep` structs |
| Parser | `internal/flow/` | Entire package (`parse.go`, `parse_test.go`) |
| Schema | `internal/store/schema.sql` | `flows` table, `flow_steps` table, `idx_flow_steps_type_name` index |
| Store | `internal/store/store.go` | `InsertFlow`, `InsertFlowStep`, `DeleteFlow`, `RenameFlow` methods |
| Loader | `internal/loader/loader.go` | `yamlFlow` struct, flow import logic |
| Show | `internal/show/show.go` | `Flow`, `FlowStep` show types, `loadFlowSteps()`, flow loading in `Load()` |
| Query | `internal/query/query.go` | `"flows"` named query, `Steps()` function |
| Diagram | `internal/diagram/diagram.go` | `Flow()` Mermaid generator |
| CLI | `cmd/sft/main.go` | `add flow`, `query flows`, `query steps`, `rename flow`, `rm flow`, `diagram flow` |
| Web route | `web/apps/web/src/routes/flows.$name.tsx` | Flow viewer page |
| Web component | `web/apps/web/src/components/flow-step-strip.tsx` | Step navigation strip |
| Web nav | sidebar/nav references to flows | Links and menu items |
| Examples | `examples/*.sft.yaml` | `flows:` sections |
| Tests | various `_test.go` | Flow-related test cases |

### Unchanged

- Screen + Region model
- State machines (transitions, state_fixtures, state_regions)
- Events, contexts, fixtures, components, tags
- Diagram generators: `States()`, `Nav()`
- View server (minus flow-specific handling)
- Validator, diff, format, render (no flow dependencies)

## DB Migration

Add to schema initialization:

```sql
DROP TABLE IF EXISTS flow_steps;
DROP TABLE IF EXISTS flows;
```

This cleans up existing `.sft/db` files on next use. No data loss risk — flows are derived from YAML specs and can be re-imported if needed in the future.

## Verification

- `go build ./cmd/sft` compiles cleanly
- `go test ./...` passes — all flow tests removed, no other tests depend on flows
- `go vet ./...` clean
- Example YAML files load without errors
- View/playground renders without flow routes
- No dead imports or unused variables
