import type { SkinProps } from './types'

function formatLabel(event: string): string {
  return event.split('(')[0].replace(/_/g, ' ')
}

export function ActionBar({ region, taste }: SkinProps) {
  const events = region.events ?? []

  if (events.length === 0) return null

  const dark = taste?.mode === 'dark'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'
  const itemClass = dark
    ? 'bg-neutral-700 text-neutral-300'
    : 'bg-neutral-100 text-neutral-600'
  const densityClass = taste?.density === 'compact' ? 'px-1.5 py-0' : taste?.density === 'spacious' ? 'px-2.5 py-1' : 'px-2 py-0.5'

  return (
    <div className="flex items-center gap-1 w-full">
      {events.map((ev) => (
        <div
          key={ev}
          className={`${densityClass} ${shapeClass} text-[8px] font-medium ${itemClass}`}
        >
          {formatLabel(ev)}
        </div>
      ))}
    </div>
  )
}
