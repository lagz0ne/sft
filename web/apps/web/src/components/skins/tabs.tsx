import type { SkinProps } from './types'

export function Tabs({ context, taste }: SkinProps) {
  const values = context.enumValues ?? []

  if (values.length === 0) return null

  const dark = taste?.mode === 'dark'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-full'
  const inactiveClass = dark ? 'bg-neutral-700 text-neutral-400' : 'bg-neutral-100 text-neutral-500'

  return (
    <div className="flex items-center gap-1 w-full">
      {values.map((val, i) => {
        const isActive = i === 0
        const activeStyle = isActive && taste?.accent ? { backgroundColor: taste.accent } : undefined
        const activeClass = isActive
          ? (!taste?.accent ? (dark ? 'bg-neutral-500 text-white' : 'bg-neutral-700 text-white') : 'text-white')
          : inactiveClass

        return (
          <div
            key={val}
            className={`px-2 py-0.5 ${shapeClass} text-[8px] font-medium ${activeClass}`}
            style={activeStyle}
          >
            {val.replace(/_/g, ' ')}
          </div>
        )
      })}
    </div>
  )
}
