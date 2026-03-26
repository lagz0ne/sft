import type { SkinProps } from './types'

function findTitleField(fields: Record<string, string>): string | undefined {
  for (const key of Object.keys(fields)) {
    const kl = key.toLowerCase()
    if (kl.includes('name') || kl.includes('subject') || kl.includes('title')) return key
  }
  return undefined
}

function findMetaFields(fields: Record<string, string>, titleKey?: string): [string, string][] {
  const meta: [string, string][] = []
  for (const [key, type] of Object.entries(fields)) {
    if (key === titleKey) continue
    const kl = key.toLowerCase()
    const tl = type.toLowerCase()
    if (kl.includes('date') || kl.includes('time') || tl === 'date' || tl === 'datetime' ||
        kl.includes('sender') || kl.includes('from') || kl.includes('author')) {
      meta.push([key, type])
    }
  }
  return meta
}

export function DetailCard({ region, context, fixtureData, compact, taste }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const titleKey = findTitleField(fields)
  const metaFields = findMetaFields(fields, titleKey)
  const data = fixtureData ?? {}

  const dark = taste?.mode === 'dark'
  const titleClass = dark ? 'text-neutral-200' : 'text-neutral-700'
  const metaClass = dark ? 'text-neutral-500' : 'text-neutral-400'
  const titleBarClass = dark ? 'bg-neutral-700' : 'bg-neutral-200'
  const bodyBarClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'

  return (
    <div className={`flex flex-col ${compact ? 'gap-1' : 'gap-1.5'} w-full`}>
      {/* Title */}
      {titleKey ? (
        <div className={`text-[10px] font-semibold leading-tight ${titleClass}`}>
          {data[titleKey] ? String(data[titleKey]) : (
            <div className={`h-2.5 rounded-sm w-3/4 ${titleBarClass}`} />
          )}
        </div>
      ) : (
        <div className={`h-2.5 rounded-sm w-2/3 ${titleBarClass}`} />
      )}

      {/* Meta */}
      {metaFields.length > 0 && (
        <div className="flex items-center gap-2">
          {metaFields.map(([key]) => (
            <span key={key} className={`text-[7px] ${metaClass}`}>
              {data[key] ? String(data[key]) : key.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      )}

      {/* Body placeholder lines */}
      {!compact && (
        <div className="flex flex-col gap-1 pt-0.5">
          <div className={`h-1.5 rounded-sm w-full ${bodyBarClass}`} />
          <div className={`h-1.5 rounded-sm w-5/6 ${bodyBarClass}`} />
          <div className={`h-1.5 rounded-sm w-4/6 ${bodyBarClass}`} />
        </div>
      )}
    </div>
  )
}
