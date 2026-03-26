import type { SkinProps } from './types'

export function ActionButton({ region, taste }: SkinProps) {
  const events = region.events ?? []
  const event = events[0]
  if (!event) return null

  const label = event.split('(')[0].replace(/_/g, ' ')
  const tags = region.tags ?? []
  const isDestructive = tags.some((t) => t.toLowerCase().includes('destructive'))

  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'
  const densityClass = taste?.density === 'compact' ? 'px-2 py-0.5' : taste?.density === 'spacious' ? 'px-4 py-1.5' : 'px-3 py-1'

  const accentStyle = !isDestructive && taste?.accent ? { backgroundColor: taste.accent } : undefined
  const defaultBgClass = isDestructive ? 'bg-red-500' : (taste?.mode === 'dark' ? 'bg-neutral-500' : 'bg-neutral-700')

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
