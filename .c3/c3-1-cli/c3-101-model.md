---
id: c3-101
c3-version: 4
title: model
type: component
category: foundation
parent: c3-1
goal: Define the domain vocabulary as Go structs shared by all packages
summary: Pure data types — App, Screen, Region, Tag, Event, Transition, Flow, FlowStep — with no behavior
uses: [ref-event-model]
---

# model

## Goal

Define the domain vocabulary as Go structs shared by all packages. No methods, no logic — just the shape of data.

## Container Connection

**Parent:** c3-1 (cli)
**Contributes:** Shared type definitions that every other component imports.
**Depended on by:** store, loader, flow, show

## Key Entities

| Type | Key Fields | Role |
|------|-----------|------|
| App | name, description | Top-level boundary (1 per project) |
| Screen | name, description, app_id | Viewport grouping of regions |
| Region | name, description, parent_type, parent_id | Hierarchical building block |
| Tag | entity_type, entity_id, tag | Bracket annotations on screens/regions |
| Event | region_id, name | Declared on emitting region |
| Transition | owner_type, owner_id, on_event, from/to_state, action | State machine rule |
| Flow | name, description, on_event, sequence | Named user journey |
| FlowStep | flow_id, position, raw, type, name, history, data | Parsed flow token |

## Dependencies

| Direction | What | From/To |
|-----------|------|---------|
| OUT (provides) | Domain types (App, Screen, Region, etc.) | c3-102, c3-110, c3-116, c3-111 |

## Related Refs

| Ref | Relevance |
|-----|-----------|
| ref-event-model | Model types encode the event bubbling hierarchy |
