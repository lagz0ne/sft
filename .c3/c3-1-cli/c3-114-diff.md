---
id: c3-114
c3-version: 4
title: diff
type: component
category: feature
parent: c3-1
goal: Compare two specs and produce a structured change list
summary: Recursive comparison of Spec trees producing +/-/~ changes for all entity types
uses: []
---

# diff

## Goal

Compare two specs and produce a structured change list with add (+), remove (-), and modify (~) operations.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft diff <file.yaml>` command.
**Depends on:** show (Spec types + Load), store (in-memory store for target), loader (import target YAML)

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | Spec tree, Load | c3-111 |
| IN (uses) | In-memory store | c3-102 |
| IN (uses) | Import target YAML | c3-110 |
| OUT (provides) | Change list | c3-117 |

## Key Aspects

### Comparison Strategy
Recursive name-based matching: screens → regions → events/transitions/tags. Transitions keyed by `(OnEvent, FromState)`.

### Workflow
1. Load current spec from DB via `show.Load`
2. Load target YAML into `store.OpenMemory()` → `show.Load`
3. `Compare(current, target)` → `[]Change`

### Change Types
`Change{Op, Entity, Name, In, Detail}` — entity types: screen, region, event, transition, flow, tag, app.
