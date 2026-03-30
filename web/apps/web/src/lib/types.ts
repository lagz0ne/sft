export interface Transition {
  on_event: string
  from_state?: string
  to_state?: string
  action?: string
}

export interface Region {
  name: string
  description?: string
  tags?: string[]
  discovery_layout?: string[]
  delivery_classes?: string[]
  delivery_component?: string
  events?: string[]
  transitions?: Transition[]
  attachments?: string[]
  regions?: Region[]
  states?: string[]
  state_regions?: Record<string, string[]>
  state_fixtures?: Record<string, string>
  ambient?: Record<string, string>
  region_data?: Record<string, string>
  component?: string
  component_props?: string
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

export interface Fixture {
  name: string
  extends?: string
  data: Record<string, any>
}

export interface Spec {
  app: App
  screens: Screen[]
  fixtures?: Fixture[]
  layouts?: Record<string, string[]>
}

export interface RenderElement {
  type: string
  [key: string]: any
}

export interface RenderSpec {
  elements?: Record<string, RenderElement>
}
