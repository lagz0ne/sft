---
id: c3-113
c3-version: 4
title: validator
type: component
category: feature
parent: c3-1
goal: Detect spec inconsistencies via rule-based validation
summary: 10 validation rules as SQL queries checking orphan events, unreachable states, dangling navigates, etc.
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
| dangling-navigate | error | `navigate(X)` targeting unknown entities |
| ambiguous-region-name | warning | Same region name in multiple parents |
| unhandled-event | warning | Events emitted but never handled |

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | DB handle | c3-102 |
| OUT (provides) | Findings | c3-117 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-event-model | Rules enforce event bubbling invariants |
| ref-sqlite-persistence | Rules are pure SQL queries |
