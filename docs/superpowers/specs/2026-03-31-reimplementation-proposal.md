# SFT Reimplementation Proposal

## Goal

SFT becomes a **composition engine**: the human describes intent, the agent composes specs across all dimensions (structure, data, tokens, behavior), and the tool guarantees exhaustiveness mechanically so the human only judges meaning.

## Core Abstractions (6)

| Abstraction | What it is | Composes into |
|---|---|---|
| **Entity** | A typed data shape (`email`, `contact`) with relationship cardinality | Fixtures, contexts, ambient refs |
| **Fixture** | A frozen data snapshot — references a shared entity pool, not inline copies | States (via state_fixtures) |
| **Region** | A rectangular UI slot — recursive, carries component binding + delivery tokens | Screens, other regions |
| **State Machine** | Named states with event-driven transitions on a screen or region | Screen behavior |
| **Experiment** | A named overlay (scope + delta) that modifies any region/screen/token without forking the base spec | PM review artifacts |
| **Component** | A render type + prop schema — exists independently, bound to regions at use site | Region content |

Everything else (screens, events, transitions, tags, ambient refs, enums, contexts) is derived from or subordinate to these six.

**Key composition rule:** Entities compose into fixtures. Fixtures bind to states. States drive region visibility. Regions bind components. Components consume entity-shaped props. The chain is: `entity -> fixture -> state -> region -> component`. Every link is validatable.

## Iteration Loops (5)

**1. Entity-first** (domain modeling)
`define entity -> generate fixtures -> bind to screen context -> see rendered preview -> refine entity fields`

**2. Component-first** (workbench)
`define component + prop schema -> render standalone with synthetic data -> bind to region -> see in screen context`

**3. Structure-first** (screen building)
`sketch screens + regions -> assign state machines -> bind fixtures -> render -> restructure regions`

**4. Token-first** (delivery refinement)
`adjust delivery classes on region -> re-render -> compare side-by-side via experiment overlay`

**5. Closure loop** (exhaustiveness)
`run validate -> fix gaps -> run validate -> 0 errors`

All loops are interruptible. Jump from 3 to 1 (need a new entity) to 2 (need a new component) and back. The tool never blocks you.

## Completeness Contract

**Tool checks (mechanical, zero judgment):**

1. Every state has a fixture. Every fixture validates against its entity schema.
2. Every event has a transition. Every `navigate()` target exists.
3. Every screen is reachable from the entry screen.
4. No `extends:` cycles in fixtures.
5. Every leaf region has a component or children.
6. Every data type is referenced by at least one context.
7. Every experiment scope resolves to an existing region/screen.

**Human judges (semantic):**
Fixtures feel realistic. State machines capture all meaningful paths. Component choices match design intent. Experiments cover the alternatives worth comparing.

The tool's job: get the mechanical list to zero. The human's job: sign off on meaning.

## Format Changes

### Shared entity pool (kills fixture duplication)

```yaml
entities:
  sarah: { _type: contact, name: "Sarah Chen", email: "sarah@acme.com" }
  deal_acme: { _type: deal, name: "Acme Corp", value: 50000, rep: $sarah }

fixtures:
  pipeline_full:
    pipeline:
      deals: [$deal_acme, ...]     # reference, not copy
  pipeline_empty:
    extends: pipeline_full
    pipeline: { deals: [] }
```

`$name` references resolve at load time. Entities carry `_type`, validated against `data:` schemas. This eliminates the 3-8x duplication found in every archetype eval.

### Entry screen

```yaml
screens:
  - name: inbox
    entry: true
```

One line. Unlocks navigation graph reachability validation.

### Experiments (first-class)

```yaml
experiments:
  compact_sidebar:
    scope: app.main_nav
    overlay:
      delivery:
        classes: [w-12, items-center]
```

Overlays on top of a complete, valid base spec. `sft view --experiment compact_sidebar` renders the variant. `sft experiment commit compact_sidebar` merges into base.

### Component schemas (progressive)

```yaml
components:
  metric-card:
    props:
      label: string
      value: string
      trend: number?
```

When present, region prop bindings validate against the schema. When absent, props remain opaque JSON. Backward compatible.

## Toolchain

| Command | Purpose |
|---|---|
| `sft init` | Import YAML, create `.sft/db` |
| `sft show` | Inspect any object |
| `sft validate` | Run all 28 mechanical checks, exit 1 on error |
| `sft diff` | Compare db vs YAML — ALL entity types (currently 40% coverage) |
| `sft clone screen\|region <src> <dst>` | Deep-copy with ID remapping |
| `sft experiment create\|apply\|commit\|discard\|list` | Manage overlays |
| `sft component workbench <name>` | Render component standalone |
| `sft set screen <name> --entry` | Mark entry screen |
| `sft view` | Browser viewer with experiment switcher + validation badge |

**Viewer additions:** experiment toggle in dock, fixture inspector (resolved entity data per state), per-screen validation badge (green/red, click for failures).

## Priority

### MVP — makes the methodology work

| # | What | Why |
|---|---|---|
| 1 | **Complete diff** (all 11 entity types) | Agent cannot verify its own mutations without this. Every iteration loop is broken. |
| 2 | **Shared entity pool** (`entities:` + `$ref`) | Fixture authoring is 3x more painful than needed. Blocks realistic data at scale. |
| 3 | **Entry screen + reachability validator** | Trivial schema change + 1 SQL rule. Unlocks nav graph completeness. |
| 4 | **8 missing validator rules** | The closure loop literally cannot close without them. |

### Phase 2 — makes the methodology shine

| # | What | Why |
|---|---|---|
| 5 | **Experiments** (loader + model + CLI + viewer) | PM-reviewable alternatives. Exploration concern has zero tooling today. |
| 6 | **Clone** | Deep-copy for structural exploration. |
| 7 | **Component workbench** | Standalone component rendering + prop schema validation. |
| 8 | **Fixture-validates-against-schema** | Type-safety bridge. Catches stale fixture keys after entity changes. |

MVP is approximately 4 moderate work items. Phase 2 is 4 more. No item in phase 2 blocks the core methodology from functioning — phase 2 makes it faster and more pleasant.
