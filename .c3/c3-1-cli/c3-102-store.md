---
id: c3-102
c3-version: 4
title: store
type: component
category: foundation
parent: c3-1
goal: Persist and query all entities in SQLite with CRUD, impact analysis, and migrations
summary: SQLite store with embedded schema, resolve helpers, cascade deletes, component/attachment support
uses: [ref-sqlite-persistence, ref-entity-resolution]
---

# store

## Goal

Persist and query all entities in SQLite with CRUD operations, impact analysis, and schema migrations.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** The single persistence layer — every command reads/writes through Store.
**Depends on:** model (types), flow (sequence parsing on insert)
**Depended on by:** loader, show, query, validator, render, entrypoint

## Key Aspects

### Schema
Embedded via `//go:embed schema.sql`. 10 tables + 4 views. Auto-migration for position column and scoped region uniqueness.

### Entity Resolution
`ResolveParent(name)`: apps → screens → regions. `ResolveRegion`: handles ambiguity with `--in` scoping. Cross-table collision checks on insert.

### Cascade Deletes
`collectDescendantRegions` recursively finds children. `deleteRegionIDs` cascades: events, tags, transitions, components, attachments, flow_steps.

### Impact Analysis
`ImpactScreen`/`ImpactRegion`: enumerates child regions, events, transitions, flows, tags, components, attachments, incoming `navigate()` references.

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | Domain types | c3-101 |
| IN (uses) | Flow sequence parser | c3-116 |
| OUT (provides) | Store CRUD, resolve, impact | c3-110, c3-111, c3-112, c3-113, c3-115, c3-117 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-sqlite-persistence | Governs schema management and data storage |
| ref-entity-resolution | Name-to-ID resolution strategy |
