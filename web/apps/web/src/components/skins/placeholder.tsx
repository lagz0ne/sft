import type { SkinProps } from './types'

function isNavLike(desc: string): boolean {
  const dl = desc.toLowerCase()
  return dl.includes('nav') || dl.includes('sidebar') || dl.includes('menu') || dl.includes('toolbar')
}

function NavItems({ compact, dark }: { compact?: boolean; dark?: boolean }) {
  const count = compact ? 3 : 4
  const iconActiveClass = dark ? 'bg-blue-900' : 'bg-blue-100'
  const iconInactiveClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'
  const barActiveClass = dark ? 'bg-blue-700' : 'bg-blue-200'
  const barInactiveClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'

  return (
    <div className="flex flex-col gap-1.5 w-full">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <div className={`w-3 h-3 rounded-sm ${i === 0 ? iconActiveClass : iconInactiveClass}`} />
          <div className={`h-1.5 rounded-sm ${i === 0 ? barActiveClass : barInactiveClass} ${i === 1 ? 'w-1/2' : i === 2 ? 'w-3/5' : 'w-2/5'} ${i === 0 ? 'w-2/3' : ''}`} />
        </div>
      ))}
    </div>
  )
}

function ContentLines({ compact, dark }: { compact?: boolean; dark?: boolean }) {
  const headingClass = dark ? 'bg-neutral-600' : 'bg-neutral-200'
  const lineClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'

  return (
    <div className="flex flex-col gap-1 w-full">
      <div className={`h-2 rounded-sm w-1/2 mb-0.5 ${headingClass}`} />
      <div className={`h-1.5 rounded-sm w-full ${lineClass}`} />
      <div className={`h-1.5 rounded-sm w-5/6 ${lineClass}`} />
      {!compact && (
        <>
          <div className={`h-1.5 rounded-sm w-4/6 ${lineClass}`} />
          <div className={`h-1.5 rounded-sm w-3/4 ${lineClass}`} />
        </>
      )}
    </div>
  )
}

export function Placeholder({ region, compact, taste }: SkinProps) {
  const desc = region.description ?? ''
  const nav = isNavLike(desc)
  const dark = taste?.mode === 'dark'

  return nav ? <NavItems compact={compact} dark={dark} /> : <ContentLines compact={compact} dark={dark} />
}
