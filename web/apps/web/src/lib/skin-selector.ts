import type { App, Region, Screen } from './types'
import {
  parseEventAnnotation,
  resolveAmbientType,
  resolveType,
  type ResolvedAmbient,
} from './type-resolver'

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
  ambientType?: ResolvedAmbient | null
  matchedEvent?: string
  enumValues?: string[]
  fields?: Record<string, string>
}

export function selectSkin(region: Region, app: App, screen?: Screen): SkinContext {
  const events = region.events ?? []
  const ambientEntries = Object.entries(region.ambient ?? {})
  const regionData = region.region_data ?? {}
  const screenContext = screen?.context ?? app.context ?? {}

  // Resolve all ambient types up front (needed by multiple rules)
  const resolvedAmbients = ambientEntries.map(([key, val]) => ({
    key,
    resolved: resolveAmbientType(val, screenContext, app.data_types, app.enums),
  }))

  // Rule 1: data-list — ambient binds array T[] AND an event is annotated with same type T
  for (const { resolved } of resolvedAmbients) {
    if (resolved?.isArray) {
      const T = resolved.baseTypeName
      for (const ev of events) {
        const parsed = parseEventAnnotation(ev)
        if (parsed.paramType === T) {
          return {
            skin: 'data-list',
            ambientType: resolved,
            matchedEvent: ev,
            fields: resolved.resolved?.fields,
          }
        }
      }
    }
  }

  // Rule 1b: data-list (read-only) — ambient binds array T[] of data_type, no matching event
  for (const { resolved } of resolvedAmbients) {
    if (resolved?.isArray && resolved.resolved?.kind === 'data_type' && resolved.resolved.fields) {
      return {
        skin: 'data-list',
        ambientType: resolved,
        fields: resolved.resolved.fields,
      }
    }
  }

  // Rule 2: form — region_data has 2+ fields (expand composite types from data_types)
  const expandedFields = expandFormFields(regionData, app)
  if (Object.keys(expandedFields).length >= 2) {
    return {
      skin: 'form',
      fields: expandedFields,
    }
  }

  // Rule 3: tabs — event annotation resolves to an enum AND region has ambient refs
  if (ambientEntries.length > 0) {
    for (const ev of events) {
      const parsed = parseEventAnnotation(ev)
      if (parsed.paramType) {
        const resolved = resolveType(parsed.paramType, app.data_types, app.enums)
        if (resolved.kind === 'enum' && resolved.values) {
          return {
            skin: 'tabs',
            matchedEvent: ev,
            enumValues: resolved.values,
          }
        }
      }
    }
  }

  // Rule 4: detail-card — ambient binds single T (not array), data_type with fields, no region_data
  if (Object.keys(regionData).length === 0) {
    for (const { resolved } of resolvedAmbients) {
      if (
        resolved &&
        !resolved.isArray &&
        resolved.resolved?.kind === 'data_type' &&
        resolved.resolved.fields
      ) {
        return {
          skin: 'detail-card',
          ambientType: resolved,
          fields: resolved.resolved.fields,
        }
      }
    }
  }

  // Rule 5: action-bar — 2+ events, majority without type annotations
  if (events.length >= 2) {
    const unannotated = events.filter((ev) => parseEventAnnotation(ev).paramType === null).length
    if (unannotated > events.length / 2) {
      return { skin: 'action-bar' }
    }
  }

  // Rule 6: action-button — exactly 1 event
  if (events.length === 1) {
    return { skin: 'action-button', matchedEvent: events[0] }
  }

  // Rule 7: search-input — description contains "search"
  if (region.description?.toLowerCase().includes('search')) {
    return { skin: 'search-input' }
  }

  // Rule 8: placeholder — fallback
  return { skin: 'placeholder' }
}

/**
 * Expand region_data fields: if a field's type references a data_type, inline its fields.
 * Otherwise keep the field as-is.
 */
function expandFormFields(
  regionData: Record<string, string>,
  app: App
): Record<string, string> {
  const result: Record<string, string> = {}
  for (const [fieldName, typeName] of Object.entries(regionData)) {
    const resolved = resolveType(typeName, app.data_types, app.enums)
    if (resolved.kind === 'data_type' && resolved.fields) {
      for (const [subField, subType] of Object.entries(resolved.fields)) {
        result[`${fieldName}.${subField}`] = subType
      }
    } else {
      result[fieldName] = typeName
    }
  }
  return result
}
