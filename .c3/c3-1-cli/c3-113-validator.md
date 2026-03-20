---
id: c3-113
c3-version: 4
title: validator
type: component
category: feature
parent: c3-1
goal: Detect spec inconsistencies via rule-based validation
summary: 20 validation rules as SQL queries checking orphan events, unreachable states, dangling navigates, invalid annotations, state-region visibility, enum collisions, emit targets, etc.
uses: [ref-event-model, ref-sqlite-persistence]
---

# validator

## Goal

Detect spec inconsistencies via rule-based validation — each rule is a SQL query + result formatter.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft validate` / `sft check` commands.
**Depends on:** store (DB handle)

## Validation Rules

| Rule | Severity | Catches |
|------|----------|---------|
| missing-description | error | Empty description on screen/region |
| orphan-emit | error | `emit(X)` with no handler |
| unreachable-state | error | States not reachable from any `to_state` |
| duplicate-transition | error | Same (owner, event, from_state) twice |
| nesting-depth | warning | Regions nested 3+ levels |
| invalid-flow-ref | error | Flow steps referencing nonexistent entities |
| orphan-event | warning | Transitions handling undeclared events |
| dead-end | warning | Terminal states with no outgoing transitions |
| guard-ambiguity | warning | Same event+from_state 2+ times without guard |
| dangling-navigate | error | `navigate(X)` targeting unknown entities |
| ambiguous-region-name | warning | Same region name in multiple parents |
| unhandled-event | warning | Events emitted but never handled |
| undefined-data-type | error | Context/region_data field type not in data_types or enums |
| fixture-not-found | error | State references undefined fixture |
| orphan-fixture | warning | Fixture not referenced by any state |
| invalid-ambient-path | error | Ambient ref with bad source or query path |
| invalid-event-annotation | warning | Event annotation type not a builtin or defined type/enum |
| emit-missing-target | warning | `emit()` without `target:` specifier |
| invalid-state-region | error | State references non-child region |
| enum-data-collision | warning | Enum name collides with data type name |

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | DB handle | c3-102 |
| OUT (provides) | Findings | c3-117 |

### Extension Pattern

To add a validation rule: append a `rule` struct to the `var rules` slice. Each rule has:

- **`id`** — kebab-case name (e.g. `"orphan-emit"`)
- **`severity`** — `Error` or `Warning`
- **`query`** — SQL string run against the schema + views
- **`format`** — `func(rows *sql.Rows) ([]Finding, error)` that scans rows into `Finding` structs

Rules are pure SQL — no Go logic beyond row scanning. Use the `ownerCase` constant (assumes table alias `t`) or `ownerCaseAlias(alias)` helper to resolve polymorphic `owner_type`+`owner_id` to a human-readable entity name in your SQL.

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-event-model | Rules enforce event bubbling invariants |
| ref-sqlite-persistence | Rules are pure SQL queries |
