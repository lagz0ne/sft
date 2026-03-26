import type { SkinProps } from './types'

function MagnifierIcon() {
  return (
    <svg
      width="10"
      height="10"
      viewBox="0 0 16 16"
      fill="none"
      className="text-neutral-400 shrink-0"
    >
      <circle cx="7" cy="7" r="5" stroke="currentColor" strokeWidth="1.5" />
      <line x1="11" y1="11" x2="14" y2="14" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

export function SearchInput({ compact }: SkinProps) {
  return (
    <div className={`flex items-center gap-1.5 border border-neutral-200 rounded-sm bg-white w-full ${compact ? 'px-1.5 py-0.5' : 'px-2 py-1'}`}>
      <MagnifierIcon />
      <div className="flex-1 h-2" />
    </div>
  )
}
