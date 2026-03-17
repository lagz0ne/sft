# Proposal: SFT as Behavioral Index for Generative UI

## One-liner

SFT owns the story (what the user sees and does). json-render owns the rendering (what components draw it). Attachments carry everything else. One portable file.

## The Problem

Building generative UI today goes straight from prompt to component JSON. Nobody agrees on what screens exist, what the user can do, or how screens connect before components get picked. The behavioral intent is implicit — not reviewable, not versioned, not debatable.

When things change, you edit 200 lines of JSON and hope the structure holds. There's no way to ask "what breaks if I remove this screen?" or "show me all the events in the system."

## The Solution

Three independent layers in one `.sft/db` file:

| Layer | What it captures | Who owns it | How it changes |
|-------|-----------------|-------------|----------------|
| **Story** (SFT entities) | Screens, regions, events, state machines, flows, tags | PM / Designer / Engineer | `sft add/rm/mv` |
| **Rendering** (components) | json-render element type, props, visibility, actions | Design system / Engineer | `sft component` |
| **Content** (attachments) | Copy rules, design tokens, reference docs, anything | Anyone | `sft attach` |

All three live in one SQLite file. Copy the file, you copy the entire spec.

## The Flow

### Provisioning (new spec)

```
1. sft add ...          define screens, regions, events, transitions, flows
2. sft show             review the behavioral story — no components yet
3. sft component ...    bind json-render types to entities
4. sft render           generate a valid json-render spec
5. sft attach ...       add supporting content (copy rules, tokens, etc.)
```

### Iteration (change something)

**Behavior changes, components stay:**
```
sft add region TransferReview "Summary" --in Transfer
sft add event confirm --in TransferReview
sft show                          # story updated
sft component TransferReview Card --props '...'
sft render                        # new element in spec
```

**Design changes, behavior stays:**
```
sft component BalanceCard AreaChart --props '{"dataKey":"history"}'
sft render                        # same structure, different component
```

**Impact before destructive changes:**
```
sft impact screen Transfer        # shows everything that depends on it
sft rm screen Transfer            # cascades cleanly
sft render                        # spec regenerates without it
```

### LLM workflow

```
sft show --json       → LLM reads structure
sft list --json       → LLM sees available content
sft cat _ tokens.json → LLM reads design constraints
sft component X Y ... → LLM binds components
sft render            → LLM generates json-render spec
```

`show` is the story. `list` is the menu. `cat` fetches specifics. `render` produces output.

## How `render` works

Walks the SFT tree and assembles a flat json-render spec:

| SFT concept | json-render output |
|-------------|-------------------|
| Screen | Root element, type from `component` (default: Card) |
| Region | Child element, type from `component` (default: Stack) |
| SFT hierarchy | Wires `children` arrays |
| Entity names | Element keys (already unique) |
| Component props | `props` field (supports $state, $bindState, $computed) |
| Component visible | `visible` field (supports $state conditions) |
| Component on | `on` field (action bindings) |

Entities without component bindings get defaults. The SFT structure provides the skeleton; component bindings provide the flesh.

## What's in the database

```sql
-- Behavioral layer (SFT)
apps, screens, regions, tags, events, transitions, flows, flow_steps

-- Rendering layer (json-render)
components (entity_type, entity_id, component, props, on_actions, visible)

-- Content layer (freestyle)
attachments (entity, name, content BLOB)

-- Cross-cutting views
event_index, state_machines, tag_index, region_tree
```

7 validation rules catch structural issues: missing descriptions, orphan emits, unreachable states, duplicate transitions, invalid flow refs, orphan events, deep nesting.

## Key Design Decisions

1. **Entity names are element keys.** SFT enforces unique names (UNIQUE constraint). json-render uses them directly — no ID mapping, no key generation.

2. **Components are optional.** An SFT spec without components is still useful — it's the behavioral contract. Components layer on when you're ready to render.

3. **Props store $-expressions as-is.** `{"value": {"$state": "/balance"}}` goes into the DB as JSON text. `render` outputs it unchanged. No interpretation — json-render's runtime resolves expressions.

4. **One file.** Everything in SQLite. Attachments are BLOBs, components are rows, the spec is tables. Copy `.sft/db` and you have everything.

5. **Defaults, not errors.** Screen without a component → renders as Card. Region without a component → renders as Stack. You refine progressively, not all-at-once.

## CLI Surface

```
sft show                              behavioral story (text or --json)
sft render                            json-render spec (always JSON)
sft query <name|SELECT>               drill into specifics
sft validate                          structural integrity check

sft add <type> ...                    create entity
sft rm <screen|region> <name>         delete (shows impact, cascades)
sft mv region <name> --to <parent>    reparent
sft impact <screen|region> <name>     show dependents

sft component <entity> [Type]         set/view json-render binding
sft component <entity> --rm           remove binding

sft attach <entity> <file>            attach content
sft detach <entity> <name>            remove content
sft list [entity]                     show all attachments
sft cat <entity> <name>               read content
```

## What this enables

- **Review behavior without visuals.** `sft show` is readable by PMs, designers, engineers, and LLMs.
- **Review visuals without re-reading behavior.** `sft render` outputs a json-render spec that stands alone.
- **Impact analysis before changes.** `sft impact` shows exactly what breaks.
- **LLM-driven iteration.** The LLM reads the story, discovers content via `list`, binds components, renders. Humans review the output.
- **Portable specs.** One 78KB file holds screens, events, state machines, flows, component bindings, copy rules, design tokens — everything.
- **Independent evolution.** Behavior, rendering, and content change at different speeds by different people without stepping on each other.
