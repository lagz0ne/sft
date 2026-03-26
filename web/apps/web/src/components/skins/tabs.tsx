import type { SkinProps } from './types'

export function Tabs({ context }: SkinProps) {
  const values = context.enumValues ?? []

  if (values.length === 0) return null

  return (
    <div className="flex items-center gap-1 w-full">
      {values.map((val, i) => (
        <div
          key={val}
          className={`px-2 py-0.5 rounded-full text-[8px] font-medium ${
            i === 0
              ? 'bg-neutral-700 text-white'
              : 'bg-neutral-100 text-neutral-500'
          }`}
        >
          {val.replace(/_/g, ' ')}
        </div>
      ))}
    </div>
  )
}
