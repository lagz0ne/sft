---
id: c3-115
c3-version: 4
title: render
type: component
category: feature
parent: c3-1
goal: Generate json-render element tree from spec with component hydration
summary: FromSFT builds skeleton (screens→Card, regions→Stack), Hydrate overrides with stored component bindings
uses: []
---

# render

## Goal

Generate a json-render-compatible element tree from the spec, then hydrate with stored component bindings.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft render` command.
**Depends on:** show (Spec tree), store (component lookup via callback)

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | Spec tree | c3-111 |
| IN (uses) | Component lookup callback | c3-102 |
| OUT (provides) | json-render Spec | c3-117 |

## Key Aspects

### Two-Phase Pipeline
1. **FromSFT (skeleton):** Screens → `Card`, regions → `Stack`, children wired by hierarchy
2. **Hydrate:** Walks elements, calls `getComp(name)` callback to override type/props/on/visible

### Output
```json
{ "root": "AppName", "elements": { "Name": { "type", "props", "children", "on", "visible" } } }
```
