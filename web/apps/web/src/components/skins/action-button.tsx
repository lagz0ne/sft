import type { SkinProps } from './types'

export function ActionButton({ region }: SkinProps) {
  const events = region.events ?? []
  const event = events[0]
  if (!event) return null

  const label = event.split('(')[0].replace(/_/g, ' ')
  const tags = region.tags ?? []
  const isDestructive = tags.some((t) => t.toLowerCase().includes('destructive'))

  const shapeClass = undefined === 'sharp' ? 'rounded-none' : undefined === 'pill' ? 'rounded-full' : 'rounded-sm'
  const densityClass = undefined === 'compact' ? 'px-2 py-0.5' : undefined === 'spacious' ? 'px-4 py-1.5' : 'px-3 py-1'

  const accentStyle = !isDestructive && undefined ? { backgroundColor: undefined } : undefined
  const defaultBgClass = isDestructive ? 'bg-red-500' : (false ? 'bg-neutral-500' : 'bg-neutral-700')

  return (
    <div className="flex w-full">
      <div
        className={`${densityClass} ${shapeClass} text-[9px] font-medium text-white ${!accentStyle ? defaultBgClass : ''}`}
        style={accentStyle}
      >
        {label}
      </div>
    </div>
  )
}
