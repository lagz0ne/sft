import type { SkinProps } from './types'

function resolveItem(fixtureData: Record<string, any> | null | undefined, screenName?: string): Record<string, any> | null {
	if (!fixtureData) return null

	// Drill into screen-level data
	let data = fixtureData
	if (screenName && fixtureData[screenName] && typeof fixtureData[screenName] === 'object') {
		data = fixtureData[screenName]
	}

	// Find a single object (not array) — the detail record
	if (typeof data === 'object' && !Array.isArray(data)) {
		// If there's a nested non-array, non-primitive object, that's likely the detail item
		for (const val of Object.values(data)) {
			if (val && typeof val === 'object' && !Array.isArray(val)) return val
		}
		// Otherwise data itself might be the item
		return data
	}
	return null
}

function resolveValue(item: any, field: string): string {
	const val = item?.[field]
	if (val == null) return ''
	if (typeof val === 'object' && 'name' in val) return val.name
	return String(val)
}

export function DetailCard({ context, fixtureData, screenName, compact }: SkinProps) {
	const fields = context.fields ?? {}
	const item = resolveItem(fixtureData, screenName)

	const dark = false
	const titleClass = dark ? 'text-neutral-200' : 'text-neutral-800'
	const metaClass = dark ? 'text-neutral-500' : 'text-neutral-400'
	const barClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'
	const barLightClass = dark ? 'bg-neutral-800' : 'bg-neutral-50'

	// Categorize fields
	const entries = Object.entries(fields).filter(([k]) => !/^(id|key)$/i.test(k))
	const titleField = entries.find(([k]) => /^(subject|title|name|label)$/i.test(k))
	const contactField = entries.find(([, v]) => /^(contact|user|person|author)$/i.test(v.replace(/[?\[\]]/g, '')))
	const dateField = entries.find(([k, v]) => /date|time|created/i.test(k) || /^(date|datetime)$/i.test(v))
	const bodyField = entries.find(([k]) => /^(body|content|description|text|message)$/i.test(k))
	const metaFields = [contactField, dateField].filter(Boolean) as [string, string][]

	const titleValue = item && titleField ? resolveValue(item, titleField[0]) : null
	const hasBody = bodyField != null

	return (
		<div className={`flex flex-col ${compact ? 'gap-1' : 'gap-2'} w-full`}>
			{/* Title */}
			{titleValue ? (
				<div className={`${compact ? 'text-xs' : 'text-sm'} font-semibold leading-tight ${titleClass}`}>
					{titleValue}
				</div>
			) : (
				<div className={`${compact ? 'h-2' : 'h-2.5'} rounded-sm w-3/5 ${dark ? 'bg-neutral-600' : 'bg-neutral-200'}`} />
			)}

			{/* Meta line: sender + date */}
			{metaFields.length > 0 && (
				<div className={`flex items-center gap-1 text-[9px] ${metaClass}`}>
					{metaFields.map(([key], i) => (
						<span key={key}>
							{i > 0 && <span className="mx-0.5">·</span>}
							{item ? resolveValue(item, key) : key.replace(/_/g, ' ')}
						</span>
					))}
				</div>
			)}

			{/* Separator */}
			{!compact && <div className={`h-px w-full ${dark ? 'bg-neutral-700' : 'bg-neutral-200'}`} />}

			{/* Body text lines */}
			{!compact && (
				<div className="flex flex-col gap-1.5">
					{hasBody && item && item[bodyField![0]] ? (
						<div className={`text-[10px] leading-relaxed ${dark ? 'text-neutral-300' : 'text-neutral-600'}`}>
							{String(item[bodyField![0]]).slice(0, 120)}
						</div>
					) : (
						<>
							{[100, 95, 88, 72, 60].map((w, i) => (
								<div key={i} className={`h-[5px] rounded-sm ${i < 2 ? barClass : barLightClass}`} style={{ width: `${w}%` }} />
							))}
						</>
					)}
				</div>
			)}
		</div>
	)
}
