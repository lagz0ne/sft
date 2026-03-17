---
id: c3-103
c3-version: 4
title: format
type: component
category: foundation
parent: c3-1
goal: Provide unified output rendering in JSON, table, and ANSI-colored text
summary: TTY detection, JSON/table formatters, status icons, impact/findings display
uses: []
---

# format

## Goal

Provide unified output rendering — JSON mode for machines, ANSI-colored tables for humans, with automatic TTY detection.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** Every command's output goes through format. Ensures consistent UX.
**Depended on by:** entrypoint (all run* functions use format)

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| OUT (provides) | JSON/table/status output | c3-117 |

## Key Aspects

### Output Modes
- `--json` → `JSONMode = true` → structured JSON via `format.JSON(v)`
- TTY detected → colored table output via `format.Table(queryName, rows)`
- Non-TTY → falls back to JSON

### Formatters
| Function | Purpose |
|----------|---------|
| `Table` | Column-aligned output with configurable widths, bool rendering |
| `Findings` | Validation results with error/warning icons + counts |
| `Impacts` / `ImpactInfo` | Tree-style impact display with box-drawing |
| `OK` / `Warn` / `Err` | Status line helpers (stderr) |
| `JSON` | Pretty-printed JSON (stdout) |
