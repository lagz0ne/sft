---
id: c3-110
c3-version: 4
title: loader
type: component
category: feature
parent: c3-1
goal: Bridge between YAML spec files and the SQLite store
summary: Parse YAML into store inserts (Load), serialize Spec tree back to YAML (Export)
uses: [ref-yaml-format]
---

# loader

## Goal

Bridge between YAML spec files and the SQLite store — import YAML into DB, export DB back to YAML with round-trip fidelity.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft import` and `sft export` commands.
**Depends on:** store (inserts), show (Spec tree for export), model (types)

## Key Aspects

### Import (Load)
Handles both single app (`mapping`) and multi-app (`sequence`, imports first only). Walks: App → app-level regions → screens → screen regions → flows. Component bindings preserved.

### Export
Takes `show.Spec` tree → yaml types → `yaml.NewEncoder` at indent 2.

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | Store inserts | c3-102 |
| IN (uses) | Spec tree types | c3-111 |
| IN (uses) | Domain types | c3-101 |
| OUT (provides) | Load, Export | c3-117 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-yaml-format | Defines the YAML schema this component implements |
