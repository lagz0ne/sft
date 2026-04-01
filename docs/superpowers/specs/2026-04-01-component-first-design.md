# Component-First Design: json-render + shadcn/ui

## The Shift

Components are the primary unit of work. Screens are pure composition. The component catalog is the design system — swappable for discovery, experiments, and themes.

**Before:** 8 hardcoded wireframe shapes. Components are labels on regions.
**After:** shadcn/ui as the default catalog. Custom components compose shadcn primitives. json-render renders the spec. Catalog swap = theme change.

## Architecture

```
SFT spec (YAML)
  → regions declare component, props, and child regions
  → loader resolves bindings + validates component names against the pinned registry
  → template compiler produces cached registry modules
  → viewer loads compiled modules and applies typed experiments
```

### Three catalog layers

| Layer | What | When defined |
|-------|------|-------------|
| **shadcn/ui base** | Button, Card, Input, Table, Tabs, Avatar, Badge, Dialog... | Built-in registry manifest — no YAML needed |
| **App-specific** | recipe-card, metric-widget, email-row — compose shadcn primitives | Defined in YAML `catalog:` when needed (YAGNI) |
| **Theme/experiment** | Overlay that changes how any component renders | Defined in `experiments:` for PM review |

Most specs start with zero catalog YAML. shadcn is the default. Add custom components only when you need a domain-specific composition.

### Built-in Registry

Each project ships a pinned static JSON manifest generated from `@json-render/shadcn`. That manifest defines the built-in component names and prop schemas that exist before any `app.catalog` YAML is loaded.

Rules:

- The manifest is generated from the exact installed `@json-render/shadcn` package version.
- The manifest is pinned per project so validation, viewer behavior, and exports are deterministic.
- Custom catalog entries layer on top of the built-in registry; they do not replace it.
- Unknown component names fail validation immediately instead of falling through to best-effort rendering.

## Format

### YAML-native props

`props:` is authored as plain YAML. Maps, arrays, numbers, booleans, and strings stay YAML-native all the way through authoring. Never encode nested props or component structures as JSON strings inside YAML.

### Binding Grammar

Bindings use `$entity.field.path` syntax.

Rules:

- Start with `$`, then read the entity name from the first segment.
- Split the remaining path on dots and walk the resolved entity data segment by segment.
- Arrays use zero-based `[index]` access on the current value.
- Missing entities, missing fields, or out-of-range indexes are load-time errors.

Examples:

- `$pho.author.name`
- `$pho.ingredients[0].name`

Resolution happens at load time, not at render time.

### Composition Model

Component composition uses exactly two authoring primitives:

| Primitive | YAML authoring | Use for |
|-----------|----------------|---------|
| **Region children** | `regions:` | Ordered structural composition |
| **Named slots** | Component-valued props | Semantic slots like `save_button`, `trailing_action`, `empty_state` |

Do not author nested component trees inside `props.children`. If a component needs structural children, nest child regions under `regions:`. If it needs a named slot, pass a component-valued prop. Scalar leaf values such as `children: Save` are still plain props, not a third composition primitive.

### Using shadcn components directly (no catalog entry needed)

```yaml
app:
  entities:
    metrics:
      revenue: $2.4M

screens:
  - name: dashboard
    entry: true
    description: Overview
    regions:
      - name: header
        component: div
        props:
          className: flex items-center justify-between p-4 border-b
        regions:
          - name: title
            component: div
            props:
              className: text-xl font-semibold
              children: Dashboard
          - name: search
            component: Input
            props:
              placeholder: Search...
              type: search

      - name: kpi_strip
        component: div
        props:
          className: grid grid-cols-4 gap-4 p-4
        regions:
          - name: revenue
            component: Card
            regions:
              - name: revenue_header
                component: CardHeader
                regions:
                  - name: revenue_title
                    component: CardTitle
                    props:
                      children: Revenue
              - name: revenue_body
                component: CardContent
                regions:
                  - name: revenue_value
                    component: div
                    props:
                      className: text-2xl font-semibold
                      children: $metrics.revenue
```

shadcn types (`Card`, `Button`, `Input`, `Table`, `Badge`, `Avatar`, `Tabs`, `Dialog`) work out of the box. No catalog entry required.

### Defining app-specific components (when shadcn alone isn't enough)

```yaml
app:
  catalog:
    recipe-card:
      props:
        title: string
        image: string
        author_name: string
        author_avatar: string
        time: string
        save_button: component?
      template: |
        <Card className="overflow-hidden hover:shadow-md transition-shadow cursor-pointer">
          <div className="relative">
            <img src={props.image} className="w-full aspect-[4/3] object-cover" />
            {props.save_button && <div className="absolute top-2 left-2">{props.save_button}</div>}
          </div>
          <CardContent className="p-3">
            <h3 className="font-semibold text-sm line-clamp-2">{props.title}</h3>
            <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
              <Avatar src={props.author_avatar} size="xs" />
              <span>{props.author_name}</span>
              <span>·</span>
              <span>{props.time}</span>
            </div>
          </CardContent>
        </Card>

    metric-widget:
      props:
        label: string
        value: string
        trend: number?
        change: string?
      template: |
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">{props.label}</p>
            <p className="text-2xl font-bold">{props.value}</p>
            {props.change && (
              <p className={props.trend > 0 ? "text-green-600 text-xs" : "text-red-600 text-xs"}>
                {props.change}
              </p>
            )}
          </CardContent>
        </Card>
```

Custom components use shadcn primitives inside their templates. The template is JSX and becomes the real React component when you export to production.

### Binding entity fields into props

Region props can bind directly to entity data via `$entity.field.path`.

```yaml
app:
  entities:
    pho:
      _type: recipe
      title: Pho Bo Hanoi
      cover_image: /recipes/pho-bo.jpg
      author:
        name: Mai Nguyen
        avatar_url: /avatars/mai.jpg
      ingredients:
        - name: Rice noodles
        - name: Beef broth

  screens:
    - name: home
      entry: true
      regions:
        - name: feed
          component: div
          props:
            className: grid grid-cols-2 lg:grid-cols-3 gap-4 p-4
          regions:
            - name: card_pho
              component: recipe-card
              props:
                title: $pho.title
                image: $pho.cover_image
                author_name: $pho.author.name
                author_avatar: $pho.author.avatar_url
                time: 6h 30m
            - name: featured_ingredient
              component: Badge
              props:
                children: $pho.ingredients[0].name
```

`$pho.author.name` and `$pho.ingredients[0].name` both resolve before the component ever renders. If any segment is missing, the loader fails the spec instead of rendering a partial view.

### Named slot composition

A prop value can itself be another component. That component is authored as structured YAML and passed through the same component resolution pipeline.

```yaml
- name: card_pho
  component: recipe-card
  props:
    title: $pho.title
    image: $pho.cover_image
    author_name: $pho.author.name
    author_avatar: $pho.author.avatar_url
    time: 6h 30m
    save_button:
      component: Button
      props:
        children: Save
        variant: ghost
```

The component template renders it with `{props.save_button}`. This is slot composition, not structural composition: the slot stays explicit and named, and the screen still owns the data binding.

### Data boundary: components are pure

Components are pure functions of props. Screens and regions do the data wiring. Templates render only what arrives via `props`, including named slots and the structural children produced from region nesting.

| Level | Can reference | Cannot reference |
|-------|---------------|------------------|
| **Screen** | entities (`$pho`), fixtures, context | — |
| **Region props** | `$entity.field.path`, static values, component-as-prop | — |
| **Component template** | `props.*` only | entities, fixtures, ambient data, context, anything outside `props` |

This is the React mental model. Parent layers bind data into props. Catalog components never reach into the data layer directly.

## Typed Experiments

Experiments are discriminated objects with two allowed types:

| Type | Scope | Overlay shape |
|------|-------|---------------|
| `catalog_variant` | `catalog.component-name` or `catalog` | Template overlay |
| `region_patch` | `screen.region` | Patch over `component`, `props`, and/or `delivery` |

Rules:

- `catalog_variant` with `scope: catalog.component-name` overlays exactly one `template`.
- `catalog_variant` with `scope: catalog` overlays a map of component names, each with a `template`.
- `region_patch` may override only `component`, `props`, and `delivery` for the targeted region.

### `catalog_variant`: single component

```yaml
  experiments:
    horizontal_cards:
      type: catalog_variant
      scope: catalog.recipe-card
      description: "Horizontal layout — image left, text right"
      overlay:
        template: |
          <Card className="relative flex gap-3 p-2">
            {props.save_button && <div className="absolute top-2 right-2">{props.save_button}</div>}
            <img src={props.image} className="w-24 h-24 rounded-lg object-cover shrink-0" />
            <div className="flex flex-col justify-center min-w-0">
              <h3 className="font-semibold text-sm line-clamp-2">{props.title}</h3>
              <div className="flex items-center gap-1.5 text-xs text-gray-500 mt-1">
                <Avatar src={props.author_avatar} size="xs" />
                <span>{props.author_name} · {props.time}</span>
              </div>
            </div>
          </Card>
```

Toggle `horizontal_cards` in the viewer and every `recipe-card` on every screen flips to horizontal. One change, all instances.

### `catalog_variant`: full catalog swap

```yaml
    minimal_theme:
      type: catalog_variant
      scope: catalog
      description: "Text-only minimal — no images, no cards"
      overlay:
        recipe-card:
          template: |
            <div className="py-2 border-b flex items-center justify-between gap-3">
              <div className="min-w-0">
                <span className="font-medium text-sm">{props.title}</span>
                <span className="text-muted-foreground text-xs ml-2">{props.author_name} · {props.time}</span>
              </div>
              <div className="flex items-center gap-2">
                {props.save_button}
              </div>
            </div>
        metric-widget:
          template: |
            <div className="flex gap-2 text-sm">
              <span className="text-muted-foreground">{props.label}:</span>
              <span className="font-semibold">{props.value}</span>
            </div>
```

Toggle `minimal_theme` and the entire app switches to text-only. Every component, every screen. A full catalog swap is just a `catalog_variant` with `scope: catalog`.

### `region_patch`: region override

```yaml
    icon_nav:
      type: region_patch
      scope: dashboard.sidebar
      description: "Icon-only sidebar"
      overlay:
        component: div
        props:
          className: w-12 flex flex-col items-center gap-3 py-4
        delivery:
          classes:
            - icon-only
            - compact
```

Region-level patches still work for one-off layout changes. Catalog variants are the primary experiment surface.

## Discovery → Delivery Pipeline

```
1. Start with shadcn defaults        → functional, clean, no design effort
2. Add app-specific components       → recipe-card, email-row (YAGNI — only when needed)
3. Experiment with catalog variants  → PM compares themes/variants
4. Commit the winner                 → base catalog updated
5. Export catalog                    → production React components (.tsx files)
```

The same spec at every stage. Structure never changes. Only the catalog evolves.

## What changes in the SFT format

### New: `catalog:` under `app:`

```yaml
app:
  catalog:
    component-name:
      props:
        field: type
      template: |
        <JSX using shadcn primitives + props.field references>
```

### Changed: `props:` are YAML-native

Author props as structured YAML:

```yaml
props:
  label: Send
  variant: primary
```

### New: `$entity.field.path` inside region props

```yaml
props:
  title: $pho.title
  author_name: $pho.author.name
  featured_ingredient: $pho.ingredients[0].name
```

### New: composition model

```yaml
- name: parent
  component: Card
  regions:
    - name: body
      component: CardContent
      regions:
        - name: copy
          component: div
          props:
            children: Body copy
  props:
    trailing_action:
      component: Button
      props:
        children: Save
        variant: ghost
```

### Changed: `component:` on regions

Previously: one of 8 hardcoded types (`text`, `button`, `image`...).
Now: any shadcn type (`Card`, `Button`, `Input`, `Badge`...) or any custom catalog type (`recipe-card`, `metric-widget`...) or `div` for pure layout containers.

### Changed: component data access

Previously: components were effectively thin labels on regions, so data access rules were loose.
Now: components are pure render functions. The screen/region layer binds data into props, structural children come from `regions:`, and templates read only what arrives through `props`.

### New: built-in registry

Every project pins a generated JSON manifest from `@json-render/shadcn`. Unknown component names fail validation against that manifest plus the custom catalog.

### Changed: `experiments:` shape

Previously: `scope: screen.region` only and overlays were effectively untyped.
Now: experiments are either `catalog_variant` (`scope: catalog.component-name` or `scope: catalog`) or `region_patch` (`scope: screen.region`), with overlay keys validated by type.

## Implementation Notes

### Viewer

- Replace the 8 hardcoded shape components with `@json-render/react` + `@json-render/shadcn`
- The viewer's `<Renderer>` receives the spec as a json-render spec, with the built-in registry manifest plus the compiled custom catalog as the registry
- Experiment toggle in dock switches which typed experiment overlay is active
- The viewer consumes compiled registry modules; it never parses raw template strings during view rendering

### Template compilation

- Catalog templates compile at `sft init` / `sft validate` time, not at render time
- Compiled output is cached by content hash so unchanged templates reuse the same module
- Syntax errors fail init/validate instead of surfacing only when a screen is opened
- The viewer loads the compiled registry modules produced by that cache

### CLI

- Project init pins the generated shadcn built-in registry manifest for the current project
- `sft add catalog <name>` — add a component to the catalog
- `sft set catalog <name> --template "..."` — update template
- `sft set catalog <name> --props "field:type,field:type"` — update prop schema
- Existing region component commands should store structured props, not JSON-in-YAML strings

### Validator

- `component-not-in-catalog` (error): region references a type missing from the pinned shadcn registry and custom catalog
- `component-prop-unknown` (error): checks props against the built-in manifest or custom catalog schema
- `experiment-scope-invalid` (error): validates `catalog.component-name`, `catalog`, and `screen.region` scopes against experiment type
- `experiment-overlay-invalid` (error): validates `catalog_variant` vs `region_patch` overlay shapes
- `template-compile-failed` (error): template failed to compile during init/validate

### Export

- `sft export components` — extract catalog entries as `.tsx` files
- Each custom component becomes a standalone React component importing from shadcn/ui
- Props interface generated from the schema
- Template becomes the JSX body
