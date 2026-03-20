export interface Transition {
  on_event: string
  from_state?: string
  to_state?: string
  action?: string
}

export interface Region {
  name: string
  description?: string
  events?: string[]
  transitions?: Transition[]
  attachments?: string[]
  regions?: Region[]
  states?: string[]
  state_regions?: Record<string, string[]>
  state_fixtures?: Record<string, string>
}

export interface Screen {
  name: string
  description?: string
  tags?: string[]
  context?: Record<string, string>
  attachments?: string[]
  regions?: Region[]
  transitions?: Transition[]
  states?: string[]
  state_regions?: Record<string, string[]>
  state_fixtures?: Record<string, string>
}

export interface App {
  name: string
  description: string
  data_types?: Record<string, Record<string, string>>
  enums?: Record<string, string[]>
  context?: Record<string, string>
  regions?: Region[]
  transitions?: Transition[]
}

export interface FlowStep {
  position: number
  type: 'screen' | 'region' | 'event' | 'back' | 'action' | 'activate'
  name: string
  history?: number
  data?: string
}

export interface Flow {
  name: string
  description?: string
  on_event?: string
  sequence: string
  steps?: FlowStep[]
}

export interface Spec {
  app: App
  screens: Screen[]
  flows?: Flow[]
}

export interface RenderElement {
  type: string
  [key: string]: any
}

export interface RenderSpec {
  elements?: Record<string, RenderElement>
}
