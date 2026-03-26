import type { SkinProps } from './types'
import type { TasteTokens } from '../../lib/types'

function isArrayType(type: string): boolean {
  return type.endsWith('[]')
}

function FieldInput({ name, type, compact, dark, shapeClass }: { name: string; type: string; compact?: boolean; dark?: boolean; shapeClass: string }) {
  const label = name.replace(/_/g, ' ')
  const labelClass = dark ? 'text-neutral-400' : 'text-neutral-500'
  const borderClass = dark ? 'border-neutral-700 bg-neutral-800' : 'border-neutral-200 bg-white'

  if (type === 'boolean' || type === 'bool') {
    return (
      <div className="flex items-center justify-between">
        <span className={`text-[8px] ${labelClass}`}>{label}</span>
        <div className={`w-5 h-2.5 rounded-full relative ${dark ? 'bg-neutral-700' : 'bg-neutral-200'}`}>
          <div className="absolute left-0.5 top-0.5 w-1.5 h-1.5 rounded-full bg-white" />
        </div>
      </div>
    )
  }

  if (isArrayType(type)) {
    return (
      <div className="flex flex-col gap-0.5">
        <span className={`text-[8px] ${labelClass}`}>{label}</span>
        <div className="flex items-center gap-1">
          <div className={`px-1.5 py-0.5 rounded text-[7px] ${dark ? 'bg-neutral-700 text-neutral-400' : 'bg-neutral-100 text-neutral-500'}`}>tag1</div>
          <div className={`px-1.5 py-0.5 rounded text-[7px] ${dark ? 'bg-neutral-700 text-neutral-400' : 'bg-neutral-100 text-neutral-500'}`}>tag2</div>
          <span className="text-[8px] text-blue-400">+ add</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-0.5">
      <span className={`text-[8px] ${labelClass}`}>{label}</span>
      <div className={`w-full border ${borderClass} ${shapeClass} ${compact ? 'h-4' : 'h-5'}`} />
    </div>
  )
}

function EventButton({ event, taste }: { event: string; taste?: TasteTokens }) {
  const label = event.split('(')[0].replace(/_/g, ' ')
  const ll = label.toLowerCase()
  const isSubmit = ll.includes('send') || ll.includes('submit') || ll.includes('save')
  const isCancel = ll.includes('cancel') || ll.includes('discard') || ll.includes('close')

  const dark = taste?.mode === 'dark'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'

  if (isSubmit) {
    const accentStyle = taste?.accent ? { backgroundColor: taste.accent } : undefined
    const defaultBg = dark ? 'bg-neutral-500' : 'bg-neutral-700'
    return (
      <button
        className={`px-3 py-1 text-white text-[8px] ${shapeClass} font-medium ${!accentStyle ? defaultBg : ''}`}
        style={accentStyle}
      >
        {label}
      </button>
    )
  }

  if (isCancel) {
    const borderClass = dark ? 'border-neutral-600 text-neutral-400 bg-neutral-800' : 'border-neutral-200 text-neutral-500 bg-white'
    return (
      <button className={`px-3 py-1 border text-[8px] ${shapeClass} ${borderClass}`}>
        {label}
      </button>
    )
  }

  return null
}

export function FormLayout({ region, context, compact, taste }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const events = region.events ?? []
  const entries = Object.entries(fields)

  const dark = taste?.mode === 'dark'
  const shapeClass = taste?.shape === 'sharp' ? 'rounded-none' : taste?.shape === 'pill' ? 'rounded-full' : 'rounded-sm'

  const submitEvents = events.filter((e) => {
    const l = e.split('(')[0].replace(/_/g, ' ').toLowerCase()
    return l.includes('send') || l.includes('submit') || l.includes('save') ||
           l.includes('cancel') || l.includes('discard') || l.includes('close')
  })

  return (
    <div className={`flex flex-col ${compact ? 'gap-1.5' : 'gap-2'} w-full`}>
      {entries.map(([name, type]) => (
        <FieldInput key={name} name={name} type={type} compact={compact} dark={dark} shapeClass={shapeClass} />
      ))}

      {submitEvents.length > 0 && (
        <div className="flex items-center gap-1.5 pt-1">
          {submitEvents.map((ev) => (
            <EventButton key={ev} event={ev} taste={taste} />
          ))}
        </div>
      )}
    </div>
  )
}
