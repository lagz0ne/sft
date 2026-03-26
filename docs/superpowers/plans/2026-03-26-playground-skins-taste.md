# Playground Wireframe Skins + Taste System

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the SFT playground from labeled boxes into recognizable wireframes by deriving visual patterns from existing spec signals (ambient, events, data, enums), and add a taste system for visual treatment exploration.

**Architecture:** The skin renderer reads each region's signals (ambient type, event annotations, region_data fields, enums) through an 8-rule decision tree to select a visual pattern (DataList, Form, Tabs, DetailCard, ActionBar, Button, Search, Placeholder). Tastes are stored in the DB and switchable in the toolbar. Tags handle layout (WHERE). Signals handle content pattern (WHAT). Fixtures fill data. Taste controls style (HOW).

**Tech Stack:** React 19 + TanStack Router (SPA), Go + SQLite (backend), NATS (real-time), Tailwind CSS

---

## Parallel Execution Map

```
Track A (frontend skins)          Track B (Go taste backend)
  Task 1: types update              Task 7: taste DB + store
  Task 2: type resolver              Task 8: taste CLI commands
  Task 3: skin selector              Task 9: taste NATS handler
  Task 4: skin components
  Task 5: wire into canvas         Track C (taste frontend)
  Task 6: SPA build + verify        Task 10: taste types + toolbar
                                     Task 11: skins read taste
                                     Task 12: SPA build + verify

Track A and B are fully parallel.
Track C depends on B completing.
Task 6 and Task 12 each require Go rebuild.
```

---

## File Map

### Track A: Frontend Skins
| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `web/apps/web/src/lib/types.ts` | Add `ambient`, `region_data`, App `data_types`/`enums` to TS types |
| Create | `web/apps/web/src/lib/type-resolver.ts` | Parse event annotations, resolve ambient refs against data_types/enums |
| Create | `web/apps/web/src/lib/skin-selector.ts` | 8-rule decision tree: region signals → skin pattern |
| Create | `web/apps/web/src/components/skins/data-list.tsx` | List rows with typed columns, fixture data + gray placeholders |
| Create | `web/apps/web/src/components/skins/form-layout.tsx` | Labeled inputs from data field types, event-driven buttons |
| Create | `web/apps/web/src/components/skins/tabs.tsx` | Horizontal pills from enum values, active from ambient |
| Create | `web/apps/web/src/components/skins/detail-card.tsx` | Key-value display from single data type instance |
| Create | `web/apps/web/src/components/skins/action-bar.tsx` | Horizontal buttons from event names |
| Create | `web/apps/web/src/components/skins/action-button.tsx` | Single prominent button from event |
| Create | `web/apps/web/src/components/skins/search-input.tsx` | Search icon + text field |
| Create | `web/apps/web/src/components/skins/placeholder.tsx` | Gray lines from description text |
| Modify | `web/apps/web/src/components/wireframe-canvas.tsx` | Replace region body with skin-selected component |
| Modify | `web/apps/web/src/routes/playground.tsx` | Pass `app.data_types`, `app.enums` down to canvas |

### Track B: Go Taste Backend
| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/store/store.go` | `tastes` table, InsertTaste, UpdateTaste, GetTaste, ListTastes |
| Modify | `cmd/sft/main.go` | `sft add taste`, `sft set taste`, `sft query tastes` |
| Modify | `internal/view/server.go` | `sft.tastes` NATS handler |
| Modify | `internal/show/show.go` | Include tastes in Spec JSON |

### Track C: Taste Frontend
| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `web/apps/web/src/lib/types.ts` | `Taste` interface |
| Modify | `web/apps/web/src/routes/playground.tsx` | Taste switcher in toolbar |
| Modify | `web/apps/web/src/components/skins/*.tsx` | Read taste tokens for density, shape, colors |

---

## Track A: Frontend Skins

### Task 1: Update TypeScript types

**Files:**
- Modify: `web/apps/web/src/lib/types.ts`

- [ ] **Step 1: Add missing fields to Region**

The Go JSON includes `ambient`, `region_data` on regions and `data_types`, `enums` on the app. Add them.

```typescript
export interface Region {
  name: string
  description?: string
  tags?: string[]
  events?: string[]
  ambient?: Record<string, string>      // ADD: local name → "data(source, .query)"
  region_data?: Record<string, string>  // ADD: field name → type (e.g., "string", "draft")
  transitions?: Transition[]
  attachments?: string[]
  regions?: Region[]
  states?: string[]
  state_regions?: Record<string, string[]>
  state_fixtures?: Record<string, string>
}
```

Also update `App` to include `data_types` and `enums`:

```typescript
export interface App {
  name: string
  description: string
  data_types?: Record<string, Record<string, string>>  // ADD: type name → {field: type}
  enums?: Record<string, string[]>                     // ADD: enum name → values
  context?: Record<string, string>
  regions?: Region[]
  transitions?: Transition[]
}
```

Also update `Screen` to include `context`:

```typescript
export interface Screen {
  name: string
  description?: string
  tags?: string[]
  context?: Record<string, string>  // field name → type name
  attachments?: string[]
  regions?: Region[]
  transitions?: Transition[]
  states?: string[]
  state_regions?: Record<string, string[]>
  state_fixtures?: Record<string, string>
}
```

- [ ] **Step 2: Verify types compile**

Run: `cd web/apps/web && npx tsc --noEmit`
Expected: No errors (or only existing ones).

- [ ] **Step 3: Commit**

```bash
git add web/apps/web/src/lib/types.ts
git commit -m "feat(playground): add ambient, region_data, data_types, enums to TS types"
```

---

### Task 2: Type resolver module

**Files:**
- Create: `web/apps/web/src/lib/type-resolver.ts`

This module resolves event annotations and ambient refs against the app's type system.

- [ ] **Step 1: Create type-resolver.ts**

```typescript
/**
 * Resolves spec type annotations against app data_types and enums.
 *
 * Key operations:
 * - parseEventAnnotation("select_email(email)") → { name: "select_email", paramType: "email" }
 * - resolveType("email", dataTypes) → { kind: "data_type", fields: {subject: "string", ...} }
 * - resolveType("category", dataTypes, enums) → { kind: "enum", values: ["primary", ...] }
 * - parseAmbientRef("data(inbox, .emails)") → { source: "inbox", query: ".emails" }
 * - resolveAmbientType(ref, screenContext, dataTypes) → { isArray: true, baseType: "email", fields: {...} }
 */

export interface ParsedEvent {
  name: string
  paramType: string | null
}

export interface ResolvedType {
  kind: 'data_type' | 'enum' | 'scalar' | 'unknown'
  fields?: Record<string, string>   // for data_type
  values?: string[]                 // for enum
  isArray?: boolean
  isOptional?: boolean
}

export interface ParsedAmbientRef {
  source: string
  query: string
}

export interface ResolvedAmbient {
  isArray: boolean
  baseTypeName: string
  resolved: ResolvedType
}

const SCALARS = new Set(['string', 'number', 'boolean', 'date', 'datetime'])

export function parseEventAnnotation(event: string): ParsedEvent {
  const match = event.match(/^([^(]+)\(([^)]+)\)$/)
  if (!match) return { name: event, paramType: null }
  return { name: match[1], paramType: match[2] }
}

export function resolveType(
  typeName: string,
  dataTypes?: Record<string, Record<string, string>>,
  enums?: Record<string, string[]>,
): ResolvedType {
  // Strip array/optional markers
  let isArray = false
  let isOptional = false
  let base = typeName

  if (base.endsWith('[]?') || base.endsWith('?[]')) {
    isArray = true
    isOptional = true
    base = base.replace(/[\[\]?]/g, '')
  } else if (base.endsWith('[]')) {
    isArray = true
    base = base.slice(0, -2)
  } else if (base.endsWith('?')) {
    isOptional = true
    base = base.slice(0, -1)
  }

  if (SCALARS.has(base)) {
    return { kind: 'scalar', isArray, isOptional }
  }
  if (enums?.[base]) {
    return { kind: 'enum', values: enums[base], isArray, isOptional }
  }
  if (dataTypes?.[base]) {
    return { kind: 'data_type', fields: dataTypes[base], isArray, isOptional }
  }
  return { kind: 'unknown', isArray, isOptional }
}

export function parseAmbientRef(ref: string): ParsedAmbientRef | null {
  const match = ref.match(/^data\(([^,]+),\s*(.+)\)$/)
  if (!match) return null
  return { source: match[1].trim(), query: match[2].trim() }
}

export function resolveAmbientType(
  ambientValue: string,
  screenContext?: Record<string, string>,
  dataTypes?: Record<string, Record<string, string>>,
  enums?: Record<string, string[]>,
): ResolvedAmbient | null {
  const parsed = parseAmbientRef(ambientValue)
  if (!parsed || !screenContext) return null

  // Find the context field type for the query
  // query is like ".emails" or ".emails[0]" or ".active_category"
  let fieldName = parsed.query.replace(/^\./, '').replace(/\[\d+\]$/, '')
  const hasIndex = /\[\d+\]$/.test(parsed.query)

  const contextType = screenContext[fieldName]
  if (!contextType) return null

  const resolved = resolveType(contextType, dataTypes, enums)

  // If query has array index like [0], unwrap array to single instance
  if (hasIndex && resolved.isArray) {
    return {
      isArray: false,
      baseTypeName: contextType.replace(/\[\]?\??$/, ''),
      resolved: { ...resolved, isArray: false },
    }
  }

  return {
    isArray: resolved.isArray ?? false,
    baseTypeName: contextType.replace(/\[\]?\??$/, ''),
    resolved,
  }
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd web/apps/web && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add web/apps/web/src/lib/type-resolver.ts
git commit -m "feat(playground): type resolver for event annotations and ambient refs"
```

---

### Task 3: Skin selector (decision tree)

**Files:**
- Create: `web/apps/web/src/lib/skin-selector.ts`

- [ ] **Step 1: Create skin-selector.ts**

```typescript
import type { Region, App, Screen } from './types'
import { parseEventAnnotation, resolveType, resolveAmbientType, type ResolvedAmbient } from './type-resolver'

export type SkinType =
  | 'data-list'
  | 'form'
  | 'tabs'
  | 'detail-card'
  | 'action-bar'
  | 'action-button'
  | 'search-input'
  | 'placeholder'

export interface SkinContext {
  skin: SkinType
  /** Resolved ambient type info (for data-list, detail-card, tabs) */
  ambientType?: ResolvedAmbient | null
  /** Event annotation that matched a type */
  matchedEvent?: string
  /** Enum values (for tabs) */
  enumValues?: string[]
  /** Data type fields (for data-list, detail-card, form) */
  fields?: Record<string, string>
}

export function selectSkin(
  region: Region,
  app: App,
  screen?: Screen,
): SkinContext {
  const events = region.events ?? []
  const ambient = region.ambient ?? {}
  const regionData = region.region_data ?? {}
  const dataTypes = app.data_types
  const enums = app.enums
  const screenContext = screen?.context

  // Parse all event annotations
  const parsedEvents = events.map(e => parseEventAnnotation(e))

  // Resolve all ambient refs
  const resolvedAmbients: Record<string, ResolvedAmbient | null> = {}
  for (const [key, value] of Object.entries(ambient)) {
    resolvedAmbients[key] = resolveAmbientType(value, screenContext, dataTypes, enums)
  }

  // --- Rule 1: Data List ---
  // Ambient binds array T[] AND event annotated with T
  for (const [, resolved] of Object.entries(resolvedAmbients)) {
    if (!resolved || !resolved.isArray || resolved.resolved.kind !== 'data_type') continue
    const baseType = resolved.baseTypeName
    const matchingEvent = parsedEvents.find(e =>
      e.paramType && e.paramType === baseType
    )
    if (matchingEvent) {
      return {
        skin: 'data-list',
        ambientType: resolved,
        matchedEvent: matchingEvent.name,
        fields: resolved.resolved.fields,
      }
    }
  }

  // --- Rule 2: Form ---
  // region_data has 2+ fields
  const dataFields = Object.entries(regionData)
  if (dataFields.length >= 2) {
    // Expand composite type fields
    const expandedFields: Record<string, string> = {}
    for (const [fieldName, fieldType] of dataFields) {
      const resolved = resolveType(fieldType, dataTypes, enums)
      if (resolved.kind === 'data_type' && resolved.fields) {
        for (const [subField, subType] of Object.entries(resolved.fields)) {
          expandedFields[subField] = subType
        }
      } else {
        expandedFields[fieldName] = fieldType
      }
    }
    return { skin: 'form', fields: expandedFields }
  }

  // --- Rule 3: Tabs ---
  // Event annotation resolves to enum AND ambient refs current selection
  for (const pe of parsedEvents) {
    if (!pe.paramType) continue
    const resolved = resolveType(pe.paramType, dataTypes, enums)
    if (resolved.kind === 'enum' && resolved.values) {
      // Check if there's an ambient ref (active selection indicator)
      const hasAmbient = Object.keys(ambient).length > 0
      if (hasAmbient) {
        return {
          skin: 'tabs',
          enumValues: resolved.values,
          matchedEvent: pe.name,
        }
      }
    }
  }

  // --- Rule 4: Detail Card ---
  // Ambient binds single T (not array), no region_data
  if (dataFields.length === 0) {
    for (const [, resolved] of Object.entries(resolvedAmbients)) {
      if (!resolved || resolved.isArray) continue
      if (resolved.resolved.kind === 'data_type' && resolved.resolved.fields) {
        return {
          skin: 'detail-card',
          ambientType: resolved,
          fields: resolved.resolved.fields,
        }
      }
    }
  }

  // --- Rule 5: Action Bar ---
  // 2+ events, most without data type annotations
  if (events.length >= 2) {
    const unannotated = parsedEvents.filter(e => !e.paramType).length
    if (unannotated >= events.length / 2) {
      return { skin: 'action-bar' }
    }
  }

  // --- Rule 6: Action Button ---
  // Exactly 1 event
  if (events.length === 1) {
    return { skin: 'action-button', matchedEvent: parsedEvents[0].name }
  }

  // --- Rule 7: Search Input ---
  // Description contains "search"
  if (region.description?.toLowerCase().includes('search')) {
    return { skin: 'search-input' }
  }

  // --- Rule 8: Placeholder ---
  return { skin: 'placeholder' }
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd web/apps/web && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add web/apps/web/src/lib/skin-selector.ts
git commit -m "feat(playground): 8-rule skin selector decision tree"
```

---

### Task 4: Skin components

**Files:**
- Create: `web/apps/web/src/components/skins/data-list.tsx`
- Create: `web/apps/web/src/components/skins/form-layout.tsx`
- Create: `web/apps/web/src/components/skins/tabs.tsx`
- Create: `web/apps/web/src/components/skins/detail-card.tsx`
- Create: `web/apps/web/src/components/skins/action-bar.tsx`
- Create: `web/apps/web/src/components/skins/action-button.tsx`
- Create: `web/apps/web/src/components/skins/search-input.tsx`
- Create: `web/apps/web/src/components/skins/placeholder.tsx`

Each skin takes `SkinContext`, fixture data, and renders a wireframe pattern.

- [ ] **Step 1: Create shared skin props interface**

Create `web/apps/web/src/components/skins/types.ts`:

```typescript
import type { SkinContext } from '../../lib/skin-selector'
import type { Region } from '../../lib/types'

export interface SkinProps {
  region: Region
  context: SkinContext
  fixtureData?: Record<string, any> | null
  screenName?: string
  compact?: boolean
}
```

- [ ] **Step 2: Create data-list.tsx**

```typescript
import type { SkinProps } from './types'

export function DataListSkin({ region, context, fixtureData, screenName, compact }: SkinProps) {
  const fields = context.fields ?? {}
  const fieldNames = Object.keys(fields).filter(f => f !== 'id')

  // Resolve fixture array
  let items: any[] = []
  if (fixtureData && screenName) {
    const screenData = fixtureData[screenName]
    if (screenData) {
      // Find the array in screen data that matches ambient binding
      for (const val of Object.values(screenData)) {
        if (Array.isArray(val)) { items = val; break }
      }
    }
  }

  // Pick display columns: prioritize name-like, subject-like, then first 3
  const displayFields = pickDisplayFields(fieldNames, fields)
  const realCount = Math.min(items.length, 2)
  const placeholderCount = compact ? 1 : Math.max(3 - realCount, 1)

  return (
    <div className="flex flex-col">
      {/* Real rows from fixture data */}
      {items.slice(0, realCount).map((item, i) => (
        <div key={i} className={`flex items-center gap-2 px-2 py-1.5 ${i === 0 ? 'bg-blue-50/50' : ''} ${i < items.length - 1 ? 'border-b border-neutral-100' : ''}`}>
          {hasContactField(fields) && (
            <div className="w-5 h-5 rounded-full bg-blue-200 flex items-center justify-center text-[8px] font-bold text-blue-700 shrink-0">
              {getInitial(item, fields)}
            </div>
          )}
          <div className="flex-1 min-w-0">
            {displayFields.primary && (
              <div className={`${compact ? 'text-[9px]' : 'text-[10px]'} font-semibold text-neutral-700 truncate`}>
                {resolveFieldValue(item, displayFields.primary, fields)}
              </div>
            )}
            {displayFields.secondary && !compact && (
              <div className="text-[9px] text-neutral-400 truncate">
                {resolveFieldValue(item, displayFields.secondary, fields)}
              </div>
            )}
          </div>
          {displayFields.meta && (
            <div className="text-[8px] text-neutral-400 shrink-0">
              {resolveFieldValue(item, displayFields.meta, fields)}
            </div>
          )}
          {hasBooleanField(item) && (
            <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${getBooleanDot(item)}`} />
          )}
        </div>
      ))}

      {/* Placeholder rows */}
      {Array.from({ length: placeholderCount }).map((_, i) => (
        <div key={`ph-${i}`} className={`flex items-center gap-2 px-2 py-1.5 ${realCount + i < realCount + placeholderCount - 1 ? 'border-b border-neutral-100' : ''}`}>
          {hasContactField(fields) && (
            <div className="w-5 h-5 rounded-full bg-neutral-100 shrink-0" />
          )}
          <div className="flex-1">
            <div className={`h-[6px] bg-neutral-100 rounded-sm ${compact ? 'w-[50%]' : 'w-[45%]'} mb-1`} />
            {!compact && <div className="h-[5px] bg-neutral-50 rounded-sm w-[65%]" />}
          </div>
          <div className="h-[5px] w-8 bg-neutral-50 rounded-sm shrink-0" />
        </div>
      ))}
    </div>
  )
}

interface DisplayFields { primary?: string; secondary?: string; meta?: string }

function pickDisplayFields(fieldNames: string[], fields: Record<string, string>): DisplayFields {
  const result: DisplayFields = {}
  // Primary: name, subject, title
  result.primary = fieldNames.find(f => /^(name|subject|title)$/i.test(f)) ?? fieldNames[0]
  // Meta: date, datetime, created, updated
  result.meta = fieldNames.find(f => /date|time|created|updated/i.test(f))
  // Secondary: first remaining field that isn't primary/meta and isn't boolean
  result.secondary = fieldNames.find(f =>
    f !== result.primary && f !== result.meta && fields[f] !== 'boolean'
  )
  return result
}

function hasContactField(fields: Record<string, string>): boolean {
  return Object.values(fields).some(t => t === 'contact' || t === 'user')
}

function getInitial(item: any, fields: Record<string, string>): string {
  const contactField = Object.entries(fields).find(([, t]) => t === 'contact' || t === 'user')
  if (!contactField) return '?'
  const val = item[contactField[0]]
  if (typeof val === 'object' && val?.name) return val.name[0]?.toUpperCase() ?? '?'
  if (typeof val === 'string') return val[0]?.toUpperCase() ?? '?'
  return '?'
}

function resolveFieldValue(item: any, field: string, _fields: Record<string, string>): string {
  const val = item[field]
  if (val === null || val === undefined) return '—'
  if (typeof val === 'object' && 'name' in val) return val.name
  if (typeof val === 'boolean') return val ? 'Yes' : 'No'
  return String(val)
}

function hasBooleanField(item: any): boolean {
  return Object.values(item).some(v => typeof v === 'boolean')
}

function getBooleanDot(item: any): string {
  const boolEntry = Object.entries(item).find(([, v]) => typeof v === 'boolean')
  if (!boolEntry) return 'bg-neutral-200'
  return boolEntry[1] ? 'bg-neutral-200' : 'bg-blue-400'
}
```

- [ ] **Step 3: Create form-layout.tsx**

```typescript
import type { SkinProps } from './types'

export function FormSkin({ context, region, compact }: SkinProps) {
  const fields = context.fields ?? {}
  const events = region.events ?? []

  // Find submit/cancel events
  const submitEvent = events.find(e => e.includes('send') || e.includes('submit') || e.includes('save') || e.includes('confirm'))
  const cancelEvent = events.find(e => e.includes('cancel') || e.includes('discard') || e.includes('close'))

  return (
    <div className="flex flex-col gap-1.5">
      {Object.entries(fields).slice(0, compact ? 2 : 5).map(([name, type]) => (
        <div key={name} className="flex flex-col gap-0.5">
          <div className="text-[9px] text-neutral-500">{name.replace(/_/g, ' ')}</div>
          <FieldInput type={type} compact={compact} />
        </div>
      ))}

      {(submitEvent || cancelEvent) && (
        <div className="flex gap-1.5 mt-1 justify-end">
          {cancelEvent && (
            <div className={`${compact ? 'text-[8px] px-2 py-0.5' : 'text-[9px] px-3 py-1'} bg-neutral-100 text-neutral-500 rounded`}>
              {cancelEvent.split('(')[0].replace(/_/g, ' ')}
            </div>
          )}
          {submitEvent && (
            <div className={`${compact ? 'text-[8px] px-2 py-0.5' : 'text-[9px] px-3 py-1'} bg-neutral-700 text-white rounded`}>
              {submitEvent.split('(')[0].replace(/_/g, ' ')}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function FieldInput({ type, compact }: { type: string; compact?: boolean }) {
  const h = compact ? 'h-4' : 'h-5'
  const base = type.replace(/[?\[\]]/g, '')

  if (base === 'boolean') {
    return <div className="w-6 h-3 bg-neutral-200 rounded-full relative"><div className="absolute left-0.5 top-0.5 w-2 h-2 bg-white rounded-full" /></div>
  }
  if (type.endsWith('[]')) {
    return <div className={`${h} bg-neutral-50 border border-neutral-200 rounded flex items-center px-1.5`}><div className="text-[8px] text-neutral-400">+ add</div></div>
  }
  return <div className={`${h} bg-neutral-50 border border-neutral-200 rounded`} />
}
```

- [ ] **Step 4: Create tabs.tsx**

```typescript
import type { SkinProps } from './types'

export function TabsSkin({ context, compact }: SkinProps) {
  const values = context.enumValues ?? []

  return (
    <div className="flex items-center gap-1 overflow-x-auto">
      {values.map((val, i) => (
        <div
          key={val}
          className={[
            compact ? 'text-[8px] px-2 py-0.5' : 'text-[9px] px-3 py-1',
            'rounded-full shrink-0',
            i === 0 ? 'bg-neutral-700 text-white font-semibold' : 'bg-neutral-100 text-neutral-500',
          ].join(' ')}
        >
          {val}
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 5: Create detail-card.tsx**

```typescript
import type { SkinProps } from './types'

export function DetailCardSkin({ context, fixtureData, screenName, region, compact }: SkinProps) {
  const fields = context.fields ?? {}

  // Try to resolve fixture data for this region's ambient
  let item: Record<string, any> | null = null
  if (fixtureData && screenName) {
    const screenData = fixtureData[screenName]
    if (screenData) {
      // Find first non-array object value in screen data
      for (const val of Object.values(screenData)) {
        if (val && typeof val === 'object' && !Array.isArray(val)) {
          item = val; break
        }
      }
    }
    // Also check region-scoped fixture data
    const regionKey = `${screenName}.${region.name}`
    if (fixtureData[regionKey]) item = fixtureData[regionKey]
  }

  const fieldEntries = Object.entries(fields).filter(([f]) => f !== 'id')
  const titleField = fieldEntries.find(([f]) => /^(name|subject|title)$/i.test(f))
  const bodyField = fieldEntries.find(([f]) => /^(body|content|description|text)$/i.test(f))
  const metaFields = fieldEntries.filter(([f]) => f !== titleField?.[0] && f !== bodyField?.[0]).slice(0, 3)

  return (
    <div className="flex flex-col gap-1">
      {/* Title */}
      {titleField && item ? (
        <div className={`${compact ? 'text-[10px]' : 'text-xs'} font-semibold text-neutral-700`}>
          {resolveVal(item[titleField[0]])}
        </div>
      ) : (
        <div className={`${compact ? 'h-[7px] w-[55%]' : 'h-[8px] w-[60%]'} bg-neutral-200 rounded-sm`} />
      )}

      {/* Meta line */}
      {metaFields.length > 0 && (
        <div className="flex gap-2 text-[8px] text-neutral-400">
          {metaFields.map(([f]) => (
            <span key={f}>{item ? resolveVal(item[f]) : f.replace(/_/g, ' ')}</span>
          ))}
        </div>
      )}

      {/* Body / content lines */}
      {!compact && (
        <div className="flex flex-col gap-0.5 mt-1">
          {[100, 90, 75, 85, 60].map((w, i) => (
            <div key={i} className="h-[4px] bg-neutral-100 rounded-sm" style={{ width: `${w}%` }} />
          ))}
        </div>
      )}
    </div>
  )
}

function resolveVal(v: any): string {
  if (v == null) return '—'
  if (typeof v === 'object' && 'name' in v) return v.name
  return String(v)
}
```

- [ ] **Step 6: Create action-bar.tsx**

```typescript
import type { SkinProps } from './types'

export function ActionBarSkin({ region, compact }: SkinProps) {
  const events = region.events ?? []
  return (
    <div className="flex items-center gap-1.5">
      {events.map(event => {
        const label = event.split('(')[0].replace(/_/g, ' ')
        return (
          <div key={event} className={`${compact ? 'text-[8px] px-2 py-0.5' : 'text-[9px] px-3 py-1'} bg-neutral-100 text-neutral-600 rounded`}>
            {label}
          </div>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 7: Create action-button.tsx**

```typescript
import type { SkinProps } from './types'

export function ActionButtonSkin({ region, context, compact }: SkinProps) {
  const label = (context.matchedEvent ?? region.events?.[0] ?? 'action')
    .split('(')[0].replace(/_/g, ' ')
  const isDestructive = region.tags?.includes('destructive')

  return (
    <div className={[
      compact ? 'text-[9px] px-3 py-1' : 'text-[10px] px-4 py-1.5',
      'rounded font-medium text-center',
      isDestructive ? 'bg-red-600 text-white' : 'bg-neutral-700 text-white',
    ].join(' ')}>
      {label}
    </div>
  )
}
```

- [ ] **Step 8: Create search-input.tsx**

```typescript
import type { SkinProps } from './types'

export function SearchInputSkin({ compact }: SkinProps) {
  return (
    <div className={`flex items-center gap-1.5 ${compact ? 'px-1.5 py-0.5' : 'px-2 py-1'} bg-neutral-50 border border-neutral-200 rounded`}>
      <svg className="w-3 h-3 text-neutral-400" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
        <circle cx="6.5" cy="6.5" r="4.5" />
        <line x1="10" y1="10" x2="14" y2="14" />
      </svg>
      <div className={`flex-1 ${compact ? 'h-3' : 'h-4'}`} />
    </div>
  )
}
```

- [ ] **Step 9: Create placeholder.tsx**

```typescript
import type { SkinProps } from './types'

export function PlaceholderSkin({ region, compact }: SkinProps) {
  const desc = region.description ?? region.name.replace(/_/g, ' ')
  const isNav = /nav|sidebar|menu|folder|label/i.test(desc)

  if (isNav) {
    return (
      <div className="flex flex-col gap-1">
        {[85, 60, 70, 50].map((w, i) => (
          <div key={i} className={`flex items-center gap-1.5 ${compact ? 'py-0.5' : 'py-1'} ${i === 0 ? 'opacity-100' : 'opacity-50'}`}>
            <div className={`${compact ? 'w-2.5 h-2.5' : 'w-3 h-3'} bg-neutral-200 rounded-sm shrink-0`} />
            <div className="h-[5px] bg-neutral-200 rounded-sm" style={{ width: `${w}%` }} />
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-1">
      <div className={`${compact ? 'h-[6px] w-[50%]' : 'h-[7px] w-[55%]'} bg-neutral-200 rounded-sm`} />
      {!compact && (
        <>
          <div className="h-[4px] bg-neutral-100 rounded-sm w-[90%]" />
          <div className="h-[4px] bg-neutral-100 rounded-sm w-[80%]" />
          <div className="h-[4px] bg-neutral-50 rounded-sm w-[70%]" />
        </>
      )}
    </div>
  )
}
```

- [ ] **Step 10: Commit all skins**

```bash
git add web/apps/web/src/components/skins/
git commit -m "feat(playground): 8 wireframe skin components"
```

---

### Task 5: Wire skins into wireframe canvas

**Files:**
- Modify: `web/apps/web/src/components/wireframe-canvas.tsx`
- Modify: `web/apps/web/src/routes/playground.tsx`

- [ ] **Step 1: Update WireframeCanvasProps to accept app + screen**

In `wireframe-canvas.tsx`, add `app` and `screen` to the props so the skin selector can access `data_types`, `enums`, and `screen.context`:

Add to `WireframeCanvasProps`:
```typescript
interface WireframeCanvasProps {
  screen: Screen
  currentState: string | null
  appRegions?: Region[]
  fixtures?: Fixture[]
  activeRegion?: string | null
  activeEvent?: string | null
  app: App          // ADD
}
```

Add to `WireframeRegionProps`:
```typescript
interface WireframeRegionProps {
  region: Region
  depth: number
  visibleRegions: string[] | null
  fixtureData?: Record<string, any> | null
  activeRegion?: string | null
  activeEvent?: string | null
  compact?: boolean
  isOverlay?: boolean
  app: App            // ADD
  screen: Screen      // ADD
}
```

- [ ] **Step 2: Import skin selector and skin components in wireframe-canvas.tsx**

Add imports at top:

```typescript
import type { App, Fixture, Region, Screen } from '../lib/types'
import { selectSkin } from '../lib/skin-selector'
import { DataListSkin } from './skins/data-list'
import { FormSkin } from './skins/form-layout'
import { TabsSkin } from './skins/tabs'
import { DetailCardSkin } from './skins/detail-card'
import { ActionBarSkin } from './skins/action-bar'
import { ActionButtonSkin } from './skins/action-button'
import { SearchInputSkin } from './skins/search-input'
import { PlaceholderSkin } from './skins/placeholder'
```

- [ ] **Step 3: Add SkinRenderer function in wireframe-canvas.tsx**

Add a dispatch function that maps skin type to component:

```typescript
function SkinRenderer({ region, app, screen, fixtureData, compact }: {
  region: Region, app: App, screen: Screen,
  fixtureData?: Record<string, any> | null, compact?: boolean
}) {
  const ctx = selectSkin(region, app, screen)
  const props = { region, context: ctx, fixtureData, screenName: screen.name, compact }

  switch (ctx.skin) {
    case 'data-list': return <DataListSkin {...props} />
    case 'form': return <FormSkin {...props} />
    case 'tabs': return <TabsSkin {...props} />
    case 'detail-card': return <DetailCardSkin {...props} />
    case 'action-bar': return <ActionBarSkin {...props} />
    case 'action-button': return <ActionButtonSkin {...props} />
    case 'search-input': return <SearchInputSkin {...props} />
    case 'placeholder': return <PlaceholderSkin {...props} />
  }
}
```

- [ ] **Step 4: Replace region body content with SkinRenderer**

In the `WireframeRegion` function, replace the existing Description + FixturePreview + Events block with:

```typescript
{/* Skin-rendered content */}
{!effectiveHidden && (
  <SkinRenderer
    region={region}
    app={app}
    screen={screen}
    fixtureData={fixtureData}
    compact={compact}
  />
)}
```

Remove the old `FixturePreview`, `FixtureRow`, `formatVal` functions and the description/events rendering blocks. The skins handle all of that now.

Keep the region header (name + tag badges) and the nested regions rendering.

- [ ] **Step 5: Pass `app` and `screen` through the component tree**

In `WireframeCanvas`, pass `app` and `screen` to the `regionProps` spread:

```typescript
const regionProps = { visibleRegions, fixtureData, activeRegion, activeEvent, app, screen: effectiveScreen }
```

Where `effectiveScreen` is the `screen` prop. Add `app` to the `WireframeCanvasProps` destructuring.

In `WireframeRegion`, pass `app` and `screen` to recursive child renders.

- [ ] **Step 6: Update playground.tsx to pass `app`**

In `playground.tsx`, pass `app={spec.app}` to the `WireframeCanvas`:

```tsx
<WireframeCanvas
  screen={effectiveScreen}
  currentState={effectiveState}
  appRegions={spec.app.regions}
  fixtures={spec.fixtures}
  activeRegion={activeRegion}
  activeEvent={activeEvent}
  app={spec.app}
/>
```

- [ ] **Step 7: Commit**

```bash
git add web/apps/web/src/components/wireframe-canvas.tsx web/apps/web/src/routes/playground.tsx
git commit -m "feat(playground): wire skin selector into wireframe renderer"
```

---

### Task 6: Build and verify Track A

- [ ] **Step 1: Build the SPA**

```bash
cd web/apps/web && npm run build
```

Expected: Build succeeds, playground chunk includes skin imports.

- [ ] **Step 2: Rebuild Go binary**

```bash
cd /home/lagz0ne/dev/sft && go build ./cmd/sft
```

- [ ] **Step 3: Test with gmail spec**

```bash
cd /tmp/sft-playground-test && /home/lagz0ne/dev/sft/sft view
```

Open `http://localhost:51741/playground`. Verify:
- `email_list` renders as DataList with sender avatar + subject + date rows
- `category_tabs` renders as horizontal pills (Primary, Social, Promotions, Updates)
- `reading_pane` renders as DetailCard with title + text lines
- `compose_window` renders as Form with inputs + send/discard buttons
- `search_bar` renders as search input with magnifier icon
- `bulk_action_bar` renders as action buttons (archive, delete)
- `compose_button` renders as single button
- `main_nav` renders as nav placeholder with icon squares

- [ ] **Step 4: Commit verification**

```bash
git add -A && git commit -m "build: SPA rebuild with skin renderer"
```

---

## Track B: Go Taste Backend

### Task 7: Taste DB table + store methods

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add tastes table to schema migration**

Find the `ensureSchema` or migration function in `store.go` and add:

```sql
CREATE TABLE IF NOT EXISTS tastes (
  id INTEGER PRIMARY KEY,
  app_id INTEGER NOT NULL REFERENCES apps(id),
  name TEXT NOT NULL,
  tokens TEXT NOT NULL DEFAULT '{}',
  UNIQUE(app_id, name)
);
```

The `tokens` column stores JSON with structured design decisions:
```json
{
  "density": "compact",
  "shape": "rounded",
  "mode": "light",
  "accent": "#635bff",
  "surface": "#f7f6f3",
  "list": { "avatar": "initials", "divider": "line", "active": "fill" },
  "nav": { "icon": "square", "active": "fill" }
}
```

- [ ] **Step 2: Add CRUD methods**

```go
func (s *Store) InsertTaste(appID int64, name, tokens string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO tastes (app_id, name, tokens) VALUES (?, ?, ?)", appID, name, tokens)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateTaste(appID int64, name, tokens string) error {
	_, err := s.db.Exec("UPDATE tastes SET tokens = ? WHERE app_id = ? AND name = ?", tokens, appID, name)
	return err
}

func (s *Store) GetTaste(appID int64, name string) (string, error) {
	var tokens string
	err := s.db.QueryRow("SELECT tokens FROM tastes WHERE app_id = ? AND name = ?", appID, name).Scan(&tokens)
	return tokens, err
}

func (s *Store) ListTastes(appID int64) ([]struct{ Name, Tokens string }, error) {
	rows, err := s.db.Query("SELECT name, tokens FROM tastes WHERE app_id = ? ORDER BY name", appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []struct{ Name, Tokens string }
	for rows.Next() {
		var t struct{ Name, Tokens string }
		rows.Scan(&t.Name, &t.Tokens)
		result = append(result, t)
	}
	return result, nil
}

func (s *Store) DeleteTaste(appID int64, name string) error {
	_, err := s.db.Exec("DELETE FROM tastes WHERE app_id = ? AND name = ?", appID, name)
	return err
}
```

- [ ] **Step 3: Run Go tests**

```bash
go test ./internal/store/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go
git commit -m "feat(taste): DB table and CRUD store methods"
```

---

### Task 8: Taste CLI commands

**Files:**
- Modify: `cmd/sft/main.go`

- [ ] **Step 1: Add taste to `sft add` subcommand**

In the `add` case block, add a `taste` case:

```go
case "taste":
	if len(args) < 1 {
		die("usage: sft add taste <name> [--tokens <json>]")
	}
	tasteName := args[0]
	tokens := flagVal(args, "--tokens")
	if tokens == "" {
		tokens = "{}"
	}
	appID := mustAppID(s)
	id, err := s.InsertTaste(appID, tasteName, tokens)
	must(err)
	ok("taste %s (id=%d)", tasteName, id)
```

- [ ] **Step 2: Add taste to `sft set` subcommand**

```go
case "taste":
	if len(args) < 1 {
		die("usage: sft set taste <name> --tokens <json>")
	}
	tasteName := args[0]
	tokens := flagVal(args, "--tokens")
	if tokens == "" {
		die("--tokens is required")
	}
	appID := mustAppID(s)
	must(s.UpdateTaste(appID, tasteName, tokens))
	ok("taste %s updated", tasteName)
```

- [ ] **Step 3: Add taste to `sft query` subcommand**

```go
case "tastes":
	appID := mustAppID(s)
	tastes, err := s.ListTastes(appID)
	must(err)
	for _, t := range tastes {
		fmt.Printf("%s\t%s\n", t.Name, t.Tokens)
	}
```

- [ ] **Step 4: Add taste to `sft delete` subcommand**

```go
case "taste":
	if len(args) < 1 {
		die("usage: sft delete taste <name>")
	}
	appID := mustAppID(s)
	must(s.DeleteTaste(appID, args[0]))
	ok("taste %s deleted", args[0])
```

- [ ] **Step 5: Build and test CLI**

```bash
go build ./cmd/sft
cd /tmp/sft-playground-test
./sft add taste wireframe
./sft add taste dark-compact --tokens '{"density":"compact","mode":"dark","accent":"#e94560"}'
./sft query tastes
./sft set taste wireframe --tokens '{"density":"default","mode":"light"}'
./sft query tastes
./sft delete taste wireframe
./sft query tastes
```

- [ ] **Step 6: Commit**

```bash
git add cmd/sft/main.go
git commit -m "feat(taste): CLI commands for add, set, query, delete"
```

---

### Task 9: Taste NATS handler + Spec inclusion

**Files:**
- Modify: `internal/show/show.go`
- Modify: `internal/view/server.go`

- [ ] **Step 1: Add Taste to the Spec struct in show.go**

```go
type Taste struct {
	Name   string         `json:"name"`
	Tokens map[string]any `json:"tokens"`
}
```

Add to `Spec`:
```go
type Spec struct {
	App      App       `json:"app"`
	Screens  []Screen  `json:"screens"`
	Flows    []Flow    `json:"flows,omitempty"`
	Fixtures []Fixture `json:"fixtures,omitempty"`
	Tastes   []Taste   `json:"tastes,omitempty"`
}
```

- [ ] **Step 2: Load tastes in the Load function**

After loading fixtures, add:

```go
// Tastes
spec.Tastes = loadTastes(db, appID)
```

Add the loader:

```go
func loadTastes(db *sql.DB, appID int64) []Taste {
	rows, _ := db.Query("SELECT name, tokens FROM tastes WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var tastes []Taste
	for rows.Next() {
		var t Taste
		var tokensJSON string
		rows.Scan(&t.Name, &tokensJSON)
		json.Unmarshal([]byte(tokensJSON), &t.Tokens)
		tastes = append(tastes, t)
	}
	return tastes
}
```

- [ ] **Step 3: Build and verify**

```bash
go build ./cmd/sft
cd /tmp/sft-playground-test
./sft show --json | python3 -c "import sys,json; d=json.load(sys.stdin); print('tastes:', d.get('tastes', []))"
```

Expected: Should show the tastes added in Task 8.

- [ ] **Step 4: Commit**

```bash
git add internal/show/show.go internal/view/server.go
git commit -m "feat(taste): include tastes in spec JSON via NATS"
```

---

## Track C: Taste Frontend

### Task 10: Taste types + toolbar switcher

**Files:**
- Modify: `web/apps/web/src/lib/types.ts`
- Modify: `web/apps/web/src/routes/playground.tsx`

- [ ] **Step 1: Add Taste interface to types.ts**

```typescript
export interface TasteTokens {
  density?: 'compact' | 'default' | 'spacious'
  shape?: 'sharp' | 'rounded' | 'pill'
  mode?: 'light' | 'dark'
  accent?: string
  surface?: string
  list?: { avatar?: string; divider?: string; active?: string }
  nav?: { icon?: string; active?: string }
  [key: string]: any
}

export interface Taste {
  name: string
  tokens: TasteTokens
}
```

Add to `Spec`:
```typescript
export interface Spec {
  app: App
  screens: Screen[]
  flows?: Flow[]
  fixtures?: Fixture[]
  tastes?: Taste[]
}
```

- [ ] **Step 2: Add taste state + switcher to playground.tsx**

Add state:
```typescript
const [currentTaste, setCurrentTaste] = useState<string | null>(null)
```

Resolve active taste:
```typescript
const activeTaste = currentTaste
  ? spec.tastes?.find(t => t.name === currentTaste)?.tokens ?? {}
  : {}
```

Add taste chips in the toolbar (after state chips, before the closing `</div>` of the top row):

```tsx
{/* Taste switcher */}
{spec.tastes && spec.tastes.length > 0 && (
  <>
    <div className="w-px h-4 bg-neutral-200" />
    <span className="text-[10px] text-neutral-400">taste:</span>
    <div className="flex gap-1">
      {spec.tastes.map(t => (
        <button
          key={t.name}
          onClick={() => setCurrentTaste(t.name === currentTaste ? null : t.name)}
          className={`px-2.5 py-0.5 text-[10px] rounded-full transition-colors ${
            t.name === currentTaste
              ? 'bg-neutral-800 text-white font-semibold'
              : 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200'
          }`}
        >
          {t.name}
        </button>
      ))}
    </div>
  </>
)}
```

Pass taste to canvas:
```tsx
<WireframeCanvas
  screen={effectiveScreen}
  currentState={effectiveState}
  appRegions={spec.app.regions}
  fixtures={spec.fixtures}
  activeRegion={activeRegion}
  activeEvent={activeEvent}
  app={spec.app}
  taste={activeTaste}
/>
```

- [ ] **Step 3: Add `taste` to WireframeCanvasProps**

In `wireframe-canvas.tsx`, add:

```typescript
interface WireframeCanvasProps {
  // ... existing
  taste?: TasteTokens
}
```

Import `TasteTokens` from types. Thread `taste` through to `WireframeRegion` and into `SkinRenderer`.

- [ ] **Step 4: Commit**

```bash
git add web/apps/web/src/lib/types.ts web/apps/web/src/routes/playground.tsx web/apps/web/src/components/wireframe-canvas.tsx
git commit -m "feat(taste): taste types, toolbar switcher, prop threading"
```

---

### Task 11: Skins read taste tokens

**Files:**
- Modify: `web/apps/web/src/components/skins/types.ts`
- Modify: all skin components to accept and use `taste`

- [ ] **Step 1: Add taste to SkinProps**

```typescript
import type { TasteTokens } from '../../lib/types'

export interface SkinProps {
  region: Region
  context: SkinContext
  fixtureData?: Record<string, any> | null
  screenName?: string
  compact?: boolean
  taste?: TasteTokens
}
```

- [ ] **Step 2: Apply taste tokens in skins**

In each skin, read taste tokens and apply styling changes. Key patterns:

**Density:** `taste.density === 'compact'` → tighter padding, smaller font sizes. `'spacious'` → more padding.

**Shape:** `taste.shape === 'sharp'` → `rounded-none`. `'pill'` → `rounded-full`. `'rounded'` → default.

**Mode:** `taste.mode === 'dark'` → invert backgrounds and text colors.

**Accent:** `taste.accent` → use for active states, buttons, highlights. Applied as inline `style={{ color: taste.accent }}` or `style={{ backgroundColor: taste.accent }}`.

Example for `tabs.tsx` — update the active tab to use accent:

```typescript
const isActive = i === 0
const accentBg = taste?.accent ? { backgroundColor: taste.accent } : {}
const accentStyle = isActive ? accentBg : {}

return (
  <div
    key={val}
    className={[
      compact ? 'text-[8px] px-2 py-0.5' : 'text-[9px] px-3 py-1',
      `rounded-${taste?.shape === 'pill' ? 'full' : taste?.shape === 'sharp' ? 'none' : 'full'}`,
      'shrink-0',
      isActive && !taste?.accent ? 'bg-neutral-700 text-white font-semibold' : '',
      isActive && taste?.accent ? 'text-white font-semibold' : '',
      !isActive ? 'bg-neutral-100 text-neutral-500' : '',
    ].join(' ')}
    style={accentStyle}
  >
    {val}
  </div>
)
```

Apply similar patterns across all skins. The taste tokens are hints — skins read them if present, fall back to defaults if not.

- [ ] **Step 3: Commit**

```bash
git add web/apps/web/src/components/skins/
git commit -m "feat(taste): skins read taste tokens for density, shape, mode, accent"
```

---

### Task 12: Build and verify full system

- [ ] **Step 1: Add test tastes to gmail spec**

```bash
cd /tmp/sft-playground-test
/home/lagz0ne/dev/sft/sft add taste wireframe
/home/lagz0ne/dev/sft/sft add taste dark-compact --tokens '{"density":"compact","mode":"dark","shape":"sharp","accent":"#e94560","surface":"#1a1a2e"}'
/home/lagz0ne/dev/sft/sft add taste notion-light --tokens '{"density":"default","mode":"light","shape":"rounded","accent":"#2383e2","surface":"#f7f6f3"}'
```

- [ ] **Step 2: Build SPA + Go**

```bash
cd web/apps/web && npm run build
cd /home/lagz0ne/dev/sft && go build ./cmd/sft
```

- [ ] **Step 3: Test in playground**

```bash
cd /tmp/sft-playground-test && /home/lagz0ne/dev/sft/sft view
```

Open `http://localhost:51741/playground`. Verify:
- Taste chips appear in toolbar: wireframe, dark-compact, notion-light
- Clicking "dark-compact" changes skin rendering (darker backgrounds, sharper corners, accent color)
- Clicking "notion-light" changes to light rounded style
- Clicking active taste again deselects (back to default)
- All 8 skins render correctly with and without taste

- [ ] **Step 4: Final commit**

```bash
git add -A && git commit -m "build: full playground skins + taste system"
```
