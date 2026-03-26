import type { SkinProps } from './types'

function isNavLike(desc: string): boolean {
  const dl = desc.toLowerCase()
  return dl.includes('nav') || dl.includes('sidebar') || dl.includes('menu') || dl.includes('toolbar')
}

function NavItems({ compact }: { compact?: boolean }) {
  const count = compact ? 3 : 4
  return (
    <div className="flex flex-col gap-1.5 w-full">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="flex items-center gap-1.5">
          <div className={`w-3 h-3 rounded-sm ${i === 0 ? 'bg-blue-100' : 'bg-neutral-100'}`} />
          <div className={`h-1.5 rounded-sm ${i === 0 ? 'bg-blue-200 w-2/3' : 'bg-neutral-100'} ${i === 1 ? 'w-1/2' : i === 2 ? 'w-3/5' : 'w-2/5'}`} />
        </div>
      ))}
    </div>
  )
}

function ContentLines({ compact }: { compact?: boolean }) {
  return (
    <div className="flex flex-col gap-1 w-full">
      <div className="h-2 bg-neutral-200 rounded-sm w-1/2 mb-0.5" />
      <div className="h-1.5 bg-neutral-100 rounded-sm w-full" />
      <div className="h-1.5 bg-neutral-100 rounded-sm w-5/6" />
      {!compact && (
        <>
          <div className="h-1.5 bg-neutral-100 rounded-sm w-4/6" />
          <div className="h-1.5 bg-neutral-100 rounded-sm w-3/4" />
        </>
      )}
    </div>
  )
}

export function Placeholder({ region, compact }: SkinProps) {
  const desc = region.description ?? ''
  const nav = isNavLike(desc)

  return nav ? <NavItems compact={compact} /> : <ContentLines compact={compact} />
}
