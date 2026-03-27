import type { SkinProps } from './types'

function isContactType(type: string): boolean {
	return /^(contact|user|person|author|member)$/i.test(type.replace(/[?\[\]]/g, ''))
}

function resolveValue(item: any, field: string): string {
	const val = item?.[field]
	if (val == null) return ''
	if (typeof val === 'object' && 'name' in val) return val.name
	if (typeof val === 'boolean') return val ? 'Yes' : 'No'
	return String(val)
}

function getInitial(item: any, field: string): string {
	const val = item?.[field]
	if (typeof val === 'object' && val?.name) return val.name[0]?.toUpperCase() ?? '?'
	if (typeof val === 'string') return val[0]?.toUpperCase() ?? '?'
	return '?'
}

interface Columns {
	contact?: string       // field name of contact type → avatar
	primary?: string       // subject, title, name → main text
	secondary?: string     // body, description, or another text field
	meta?: string          // date, time → right side
	booleans: string[]     // boolean fields → indicator dots
	hasCheckEvent: boolean // check_* event exists → render checkbox
}

function pickColumns(fields: Record<string, string>, events: string[]): Columns {
	const cols: Columns = { booleans: [], hasCheckEvent: events.some(e => e.startsWith('check_')) }

	for (const [k, v] of Object.entries(fields)) {
		const baseType = v.replace(/[?\[\]]/g, '')
		if (!cols.contact && isContactType(baseType)) {
			cols.contact = k
		} else if (!cols.primary && /^(subject|title|name|label)$/i.test(k)) {
			cols.primary = k
		} else if (!cols.meta && /^(date|datetime|time|created|updated)$/i.test(baseType)) {
			cols.meta = k
		} else if (/^(boolean|bool)$/i.test(baseType)) {
			cols.booleans.push(k)
		} else if (!cols.secondary && !/^(id|key)$/i.test(k)) {
			cols.secondary = k
		}
	}

	// Fallbacks
	if (!cols.primary) {
		cols.primary = Object.keys(fields).find(k =>
			k !== cols.contact && k !== cols.meta && !cols.booleans.includes(k)
		)
	}

	return cols
}

function resolveItems(fixtureData: Record<string, any> | null | undefined, screenName?: string): any[] {
	if (!fixtureData) return []

	// Drill into screen-level data first: fixtureData[screenName]
	let data = fixtureData
	if (screenName && fixtureData[screenName] && typeof fixtureData[screenName] === 'object') {
		data = fixtureData[screenName]
	}

	// Find the first non-empty array in the data
	if (Array.isArray(data)) return data
	for (const val of Object.values(data)) {
		if (Array.isArray(val) && val.length > 0) return val
	}
	return []
}

export function DataList({ region, context, fixtureData, screenName, compact }: SkinProps) {
	const fields = context.fields ?? {}
	const events = region.events ?? []
	const cols = pickColumns(fields, events)
	const items = resolveItems(fixtureData, screenName)

	const dark = false
	const rowPy = 'py-1.5'
	const rowBorder = dark ? 'border-neutral-700' : 'border-neutral-100'
	const hoverBg = dark ? 'bg-neutral-800' : 'bg-blue-50/40'

	const realCount = Math.min(items.length, compact ? 2 : 3)
	const placeholderCount = compact ? 1 : 2
	const totalRows = realCount + placeholderCount

	return (
		<div className="flex flex-col w-full">
			{Array.from({ length: totalRows }, (_, i) => {
				const item = i < realCount ? items[i] : null
				const isFirstReal = i === 0 && item
				const isUnread = item && cols.booleans.length > 0 && item[cols.booleans[0]] === false

				return (
					<div
						key={i}
						className={[
							'flex items-center gap-2 px-2',
							rowPy,
							i < totalRows - 1 ? `border-b ${rowBorder}` : '',
							isFirstReal ? hoverBg : '',
						].join(' ')}
					>
						{/* Checkbox (if check_* event exists) */}
						{cols.hasCheckEvent && (
							<div className={[
								'w-3 h-3 rounded-sm border shrink-0',
								dark ? 'border-neutral-600' : 'border-neutral-300',
							].join(' ')} />
						)}

						{/* Avatar (if contact field) */}
						{cols.contact && (
							item ? (
								<div
									className="w-6 h-6 rounded-full flex items-center justify-center text-[9px] font-semibold shrink-0"
									style={{
										backgroundColor: dark ? '#334' : '#dbeafe',
										color: (dark ? '#8ab4f8' : '#2563eb'),
									}}
								>
									{getInitial(item, cols.contact)}
								</div>
							) : (
								<div className={`w-6 h-6 rounded-full shrink-0 ${dark ? 'bg-neutral-700' : 'bg-neutral-100'}`} />
							)
						)}

						{/* Text content */}
						<div className="flex-1 min-w-0">
							{item ? (
								<>
									<div className="flex items-baseline gap-1.5">
										{/* Contact name (bold) */}
										{cols.contact && (
											<span className={`text-[10px] shrink-0 ${
												isUnread ? 'font-bold' : 'font-medium'
											} ${dark ? 'text-neutral-200' : 'text-neutral-700'}`}>
												{resolveValue(item, cols.contact)}
											</span>
										)}
										{/* Primary field (subject) */}
										{cols.primary && (
											<span className={`text-[10px] truncate ${
												isUnread && !cols.contact ? 'font-bold' : ''
											} ${dark ? 'text-neutral-300' : 'text-neutral-600'}`}>
												{cols.contact ? '— ' : ''}{resolveValue(item, cols.primary)}
											</span>
										)}
									</div>
									{/* Secondary line */}
									{cols.secondary && !compact && (
										<div className={`text-[9px] truncate ${dark ? 'text-neutral-500' : 'text-neutral-400'}`}>
											{resolveValue(item, cols.secondary)}
										</div>
									)}
								</>
							) : (
								<>
									<div className={`h-[7px] rounded-sm mb-1 ${dark ? 'bg-neutral-700' : 'bg-neutral-100'}`}
										style={{ width: `${45 + (i % 3) * 15}%` }} />
									{!compact && (
										<div className={`h-[6px] rounded-sm ${dark ? 'bg-neutral-800' : 'bg-neutral-50'}`}
											style={{ width: `${55 + (i % 2) * 20}%` }} />
									)}
								</>
							)}
						</div>

						{/* Date meta */}
						{cols.meta && (
							<div className="shrink-0">
								{item ? (
									<span className={`text-[9px] ${dark ? 'text-neutral-500' : 'text-neutral-400'}`}>
										{resolveValue(item, cols.meta)}
									</span>
								) : (
									<div className={`w-8 h-[6px] rounded-sm ${dark ? 'bg-neutral-700' : 'bg-neutral-100'}`} />
								)}
							</div>
						)}

						{/* Unread dot */}
						{cols.booleans.length > 0 && (
							<div
								className={`w-1.5 h-1.5 rounded-full shrink-0 ${
									isUnread ? '' : dark ? 'bg-neutral-700' : 'bg-transparent'
								}`}
								style={isUnread ? { backgroundColor: '#3b82f6' } : undefined}
							/>
						)}
					</div>
				)
			})}
		</div>
	)
}
