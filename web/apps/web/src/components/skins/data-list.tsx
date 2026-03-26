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

function PlaceholderBar({ width, dark }: { width: string; dark?: boolean }) {
  return <div className={`h-1.5 rounded-sm ${width} ${dark ? 'bg-neutral-700' : 'bg-neutral-100'}`} />
}

export function DataList({ region, context, fixtureData, compact, taste }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const cols = pickColumns(fields)
  const totalRows = compact ? 3 : 4

  const dark = taste?.mode === 'dark'
  const rowPy = taste?.density === 'compact' ? 'py-0.5' : taste?.density === 'spacious' ? 'py-1.5' : 'py-1'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'
  const activeRowBg = dark ? 'bg-neutral-700' : 'bg-blue-50'
  const primaryTextClass = dark ? 'text-neutral-200' : 'text-neutral-700'
  const secondaryTextClass = dark ? 'text-neutral-400' : 'text-neutral-400'
  const metaTextClass = dark ? 'text-neutral-500' : 'text-neutral-400'
  const metaBarClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'

  // Build real rows from fixture data
  const realRows: Record<string, any>[] = []
  if (fixtureData) {
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

  const accentDotStyle = taste?.accent ? { backgroundColor: taste.accent } : undefined

  return (
    <div className="flex flex-col gap-0.5 w-full">
      {rows.map(({ data, isFirst, index }) => (
        <div
          key={index}
          className={`flex items-center gap-1.5 px-1.5 ${rowPy} ${shapeClass} ${isFirst ? activeRowBg : ''}`}
        >
          {/* Boolean indicator / unread dot */}
          {cols.booleans.length > 0 && (
            <div
              className={`w-1.5 h-1.5 rounded-full shrink-0 ${!accentDotStyle && index === 0 ? 'bg-blue-400' : 'bg-neutral-200'}`}
              style={index === 0 ? accentDotStyle : undefined}
            />
          )}

          {/* Avatar for contact-type primary */}
          {cols.primary && isContactType(cols.primary[1]) && data ? (
            <Avatar initial={String(data[cols.primary[0]] ?? '?')[0]} />
          ) : cols.primary && isContactType(cols.primary[1]) ? (
            <div className={`w-4 h-4 rounded-full shrink-0 ${dark ? 'bg-neutral-700' : 'bg-neutral-100'}`} />
          ) : null}

          {/* Content */}
          <div className="flex-1 min-w-0 flex flex-col gap-0.5">
            {data && cols.primary ? (
              <span className={`text-[9px] font-medium truncate leading-tight ${primaryTextClass}`}>
                {String(data[cols.primary[0]] ?? '')}
              </span>
            ) : (
              <PlaceholderBar width={index % 2 === 0 ? 'w-3/4' : 'w-1/2'} dark={dark} />
            )}
            {!compact && (
              data && cols.secondary ? (
                <span className={`text-[8px] truncate leading-tight ${secondaryTextClass}`}>
                  {String(data[cols.secondary[0]] ?? '')}
                </span>
              ) : (
                <PlaceholderBar width={index % 2 === 0 ? 'w-1/2' : 'w-2/3'} dark={dark} />
              )
            )}
          </div>

          {/* Meta (date) */}
          {cols.meta && (
            <div className="shrink-0">
              {data ? (
                <span className={`text-[7px] ${metaTextClass}`}>{String(data[cols.meta[0]] ?? '')}</span>
              ) : (
                <div className={`w-6 h-1.5 rounded-sm ${metaBarClass}`} />
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
