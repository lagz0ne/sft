---
id: c3-111
c3-version: 4
title: show
type: component
category: feature
parent: c3-1
goal: Assemble full spec tree from DB and render as text
summary: Load Spec tree from SQLite with enrichment (attachments, components); text rendering for sft show
uses: [ref-entity-resolution]
---

# show

## Goal

Assemble the full spec tree from the database and render it as human-readable text.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft show` + Spec tree used by diff, render, loader/export.
**Depends on:** store (DB queries, Enricher interface)
**Depended on by:** diff, render, loader/export

## Key Aspects

### Spec Tree
Nested: `Spec { App { Regions }, Screens[] { Regions[], Transitions[] }, Flows[] }`. Mirrors YAML structure.

### Enricher Interface
`show.Load(db, enricher)` — `Enricher` provides attachments and components. Store implements it; nil accepted (diff's in-memory store).

### Text Rendering
`show.Render(w, spec)` — indented text with component type, tags, events, transitions, flows.

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | DB handle, Enricher | c3-102 |
| OUT (provides) | Spec tree, Render | c3-114, c3-115, c3-110 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-entity-resolution | Enricher uses entity resolution for component/attachment lookup |
