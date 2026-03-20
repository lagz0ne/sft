---
id: ref-event-model
c3-version: 4
title: Event Bubbling Model
goal: Define how events propagate through the entity hierarchy
scope: [c3-101, c3-113]
---

# Event Bubbling Model

## Goal

Define how events propagate through the SFT entity hierarchy — the core behavioral model.

## Choice

Hierarchical event bubbling: events declared on regions, bubble up Sub-Region → Region → Screen → App. Handled = consumed. `emit(event-name)` transforms and re-sends to parent.

## Why

- Maps directly to UI event propagation (DOM-like)
- Enables layered state machines — each level handles what it cares about
- `emit()` allows child to handle locally AND notify parent with a domain-level event name

## How

| Concept | Mechanism |
|---------|-----------|
| Event declaration | `events:` list on Region |
| Event annotations | `name(type)` — payload type annotation on events (validated against data types + enums) |
| Bubbling | Unhandled events propagate up: Sub-Region → Region → Screen → App |
| Handling | `transitions.on_event` matches; handled = consumed |
| Emit | `action: emit(event-name)` — handle locally, send named event to parent |
| Emit targets | `emit(event-name, target:[...])` — explicit target routing; validated by `emit-missing-target` rule |
| Ambient events | Keyboard shortcuts appear in `states` without Region declaration |
| Validation | `orphan-emit`, `orphan-event`, `unhandled-event`, `invalid-event-annotation`, `emit-missing-target` rules enforce invariants |

## Scope

**Applies to:** c3-101 (model types encode hierarchy), c3-113 (validator enforces invariants)

**Does NOT apply to:** Store CRUD (no runtime event dispatch), render (static output)

## Cited By

- c3-101 (model)
- c3-113 (validator)
