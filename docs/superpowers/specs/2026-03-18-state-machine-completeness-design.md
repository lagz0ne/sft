# State Machine Completeness for SFT Screen Inventory

**Date:** 2026-03-19
**Status:** Proposed

## Problem

1. **States are implicit** — no explicit declaration, no completeness proof
2. **No data model** — regions describe behavior but not what data they need
3. **No fixtures** — no concrete data proving what each state looks like

## Goal

Every state named, every transition deterministic, every region's data needs met, every state visually demonstrable. CLI enforces. LLM authors.

## Naming Convention

Everything is `lower_snake`. No exceptions. Every named entity is a map key — no `name:` fields.

## Concept

```
state_machine  →  WHAT states exist
context        →  WHAT data the screen/app holds
ambient        →  WHERE each region gets its data: { name: data(source, query) }
data           →  WHAT the region owns locally
fixture        →  WHAT the context looks like in each state
component      →  HOW to render (already exists)
```

CLI validates the full chain:
1. Every state named and reachable
2. Every `ambient` value `data(source, .path)` resolves to a real context field
3. Every state's fixture provides data for all ambient-dependent regions
4. Every fixture value matches the declared shape
5. Navigate params match target screen context
6. Missing data for a required region = error

## State Machine

### Format: `states:` list → `state_machine:` map

```yaml
state_machine:
  start:                              # built-in initial
    on:
      check_email: selecting
      select_email: { action: navigate(thread_view, { email: data(inbox, .selected) }) }
  selecting:
    on:
      check_email: .                  # stay — event acknowledged
      archive_selected: start
      escape: start
  empty: {}                           # terminal — no transitions out
```

### Built-in States

| State | Meaning |
|-------|---------|
| `start` | initial — where the machine begins |
| `end` | final — triggers `child_name:done` to parent |

### Built-in Events

| Event | Trigger | Direction |
|-------|---------|-----------|
| `child_name:done` | child reaches `end` | bottom-up (auto) |

### Transition Values

| Pattern | Syntax |
|---------|--------|
| Target state | `check_email: selecting` |
| Stay | `check_email: .` |
| Action only | `select_email: { action: navigate(thread_view) }` |
| State + action | `send_reply: { to: start, action: emit(reply_sent) }` |
| Guarded | `submit: [{ guard: "valid", to: saving }, { guard: "invalid", to: . }]` |
| Navigate with params | `select_order: { action: navigate(order_detail, { order: data(order_list, .selected) }) }` |

### Guards

Descriptive strings, not executable. CLI validates: same event in same state without distinguishing guards = warning.

```yaml
state_machine:
  start:
    on:
      submit_pin:
        - { guard: "pin_correct", to: end, action: navigate(home) }
        - { guard: "pin_incorrect", to: error }
        - { guard: "attempts_exceeded", to: locked }
```

### State Machine Templates (Phase 4)

Reusable patterns via `extends:`:

```yaml
app:
  state_templates:
    crud_loadable:
      start:
        on: { load: loading }
      loading:
        on: { load_success: loaded, load_error: error }
      loaded:
        on: { edit: editing }
      editing:
        on: { save: saving, cancel: loaded }
      saving:
        on: { save_success: loaded, save_error: editing }
      error:
        on: { retry: loading }

  screens:
    order_detail:
      state_machine:
        extends: crud_loadable
        loaded:
          on:
            delete: { action: navigate(order_list) }
```

### No Implicit Reset

Parent state changes do NOT auto-reset children. Reset happens through:
1. `child_name:done` — child reaches `end`, parent reacts
2. `navigate()` — target screen starts at `start`
3. Explicit `emit()` — parent emits, child handles in its own `on:`

Preserves `(H)` history in flows.

### Top-Down Coordination

Parent emits, child handles — same mechanism as bottom-up, reversed:

```yaml
# parent
state_machine:
  viewing:
    on:
      edit: { to: editing, action: emit(enter_edit_mode) }

# child toolbar reacts
state_machine:
  start:
    on:
      enter_edit_mode: text_mode
  text_mode:
    on:
      exit_edit_mode: start
```

## Data Model

### App-Level Data Vocabulary (Phase 2)

Domain types defined once, referenced everywhere:

```yaml
app:
  data:
    email:
      subject: string
      sender: contact
      date: datetime
      read: boolean
      body: string
    contact:
      name: string
      email: string
```

Business-level, not DB schema. Composable — shapes reference shapes.

### Context Hierarchy

Data containers at app and screen level:

```yaml
app:
  context:
    current_user: user
    permissions: permission[]

  screens:
    inbox:
      context:
        emails: email[]
        selected: email[]
        active_category: string
```

### Region Data: `ambient` + `data`

```yaml
regions:
  email_list:
    ambient:
      emails: data(inbox, .emails)
    events:
      select_email:
      check_email:

  unread_badge:
    ambient:
      count: data(inbox, .emails[?read==false] | length)

  bulk_action_bar:
    ambient:
      selected: data(inbox, .selected)
      permissions: data(app, .permissions)
    data:
      action_label: string

  search_bar:
    data:
      query: string
      suggestions: string[]
```

- **`ambient:`** — map of named data from the environment. Each key is the region's local name, each value is a `data(source, query)` reference.
- **`data:`** — region's own local state.
- Parallels **ambient events** (keyboard/system events with no region source).

### Navigation with Parameters (Phase 4)

```yaml
state_machine:
  start:
    on:
      select_order: { action: navigate(order_detail, { order: data(order_list, .selected) }) }

# target declares what it expects
screens:
  order_detail:
    context:
      order: order
```

## Fixtures (Phase 3)

### Named Data Snapshots Bound to States

```yaml
fixtures:
  inbox_full:
    inbox:
      emails:
        - { subject: "Welcome", sender: { name: "HR", email: "hr@acme.co" }, read: false }
        - { subject: "Q1 Report", sender: { name: "Finance" }, read: true }
      selected: []
      active_category: "primary"

  inbox_empty:
    inbox:
      emails: []
      selected: []
      active_category: "primary"

  inbox_selecting:
    extends: inbox_full
    inbox:
      selected:
        - { subject: "Welcome", sender: { name: "HR" }, read: false }
    inbox.bulk_action_bar:
      action_label: "Archive 1 item"
```

### State ↔ Fixture Binding

```yaml
state_machine:
  start:
    fixture: inbox_full
    on:
      check_email: selecting
  selecting:
    fixture: inbox_selecting
    on:
      escape: start
  empty:
    fixture: inbox_empty
```

## Validation Rules

**Additive** — new rules join the existing 10.

### Phase 1 Rules (state machine)

| Rule | Severity | Check |
|------|----------|-------|
| `undeclared_state` | error | transition target must exist as a state key |
| `dead_end` | warning | state with no `on:` — CLI infers terminal |
| `guard_ambiguity` | warning | same event + state without distinguishing guards |
| `cycle_in_emit` | error | emit chains must be acyclic (low priority) |

### Phase 2 Rules (data model)

| Rule | Severity | Check |
|------|----------|-------|
| `undefined_data_type` | error | region data references type not in `app.data` |
| `invalid_ambient_path` | error | `data(source, .path)` doesn't resolve |

### Phase 3 Rules (fixtures)

| Rule | Severity | Check |
|------|----------|-------|
| `fixture_type_mismatch` | error | fixture data doesn't match declared shape |
| `data_starved_region` | warning | state's fixture missing data for ambient-dependent region |
| `orphan_fixture` | warning | fixture not referenced by any state (strict mode only) |

### Phase 4 Rules (templates + navigate)

| Rule | Severity | Check |
|------|----------|-------|
| `navigate_params_mismatch` | error | navigate params don't match target screen context |
| `template_override_invalid` | error | extends overrides non-existent state |

### Existing Rules (kept unchanged)

`missing_description`, `orphan_emit`, `unreachable_state`, `duplicate_transition`, `nesting_depth`, `invalid_flow_ref`, `orphan_event`, `dangling_navigate`, `ambiguous_region_name`, `unhandled_event`

Note: `unhandled_event` absorbs `event_gap` — graduated severity (error when zero coverage, warning when partial).

## Full Example: Gmail

```yaml
app:
  name: gmail
  description: email client

  data:
    email:
      subject: string
      sender: contact
      date: datetime
      read: boolean
      body: string
    contact:
      name: string
      email: string
    label:
      name: string
      color: string

  context:
    current_user: contact
    compose_draft: email

  screens:
    inbox:
      description: primary email list with category tabs
      context:
        emails: email[]
        selected: email[]
        active_category: string

      regions:
        category_tabs:
          ambient:
            category: data(inbox, .active_category)
          events:
            switch_category:

        email_list:
          ambient:
            emails: data(inbox, .emails)
          events:
            select_email:
            check_email:

        unread_badge:
          ambient:
            count: data(inbox, .emails[?read==false] | length)

        reading_pane:
          ambient:
            email: data(inbox, .emails[0])
          tags: [preview_pane_enabled]

        bulk_action_bar:
          ambient:
            selected: data(inbox, .selected)
            user: data(app, .current_user)
          data:
            action_label: string
          events:
            archive_selected:
            delete_selected:

      state_machine:
        start:
          fixture: inbox_full
          on:
            select_email: { action: navigate(thread_view, { email: data(inbox, .emails[0]) }) }
            check_email: selecting
            switch_category: .
        selecting:
          fixture: inbox_selecting
          on:
            check_email: .
            archive_selected: start
            delete_selected: start
            escape: start
        empty:
          fixture: inbox_empty

    thread_view:
      description: full conversation view
      context:
        email: email

      regions:
        message_list:
          ambient:
            email: data(thread_view, .email)

        reply_composer:
          events:
            start_reply:
            send_reply:
            discard_reply:
          data:
            draft_body: string
          state_machine:
            start:
              on:
                start_reply: expanded
            expanded:
              on:
                send_reply: { to: end, action: emit(reply_sent) }
                discard_reply: start

      state_machine:
        start:
          on:
            reply_sent: .
            reply_composer:done: .

  fixtures:
    inbox_full:
      inbox:
        emails:
          - { subject: "Welcome", sender: { name: "HR", email: "hr@acme.co" }, read: false }
          - { subject: "Q1 Report", sender: { name: "Finance", email: "fin@acme.co" }, read: true }
        selected: []
        active_category: "primary"

    inbox_empty:
      inbox:
        emails: []
        selected: []
        active_category: "primary"

    inbox_selecting:
      extends: inbox_full
      inbox:
        selected:
          - { subject: "Welcome", sender: { name: "HR" }, read: false }
      inbox.bulk_action_bar:
        action_label: "Archive 1 item"

  flows:
    email_from_inbox:
      description: read an email and return with scroll preserved
      sequence: "inbox → thread_view → [Back] → inbox(H)"
```

## Migration

1. **Phase 1**: `state_machine:` format + built-in states + Phase 1 validation
2. **Phase 2**: `app.data:` shapes + context hierarchy + `ambient` + Phase 2 validation
3. **Phase 3**: fixtures + composition + state-fixture binding + Phase 3 validation
4. **Phase 4**: state machine templates + navigate params + Phase 4 validation

Each phase independently shippable.

## Decisions Log

| Decision | Rationale | Round |
|----------|-----------|-------|
| `lower_snake` everywhere | YAML-friendly, no quoting | 5 |
| Maps for all named entities | Consistent structure, key = identity, no `name:` fields | 7 |
| Keep event bubbling | Architecturally sound | 2 |
| Validation additive | Dropping existing rules = regression | 3 |
| 5 value forms in `on:` | Real examples need all patterns | 3 |
| No implicit reset | Breaks 6+ flows using `(H)` history | 4 |
| `child:done` built-in | Validated by SCXML/XState | 4 |
| Guards descriptive only | Spec tool, not runtime | 4 |
| App-level context | Global data accessible everywhere | 5 |
| `ambient` as map + `data(source, query)` | Regions name their data, express precise contracts | 6 |
| Navigate with params | Cross-screen data transfer explicit | 5 |
| State machine templates | CRUD repeats 5+ times | 5 |
| Fixture `extends:` | Deep wizards duplicate without inheritance | 4 |
| `unhandled_event` absorbs `event_gap` | Graduated severity instead of separate rule | 6 |
| `dead_end` infers terminal | Empty `on:` = terminal, no tag needed | 6 |
| `orphan_fixture` strict-mode only | Noisy during iterative authoring | 6 |

## Appendix: Statechart Equivalences

| SFT | Statechart |
|-----|-----------|
| Regions | Parallel states |
| `emit()` | `sendParent()` |
| `(H)` in flows | History states |
| `state_machine:` map | Compound states |
| `start` / `end` | Initial / final |
| `child:done` | SCXML `done.state.id` |
| `guard:` | SCXML `cond` |
| `app.data:` | XState context |
| `ambient` + `data()` | Computed/derived |
| Fixtures | Storybook args |

## Appendix: Patterns Validated

45 patterns tested across 5 rounds. Expressible: loading/retry, optimistic updates, undo/redo, debounced search, error recovery, concurrent machines, branching flows, machine restart, conditional entry, nested forms, dashboards, modal stacks, infinite scroll, role-based views, collaborative editing, tab preservation, wizard with conditional steps, drag-drop, file upload.

Out of scope: time-based transitions, event payloads, intra-state continuous changes, race conditions.
