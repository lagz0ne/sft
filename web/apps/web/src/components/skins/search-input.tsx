import type { SkinProps } from './types'

function MagnifierIcon({ dark }: { dark?: boolean }) {
  return (
    <svg
      width="10"
      height="10"
      viewBox="0 0 16 16"
      fill="none"
      className={`shrink-0 ${dark ? 'text-neutral-500' : 'text-neutral-400'}`}
    >
      <circle cx="7" cy="7" r="5" stroke="currentColor" strokeWidth="1.5" />
      <line x1="11" y1="11" x2="14" y2="14" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

export function SearchInput({ compact, taste }: SkinProps) {
  const dark = taste?.mode === 'dark'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'
  const densityClass = taste?.density === 'compact' ? 'px-1 py-0' : taste?.density === 'spacious' ? 'px-2.5 py-1.5' : (compact ? 'px-1.5 py-0.5' : 'px-2 py-1')
  const bgBorderClass = dark
    ? 'border-neutral-700 bg-neutral-800'
    : 'border-neutral-200 bg-white'

  return (
    <div className={`flex items-center gap-1.5 border ${bgBorderClass} ${shapeClass} ${densityClass} w-full`}>
      <MagnifierIcon dark={dark} />
      <div className="flex-1 h-2" />
    </div>
  )
}
