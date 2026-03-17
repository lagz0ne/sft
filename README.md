# SFT — Screens, Flows, Transitions

A lightweight vocabulary for making implicit UI structure explicit. Event-driven, layered state machines in YAML.

SFT sits between Figma (visual design) and PRDs (requirements) — a **behavioral contract layer** that captures what the user sees and does at the screen/region level. Write it once, hand it to designers, engineers, and PMs.

## Install

```bash
npm i -g sft-cli
```

Binaries also available from [GitHub Releases](https://github.com/lagz0ne/sft/releases).

## Quick Start

```bash
# Import an existing spec
sft import examples/linear.sft.yaml

# Explore
sft show                   # full spec tree (human + LLM readable)
sft show --json            # structured JSON
sft query screens          # list all screens
sft query events           # list all events
sft query states Home      # transitions for a screen

# Validate
sft validate               # check for orphans, dead events, cycles

# Round-trip
sft export spec.yaml       # serialize back to YAML
sft diff examples/linear.sft.yaml   # compare current vs a file
```

## CLI Reference

All commands support `--json` for structured output. The spec lives in `.sft/db` (auto-created).

### Reading

```bash
sft show                             # full spec tree
sft query <type>                     # screens | regions | events | flows | tags
sft query states <name>              # transitions for a screen/region
sft query steps <flow>               # parsed flow steps
sft query "SELECT ..."               # raw SQL against the spec DB
sft impact <screen|region> <name>    # what depends on this entity
```

### Mutating

```bash
sft add app <name> <desc>
sft add screen <name> <desc>
sft add region <name> <desc> --in <parent>
sft add event <name> --in <region>
sft add transition --on <event> --in <owner> [--from <s>] [--to <s>] [--action <a>]
sft add tag <tag> --on <entity>
sft add flow <name> <sequence> [--description <d>] [--on <event>]

sft set <screen|region> <name> --description <new> [--in <parent>]
sft rename <screen|region|flow> <old> <new> [--in <parent>]
sft rm <screen|region|event|transition|tag|flow> <name> [--in/--on <parent>]
sft mv region <name> --to <parent> [--in <current-parent>]
sft reorder <parent> <child1> <child2> ...
```

### Components

Bind UI component types to screens or regions for code generation:

```bash
sft component Home                          # show bound component (JSON)
sft component Home Dashboard --props '{"layout":"grid"}'
sft component Sidebar NavPanel --props-file sidebar.json --on handleClick --visible auth
sft component Home --rm                     # unbind
```

Components round-trip through `export`/`import` — they're preserved in YAML.

### Import / Export / Diff

```bash
sft import spec.yaml          # load YAML into fresh DB
sft export [file.yaml]        # serialize to YAML (stdout if no file)
sft diff spec.yaml            # compare current spec vs YAML file
```

### Attachments

```bash
sft attach Home mockup.png --as wireframe.png
sft detach Home wireframe.png
sft list                      # all attachments
sft list Home                 # attachments on Home
sft cat Home wireframe.png    # read attachment content
```

### Scoped Regions

Region names can repeat across different parents. Use `--in` to disambiguate:

```bash
sft add region Header "Top bar" --in Settings
sft add region Header "Top bar" --in Profile    # same name, different parent
sft rm region Header --in Settings
sft rename region Header TopBar --in Profile
```

### Rendering (json-render)

`sft render` generates a [json-render](https://github.com/nicholasgasior/json-render)-compatible spec — a flat element tree that a UI runtime can consume directly.

```bash
sft render                    # full json-render spec to stdout
sft render | jq '.elements.Home'   # inspect one element
```

The pipeline:
1. **Skeleton** — every screen becomes a `Card`, every region a `Stack`, children wired by hierarchy
2. **Hydrate** — components bound via `sft component` override the element type, props, `on` handlers, and `visible` conditions

```bash
# Without components — generic skeleton
sft render | jq '.elements.MyIssues'
# {"type": "Card", "props": {"title": "MyIssues"}, "children": ["IssueList", "FilterBar", ...]}

# Bind a component
sft component MyIssues DataTable --props '{"cols":["title","status"],"selectable":true}'

# Now render picks it up
sft render | jq '.elements.MyIssues'
# {"type": "DataTable", "props": {"cols":["title","status"],"selectable":true}, "children": [...]}
```

Full component binding with event handlers and visibility:

```bash
sft component IssueDetail DetailPanel \
  --props '{"layout":"split"}' \
  --on '{"save":"handleSave","delete":"handleDelete"}' \
  --visible '{"role":"member"}'
```

Output schema:
```json
{
  "root": "AppName",
  "elements": {
    "ScreenOrRegionName": {
      "type": "ComponentType",
      "props": { ... },
      "children": ["ChildName", ...],
      "on": { "event": "handler" },
      "visible": { "condition": "value" }
    }
  }
}
```

## The Model

### Structure — what exists, what events it emits

```yaml
app:
  regions:          # app-level → persist across all screens
    - name: MainNav
    - name: ComposeWindow
      tags: [composing, overlay]
      events: [send-email]

  screens:
    - name: Inbox
      regions:      # screen-scoped
        - name: EmailList
          events: [select-email, check-email]
```

- **App** — top-level boundary. One deployable app = one App.
- **Screen** — what the user sees. A viewport grouping of Regions.
- **Region** — building block. Own content, own events. May contain sub-regions (1 level max).
- **Events** — declared on the Region that emits them.
- **Tags** — `[condition]` for existence, `[overlay]` for rendering.
- **Placement = scope** — Region under App persists. Region under Screen is scoped.

### States — layered state machines

State machines live at the layer they belong to. Events bubble up: Sub-Region → Region → Screen → App. Valid states are inferred from `from:` and `to:` values in transitions. The first `from:` value is the initial state.

```yaml
screens:
  - name: Inbox
    regions:
      - name: EmailList
        events: [select-email, check-email]

    # screen-level state machine
    states:
      - on: select-email          # event from EmailList (bubbled up)
        from: browsing
        action: navigate(ThreadView)
      - on: check-email
        from: browsing
        to: selecting
      - on: Escape                # ambient event (keyboard)
        from: selecting
        to: browsing

  - name: ThreadView
    regions:
      - name: ReplyComposer
        events: [start-reply, send-reply]

        # region-level state machine (sub-machine)
        states:
          - on: start-reply       # consumed here, doesn't bubble
            from: collapsed
            to: expanded
          - on: send-reply
            from: expanded
            to: collapsed
            action: emit(reply-sent)   # handle locally AND send reply-sent to parent

    # screen-level state machine
    states:
      - on: reply-sent            # emitted from ReplyComposer
        from: reading
```

**Event bubbling**: Region emits event → Region's state machine gets first look → if unhandled, bubbles to Screen → if unhandled, bubbles to App. Handled = consumed. With nested regions, the chain deepens: Sub-Region → Region → Screen → App.

**`emit(event-name)` action**: When a state machine handles an event locally but the parent also needs to know — with a *named* event:
```yaml
- on: send-reply
  from: expanded
  to: collapsed
  action: emit(reply-sent)   # handle locally AND send reply-sent to parent
```
The parent must have a matching `on: reply-sent` transition. Unlike automatic bubbling (which re-sends the same event), `emit` transforms the event — the parent sees a domain-level event, not the child's internal event.

### Transitions

- `on:` — event that triggers it
- `from:` — guard (required current state)
- `to:` — target state (omit to handle the event without changing state, e.g., continuing to select items while already in selecting)
- `action:` — built-in side effect

Built-in actions:
| Action | What it does |
|--------|-------------|
| `navigate(Screen)` | Exit current screen, enter target |
| `emit(event-name)` | Handle locally AND send named event to parent layer |

### Flows — named journeys

Flows document key user journeys worth communicating to the team. Not every navigation needs a flow — only journeys with handoff-relevant detail (back behavior, state preservation, data dependencies, error paths).

```yaml
flows:
  - name: EmailFromInbox
    description: Read an email and return with scroll position preserved
    sequence: "Inbox → ThreadView → [Back] → Inbox(H)"

  - name: RefundPayment
    description: Issue a refund with amount and reason confirmation
    on: start-refund
    sequence: "PaymentDetail{payment ID} → RefundConfirmation{refund amount + reason} → confirm-refund → PaymentDetail{updated status}"
```

- **`sequence`** — arrow notation showing the journey path. Authoritative representation.
- **`(H)`** — history re-entry. Resume prior sub-state (scroll position, selection, tab).
- **`on:`** — event that triggers the flow. Omit when the flow starts from a screen the user navigates to normally.
- **`{data}`** — inline data annotation on a step. Shows what data is available or produced at that step. Use when data other than the entity ID must survive a screen change.
- **`activates`** — in a sequence, means an independently-triggered overlay becomes visible without screen navigation: `"ComposeWindow activates → fill → send-email"`. State-machine-controlled overlays (confirmation dialogs) are referenced directly: `"PaymentDetail → RefundConfirmation → confirm-refund → PaymentDetail"` — the parent state machine governs visibility.

Sequence elements follow naming conventions: `ScreenName` or `RegionName` (PascalCase), `event-name` (kebab-case), prose action (lowercase, e.g., `fill`, `await resolution`), `[Back]` (bracketed navigation), `RegionName activates` (overlay activation), `ScreenName(H)` (history re-entry), `Step{data}` (data annotation).

### Components

Components bind a UI component type to a screen or region, bridging behavioral spec to implementation:

```yaml
screens:
  - name: Dashboard
    description: Main dashboard
    component: DataGrid
    props: '{"cols":5}'
    on_actions: handleClick
    visible: admin
```

Components survive `import` → `export` round-trips and are detected by `sft diff`.

## Conventions

| Convention | How it works |
|-----------|-------------|
| **App-level Regions** | Regions under App persist across all screens |
| **Naming** | Screen names imply parameterization. `ProductDetail` = per-product |
| **Tags** | Untyped bracket annotations. Categories by naming convention: `[overlay]` rendering, `[per-X]` parameterization, `[has-X]`/`[no-X]`/`[loading]`/`[error]` data states, `[primary]`/`[destructive]` action weight, role names (`[admin]`) permissions, domain predicates (`[fulfilled]`) visibility conditions, `[contains:AppName]` cross-app composition. They compose. |
| **Sub-machine** | Region with `states` block. No marker needed. |
| **Nested regions** | Region with `regions` block. Cap at 1 level of nesting. Deeper nesting signals a Region should be its own Screen. |
| **Ambient events** | Keyboard shortcuts appear in state machines without a Region declaring them |
| **Bubbling** | Unhandled events bubble Sub-Region → Region → Screen → App automatically |
| **Tags vs descriptions** | Tags for external conditions (data presence, user role, feature flag). Description prose for internal state references ("appears in selecting state"). |
| **Action weight** | Optional `[primary]`, `[secondary]`, `[destructive]` on event-emitting Regions. Behavioral priority, not visual styling. |
| **Confirmation dialogs** | Overlay Region with confirm/cancel events. Parent state machine controls the flow (e.g., `viewing → deleting → viewing`). |
| **Flow naming** | Navigation: `EntityFromSource`. Action: verb-led (`RefundPayment`, `CancelSubscription`, `ComposeAndSend`). Unhappy path: `VerbFailed` or `FailedEntity`. Domain language wins over rigid templates. |
| **Paired flows** | Happy/unhappy variants share the same `on:` trigger or reference each other by naming (e.g., `RefundPayment` / `RefundFailed`). |

### Reading SFT as Given/When/Then

State machines map directly to acceptance criteria format:

```
Given [from state], When [event], Then [to state or action]
```

Example: `on: check-payment, from: browsing, to: selecting` reads as:
> Given the user is **browsing**, when they **check a payment**, then the screen enters **selecting** mode.

## Scope

| SFT covers | Use instead |
|-----------|-------------|
| Screens and their regions | |
| Events regions emit | |
| State machines (interaction modes) | |
| Flows across screens | |
| Conditional existence (tags) | |
| Components bound to screens/regions | |
| | Component internals (button variants, form fields) → **design system** |
| | Visual hierarchy (typography, spacing, color) → **Figma** |
| | Form validation rules → **acceptance criteria in PRD** |
| | Content structure (what text says) → **content spec** |
| | Responsive breakpoints → **separate SFT file per viewport** |
| | Transition guards beyond current-state (`from:`) → **acceptance criteria in PRD** |
| | Time-dependent behavior (auto-dismiss, session timeout) → **acceptance criteria or Region description** |

## YAML Schema

```
app:
  name, description
  regions: [{ name, description, tags?, events?, regions?, states?,
              component?, props?, on_actions?, visible? }]
  screens:
    [{ name, description, tags?,
       component?, props?, on_actions?, visible?,
       regions: [{ name, description, tags?, events?, regions?, states?,
                   component?, props?, on_actions?, visible? }],
       states? }]
  flows: [{ name, description?, on?, sequence }]

states:               # can appear at app, screen, or region level — list of transitions
  [{ on, from?, to?, action? }]
              # on: may reference ambient events (keyboard shortcuts, system events) without a Region declaring them
              # valid states inferred from from/to values; first from is initial state
```

Multi-app: `app:` accepts a list of apps. Cross-app: `[contains:AppName]` tag on Region.

## Examples

- [`gmail.sft.yaml`](./examples/gmail.sft.yaml) — Email. Inbox browsing/selecting, ReplyComposer sub-machine with `emit`, compose overlay.
- [`linear.sft.yaml`](./examples/linear.sft.yaml) — Project management. Issue list/board/cycle, DescriptionEditor sub-machine, keyboard shortcuts.
- [`shopify.sft.yaml`](./examples/shopify.sft.yaml) — E-commerce. Two apps, cross-app `contains`, nested FulfillmentArea, VariantEditor sub-machine.
- [`stripe.sft.yaml`](./examples/stripe.sft.yaml) — Payments. Nested RefundArea, `emit` for evidence completion, data-conditional states, unhappy path flows.
- [`bank.sft.yaml`](./examples/bank.sft.yaml) — Banking. Conditional display, wizards, multi-step flows.
- [`docs.sft.yaml`](./examples/docs.sft.yaml) — Documentation. All 36 UI patterns covered with 14 keywords.

Try them:
```bash
sft import examples/stripe.sft.yaml
sft show
sft validate
sft query flows
sft export /tmp/roundtrip.yaml
sft diff examples/stripe.sft.yaml   # should show "no changes"
```

## Quick Reference

`ScreenName` PascalCase · `event-name` kebab-case · `[tag]` bracket annotation · Region under App = persistent · Region under Screen = scoped · Regions nest 1 level deep · Unhandled events bubble up (Sub-Region → Region → Screen → App) · `emit(event-name)` sends named event to parent · Independent sub-machines run in parallel.
