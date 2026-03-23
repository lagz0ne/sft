# SFT View — Design Spec

**Date:** 2026-03-20
**Status:** Draft

## Problem

SFT captures behavioral specs (screens, regions, events, state machines, flows) via CLI and YAML. The spec data exists in SQLite but has no shared surface for collaborative review. Teams need to look at the spec together — understand it, discuss it, iterate on it, attach visual designs, and capture decisions.

## Users

PMs and Designers working together in review sessions. Developers inherit the same view as a reference.

### Group JTBD

"We need to look at the spec together, understand it, discuss it, change it, and capture where we landed."

The tool does:
- Show the spec — screens, areas, looks, journeys — readable by anyone
- Let anyone navigate — "show me Inbox", "walk me through the login"
- Show what a change affects
- Capture snapshots

The tool does NOT:
- Tell you what's missing or what to do next
- Separate workspaces per role
- Have opinions about what you should do

### Collaboration Model

Shared data, not shared cursor. Everyone opens the same URL and sees the same spec. State machine simulation is local per browser. Changes via CLI are reflected in all open views via NATS.

## Language

The view translates SFT vocabulary for its audience:

| SFT internal | View renders as |
|---|---|
| regions | "areas" |
| states | "looks" |
| flows | "user journeys" |
| events | label from event name (`select_email` → "Select email") |
| transitions | "what happens next" |

## Design

### 1. Maturity Model

Each entity's maturity is automatically derived — never manually set. A property, not a progress tracker.

**Structured** → **Validated** → **Skinned**

- **Structured:** Has regions, events, and/or states defined.
- **Validated:** Zero validation errors scoped to this entity (not transitive).
- **Skinned:** Mockups attached for every state.

Maturity can regress — adding a new state drops "skinned" back to "validated."

### 2. Shared Surface

One surface. Two entry points, plus app-level regions.

**Screens tab** — every screen, with region count and state count. Click to expand.

**Flows tab** — every named flow with screen sequence. Click to walk through.

**App-level regions** — global regions (overlays, modals, persistent navigation) listed alongside screens. They have their own state machines and participate in flows.

#### Screen Detail

1. **State machine strip** — compact horizontal flow at the top. See Section 3.
2. **Current look** — mockup for the current state, or "no mockup."
3. **Regions** — stacked list grouped by nesting. Events described as labels. Hidden regions faded.
4. **Flows through this screen** — which flows pass through here.

#### URL Routing

- `/screens/:name` — screen detail
- `/flows/:name` — flow walkthrough
- `/flows/:name/:step` — flow at specific step
- `/snapshots` — snapshot list

### 3. State Machine — Compact Interactive Strip

When a screen has a state machine, a compact horizontal strip at the top.

**Visual:**
- States as inline chips connected by labeled edges
- Current state: bold. Reachable transitions: bold + clickable. Unreachable: dim.
- Initial state: dot indicator
- Self-transitions and cross-screen navigates: secondary text below
- Overflow: scrolls horizontally when states exceed available width

**Interaction:**
- Click a reachable transition → state advances, mockup swaps, regions update
- Cross-screen transitions navigate to the target screen in its initial state
- Reset returns to initial state

### 4. Flows — Entry Point + Suggested Path

Flows are a starting point, not a rigid script. The state machine is always running — at any point, the user can go off-script and trigger any valid transition.

This is what makes SFT better than Figma prototyping: the spec is always complete. Every valid transition is available. Error paths and "what ifs" are explored by interacting with the state machine, not by writing more flows.

#### Flow Format

New format (breaking change from the current `sequence:` string):

```yaml
flows:
  - name: login
    description: User logs into the app
    starts: login
    steps:
      - submit credentials
      - auth_success

  - name: read_email
    starts: inbox
    steps:
      - select email
      - go back
```

- `starts` — screen where the flow begins
- `steps` — event names. State machine resolves which transitions fire and which screens they navigate to.
- No screens in steps, no branching, no conditions — the state machine handles all of that.

#### Walkthrough

Click a flow → walkthrough mode.

**Step strip** shows the suggested event sequence. Current step bold.

**Below:** current screen's full detail — mockup, regions, state machine strip.

- "Next step" fires the suggested event through the state machine
- User can go off-script — trigger any valid event in the state machine strip. The walkthrough visually indicates they've left the suggested path.
- "Back to journey" returns to the suggested path
- Back steps backward through the flow

### 5. Snapshots

Named copies of the spec database at a point in time.

- **Save:** name it, save the full database state
- **Compare:** pick two snapshots, see what changed (uses existing diff engine)
- **Restore:** roll back to any snapshot

Accessible from the view as a secondary surface — not primary navigation.

### 6. Ambient Impact

Connections between things are always visible:

- Screen → which flows pass through it
- Flow → screens in sequence
- Region → what user actions it triggers and where they lead

### 7. Mockup-per-State Binding

Add optional `state` column to attachments. When set, the mockup binds to that state. When empty, it binds to the entity regardless of state (current behavior, preserved as fallback).

- `sft attach <screen> <file> --state <state-name>` — binds to a specific look
- `sft attach <screen> <file>` — binds to the entity

Mockups swap when the state machine strip advances.

## Architecture

### Backend

- Existing Go server, embedded NATS, HTTP, port 51741
- `show.Spec` is the primary data contract — needs parsed flow steps and state lists added
- Maturity computation is server-side
- Snapshots are new backend work (CLI commands + NATS subjects)

### Frontend

React 19 + TanStack Router, replacing the experimental canvas. Built from scratch within existing infrastructure (server, NATS client, Vite setup, shadcn/ui primitives).

## Known Trade-offs (from triage)

Decisions made during design that have costs:

1. **New flow format is a breaking change.** All existing flows in all examples must be migrated. The parser must be rewritten. The benefit: flows become pure event sequences driven by the state machine. The cost: migration effort and loss of `{data}` annotations and `[Back]` markers from the old format.

2. **States are not first-class entities.** The schema has no `states` table — states are inferred from transitions, state_regions, and state_fixtures. Every feature touching state machines (strip, maturity, mockup-per-state) needs this derivation. Adding a `States` field to `show.Spec` is the pragmatic fix; a proper `states` table is the structural fix.

3. **Event labels are derived from names, not authored.** The view shows `select_email` as "Select email" — mechanical, not hand-crafted. Good enough for most event names, insufficient for complex ones (`escape`, `check_email`). Adding an event description field was considered and rejected to avoid scope creep.

4. **Off-script walkthrough behavior is intentionally loose.** The spec describes WHAT happens ("walkthrough indicates you've left the path", "back to journey returns you"), not HOW. The implementation plan should specify the mechanism (snapshot/replay, state reconciliation, etc).

5. **Compact strip doesn't scale to 7+ states.** Horizontal scroll is the fallback, but the real constraint is that complex state machines (branching, cycles) don't linearize well. Screens with 6-7 states are the complexity cliff.

## Principles

1. **Shared surface** — one view, same spec data
2. **No opinions** — show the spec, users decide
3. **CLI handles correctness** — view trusts the data
4. **Plain language** — areas, looks, journeys, event labels
5. **Compact interaction** — strips, not panels
6. **Ambient impact** — connections always visible
7. **Explore freely** — flows suggest a path; state machine is always interactive
