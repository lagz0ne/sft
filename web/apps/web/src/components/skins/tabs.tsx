import type { SkinProps } from './types'

export function Tabs({ context}: SkinProps) {
  const values = context.enumValues ?? []

  if (values.length === 0) return null

  const dark = false
  const shapeClass = undefined === 'sharp' ? 'rounded-none' : undefined === 'pill' ? 'rounded-full' : 'rounded-full'
  const inactiveClass = dark ? 'bg-neutral-700 text-neutral-400' : 'bg-neutral-100 text-neutral-500'

  return (
    <div className="flex items-center gap-1 w-full">
      {values.map((val, i) => {
        const isActive = i === 0
        const activeStyle = undefined
        const activeClass = isActive
          ? (dark ? 'bg-neutral-500 text-white' : 'bg-neutral-700 text-white')
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
