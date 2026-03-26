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

export function DetailCard({ region, context, fixtureData, compact }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const titleKey = findTitleField(fields)
  const metaFields = findMetaFields(fields, titleKey)
  const data = fixtureData ?? {}

  return (
    <div className={`flex flex-col ${compact ? 'gap-1' : 'gap-1.5'} w-full`}>
      {/* Title */}
      {titleKey ? (
        <div className="text-[10px] font-semibold text-neutral-700 leading-tight">
          {data[titleKey] ? String(data[titleKey]) : (
            <div className="h-2.5 bg-neutral-200 rounded-sm w-3/4" />
          )}
        </div>
      ) : (
        <div className="h-2.5 bg-neutral-200 rounded-sm w-2/3" />
      )}

      {/* Meta */}
      {metaFields.length > 0 && (
        <div className="flex items-center gap-2">
          {metaFields.map(([key]) => (
            <span key={key} className="text-[7px] text-neutral-400">
              {data[key] ? String(data[key]) : key.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      )}

      {/* Body placeholder lines */}
      {!compact && (
        <div className="flex flex-col gap-1 pt-0.5">
          <div className="h-1.5 bg-neutral-100 rounded-sm w-full" />
          <div className="h-1.5 bg-neutral-100 rounded-sm w-5/6" />
          <div className="h-1.5 bg-neutral-100 rounded-sm w-4/6" />
        </div>
      )}
    </div>
  )
}
