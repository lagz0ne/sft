import type { App } from './types'

export interface ParsedEvent {
  name: string
  paramType: string | null
}

export interface ResolvedType {
  kind: 'scalar' | 'data_type' | 'enum'
  fields?: Record<string, string>
  values?: string[]
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
  resolved: ResolvedType | null
}

const SCALARS = new Set(['string', 'number', 'boolean', 'date', 'datetime'])

export function parseEventAnnotation(event: string): ParsedEvent {
  const match = event.match(/^(\w+)\((\w+)\)$/)
  if (match) {
    return { name: match[1], paramType: match[2] }
  }
  return { name: event.trim(), paramType: null }
}

export function resolveType(
  typeName: string,
  dataTypes?: App['data_types'],
  enums?: App['enums']
): ResolvedType {
  let name = typeName.trim()
  const isOptional = name.endsWith('?')
  if (isOptional) name = name.slice(0, -1)
  const isArray = name.endsWith('[]')
  if (isArray) name = name.slice(0, -2)

  const base: Partial<ResolvedType> = {}
  if (isArray) base.isArray = true
  if (isOptional) base.isOptional = true

  if (SCALARS.has(name)) {
    return { kind: 'scalar', ...base }
  }
  if (dataTypes && name in dataTypes) {
    return { kind: 'data_type', fields: dataTypes[name], ...base }
  }
  if (enums && name in enums) {
    return { kind: 'enum', values: enums[name], ...base }
  }
  // Unknown — treat as scalar fallback
  return { kind: 'scalar', ...base }
}

export function parseAmbientRef(ref: string): ParsedAmbientRef | null {
  const match = ref.match(/^data\((\w+),\s*(\.[^\)]+)\)$/)
  if (!match) return null
  return { source: match[1], query: match[2] }
}

export function resolveAmbientType(
  ambientValue: string,
  screenContext?: Record<string, string>,
  dataTypes?: App['data_types'],
  enums?: App['enums']
): ResolvedAmbient | null {
  const parsed = parseAmbientRef(ambientValue)
  if (!parsed) return null

  const { query } = parsed

  // Extract field name from query: ".emails" → "emails", ".emails[0]" → "emails"
  const fieldMatch = query.match(/^\.(\w+)/)
  if (!fieldMatch) return null
  const fieldName = fieldMatch[1]

  // Determine if indexed access e.g. ".emails[0]"
  const isIndexed = /\[\d+\]/.test(query)

  // Resolve field type from screen context
  const context = screenContext ?? {}
  const rawType = context[fieldName]
  if (!rawType) return null

  let typeName = rawType.trim()
  const isOptional = typeName.endsWith('?')
  if (isOptional) typeName = typeName.slice(0, -1)
  const isArray = typeName.endsWith('[]')
  if (isArray) typeName = typeName.slice(0, -2)

  // If indexed access, dereference the array → single item
  const resultIsArray = isIndexed ? false : isArray

  const resolved = resolveType(typeName, dataTypes, enums)

  return {
    isArray: resultIsArray,
    baseTypeName: typeName,
    resolved,
  }
}
