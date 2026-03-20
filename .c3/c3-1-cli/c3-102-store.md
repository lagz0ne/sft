---
id: c3-102
c3-version: 4
title: store
type: component
category: foundation
parent: c3-1
goal: Persist and query all entities in SQLite with CRUD, impact analysis, and migrations
summary: SQLite store with embedded schema, resolve helpers, cascade deletes, component/attachment support, enums + state_regions tables
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
Embedded via `//go:embed schema.sql`. 19 tables + 4 views. Auto-migration for position column, annotation column on events, and scoped region uniqueness. InsertEnum, InsertStateRegion for Phase 5 entities.

### Entity Resolution
`ResolveParent(name)`: apps → screens → regions. `ResolveRegion`: handles ambiguity with `--in` scoping. Cross-table collision checks on insert.

### Cascade Deletes
`collectDescendantRegions` recursively finds children. `deleteRegionIDs` cascades: events, tags, transitions, components, attachments, flow_steps.

### Impact Analysis
`ImpactScreen`/`ImpactRegion`: enumerates child regions, events, transitions, flows, tags, components, attachments, incoming `navigate()` references.

### Schema Structure

**Core entities (Phase 1):**

| Table | Key Columns | Relationships |
|-------|-------------|---------------|
| apps | id, name, description | Root entity |
| screens | id, app_id, name, description | FK apps(id) |
| regions | id, app_id, parent_type, parent_id, name, description, position | FK apps(id); polymorphic parent (app/screen/region); UNIQUE(parent_type, parent_id, name) |
| tags | id, entity_type, entity_id, tag | Polymorphic owner (screen/region) |
| events | id, region_id, name, annotation | FK regions(id) |
| transitions | id, owner_type, owner_id, on_event, from_state, to_state, action | Polymorphic owner (app/screen/region) |
| flows | id, app_id, name, description, on_event, sequence | FK apps(id) |
| flow_steps | id, flow_id, position, raw, type, name, history, data | FK flows(id); type CHECK (screen/region/event/back/action/activate) |
| components | id, entity_type, entity_id, component, props, on_actions, visible | Polymorphic owner (app/screen/region); UNIQUE(entity_type, entity_id) |
| attachments | id, entity, name, content (BLOB) | Entity by name string; UNIQUE(entity, name) |

**Data model (Phase 2):**

| Table | Key Columns | Relationships |
|-------|-------------|---------------|
| data_types | id, app_id, name, fields | FK apps(id); UNIQUE(app_id, name) |
| contexts | id, owner_type, owner_id, field_name, field_type | Polymorphic owner (app/screen); UNIQUE(owner_type, owner_id, field_name) |
| ambient_refs | id, region_id, local_name, source, query | FK regions(id); UNIQUE(region_id, local_name) |
| region_data | id, region_id, field_name, field_type | FK regions(id); UNIQUE(region_id, field_name) |

**Fixtures (Phase 3):**

| Table | Key Columns | Relationships |
|-------|-------------|---------------|
| fixtures | id, app_id, name, extends, data | FK apps(id); UNIQUE(app_id, name) |
| state_fixtures | id, owner_type, owner_id, state_name, fixture_name | Polymorphic owner (app/screen/region); UNIQUE(owner_type, owner_id, state_name) |

**State templates (Phase 4):**

| Table | Key Columns | Relationships |
|-------|-------------|---------------|
| state_templates | id, app_id, name, definition | FK apps(id); UNIQUE(app_id, name) |

**Enums + state-region visibility (Phase 5):**

| Table | Key Columns | Relationships |
|-------|-------------|---------------|
| enums | id, app_id, name, "values" | FK apps(id); UNIQUE(app_id, name) |
| state_regions | id, owner_type, owner_id, state_name, region_name | Polymorphic owner (app/screen/region); UNIQUE(owner_type, owner_id, state_name, region_name) |

**Views (4):**

| View | Purpose |
|------|---------|
| event_index | Events joined with emitting region and handling transitions |
| state_machines | Transitions with resolved owner names, ordered by owner |
| tag_index | Tags with resolved entity names |
| region_tree | Regions with resolved parent names, event count, has_states flag |

**Polymorphic owner pattern:** Tables `transitions`, `state_regions`, `state_fixtures`, `components`, `contexts`, `tags` use `(owner_type, owner_id)` or `(entity_type, entity_id)` columns with CHECK constraints to reference apps, screens, or regions without foreign keys. Resolution goes through `ResolveParent`/`ResolveOwner`.

### CRUD Pattern

Insert methods follow a uniform pattern: collision check (optional) → `db.Exec(INSERT)` → `res.LastInsertId()` → assign to struct `.ID`.

Reference: `InsertScreen`:
```go
func (s *Store) InsertScreen(sc *model.Screen) error {
    // Cross-table collision check
    var regionCount int
    s.DB.QueryRow("SELECT COUNT(*) FROM regions WHERE name = ?", sc.Name).Scan(&regionCount)
    if regionCount > 0 {
        return fmt.Errorf("name %q already used by a region", sc.Name)
    }
    res, err := s.DB.Exec("INSERT INTO screens (app_id, name, description) VALUES (?, ?, ?)",
        sc.AppID, sc.Name, sc.Description)
    if err != nil {
        return err
    }
    sc.ID, _ = res.LastInsertId()
    return nil
}
```

Some inserts have side effects: `InsertFlow` parses the sequence string and auto-inserts `flow_steps`. `SetComponent` uses `INSERT ... ON CONFLICT DO UPDATE` for upsert semantics.

### In-Memory Store

`OpenMemory()` creates an in-memory SQLite store for diff comparison and tests. Uses shared-cache mode with an atomic counter (`memSeq`) to ensure unique DSN names across concurrent stores: `file:sft_mem_<N>?mode=memory&cache=shared`. Same schema, no migrations (no ALTER TABLE calls), no WAL/busy_timeout pragmas.

### Migration Pattern

Idempotent `ALTER TABLE` calls in `Open()` after `db.Exec(schema)`. SQLite silently ignores duplicate column adds. Existing migrations:
- `ALTER TABLE regions ADD COLUMN position` — region ordering
- `ALTER TABLE events ADD COLUMN annotation` — Phase 5 event annotations
- `migrateRegionScope()` — converts old global `UNIQUE(name)` on regions to scoped `UNIQUE(parent_type, parent_id, name)` via table rebuild in a transaction

Migrations run on every `Open()` and are no-ops on current-schema databases.

### Test Patterns

`OpenMemory()` with `t.Cleanup(s.Close)`. Helper `mustOpen(t)` + `seedApp(t, s)` for common setup. Tests construct model structs directly, call CRUD methods, and assert on ID assignment, unique constraint errors, and query results. No fixtures or test databases — each test gets a fresh in-memory store.

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
