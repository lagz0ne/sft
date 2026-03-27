# SFT Playground Manual

The playground is an interactive wireframe viewer for SFT behavioral specs. It renders screens, regions, state machines, and flows as spatial wireframes that PMs and designers can walk through with stakeholders.

`sft view` opens the playground at `http://localhost:51741`.

## Three Independent Axes

The playground separates three concerns. Each can change without affecting the others.

### 1. Tags — Layout (where things sit)

Tags are explicit layout instructions on regions. Tailwind-like colon syntax.

**Position tags** — which grid slot the region occupies:

```
sidebar           header            toolbar
footer            bottomnav         aside
overlay           modal             drawer
banner            split
```

**Modifiers** — size variants, colon-separated:

```
sidebar:narrow    sidebar:wide
aside:narrow      aside:wide
split:narrow      split:wide
```

**Visual** — styling without changing position:

```
elevated          # shadow, no border, no header label
```

**Compositions** — alternative layouts, prefixed by name:

```
mobile:bottomnav          # in "mobile" composition, this region is bottomnav
tablet:sidebar:narrow     # in "tablet" composition, narrow sidebar
```

A region can have multiple tags for different compositions:

```yaml
tags: [sidebar, mobile:bottomnav, tablet:sidebar:narrow]
```

Default (unprefixed) tags apply when no composition is selected. The playground discovers compositions from tag prefixes and shows them in the dock.

**CLI:**

```bash
sft add tag sidebar --on main_nav
sft add tag mobile:bottomnav --on main_nav
sft add tag elevated --on compose_button
sft rm tag sidebar --on main_nav
```

### 2. Component — Content (what each region renders as)

Component bindings declare what a region IS — using json-render component types and props.

```bash
sft component search_input Input --props '{"placeholder":"Search..."}'
sft component email_list Table --props '{"columns":["sender","subject","date"]}'
sft component submit_btn Button --props '{"variant":"primary","label":"Sign up"}'
sft component hero_image Image --props '{"aspect":"video"}'
sft component volume Slider --props '{"label":"Volume"}'
sft component remember Toggle --props '{"label":"Remember me"}'
sft component category Select --props '{"options":["Rock","Jazz","Pop"]}'
```

The playground maps any json-render component type to one of 8 wireframe shapes:

| Shape | Component types | What it renders |
|-------|----------------|-----------------|
| input | Input, Textarea, Slider | Label + input box (or slider track) |
| select | Select, Checkbox, Radio, Toggle | Label + dropdown/toggle |
| button | Button, ButtonGroup | Button rectangle with label |
| image | Image, Avatar | Gray rectangle with landscape icon |
| text | Text, Heading, Badge, Alert | Heading bar + body lines |
| list | Table, Stack | Data rows from fixtures |
| card | Card, Grid | Placeholder cards |
| tabs | Tabs, Accordion | Horizontal pills |

Props enhance the wireframe — `placeholder` shows in the input box, `label` shows above it, `variant:"primary"` makes the button dark-filled, `aspect:"square"` changes the image ratio.

**No component set** → the region shows a prompt: `sft component region_name Type`

### 3. Component Set — Implementation (how it renders)

Component sets are named rendering implementations. Same spec, different visual treatment.

The playground ships with three built-in sets:

- **wireframe** — gray shapes, no color, structural only
- **styled** — colors, shadows, closer to production
- **compact** — tighter spacing, smaller elements

Switch sets in the dock. The set changes how ALL component shapes render without changing what they are.

In production, component sets map to json-render registries — `@json-render/react` (shadcn), `@json-render/react-native`, `@json-render/vue`, etc. Same spec, different platform.

## The Dock

A compact floating control bar at the bottom of the viewport.

```
[layers][play] | home +2 | browsing | [full][desktop][tablet][phone] | default mobile | wireframe styled compact
```

| Segment | What it controls |
|---------|-----------------|
| Mode (layers/play icons) | Screen mode vs Flow mode |
| Screen | Which screen is displayed |
| State | Which state the screen is in (changes region visibility) |
| Viewport | Canvas width constraint + linked composition |
| Layout | Which composition is active (default, mobile, tablet) |
| Component set | wireframe, styled, compact |

**Viewport ↔ composition linking:** Clicking the phone icon (375px) auto-switches to the "mobile" composition. Clicking full width switches back to "default". The viewport constraint and the layout composition are the same gesture.

**Overflow:** When a segment has 5+ items, it shows the active item + a chevron. Click to open an upward popover with all options.

**Flow mode:** The state segment is replaced by a flow step strip with prev/next navigation.

## Progressive Fidelity

The playground renders whatever information exists in the spec. Early specs show less; mature specs show more.

| Stage | What the playground shows |
|-------|--------------------------|
| Screens + regions only | Empty boxes with names, layout from tags |
| + component bindings | Wireframe shapes (inputs, buttons, lists, images) |
| + fixtures | Real data fills the shapes (sender names, subjects, dates) |
| + attachments | Mockup images replace wireframe shapes |
| + compositions | Alternative layouts switchable in the dock |
| + component sets | Different visual treatments |

## Spec Lifecycle

```
Discovery       → rough screens, regions, tags
Refinement      → events, state machines, flows, fixtures
Component       → sft component bindings, wireframe shapes
Composition     → mobile/tablet alternative layouts via tag prefixes
Polish          → component sets, attachments
Handoff         → sft render outputs json-render JSON for any platform
```

At every stage, the playground shows the current fidelity. Nothing is thrown away — each layer builds on what stuck.

## Key Commands

```bash
# Build and view
go build ./cmd/sft
sft import examples/spotify.sft.yaml
sft view

# Layout tags
sft add tag sidebar --on nav_panel
sft add tag split:wide --on content
sft add tag mobile:bottomnav --on nav_panel

# Component bindings
sft component search Input --props '{"placeholder":"Search..."}'
sft component list Table
sft component hero Image --props '{"aspect":"video"}'

# Inspect
sft show                    # text tree
sft show --json             # full JSON
sft render                  # json-render output
sft query tags              # all tags
sft component region_name   # read component binding
```

## Architecture

```
Browser (React SPA)
  ↓ WebSocket (nats.ws)
Embedded NATS Server
  ↓ request sft.spec / sft.render
Go HTTP Server (port 51741)
  ↓ show.Load(db) / render.FromSFT(spec)
SQLite DB (.sft/db)
```

The SPA is embedded in the Go binary via `//go:embed`. Single binary, single port, no external dependencies.

Tags are stored in the `tags` table. Component bindings in the `components` table. Both flow through the spec JSON to the browser via NATS. The playground reads `region.tags` for layout and `region.component` + `region.component_props` for wireframe shapes.
