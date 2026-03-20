---
id: c3-111
c3-version: 4
title: show
type: component
category: feature
parent: c3-1
goal: Assemble full spec tree from DB and render as text
summary: Load Spec tree from SQLite with enrichment (attachments, components); loads Enums, StateRegions on Screen/Region, event annotations via loadEvents; text rendering for sft show
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

### Load() Assembly

`Load(db, enricher)` builds the full `Spec` tree in this order:

1. **App** — query single row from `apps`; populate `appID`, name, description.
2. **DataTypes** — `loadDataTypes(db, appID)` — JSON-stored field maps keyed by type name.
3. **Enums** — `loadEnums(db, appID)` — JSON-stored string slices keyed by enum name.
4. **Context** — `loadContext(db, "app", appID)` — field-name→type pairs.
5. **App Regions** — `loadRegions(db, "app", appID, enricher)` — recursive (see below).
6. **App Transitions** — `loadTransitions(db, "app", appID)`.
7. **Screens** — iterate `SELECT id, name, description FROM screens ORDER BY id`:
   - `loadTags` — string slice from `tags` table.
   - `loadContext` — field-name→type from `contexts` table.
   - `loadRegions` — recursive, same as app.
   - `loadTransitions` — on_event / from / to / action rows.
   - `loadStateFixtures` — state-name→fixture-name map.
   - `loadStateRegions` — state-name→region-name[] map.
   - **Enricher** (if non-nil): `AttachmentsFor(name)`, `ComponentFor("screen", id)`, `ComponentInfoFor("screen", id)` for props/on/visible.
8. **Flows** — `SELECT name, description, on_event, sequence FROM flows ORDER BY id`.
9. **Fixtures** — `loadFixtures(db, appID)` — name, extends, JSON data.

`loadRegions` recurses: for each region it loads tags, events (with annotation reconstruction), ambient refs, region_data, child regions, transitions, state fixtures, state regions, and enricher data. Recursion key: `loadRegions(db, "region", id, al)`.

### Enricher Contract

Interface that decouples `show` from `store` for hydration concerns. Three methods:

| Method | Signature | Returns |
|--------|-----------|---------|
| `AttachmentsFor` | `(entity string) []string` | Attachment file names for a named entity |
| `ComponentFor` | `(entityType string, entityID int64) string` | Component type string, or `""` |
| `ComponentInfoFor` | `(entityType string, entityID int64) *store.ComponentInfo` | Full detail: Component, Props, OnActions, Visible — or `nil` |

`store.Store` implements `Enricher`. Passing `nil` is valid (used by diff's in-memory store) — all enrichment is skipped via `if al != nil` guards.

### Spec Tree Shape

```
Spec
├── App
│   ├── Name, Description
│   ├── DataTypes        map[name]map[field]type
│   ├── Enums            map[name][]value
│   ├── Context          map[field]type
│   ├── Regions[]        (recursive)
│   └── Transitions[]
├── Screens[]
│   ├── Name, Description, Tags[], Context
│   ├── Component, ComponentProps, ComponentOn, ComponentVis
│   ├── Regions[]        (recursive)
│   │   ├── Name, Description, Tags[]
│   │   ├── Component, ComponentProps, ComponentOn, ComponentVis
│   │   ├── Events[]     (name or name(annotation))
│   │   ├── Ambient      map[name]"data(source, query)"
│   │   ├── RegionData   map[field]type
│   │   ├── Regions[]    ← recurse
│   │   ├── Transitions[]
│   │   ├── StateFixtures  map[state]fixture
│   │   ├── StateRegions   map[state][]region
│   │   └── Attachments[]
│   ├── Transitions[]
│   │   └── OnEvent, FromState, ToState, Action
│   ├── StateFixtures    map[state]fixture
│   ├── StateRegions     map[state][]region
│   └── Attachments[]
├── Flows[]
│   └── Name, Description, OnEvent, Sequence
└── Fixtures[]
    └── Name, Extends, Data (any)
```

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
