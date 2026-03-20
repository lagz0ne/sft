---
id: c3-110
c3-version: 4
title: loader
type: component
category: feature
parent: c3-1
goal: Bridge between YAML spec files and the SQLite store
summary: Parse YAML into store inserts (Load) with enums, event annotations (dual-format), state-region visibility, validateTypeSuffix; serialize Spec tree back to YAML (Export)
uses: [ref-yaml-format]
---

# loader

## Goal

Bridge between YAML spec files and the SQLite store — import YAML into DB, export DB back to YAML with round-trip fidelity.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft import` and `sft export` commands.
**Depends on:** store (inserts), show (Spec tree for export), model (types)

## Key Aspects

### Import (Load)
Handles both single app (`mapping`) and multi-app (`sequence`, imports first only). Walks: App → enums → app-level regions → screens → screen regions → flows. Component bindings preserved. Event annotations parsed via `parseEventName` (dual-format: sequence or mapping). `ParseStateMachine` returns 5 values (transitions, states, stateFixtures, stateRegions). `validateTypeSuffix` rejects invalid `?[]` ordering on field types.

### Import Loop

`Load(s, path)` pipeline:

1. `os.ReadFile(path)` → `yaml.Unmarshal` into `yamlFile`
2. Decode `yamlFile.App` node — mapping (single app) or sequence (first app only, warns on extras)
3. `s.InsertApp` with name + description
4. Iterate `app.Data` → `json.Marshal` fields → `s.InsertDataType`
5. Iterate `app.Enums` → `json.Marshal` values → `s.InsertEnum`
6. Iterate `app.Context` → `validateTypeSuffix` → `s.InsertContextField` (owner=app)
7. Load `state_templates` if present
8. Iterate `app.Regions` → `insertRegion` (recursive: region → component/events/tags/ambient/data/children/transitions)
9. Iterate `app.Screens` → `s.InsertScreen` → tags → context fields → component binding → regions (via `insertRegion`) → `insertTransitions` (handles `states` vs `state_machine` dual-format, template extends, state fixtures, state-region visibility)
10. Iterate `app.Flows` → `s.InsertFlow` (name, description, onEvent, sequence)
11. Load `app.Fixtures` if present → `loadFixtures` (name, extends, data as JSON)

### Export Sequence

`Export(spec, w)`: builds `yamlApp` from `show.Spec` fields (DataTypes, Enums, Context, Regions, Screens, Flows, Fixtures) using `export*` helpers. Screens and regions re-emit state machines via `exportStateMachine`. Events choose sequence or mapping format based on annotation presence. Wraps in `{App: yamlApp}` → `yaml.NewEncoder(w)` at indent 2. Reverse of import.

### Test Patterns

- Helper `mustStore` → `store.OpenMemory`, `importYAML` → write temp file + `Load(s, path)`, `loadSpec` → `show.Load(s.DB, s)`
- Round-trip: import YAML → export to buffer → re-import into fresh store → compare spec fields (app name, screens, tags, regions, transitions, flows, app-level regions)
- Separate round-trip tests for components, data types, enums, context, state machines, fixtures

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | Store inserts | c3-102 |
| IN (uses) | Spec tree types | c3-111 |
| IN (uses) | Domain types | c3-101 |
| OUT (provides) | Load, Export | c3-117 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-yaml-format | Defines the YAML schema this component implements |
