---
id: ref-yaml-format
c3-version: 4
title: YAML Spec Format
goal: Standardize the YAML schema for import/export round-trips
scope: [c3-110]
---

# YAML Spec Format

## Goal

Standardize the YAML schema for SFT spec files — the contract between teams.

## Choice

YAML via `gopkg.in/yaml.v3`. Schema: `app:` top-level key containing name, description, regions, screens, flows. See README.md `## YAML Schema` for full spec.

## Why

- Human-readable, writable by hand
- Natural fit for hierarchical spec structure
- `yaml.v3` supports node-level parsing (needed for single-app vs multi-app detection)

## How

| Guideline | Detail |
|-----------|--------|
| Single vs multi-app | `app:` as mapping = single, as sequence = list (first imported, rest warned) |
| Enums | `enums:` mapping at app level — name → list of values |
| Optional fields | `?` suffix marks optional, `[]` suffix marks array, `[]?` valid, `?[]` rejected |
| Event annotations | `events:` as sequence (`[name(type)]`) or mapping (`name(type):`) — annotation = payload type |
| State regions | `regions:` list inside state_machine states — controls child region visibility per state |
| Emit targets | `emit(event-name, target:[...])` — explicit target routing for emitted events |
| Component bindings | `component`, `props`, `on_actions`, `visible` fields on screens/regions |
| Round-trip fidelity | Export uses same yaml types as import — `import → export` must be lossless |
| Indent | `yaml.NewEncoder` with `SetIndent(2)` |

## Scope

**Applies to:** c3-110 (loader)

**Does NOT apply to:** SQLite schema, CLI output format

## Cited By

- c3-110 (loader)
