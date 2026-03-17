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

## Scope

**Applies to:** c3-102 (store), c3-112 (query), c3-113 (validator)

**Does NOT apply to:** YAML format (ref-yaml-format governs that)

## Cited By

- c3-102 (store)
- c3-112 (query)
- c3-113 (validator)
