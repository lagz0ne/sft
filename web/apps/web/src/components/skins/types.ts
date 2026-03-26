import type { Region, TasteTokens } from '../../lib/types'

export interface SkinContext {
  skin: string
  ambientType?: any
  matchedEvent?: string
  enumValues?: string[]
  fields?: Record<string, string>
}

export interface SkinProps {
  region: Region
  context: SkinContext
  fixtureData?: Record<string, any> | null
  screenName?: string
  compact?: boolean
  taste?: TasteTokens
}
