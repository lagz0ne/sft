---
id: ref-sqlite-persistence
c3-version: 4
title: SQLite Persistence
goal: Govern how all data is stored, queried, and migrated
scope: [c3-102, c3-112, c3-113]
---

# SQLite Persistence

## Goal

Govern how all SFT data is stored, queried, and migrated — single-file SQLite database at `.sft/db`.

## Choice

Pure-Go SQLite via `modernc.org/sqlite`. Schema embedded with `//go:embed schema.sql`. All data in one file, no external dependencies.

## Why

- Zero runtime dependencies — single static binary
- `database/sql` interface — standard Go patterns
- Raw SQL queries via `sft query "SELECT ..."` — power users get direct DB access
- Views (`event_index`, `state_machines`, `tag_index`, `region_tree`) for denormalized reads

## How

| Guideline | Detail |
|-----------|--------|
| Schema embedded | `//go:embed schema.sql` in store package |
| Migrations inline | `db.Exec("ALTER TABLE ...")` in `Open()`, guarded by checks |
| Foreign keys enabled | `PRAGMA foreign_keys = ON` on every open |
| In-memory store | `store.OpenMemory()` for diff's target spec |
| Views for reads | Named queries hit views, not raw joins |

### Schema Overview

**Core entities** — the primary spec hierarchy:

| Table | Key columns | Notes |
|-------|-------------|-------|
| `apps` | name, description | Top-level container |
| `screens` | app_id, name, description | FK → apps |
| `regions` | app_id, parent_type, parent_id, name, position | Polymorphic parent (app/screen/region), recursive nesting |
| `tags` | entity_type, entity_id, tag | Polymorphic (screen/region) |
| `events` | region_id, name, annotation | Emitted by regions |
| `transitions` | owner_type, owner_id, on_event, from_state, to_state, action | State machine edges — **polymorphic owner** |
| `flows` | app_id, name, on_event, sequence | User journey definitions |
| `flow_steps` | flow_id, position, raw, type, name, history, data | Parsed steps within a flow |

**Components & attachments:**

| Table | Key columns | Notes |
|-------|-------------|-------|
| `components` | entity_type, entity_id, component, props, on_actions, visible | One component per entity (UNIQUE entity_type+entity_id) |
| `attachments` | entity, name, content (BLOB) | Binary blobs keyed by entity+name |

**Data model (Phase 2–5):**

| Table | Phase | Key columns | Notes |
|-------|-------|-------------|-------|
| `data_types` | 2 | app_id, name, fields (JSON) | Named type definitions |
| `contexts` | 2 | owner_type, owner_id, field_name, field_type | Scoped data context (app/screen) |
| `ambient_refs` | 2 | region_id, local_name, source, query | Derived data bindings |
| `region_data` | 2 | region_id, field_name, field_type | Region-local fields |
| `enums` | 5 | app_id, name, values (JSON array) | Enumerated types |
| `state_templates` | 4 | app_id, name, definition | Reusable state machine templates |
| `fixtures` | 3 | app_id, name, extends, data (JSON) | Test data fixtures, optional inheritance |
| `state_fixtures` | 3 | owner_type, owner_id, state_name, fixture_name | Bind fixture to state — **polymorphic owner** |
| `state_regions` | 5 | owner_type, owner_id, state_name, region_name | State-driven region visibility — **polymorphic owner** |

**Polymorphic owner pattern:** `transitions`, `state_fixtures`, and `state_regions` all use `(owner_type, owner_id)` with CHECK constraint `IN ('app','screen','region')` to attach to any entity level. `contexts` uses the same pattern scoped to `('app','screen')`.

**Views** — denormalized reads, used by named queries:

| View | Purpose |
|------|---------|
| `event_index` | Joins events → regions → transitions to show emit/handle pairs |
| `state_machines` | Resolves transition owner names via CASE subqueries |
| `tag_index` | Resolves tag entity names |
| `region_tree` | Full region hierarchy with parent name, event_count, has_states flag |

**Indexes:**

| Index | Columns |
|-------|---------|
| `idx_events_region` | events(region_id) |
| `idx_transitions_owner` | transitions(owner_type, owner_id) |
| `idx_transitions_on_event` | transitions(on_event) |
| `idx_tags_entity` | tags(entity_type, entity_id) |
| `idx_flow_steps_type_name` | flow_steps(type, name) |

### Migration Pattern

Migrations run inline in `Open()` immediately after `db.Exec(schemaSQL)`. They rely on SQLite's idempotent behavior: `ALTER TABLE ... ADD COLUMN` silently fails if the column already exists (error is discarded). This means migrations are safe to re-run on every open.

Current migrations in `Open()`:

1. **`position` on regions** — `ALTER TABLE regions ADD COLUMN position INTEGER NOT NULL DEFAULT 0`
2. **`annotation` on events** — `ALTER TABLE events ADD COLUMN annotation TEXT`
3. **`migrateRegionScope()`** — Detects old `UNIQUE(name)` index on regions (single-column `sqlite_autoindex_regions_1`), then rebuilds the table via create-new/insert/drop/rename to get `UNIQUE(parent_type, parent_id, name)`. Runs in a transaction, no-ops if already migrated.

## Scope

**Applies to:** c3-102 (store), c3-112 (query), c3-113 (validator)

**Does NOT apply to:** YAML format (ref-yaml-format governs that)

## Cited By

- c3-102 (store)
- c3-112 (query)
- c3-113 (validator)
