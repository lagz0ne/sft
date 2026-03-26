import type { SkinProps } from './types'

function pickColumns(fields: Record<string, string>) {
  const entries = Object.entries(fields)
  let primary: [string, string] | undefined
  let secondary: [string, string] | undefined
  let meta: [string, string] | undefined
  const booleans: [string, string][] = []

  for (const [k, v] of entries) {
    const kl = k.toLowerCase()
    const vl = v.toLowerCase()
    if (!primary && (kl.includes('name') || kl.includes('subject') || kl.includes('title'))) {
      primary = [k, v]
    } else if (!meta && (vl === 'date' || vl === 'datetime' || kl.includes('date') || kl.includes('time'))) {
      meta = [k, v]
    } else if (vl === 'boolean' || vl === 'bool') {
      booleans.push([k, v])
    } else if (!secondary && vl !== 'boolean' && vl !== 'bool') {
      secondary = [k, v]
    }
  }

  if (!primary && entries.length > 0) primary = entries[0]
  if (!secondary) {
    const remaining = entries.find(([k]) => k !== primary?.[0] && k !== meta?.[0] && !booleans.some(([bk]) => bk === k))
    if (remaining) secondary = remaining
  }

  return { primary, secondary, meta, booleans }
}

function isContactType(type: string): boolean {
  const t = type.toLowerCase()
  return t === 'contact' || t === 'user' || t === 'person' || t === 'author' || t === 'sender'
}

function Avatar({ initial }: { initial: string }) {
  return (
    <div className="w-4 h-4 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-[7px] font-medium shrink-0">
      {initial.toUpperCase()}
    </div>
  )
}

function PlaceholderBar({ width }: { width: string }) {
  return <div className={`h-1.5 bg-neutral-100 rounded-sm ${width}`} />
}

export function DataList({ region, context, fixtureData, compact }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const cols = pickColumns(fields)
  const totalRows = compact ? 3 : 4

  // Build real rows from fixture data
  const realRows: Record<string, any>[] = []
  if (fixtureData) {
    // Fixture data may be an array or an object with array values
    const arr = Array.isArray(fixtureData)
      ? fixtureData
      : Object.values(fixtureData).find(Array.isArray) as any[] | undefined
    if (arr) {
      realRows.push(...arr.slice(0, 2))
    } else if (typeof fixtureData === 'object') {
      realRows.push(fixtureData)
    }
  }

  const rows = Array.from({ length: totalRows }, (_, i) => {
    const data = realRows[i] ?? null
    const isFirst = i === 0 && data !== null
    return { data, isFirst, index: i }
  })

  return (
    <div className="flex flex-col gap-0.5 w-full">
      {rows.map(({ data, isFirst, index }) => (
        <div
          key={index}
          className={`flex items-center gap-1.5 px-1.5 py-1 rounded-sm ${isFirst ? 'bg-blue-50' : ''}`}
        >
          {/* Boolean indicator */}
          {cols.booleans.length > 0 && (
            <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${index === 0 ? 'bg-blue-400' : 'bg-neutral-200'}`} />
          )}

          {/* Avatar for contact-type primary */}
          {cols.primary && isContactType(cols.primary[1]) && data ? (
            <Avatar initial={String(data[cols.primary[0]] ?? '?')[0]} />
          ) : cols.primary && isContactType(cols.primary[1]) ? (
            <div className="w-4 h-4 rounded-full bg-neutral-100 shrink-0" />
          ) : null}

          {/* Content */}
          <div className="flex-1 min-w-0 flex flex-col gap-0.5">
            {data && cols.primary ? (
              <span className="text-[9px] text-neutral-700 font-medium truncate leading-tight">
                {String(data[cols.primary[0]] ?? '')}
              </span>
            ) : (
              <PlaceholderBar width={index % 2 === 0 ? 'w-3/4' : 'w-1/2'} />
            )}
            {!compact && (
              data && cols.secondary ? (
                <span className="text-[8px] text-neutral-400 truncate leading-tight">
                  {String(data[cols.secondary[0]] ?? '')}
                </span>
              ) : (
                <PlaceholderBar width={index % 2 === 0 ? 'w-1/2' : 'w-2/3'} />
              )
            )}
          </div>

          {/* Meta (date) */}
          {cols.meta && (
            <div className="shrink-0">
              {data ? (
                <span className="text-[7px] text-neutral-400">{String(data[cols.meta[0]] ?? '')}</span>
              ) : (
                <div className="w-6 h-1.5 bg-neutral-100 rounded-sm" />
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
