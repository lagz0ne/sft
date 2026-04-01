# SFT Authoring Methodology

A protocol for building exhaustive UI behavioral specs. An AI agent drives authoring collaboratively with a human who describes what they want. The spec is the artifact — stands on its own, independent of who authored it.

## 5 Concerns, Not 5 Stages

The methodology addresses 5 concerns. These are NOT sequential stages — they're lenses you apply in whatever order the work demands. Start from a component, a screen, a token, or a data type. Loop between concerns freely. The only rule: when the spec is done, all 5 concerns are addressed.

| Concern | What it addresses | What it produces |
|---------|-------------------|------------------|
| **Domain** | The vocabulary of the product | `data:` types, `enums:`, `context:` |
| **Fixtures** | Realistic frozen-snapshot data | `fixtures:` with production-feeling content |
| **Structure** | Screens, regions, state machines | `screens:`, `regions:`, `state_machine:`, `events:` |
| **Exploration** | "What if?" at any level | Experiments, clones, token/component/layout variants |
| **Closure** | Exhaustiveness — no unanswered questions | Validator passes, PM signs off against ACs |

### Any entry point works

```
component → screen → component → token       ✓
screen → region → fixture → data type        ✓  
token palette → region → screen → fixture    ✓
data type → fixture → screen → component     ✓
```

You might start by designing a card component, then build a screen around it, then realize you need a new data type to populate it, then seed fixtures, then experiment with a different card layout. The concerns name WHAT you're addressing, not WHEN.

### Iteration levels

| Level | Iterate on | Example |
|-------|------------|---------|
| **Token** | Spacing, colors, typography | Named delivery class sets: `[bg-gray-900, text-white, p-4]` |
| **Component** | Shape + props | A `metric-card` with `{label, value, trend}`, independent of any screen |
| **Region** | Structure + placement | `kpi_strip` composing 4 `metric-card` components with grid layout |
| **Screen** | Composition + state machines | `dashboard` arranging regions with state-driven visibility |
| **Flow** | Cross-screen navigation | `inbox → thread_view → compose` with transitions |

All levels are valid starting points. All loop back to each other.

## The 5 Concerns in Detail

### Domain

Derive entities and types from the product category + known user stories. This is a living model refined throughout.

**Relationship nature matters.** Entity relationships capture business structure: `order` HAS MANY `order_item[]`, `invoice` HAS MANY `invoice_item[]`. These drive fixture structure (creating an order implies creating its items) and eliminate redundant seeding.

```yaml
data:
  order:
    id: number
    customer: customer
    items: order_item[]      # HAS MANY — drives fixture shape
    total: number
    status: order_status
  order_item:
    product: string
    quantity: number
    price: number
  customer:
    name: string
    email: string            # semantic hint: faker can generate realistic emails
    phone: string?
```

**Semantic field hints.** Fields named `email`, `name`, `phone`, `avatar_url`, `published_at` carry inherent meaning. The agent can generate realistic fake data from field name + type alone — `email: string` → `"john.doe@example.com"`, `name: string` → `"Nguyen Van A"`. This gives the "frozen snapshot" feel even before explicit fixture authoring.

**Batch thinking.** Domain modeling happens in correlated batches — types, enums, and screens emerge together. The YAML format supports this naturally (`sft init` imports everything at once). For incremental work, batch correlated CLI calls.

**Dashboard archetype note.** Dashboard apps produce two classes of types: domain entities (deal, activity) and view models (metric_snapshot, pipeline_summary). View models are computed projections, not CRUD entities. The type system treats them identically — this is serviceable but semantically imprecise. Name view-model types descriptively (`metric_snapshot`, `forecast_summary`) so their nature is clear from the name.

### Fixtures

Populate realistic, production-feeling sample data. Not lorem ipsum — frozen snapshots of what the real product looks like. Realism exposes design problems and enables stakeholder communication.

**Semantic seeding eliminates boilerplate.** When the domain model declares `email: string`, the agent can auto-generate `"user@example.com"` without explicit fixture authoring. Relationship nature drives structure: populating an `order` automatically generates its `order_item[]` children.

**States are cheap for data variants.** Gmail models `browsing` (normal), `selecting` (bulk actions), and `empty` (no emails) as three distinct states, each with their own fixture. This is the canonical pattern — don't fight the one-fixture-per-state constraint, use more states.

**Events on forms are NOT state-transitioning events.** Form mutations like `update_title`, `add_ingredient`, `upload_photo` are side effects that don't drive state transitions. Don't declare these as events — they're implied by the component type. Events are for behavioral transitions: `select_email`, `start_compose`, `send_invoice`. Gmail follows this convention: the compose window has `send_email` and `discard_draft` but NOT `update_subject` or `update_body`.

```yaml
fixtures:
  inbox_full:
    inbox:
      emails: [...]
      active_category: "primary"
  inbox_empty:
    extends: inbox_full
    inbox:
      emails: []
```

### Structure

User stories drive screens. Regions start at whatever granularity makes sense — YAGNI for navigation-centric apps (start coarse, destructure later), but data-dense screens like dashboards need granular regions from day one because each visualization is a different component with different data bindings.

**Progressive destructuring** for navigation apps: name `hero_zone` first, break into `hero_image` + `hero_title` + `hero_author` when layout decisions demand it.

**Upfront granularity** for data-dense apps: dashboards, analytics, and reports need one region per distinct visualization or data binding from the start. A dashboard with 8 metric cards, 2 charts, and a table needs 11+ regions immediately — there's no useful coarse starting point.

```yaml
# Navigation app: start coarse
regions:
  - name: email_list
    description: Scrollable thread list
    tags: [main]

# Dashboard app: start granular
regions:
  - name: kpi_strip
    description: Four headline metrics
    delivery:
      classes: [grid, grid-cols-4, gap-4]
    regions:
      - name: revenue_card
        component: metric-card
        props: '{"label": "Revenue", "format": "currency"}'
      - name: deals_won_card
        component: metric-card
        props: '{"label": "Deals Won"}'
```

### Exploration

"What if?" applied to anything — component, layout, props, structure, tokens, screen flows. Experiments can be committed or discarded.

**Experiments are embedded in the format, not hidden in git branches.** A PM reviewing the spec needs to see "here are 3 sidebar variants" in the spec itself. Experiments are a review artifact.

An experiment is a named overlay targeting a scope (screen, region, app) describing what changes. The base spec is always complete and valid.

```yaml
experiments:
  compact_sidebar:
    description: "Narrower sidebar, icons only"
    scope: app.main_nav
    overlay:
      delivery:
        classes: [w-12, shrink-0, bg-gray-50, flex, flex-col, items-center]
      regions:
        - name: inbox_nav
          component: button
          props: '{"icon": "inbox", "variant": "ghost"}'

  masonry_feed:
    description: "Pinterest-style masonry grid"
    scope: home_feed.recipe_grid
    overlay:
      delivery:
        classes: [columns-2, "lg:columns-3", gap-4]

  condensed_kpi:
    description: "Inline metrics instead of cards"
    scope: dashboard.kpi_strip
    overlay:
      delivery:
        classes: [flex, gap-6, py-2, border-b]
```

**Clone for structural variants:** `sft clone screen inbox inbox_alt` deep-copies a screen with all children, then mutate selectively. Clone operates at screen or region level.

**Git for working state:** checkpoint with `cp .sft/db` before risky changes. Use git branches for bigger experiments. But the PM-facing artifacts are `experiments:`, not branches.

### Closure

The spec is exhaustive by design — no question about the UI goes unanswered. The tool ensures structural integrity; semantic completeness is PM-judged against acceptance criteria.

**What "exhaustive" means:**

1. Every screen has states, every state has a fixture — you can render any state with real data
2. Every event goes somewhere — no orphan events, every transition has a destination
3. Every region has content or children — no empty leaf regions without components
4. Every data type is used — no phantom types that never appear on screen
5. Every screen is reachable — navigation graph has no islands from the entry screen

## Key Properties

- **The spec is the artifact** — stands on its own, independent of who authored it
- **Concerns are concurrent, not sequential** — address domain, fixtures, structure, exploration, closure in any order
- **Realism is non-negotiable** — the spec feels like the real product frozen in time
- **Exploration is first-class** — experiments embedded in format, reviewable by PMs
- **Any granularity is a valid starting point** — component, token, region, screen, or flow

## Eval Findings

Tested against 3 product archetypes: CRUD/transactional (invoicing), content/social (recipe sharing), dashboard/analytics (sales dashboard).

### Cross-archetype friction

| # | Friction | Severity |
|---|----------|----------|
| F1 | **Fixture entity duplication** — same object (invoice, recipe, rep) copy-pasted in full across fixtures. No `$ref`, no anchors, no shared entity pool. Fixtures are 3x larger than needed. | HIGH |
| F2 | **Experiments format unimplemented** — `experiments:` is proposed but has zero implementation in loader/model/validator. Exploration (concern 4) is untestable. | HIGH |
| F3 | **Form events generate false positives** — `add_ingredient`, `update_title` are form mutations without state transitions. Validator flags as `unhandled-event`. | MEDIUM |

### Archetype-specific findings

| # | Finding | Archetype |
|---|---------|-----------|
| F4 | Data-dense screens need upfront granularity, not progressive destructuring | Dashboard |
| F5 | View-model types (metric_snapshot) are semantically distinct from domain entities but treated identically | Dashboard |
| F6 | Same data needs both form and display modes, doubling region definitions | CRUD |

### Resolutions incorporated above

- **F3** → documented: form mutations are NOT events (see Fixtures concern)
- **F4** → documented: dashboards skip coarse phase (see Structure concern)
- **F5** → documented: name view-model types descriptively (see Domain concern)
- **F1, F2** → tracked as format gaps below

## Format Gaps

### HIGH

| ID | Gap | Impact |
|----|-----|--------|
| H1 | Fixture data is opaque JSON — no validation against declared types | "Fixture exists" ≠ "fixture is correct." Renaming a context field silently leaves fixtures with stale keys. |
| H3 | `sft diff` ignores 10+ entity types | Changes to types, enums, fixtures, layouts invisible to diff. Agent can't verify its own mutations. |
| H4 | 8 of 28 designed validator rules unimplemented | The missing rules are the coverage closure rules. |
| F1 | Fixture entity duplication across fixtures | Same object copy-pasted 3-8x. No cross-fixture reference mechanism. Fixtures 3x larger than needed. |
| F2 | Experiments format unimplemented | `experiments:` has no loader, no model, no CLI, no viewer support. |

### MEDIUM

| ID | Gap | Impact |
|----|-----|--------|
| M2 | Component props are opaque text | No schema for what props a component accepts. |
| M3 | No entry-screen concept, screen reachability not validated | Unreachable screens are invisible dead code. |

### Resolved by existing patterns

| Concern | Resolution |
|---------|------------|
| One fixture per state | States are cheap. Model data variants as separate states. |
| No undo/snapshot | `cp .sft/db` is trivial. Git handles file-level versioning. |
| Can't re-import YAML | By design: CLI-as-sole-writer. `rm -rf .sft && sft init` for full reimport. |
| Stateless screens | Convention: single-state machine. `search_results.browsing` pattern. |
| Form events noisy | Convention: don't declare form field mutations as events. |

## Solutions

### S1: Fixture validation rule (H1)

Go-level validator rule cross-referencing `fixtures.data` JSON keys against `contexts` field declarations. Extends the `rule` struct with an `fn` field for Go-level validation alongside SQL queries. Severity: warning.

### S2: Entry screen (M3)

`ALTER TABLE screens ADD COLUMN entry INTEGER NOT NULL DEFAULT 0;`
YAML: `entry: true` on screen. CLI: `sft set screen inbox --entry`. Validator: warning if no entry screen, error if multiple.

### S3: Screen reachability rule (M3)

Validator rule flagging screens not targeted by `navigate()` and not the entry screen. Reuses existing navigate() parsing. Depends on S2.

### S4: Fixture extends cycle detection (H4)

Error-severity Go-level DFS cycle detection on fixture extends graph. ~25 lines.

### S5: State-region without fixture rule (H4)

Warning rule: states with `state_regions` visibility entries but no `state_fixtures` binding.

### S6: Complete diff (H3)

Extend `diff.Compare()` to walk all 11 entity types `show.Spec` carries. ~12 small diff functions following existing patterns.

### S7: Clone command

`sft clone screen <name> <new-name>` — deep-copy with ID remapping in a single transaction. Also `sft clone region`. Enables structural exploration.

### S8: Experiments in format (F2)

`experiments:` table with name, description, scope, overlay (JSON), status. CLI: `sft experiment create/apply/commit/discard/list`. Viewer: experiment switcher in dock.

### S9: Fixture entity references (F1)

Shared entity pool to eliminate cross-fixture duplication. Options under consideration:

- **YAML anchors** (`&sarah` / `*sarah`) — if the loader preserves them through parse/import
- **`shared:` block** in fixtures — define entities once, reference by key
- **`$ref` paths** — `recipe: $ref(home_popular.home_feed.featured_recipes[0])`

Needs design work to determine which approach fits the existing loader architecture.

## Implementation Priority

| Order | Solution | Effort | Unlocks |
|-------|----------|--------|---------|
| 1 | S6: Complete diff | Moderate | Trustworthy change detection — foundation |
| 2 | S7: Clone command | Moderate | Structural exploration |
| 3 | S8: Experiments | Moderate | PM-reviewable alternatives |
| 4 | S9: Fixture dedup | Moderate | Fixture authoring at scale |
| 5 | S2: Entry screen | Trivial | Navigation root |
| 6 | S3: Screen reachability | Easy | Nav graph completeness |
| 7 | S4: Fixture cycle detection | Trivial | Prevents infinite recursion |
| 8 | S5: State-region no fixture | Easy | Catches incomplete states |
| 9 | S1: Fixture validation | Moderate | Type-safety bridge |
