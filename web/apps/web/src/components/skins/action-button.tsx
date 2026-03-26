import type { SkinProps } from './types'

export function ActionButton({ region }: SkinProps) {
  const events = region.events ?? []
  const event = events[0]
  if (!event) return null

  const label = event.split('(')[0].replace(/_/g, ' ')
  const tags = region.tags ?? []
  const isDestructive = tags.some((t) => t.toLowerCase().includes('destructive'))

  return (
    <div className="flex w-full">
      <div
        className={`px-3 py-1 rounded-sm text-[9px] font-medium text-white ${
          isDestructive ? 'bg-red-500' : 'bg-neutral-700'
        }`}
      >
        {label}
      </div>
    </div>
  )
}
