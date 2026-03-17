---
id: c3-116
c3-version: 4
title: flow
type: component
category: feature
parent: c3-1
goal: Parse flow sequence strings into classified steps
summary: Tokenize arrow notation, classify each token as screen/region/event/back/action/activate via Resolver
uses: [ref-entity-resolution]
---

# flow

## Goal

Parse flow sequence strings (arrow notation) into classified, typed steps stored in `flow_steps`.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** Called by `store.InsertFlow` to parse sequences at insert time.
**Depends on:** model (FlowStep type), Resolver interface (implemented by store)

## Key Aspects

### Tokenization
Split on `→` (or `>` fallback). Extract `{data}` suffixes and `(H)` history markers. Handle `[Back]` and `activates` syntax.

### Classification
Via `Resolver` interface: `ResolveScreen` → `ResolveRegion` → `IsEvent`. Fallback: PascalCase → screen, else → action.

### Step Types
`screen`, `region`, `event`, `back`, `action`, `activate`

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | FlowStep type | c3-101 |
| IN (uses) | Resolver interface | c3-102 |
| OUT (provides) | ParseSequence | c3-102 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-entity-resolution | Same resolution order (screen → region → event) |
