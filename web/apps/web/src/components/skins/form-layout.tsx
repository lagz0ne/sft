import type { SkinProps } from './types'

function isArrayType(type: string): boolean {
  return type.endsWith('[]')
}

function FieldInput({ name, type, compact }: { name: string; type: string; compact?: boolean }) {
  const label = name.replace(/_/g, ' ')

  if (type === 'boolean' || type === 'bool') {
    return (
      <div className="flex items-center justify-between">
        <span className="text-[8px] text-neutral-500">{label}</span>
        <div className="w-5 h-2.5 rounded-full bg-neutral-200 relative">
          <div className="absolute left-0.5 top-0.5 w-1.5 h-1.5 rounded-full bg-white" />
        </div>
      </div>
    )
  }

  if (isArrayType(type)) {
    return (
      <div className="flex flex-col gap-0.5">
        <span className="text-[8px] text-neutral-500">{label}</span>
        <div className="flex items-center gap-1">
          <div className="px-1.5 py-0.5 bg-neutral-100 rounded text-[7px] text-neutral-500">tag1</div>
          <div className="px-1.5 py-0.5 bg-neutral-100 rounded text-[7px] text-neutral-500">tag2</div>
          <span className="text-[8px] text-blue-400">+ add</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[8px] text-neutral-500">{label}</span>
      <div className={`w-full border border-neutral-200 rounded-sm bg-white ${compact ? 'h-4' : 'h-5'}`} />
    </div>
  )
}

function EventButton({ event }: { event: string }) {
  const label = event.split('(')[0].replace(/_/g, ' ')
  const ll = label.toLowerCase()
  const isSubmit = ll.includes('send') || ll.includes('submit') || ll.includes('save')
  const isCancel = ll.includes('cancel') || ll.includes('discard') || ll.includes('close')

  if (isSubmit) {
    return (
      <button className="px-3 py-1 bg-neutral-700 text-white text-[8px] rounded-sm font-medium">
        {label}
      </button>
    )
  }

  if (isCancel) {
    return (
      <button className="px-3 py-1 bg-white border border-neutral-200 text-neutral-500 text-[8px] rounded-sm">
        {label}
      </button>
    )
  }

  return null
}

export function FormLayout({ region, context, compact }: SkinProps) {
  const fields = context.fields ?? region.region_data ?? {}
  const events = region.events ?? []
  const entries = Object.entries(fields)

  const submitEvents = events.filter((e) => {
    const l = e.split('(')[0].replace(/_/g, ' ').toLowerCase()
    return l.includes('send') || l.includes('submit') || l.includes('save') ||
           l.includes('cancel') || l.includes('discard') || l.includes('close')
  })

  return (
    <div className={`flex flex-col ${compact ? 'gap-1.5' : 'gap-2'} w-full`}>
      {entries.map(([name, type]) => (
        <FieldInput key={name} name={name} type={type} compact={compact} />
      ))}

      {submitEvents.length > 0 && (
        <div className="flex items-center gap-1.5 pt-1">
          {submitEvents.map((ev) => (
            <EventButton key={ev} event={ev} />
          ))}
        </div>
      )}
    </div>
  )
}
