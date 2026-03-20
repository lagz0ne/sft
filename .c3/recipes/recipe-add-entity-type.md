---
id: recipe-add-entity-type
type: recipe
title: Add Entity Type
description: Cross-cutting pipeline for adding a new entity type spanning model → store → loader → show → diff → query → validator
sources: [c3-101, c3-102, c3-110, c3-111, c3-114, c3-112, c3-113, c3-103, c3-117]
---

# Add Entity Type

## Goal

Walk through every component that must change when introducing a new entity type to SFT. Each step references the component that owns it — load its doc via `/c3` for implementation details.

## Pipeline

### 1. model (c3-101)

Add a flat struct. No methods, no logic. Fields use `int64` for IDs, `string` for names/descriptions, nullable fields as appropriate.

### 2. store (c3-102)

- Add `CREATE TABLE` to schema (UNIQUE, FK, CHECK constraints as needed)
- Add `Insert*()` method — see CRUD pattern in c3-102
- Add `Resolve*()` if the entity is named and needs name→ID lookup
- Add `Delete*()` with cascade if it owns children
- Add migration in `Open()` if modifying existing tables — see migration pattern in c3-102
- Cross-table collision check if the name could conflict with screens/regions

### 3. loader (c3-110)

- Add YAML struct fields to the appropriate `yaml*` type
- Wire into `Load()` import loop at the right point in the iteration — see import loop in c3-110
- Wire into `Export()` to serialize back to YAML — see export sequence in c3-110
- Ensure round-trip fidelity: import → export → import must be identical

### 4. show (c3-111)

- Add fields to the appropriate Spec tree type (Screen, Region, App, or new top-level) — see Spec tree shape in c3-111
- Add query + row iteration in `Load()` assembly — see Load() assembly in c3-111
- If the entity needs enrichment (attachments, components), extend the Enricher interface

### 5. diff (c3-114)

- Add comparison logic for the new entity type
- Produce Change entries with appropriate Op (+/-/~), Entity type, Name, Detail

### 6. query (c3-112) — if queryable

- Add named query to queries map — see extension pattern in c3-112
- Add format table schema in format (c3-103) if columns are new

### 7. validator (c3-113) — if validateable

- Add SQL rule(s) to rules slice — see extension pattern in c3-113
- Consider: orphan references, missing required fields, invalid cross-entity references

### 8. entrypoint (c3-117) — if CLI-exposed

- Add `sft add <entity>` case — see extension pattern in c3-117
- Add `sft rm <entity>` case
- Add other CRUD commands as needed (set, rename, mv)

## Checklist

- [ ] model: struct added
- [ ] store: schema + CRUD + migration (if needed)
- [ ] loader: import + export + round-trip test
- [ ] show: Spec tree + Load() query
- [ ] diff: comparison logic
- [ ] query: named query (if applicable)
- [ ] validator: rules (if applicable)
- [ ] entrypoint: CLI commands (if applicable)
- [ ] tests pass: `go test ./...`
