# SFT View Rebuild — Design Spec

## Problem

The current view (wireframe-canvas.tsx, ~900 lines) grew through accumulated patches. It has two bolted-on modes (wireframe vs styled) that fight each other. The spec-to-visual pipeline has gaps at every layer.

## Core Insight

**Fidelity is a spectrum, not a toggle.** A spec author progressively adds detail:

1. **Structure** — regions + events + tags → you see position zones and region names
2. **Content** — components + props + fixtures → you see shaped content (inputs, images, text, lists) with real data
3. **Style** — delivery_classes → you see Tailwind-styled production layout
4. **Implementation** — delivery_component → you see real React components

The view should render whatever fidelity the spec has reached. No mode switch needed — a region with only tags renders as a wireframe box; the same region with delivery_classes renders with those styles; with delivery_component it renders as the real component.

## Golden Examples

Two specs drive the rebuild — both must work perfectly at all fidelity levels:

| Spec | App type | Tests |
|------|----------|-------|
| **gmail** | Utility (state-heavy) | State machines, fixture switching, overlay compose, reading pane toggle, multi-screen nav |
| **tinhte** | Content (layout-heavy) | Infinite scroll, card grids, sidebar ads, story carousel, section blocks, fixture-driven article feed |

## Spec Authoring Workflow

```
1. Define structure
   sft add screen inbox "Primary email list"
   sft add region email_list "Thread list" --in inbox
   sft add tag main --on email_list
   sft add event select_email(email) --in email_list
   → View shows: zone grid with labeled region boxes

2. Add content shape
   sft component email_list list
   sft add type email "subject:string, sender:contact, date:datetime, read:boolean"
   sft set context emails email[] --on inbox
   → View shows: list shape with column headers from type

3. Add fixture data
   sft set fixture inbox_full --data '{"inbox": {"emails": [...]}}'
   sft add state-fixture inbox_full --in inbox --state browsing
   → View shows: list populated with real email subjects and senders

4. Add delivery styling
   delivery_classes get set on regions via YAML or CLI
   → View shows: Tailwind-styled layout, no wireframe chrome

5. Bind production component
   delivery_component maps to real React component
   → View shows: actual production UI
```

## YAML Format Reference

```yaml
app:
  name: my_app
  description: "..."

  enums:
    status: [active, archived]

  data:
    article:
      title: string
      author: author
      date: datetime
    author:
      name: string
      avatar: string

  context:
    current_user: author?

  regions:  # app-level, shown on all screens
    - name: header
      tags: [header]  # wireframe position
      description: "..."
      delivery:
        classes: [sticky, top-0, bg-white, ...]  # styled position
      regions:  # children
        - name: logo
          component: text  # LOWERCASE
          props: '{"content": "My App", "level": 1}'  # props: NOT component_props:
          delivery:
            classes: [text-xl, font-bold]

  screens:  # UNDER app:, indented
    - name: home
      description: "..."
      context:
        articles: article[]
      state_machine:
        browsing:
          fixture: home_full  # binds fixture data
          on:
            select: navigate(detail)
      regions:
        - name: article_list
          tags: [main]  # wireframe position
          component: list  # lowercase
          ambient:
            articles: "data(home, .articles)"  # fixture binding
          delivery:
            classes: [space-y-3]

  fixtures:
    home_full:
      home:
        articles:
          - { title: "...", author: { name: "..." }, date: "2026-03-31" }
```

## Key Rules

1. **props: not component_props:** — the Go loader field name
2. **Lowercase component names** — text, image, button, card, input, select, list, tabs
3. **Containers don't get components** — only leaf regions. Container regions render their children.
4. **Tags for wireframe, delivery for styled** — same region, two rendering paths
5. **Flat screen children** — no wrapper divs. Direct children with tags for zone grid.
6. **Data first** — verify props reach the DB before tweaking renderers
7. **Fixtures bind via state_machine** — `state_machine: { browsing: { fixture: name } }`

## View Rebuild Architecture

### Renderer Pipeline

```
Region data
  → Fidelity detection (what does this region have?)
  → Select renderer:
      has delivery_component? → Real component
      has delivery_classes?   → Styled div with Tailwind
      has component + props?  → Shaped wireframe with content
      has tags only?          → Zone-positioned box with name label
```

### Layout Strategy

The outer layout adapts to fidelity too:
- **Structure/Content fidelity** → zone-based CSS grid (position tags drive placement)
- **Style/Production fidelity** → flat flow (delivery_classes drive placement)

Detection: if >50% of top-level regions have delivery_classes → flat flow. Otherwise → zone grid.

### Component Shapes

Shapes render real content from props/fixtures:
- `text` → actual text string at appropriate heading level
- `image` → gray placeholder with alt text and aspect ratio
- `button` → styled pill with label
- `input` → input field with placeholder
- `card` → stacked items list from props.items
- `list` → DataList auto-detecting columns from fixture data type
- `select` → dropdown with first option shown
- `tabs` → pill tabs from enum values

### Dock Controls

Minimal:
- Screen picker (Picker popover, handles many screens)
- State selector (for state machine switching → fixture changes)
- Viewport size (responsive preview)
- Fidelity override (only if needed — auto-detection should work)
