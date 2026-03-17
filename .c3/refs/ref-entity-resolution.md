---
id: ref-entity-resolution
c3-version: 4
title: Entity Resolution
goal: Standardize how names resolve to IDs across entity types
scope: [c3-102, c3-111, c3-116]
---

# Entity Resolution

## Goal

Standardize how user-provided names resolve to database IDs, with consistent priority order and disambiguation.

## Choice

Name-based resolution with fixed priority: apps → screens → regions. Ambiguous region names resolved via `--in <parent>` scoping.

## Why

- Users refer to entities by name, not ID
- Priority order prevents ambiguity: screen names are globally unique, region names scoped to parent
- `--in` flag provides escape hatch for same-name regions across different parents

## How

| Method | Priority | Used By |
|--------|----------|---------|
| `ResolveParent(name)` | apps → screens → regions | store (add region, transition, component), flow |
| `ResolveScreen(name)` | screens only | store, flow, show |
| `ResolveRegion(name)` | regions, error if ambiguous | store, flow |
| `ResolveRegionIn(name, parent)` | scoped to parent | store (--in flag) |
| `ResolveScreenOrRegion(name)` | screens → regions | store (tags, components) |

### Cross-Table Collision
`InsertScreen` checks region names. `InsertRegion` checks screen names. Prevents name collision across entity types.

## Scope

**Applies to:** c3-102 (store), c3-111 (show/Enricher), c3-116 (flow/Resolver)

**Does NOT apply to:** query (uses raw SQL), validator (uses SQL joins)

## Cited By

- c3-102 (store)
- c3-111 (show)
- c3-116 (flow)
