---
id: c3-112
c3-version: 4
title: query
type: component
category: feature
parent: c3-1
goal: Provide ad-hoc spec inspection via named queries and raw SQL
summary: Named queries (screens, events, flows, tags, regions) against SQLite views + raw SELECT passthrough
uses: [ref-sqlite-persistence]
---

# query

## Goal

Provide ad-hoc spec inspection via named queries and raw SQL against the spec database.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** `sft query <type>` and `sft query "SELECT ..."` commands.
**Depends on:** store (DB handle)

## Key Aspects

### Named Queries
| Query | Source |
|-------|--------|
| `screens` | `screens` table |
| `events` | `event_index` view |
| `flows` | `flows` table |
| `tags` | `tag_index` view |
| `regions` | `region_tree` view |
| `states <name>` | `state_machines` view |
| `steps <flow>` | `flow_steps` + `flows` join |

### Raw SQL
Any input starting with `SELECT` is passed through directly.

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| IN (uses) | DB handle | c3-102 |
| OUT (provides) | Query results | c3-117 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-sqlite-persistence | Queries run against SQLite views defined in schema |
