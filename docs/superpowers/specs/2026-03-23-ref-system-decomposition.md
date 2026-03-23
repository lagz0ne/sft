# SFT Phase 6 — Ref System + Top-Down Decomposition

**Date:** 2026-03-23
**Status:** Draft

## Problem

SFT mutations require verbose, ambiguous entity addressing (`sft set region email_list --in inbox`). Region names collide across parents. There's no stable identity for chaining commands. And the component system is a flat overlay — no variants, no tokens, no slots.

## Two changes, one spec

1. **Ref system** — every entity gets a stable `@ref` exposed in all output, usable in all commands
2. **Top-down decomposition** — regions gain component references, token references, and slot mappings

These are one spec because refs make decomposition usable. Without refs, component/token/slot bindings require verbose `--in` addressing. With refs, it's `sft set @r2 --component List:scrollable`.

## Breaking changes

- **Remove `sft import`** — CLI is the sole write path. Database is source of truth.
- **Remove `sft export`** — no round-trip. Snapshots are views, not sources.
- **`sft init <file.yaml>`** — one-time bootstrap from YAML. Only works on empty DB. Replaces `import` for migration.

## 1. Ref System

### Format

Every entity gets a ref: `@` + type prefix + database ID.

| Prefix | Entity | Example |
|---|---|---|
| `s` | screen | `@s1` |
| `r` | region | `@r5` |
| `e` | event | `@e12` |
| `t` | transition | `@t3` |
| `f` | flow | `@f2` |
| `k` | token | `@k1` |
| `c` | component type | `@c4` |

Refs are deterministic — they're the database primary key with a type prefix. They don't change when entities are renamed or moved.

### Output

Every command that outputs entities includes refs:

```
$ sft show
gmail — Google's email client
  @s1 inbox — Primary email list with category tabs
    @r1 category_tabs — Tab filter
      @e1 switch_category
    @r2 email_list — Scrollable thread list
      @e2 select_email(email)
      @e3 check_email(email)
      @e4 escape
    @r3 reading_pane — Preview
    @r4 bulk_action_bar — Bulk actions
      @e5 archive_selected
      @e6 delete_selected
  @s2 thread_view — Conversation view
    @r5 message_list
    @r6 reply_composer
```

`sft show --json` includes `"ref": "@s1"` on every entity.

`sft query` results include refs:

```
$ sft query screens
@s1  inbox        4 regions  2 states
@s2  thread_view  2 regions  1 state
@s3  settings     3 regions
```

`sft validate` warnings include refs:

```
⚠ @r4 bulk_action_bar: token $space-4 not declared
⚠ @e4 escape: unhandled by any transition
✗ @r2 email_list: slot 'item' references EmailRow — not found
```

### Input

Every command that takes an entity accepts a ref:

```bash
sft set @r2 --component List:scrollable
sft rm @e4
sft mv @r4 --to @s2
sft set @s1 --description "Updated inbox"
sft add region header "Top bar" --in @s1
sft add event tap_send --in @r6
sft add transition --on @e2 --in @s1 --action "navigate(@s2)"
```

Refs work anywhere an entity name is accepted. The CLI resolves the ref to (entity_type, entity_id) before executing.

### Implementation

Minimal — refs are a presentation/parsing concern:

- **Output:** format functions prepend `@{prefix}{id}` before entity names
- **Input:** argument parser detects `@` prefix, looks up entity type + ID from the prefix letter, resolves to (type, id)
- **No schema change** — refs are computed from existing primary keys

## 2. Top-Down Decomposition

Four layers, each independently valuable.

### Layer 1: Component reference

Regions gain a component reference with optional variant.

```bash
sft set @r2 --component List:scrollable
sft set @r4 --component ActionBar:contextual
sft set @r6 --component TextEditor:rich
```

**What changes:**
- The existing `components.component` field already stores a type name. It now supports `Type:variant` format.
- `sft show` displays the component reference inline:
  ```
  @r2 email_list [List:scrollable] — Scrollable thread list
  ```

**Validation:** Warn if variant is used but not declared. Component references are informational — they don't need to resolve to a registry (yet). The name IS the handoff.

### Layer 2: Token references in props

Props can reference tokens by `$name`.

```bash
sft set @r2 --prop spacing=$space-4
sft set @r2 --prop divider=$color-border
```

**What changes:**
- The existing `components.props` JSON field stores props. Props with `$` prefix are token references.
- `sft show` displays props with token refs:
  ```
  @r2 email_list [List:scrollable] — Scrollable thread list
    props: spacing=$space-4, divider=$color-border
  ```

**Validation:** Warn if `$token-name` is referenced but not declared. Warning, not error — top-down authoring means you reference before declaring.

### Layer 3: Token declarations

App-level named values, like enums.

```bash
sft add token space-4 "16px"
sft add token color-primary "#1a73e8"
sft add token color-border "#dadce0"
sft add token radius-md "8px"
```

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS tokens (
  id       INTEGER PRIMARY KEY,
  app_id   INTEGER NOT NULL REFERENCES apps(id),
  name     TEXT NOT NULL,
  value    TEXT NOT NULL,
  UNIQUE(app_id, name)
);
```

**What it enables:**
- Token refs in props resolve: `$space-4` → `16px`
- `sft query tokens` lists all declared tokens
- `sft validate` reports: declared-but-unused tokens, referenced-but-undeclared tokens
- `sft show` can optionally resolve tokens: `spacing=16px` vs `spacing=$space-4`

### Layer 4: Slots

Named composition points — what child region fills which slot in the parent's component.

```bash
sft set @r2 --slot item=@r7       # email_row region fills the 'item' slot
sft set @r2 --slot empty=@r8      # empty_state region fills the 'empty' slot
```

**What changes:**
- New column on regions or a join table: `slots` mapping slot names to child region refs
- `sft show` displays slots:
  ```
  @r2 email_list [List:scrollable]
    slots: item=@r7 email_row, empty=@r8 empty_state
    @r7 email_row [EmailRow]
    @r8 empty_state [EmptyState:inbox]
  ```

**Validation:** Slot target must be a child region of the entity. Warns if slot references a non-existent region.

## Query: `sft query interfaces`

New named query that projects the decomposition:

```
$ sft query interfaces
@r1 category_tabs
  component: TabBar
  events out: switch_category
  ambient in: active ← inbox.active_category

@r2 email_list
  component: List:scrollable
  props: spacing=$space-4, divider=$color-border
  events out: select_email(email), check_email(email), escape
  ambient in: emails ← inbox.emails
  slots: item=@r7, empty=@r8
  visible in: browsing, selecting, empty
  fixture: browsing→inbox_full, selecting→inbox_selecting
```

Pure SQL projection of existing + new data. Zero new logic.

## Architecture

### What's removed
- `internal/loader/` — YAML import (keep only for `sft init`)
- `sft import` command
- `sft export` command

### What's added
- Ref formatting in all output functions (show, query, validate, diff)
- Ref parsing in argument handling (detect `@` prefix, resolve to type+id)
- `tokens` table (Layer 3)
- `component:variant` parsing in `components.component` field (Layer 1)
- `$token` detection in `components.props` JSON (Layer 2)
- Slot storage — TBD: column on regions or join table (Layer 4)

### What's modified
- `sft init` — replaces `import`, only works on empty DB
- `components.component` field — now supports `Type:variant` format
- `components.props` field — now supports `$token` references
- Every output function — includes refs

## Principles

1. **CLI is the sole write path** — database is source of truth, not YAML
2. **Refs everywhere** — every entity, every output, every input
3. **Top-down decomposition** — reference before declaring, layers are independently valuable
4. **Validate at mutation time** — warnings on write, not batch-validate-after-import
5. **The spec IS the handoff** — decomposition + refs + tokens = everything a dev needs
