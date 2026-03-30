// Discriminated union for typed component props

export type TableProps = {
  columns?: string[]
  rows?: number
}

export type StackProps = {
  items?: string[]
  gap?: string
}

export type InputProps = {
  label?: string
  placeholder?: string
  type?: string
}

export type ButtonProps = {
  label?: string
  variant?: 'primary' | 'secondary' | 'outline'
}

export type CardProps = {
  title?: string
  subtitle?: string
}

export type FormProps = {
  fields?: string[]
}

export type TextProps = {
  content?: string
  size?: 'sm' | 'md' | 'lg'
}

export type MetricProps = {
  label?: string
  value?: string
  trend?: 'up' | 'down' | 'flat'
}

export type ComponentProps =
  | TableProps
  | StackProps
  | InputProps
  | ButtonProps
  | CardProps
  | FormProps
  | TextProps
  | MetricProps
  | Record<string, unknown>

/**
 * Safely parse component_props JSON string into typed props.
 * Returns empty object on parse failure.
 */
export function parseComponentProps(raw: string | undefined): Record<string, any> {
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch {
    return {}
  }
}
