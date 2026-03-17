---
name: sft
description: |
  SFT (Screens, Flows, Transitions) — manage UI behavioral specs as structured data.
  Use when the user wants to define, query, modify, or validate a UI specification
  covering screens, regions, events, state machines, tags, and flows. The spec lives
  in a local .sft/db SQLite database in the working directory.

  <example>
  user: "show me the current spec"
  assistant: runs sft show to render the full spec
  </example>

  <example>
  user: "add a Settings screen"
  assistant: runs sft add screen Settings "description"
  </example>

  <example>
  user: "what would break if I remove the Inbox screen?"
  assistant: runs sft impact screen Inbox
  </example>

  <example>
  user: "validate the spec"
  assistant: runs sft validate
  </example>

  <example>
  user: "model the checkout flow for this app"
  assistant: uses sft add to build screens, regions, events, transitions, and flows
  </example>

  <example>
  user: "query all events"
  assistant: runs sft query events
  </example>

  <example>
  user: "attach the component schema to the NavBar"
  assistant: runs sft attach NavBar component.json
  </example>

  <example>
  user: "what content is attached to this spec?"
  assistant: runs sft list to show all attachments
  </example>

  <example>
  user: "show me the NavBar component definition"
  assistant: runs sft cat NavBar component.json
  </example>
---

# SFT — Screens, Flows, Transitions

CLI: `bash ${CLAUDE_SKILL_DIR}/bin/sft.sh <command> [args] [--json]`

Pass `--json` when you need machine-readable output to parse. Omit for human-readable display. The database lives at `.sft/db` — created automatically on first use.

## Commands

```
show                                    full spec — the primary read command
query   screens|events|states|flows|tags|regions
query   states <ScreenOrRegion>         state machine for a specific owner
query   "SELECT ..."                    raw SQL
validate                                check spec integrity
add     <type> ...                      create entity
rm      <screen|region> <name>          delete (cascades children, shows impact)
mv      region <name> --to <parent>     reparent a region
impact  <screen|region> <name>          show dependents without modifying
attach  <entity> <file> [--as name]    attach content to an entity
attach  _ <file>                        attach global content (not entity-specific)
detach  <entity> <name>                remove an attachment
list    [entity]                        show all attachments (the LLM menu)
cat     <entity> <name>                read an attachment's content
```

Aliases: `q` = query, `check` = validate.

## Workflow

### 1. Orient
```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh show
```
Always start here. Returns the full nested spec — app, screens with regions, events, transitions, and flows. Use `--json` if you need to parse the structure.

### 2. Build incrementally
Order matters: app → screens → regions → events → transitions → tags → flows.

```bash
# Bootstrap
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add app "AppName" "What the app does"

# Structure
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add screen "Inbox" "Primary email list with category tabs"
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add region "EmailList" "Scrollable thread list" --in Inbox
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add region "EmailRow" "Single row in the list" --in EmailList

# Behavior
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add event "select-email" --in EmailList
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on select-email --from browsing --action "navigate(ThreadView)" --in Inbox

# Metadata
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add tag "overlay" --on ComposeWindow

# Journeys
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add flow "EmailFromInbox" "Inbox → ThreadView → [Back] → Inbox(H)" \
  --description "Read email and return with scroll preserved"
```

### 3. Before destructive changes — check impact
```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh impact screen Inbox
```
Shows child regions, events, transitions, tags, and flow references that would be affected.

### 4. After mutations — validate
```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh validate
```

## Attachments

SFT is the behavioral index. Attachments are the content that hangs off of it — component schemas, copy rules, color principles, design tokens, anything.

```bash
# Attach a json-render component schema to a region
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh attach NavBar ./navbar-component.json --as component.json

# Attach global content (not entity-specific)
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh attach _ ./copy-rules.md

# See what's available (the menu)
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh list

# Read specific content
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh cat NavBar component.json
```

`list` is the discovery command. Run it to see what content exists across the spec. Then `cat` to read what you need for the task at hand.

Attachments live at `.sft/attach/<entity>/`. Global attachments use `_` as the entity. `show` surfaces attachment names inline with their entity so the full spec view doubles as a content inventory.

## Add Reference

### Transitions (state machine entries)
```bash
# State change
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on event --from state1 --to state2 --in Owner

# Navigate to another screen (no to state — you're leaving)
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on event --from state --action "navigate(Screen)" --in Owner

# Handle event without state change (e.g., keep selecting while in selecting)
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on event --from state --in Owner

# Emit transformed event to parent
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on event --from state --to state2 --action "emit(parent-event)" --in Region

# Action-only (no state guard)
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add transition --on event --action "open Compose" --in AppName
```

Owner (`--in`) resolves automatically: app name, screen name, or region name.

### Tags
```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add tag "overlay" --on ScreenOrRegion
```

### Flows
```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh add flow "FlowName" "Screen → Region → event → Screen" \
  --description "What this journey accomplishes" \
  --on triggering-event
```

## Mutations

```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh rm screen Inbox       # cascades children
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh rm region EmailList    # cascades children
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh mv region EmailList --to Compose  # reparent
```

## Validation Rules

| Rule | Severity | Meaning |
|------|----------|---------|
| `missing-description` | error | Screen/region with empty description |
| `orphan-emit` | error | `emit(X)` where nothing handles X |
| `unreachable-state` | error | State in `from` but no transition leads `to` it |
| `duplicate-transition` | error | Same event+from_state handled twice in same owner |
| `invalid-flow-ref` | error | Flow step names a non-existent screen |
| `orphan-event` | warning | Transition handles event no region emits (may be ambient — keyboard shortcut) |
| `nesting-depth` | warning | Region nested 3+ levels deep |

## Raw SQL

Four views are available for advanced queries:
- `event_index` — events joined with their handlers
- `state_machines` — transitions with resolved owner names
- `tag_index` — tags with resolved entity names
- `region_tree` — regions with parent names, event counts, state presence

```bash
bash ${CLAUDE_SKILL_DIR}/bin/sft.sh query "SELECT * FROM event_index WHERE event = 'select-email'" --json
```

---

# SFT Domain Model

Use this knowledge when deciding how to model a UI.

## Hierarchy

- **App** — top-level boundary. One per spec.
- **Screen** — a viewport the user sees. Contains regions.
- **Region** — building block within a screen. Has events, may have sub-regions (1 level max), may own a state machine.
- **Event** — declared on the region that emits it. kebab-case.
- **Transition** — state machine entry: on event, optionally guarded by from state, optionally transitions to a new state, optionally triggers an action.
- **Tag** — bracket annotation on screen or region.
- **Flow** — named user journey with arrow-notation sequence.

## Placement = Scope

- Region under **App** → persists across all screens (nav bars, overlays, global controls)
- Region under **Screen** → scoped to that screen
- Region under **Region** → nested component (1 level max; deeper = promote to screen)

## State Machines

State machines live at the layer they belong to. Events bubble up: Sub-Region → Region → Screen → App. First `from` value is the initial state.

**Modeling rule**: if a region emits an event and the *screen* handles it (e.g., changing screen mode from browsing to selecting), the transition belongs to the **screen**, not the region. The region just declares the event.

**Sub-machines**: A region with its own `states` block. Handles events before they bubble. Use `emit(parent-event)` to handle locally AND notify parent.

## Tags — Conventions

Tags are untyped bracket annotations. Meaning comes from naming patterns:

| Pattern | Purpose | Examples |
|---------|---------|---------|
| `[overlay]` | Rendering mode | dialog, modal, floating panel |
| `[per-X]` | Parameterized | `[per-account]`, `[per-transaction]` |
| `[has-X]` / `[no-X]` | Data conditional | `[has-payments]`, `[no-accounts]` |
| `[loading]` / `[error]` | Data state | async states |
| `[primary]` / `[destructive]` | Action weight | button importance |
| `[admin]` / `[role]` | Permission | visibility guard |
| `[contains:AppName]` | Cross-app | embedded app |
| `[condition]` | Domain predicate | `[frozen]`, `[expired]`, `[fulfilled]` |

## Flows — Arrow Notation

```
sequence: "Screen → Region → event → Screen(H)"
```

| Element | Meaning |
|---------|---------|
| `ScreenName` | Navigate to screen |
| `RegionName` | Interact with region |
| `event-name` | Event fires (kebab-case) |
| `[Back]` | Back navigation |
| `Screen(H)` | History re-entry (restore scroll, selection, tab) |
| `Step{data}` | Data annotation — what's available at that step |
| `Region activates` | Overlay becomes visible without navigation |

## Common Patterns

**Browse → detail**: Screen state machine with `navigate(DetailScreen)` action.

**Multi-select / bulk actions**: `browsing → selecting` state, with a BulkActionBar region that emits action events.

**Confirmation dialog**: Overlay region with confirm/cancel events. Parent state machine: `viewing → confirming → viewing`.

**Wizard / multi-step**: Single screen with state machine stepping through stages. Each region corresponds to a step. Events advance: `step1 → step2 → step3 → completed`.

**Inline editing**: Region sub-machine: `viewing → editing` triggered by edit event, back to viewing on save/cancel.

**Persistent overlay**: App-level region with `[overlay]` tag. Flows use `activates` keyword.

## Naming

- Screens: PascalCase — `Inbox`, `ThreadView`, `AccountDetail`
- Regions: PascalCase — `EmailList`, `ComposeForm`, `BulkActionBar`
- Events: kebab-case — `select-email`, `submit-form`, `toggle-freeze`
- Screen names imply parameterization: `ProductDetail` = one per product
