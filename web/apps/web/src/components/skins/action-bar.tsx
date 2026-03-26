import type { SkinProps } from './types'

function formatLabel(event: string): string {
  return event.split('(')[0].replace(/_/g, ' ')
}

export function ActionBar({ region }: SkinProps) {
  const events = region.events ?? []

  if (events.length === 0) return null

  return (
    <div className="flex items-center gap-1 w-full">
      {events.map((ev) => (
        <div
          key={ev}
          className="px-2 py-0.5 bg-neutral-100 rounded-sm text-[8px] text-neutral-600 font-medium"
        >
          {formatLabel(ev)}
        </div>
      ))}
    </div>
  )
}
