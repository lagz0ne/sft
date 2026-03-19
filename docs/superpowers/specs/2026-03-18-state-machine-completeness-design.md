# State Machine Completeness for SFT Screen Inventory

**Date:** 2026-03-19
**Status:** Proposed

## Problem

1. **States are implicit** — no explicit declaration, no completeness proof
2. **No data model** — regions describe behavior but not what data they need
3. **No fixtures** — no concrete data proving what each state looks like

## Goal

Every state named, every transition deterministic, every region's data needs met, every state visually demonstrable. CLI enforces. LLM authors.

## Concept

Everything is `lower_snake`. `extends` is reserved — cannot be used as a state name.

```
enums          →  WHAT values a field can take (named sets)
data           →  WHAT shapes exist (types with optional fields)
context        →  WHAT data the screen/app holds
ambient        →  WHERE each region gets its data: { name: data(source, query) }
region data    →  WHAT the region owns locally
events         →  WHAT happened + what data it carries: event_name(type)
state_machine  →  WHAT states exist + which regions are active per state
fixture        →  WHAT the context looks like in each state
emit(e, target)→  WHO receives cross-machine events (exhaustive)
component      →  HOW to render (already exists)
```

## State Machine

### Format

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

Built-in states: `start` (initial), `end` (final — triggers `child_name:done` to parent). `child:done` signals generic completion. Use `emit()` when the parent needs to distinguish between different outcomes.

### Transition Values

| Pattern | Syntax |
|---------|--------|
| Target state | `check_email: selecting` |
| Stay | `check_email: .` |
| Action only | `select_email: { action: navigate(thread_view) }` |
| State + action | `send_reply: { to: start, action: emit(reply_sent, target: [thread_view]) }` |
| Guarded | `submit: [{ guard: "valid", to: saving }, { guard: "invalid", to: . }]` |
| Navigate with params | `select_order: { action: navigate(order_detail, { order: data(order_list, .selected) }) }` |

Guards are descriptive strings, not executable. CLI validates: same event in same state without distinguishing guards = warning.

### Templates

Reusable patterns via `extends:`. State overrides replace the template state entirely — `on:`, `fixture:`, and `regions:` are all replaced, not merged:

```yaml
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
```

### Reset and Coordination

Parent state changes do NOT auto-reset children. Preserves `(H)` history in flows. Reset through:
1. `child_name:done` — child reaches `end`, parent reacts
2. `navigate()` — target screen starts at `start`
3. `emit(event, target: [child])` — parent emits to named targets, child handles

Top-down coordination:

```yaml
# parent screen
state_machine:
  viewing:
    on:
      edit: { to: editing, action: emit(enter_edit_mode, target: [toolbar]) }

# child toolbar region reacts
state_machine:
  start:
    on:
      enter_edit_mode: text_mode
  text_mode:
    on:
      exit_edit_mode: start
```

Bare `emit(event)` without `target:` triggers `emit_missing_target` warning under Phase 5 validation.

## Data Model

### Enums

Named sets of valid values. Referenced by name in type fields.

```yaml
enums:
  category: [primary, social, promotions, updates]
  priority: [urgent, high, medium, low, none]
```

### Data Types

Domain types defined once, referenced everywhere. `?` suffix marks optional/nullable fields.

```yaml
data:
  email:
    subject: string
    sender: contact
    date: datetime
    read: boolean
    body: string?
    attachments: string[]?
  contact:
    name: string
    email: string
```

Suffix order: `type[]?` = optional array. `type?` = optional scalar. `type?[]` is invalid.

### Context

Data containers at app and screen level:

```yaml
context:
  current_user: user
  permissions: permission[]

screens:
  inbox:
    context:
      emails: email[]
      selected: email[]
      active_category: category      # enum type
```

### Region Data

```yaml
regions:
  email_list:
    ambient:
      emails: data(inbox, .emails)
    events:
      select_email(email):             # annotation: carries an email
      check_email(email):

  unread_badge:
    ambient:
      count: data(inbox, .emails[?read==false] | length)

  bulk_action_bar:
    ambient:
      selected: data(inbox, .selected)
      permissions: data(app, .permissions)
    data:
      action_label: string
    events:
      archive_selected(email[]):       # annotation: carries email array
      delete_selected(email[]):
```

Event name is everything before `(` — transitions, emit, and flows reference the bare name. When all events are bare, the list form `events: [a, b]` is equivalent.

## Fixtures

Fixture `extends:` merges by context key (`inbox`, `inbox.bulk_action_bar`). Within each key, child fields override parent fields. New keys are added. Fields not mentioned are inherited.

```yaml
fixtures:
  inbox_full:
    inbox:
      emails:
        - { subject: "Welcome", sender: { name: "HR", email: "hr@acme.co" }, read: false }
        - { subject: "Q1 Report", sender: { name: "Finance" }, read: true }
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

### State ↔ Fixture + Region Binding

```yaml
state_machine:
  start:
    fixture: inbox_full
    regions: [category_tabs, email_list, unread_badge]
    on:
      check_email: selecting
  selecting:
    fixture: inbox_selecting
    regions: [category_tabs, email_list, bulk_action_bar]
    on:
      escape: start
  empty:
    fixture: inbox_empty
    regions: [category_tabs, empty_state]
```

Omitting `regions:` means all child regions are visible (backwards compatible). Listing a region makes it and all its descendants visible. `regions:` is for state-driven visibility. Tags are for orthogonal conditions (feature flags, roles, data presence). Both compose: a region must satisfy its tag AND be in the state's `regions:` list.

## Validation Rules

| Rule | Severity | Check |
|------|----------|-------|
| `missing_description` | error | screen/region has no description |
| `orphan_emit` | error | emit targets event with no handler |
| `unreachable_state` | error | state not reachable from initial |
| `duplicate_transition` | error | same event+from_state appears 2+ times |
| `nesting_depth` | warning | region nested 3+ levels deep |
| `invalid_flow_ref` | error | flow step references unknown screen/region/event |
| `orphan_event` | warning | transition handles event no region emits |
| `dangling_navigate` | error | navigate targets unknown screen/region |
| `ambiguous_region_name` | warning | same name used by multiple regions |
| `unhandled_event` | warning | event emitted but no handler (error when zero coverage) |
| `undeclared_state` | error | transition target not a state key |
| `dead_end` | warning | state with no `on:` — infers terminal |
| `guard_ambiguity` | warning | same event+state without distinguishing guards |
| `undefined_data_type` | error | field references type not in `app.data` or `app.enums` |
| `invalid_ambient_path` | error | `data(source, .path)` doesn't resolve |
| `fixture_type_mismatch` | error | fixture data doesn't match declared shape |
| `data_starved_region` | warning | state's fixture missing data for ambient-dependent region |
| `orphan_fixture` | warning | fixture not referenced by any state (strict mode only) |
| `navigate_params_mismatch` | error | navigate params don't match target screen context |
| `template_override_invalid` | error | extends overrides non-existent state |
| `undefined_enum` | error | field references enum not in `app.enums` |
| `fixture_missing_required` | warning | fixture omits a non-optional field |
| `invalid_state_region` | error | state `regions:` references unknown child region |
| `emit_target_mismatch` | error | `emit(event, target: [x])` but `x` has no handler |
| `invalid_event_annotation` | warning | event annotation type not in `app.data` or `app.enums` |
| `enum_data_collision` | warning | same name in both `app.enums` and `app.data` |
| `reserved_state_name` | error | state named `extends` (reserved key) |
| `emit_missing_target` | warning | `emit(event)` without `target:` |

## Full Example: Gmail

```yaml
app:
  name: gmail
  description: email client

  enums:
    category: [primary, social, promotions, updates]

  data:
    email:
      subject: string
      sender: contact
      date: datetime
      read: boolean
      body: string?
      attachments: string[]?
    contact:
      name: string
      email: string
    label:
      name: string
      color: string

  context:
    current_user: contact
    compose_draft: email?

  screens:
    inbox:
      description: primary email list with category tabs
      context:
        emails: email[]
        selected: email[]
        active_category: category

      regions:
        category_tabs:
          description: Primary / Social / Promotions / Updates tab filter
          ambient:
            category: data(inbox, .active_category)
          events:
            switch_category(category):

        email_list:
          description: scrollable thread list with individual and bulk interactions
          ambient:
            emails: data(inbox, .emails)
          events:
            select_email(email):
            check_email(email):

        unread_badge:
          ambient:
            count: data(inbox, .emails[?read==false] | length)

        reading_pane:
          description: inline email preview — only when preview pane setting is on
          ambient:
            email: data(inbox, .emails[0])
          tags: [preview_pane_enabled]

        bulk_action_bar:
          description: archive, delete, label — appears in selecting state
          ambient:
            selected: data(inbox, .selected)
            user: data(app, .current_user)
          data:
            action_label: string
          events:
            archive_selected(email[]):
            delete_selected(email[]):

        empty_state:
          description: no emails illustration + prompt to check other categories

      state_machine:
        start:
          fixture: inbox_full
          regions: [category_tabs, email_list, unread_badge, reading_pane]
          on:
            select_email: { action: navigate(thread_view, { email: data(inbox, .emails[0]) }) }
            check_email: selecting
            switch_category: .
        selecting:
          fixture: inbox_selecting
          regions: [category_tabs, email_list, unread_badge, bulk_action_bar]
          on:
            check_email: .
            archive_selected: start
            delete_selected: start
            escape: start
        empty:
          fixture: inbox_empty
          regions: [category_tabs, empty_state]

    thread_view:
      description: full conversation view
      context:
        email: email

      regions:
        message_list:
          description: chronological messages with expand/collapse per message
          ambient:
            email: data(thread_view, .email)

        reply_composer:
          description: rich text reply editor at bottom of thread
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
                send_reply: { to: end, action: emit(reply_sent, target: [thread_view]) }
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

## Out of Scope

Time-based transitions, intra-state continuous changes, race conditions, multi-window/detachable panels, drag-and-drop phases, overlay stacking/z-ordering, form validation rules, responsive breakpoints.
