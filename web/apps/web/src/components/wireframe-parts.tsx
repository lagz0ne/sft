import type { Region } from '../lib/types'
import { parseComponentProps } from '../lib/component-props'

// --- DataList ---

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
	contact?: string
	primary?: string
	secondary?: string
	meta?: string
	booleans: string[]
	hasCheckEvent: boolean
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

	if (!cols.primary) {
		cols.primary = Object.keys(fields).find(k =>
			k !== cols.contact && k !== cols.meta && !cols.booleans.includes(k)
		)
	}

	return cols
}

function resolveItems(fixtureData: Record<string, any> | null | undefined, screenName?: string): any[] {
	if (!fixtureData) return []

	let data = fixtureData
	if (screenName && fixtureData[screenName] && typeof fixtureData[screenName] === 'object') {
		data = fixtureData[screenName]
	}

	if (Array.isArray(data)) return data
	for (const val of Object.values(data)) {
		if (Array.isArray(val) && val.length > 0) return val
	}
	return []
}

export function DataList({ region, fixtureData, screenName, compact }: {
	region: Region
	fixtureData?: Record<string, any> | null
	screenName?: string
	compact?: boolean
}) {
	const fields: Record<string, string> = {}
	const events = region.events ?? []
	const cols = pickColumns(fields, events)
	const items = resolveItems(fixtureData, screenName)

	const dark = false
	const rowPy = 'py-1.5'
	const rowBorder = dark ? 'border-neutral-700' : 'border-neutral-100'
	const hoverBg = dark ? 'bg-neutral-800' : 'bg-blue-50/40'

	const realCount = Math.min(items.length, compact ? 3 : 4)
	const placeholderCount = compact ? 1 : Math.max(1, 5 - realCount)
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
										{cols.contact && (
											<span className={`text-[10px] shrink-0 ${
												isUnread ? 'font-bold' : 'font-medium'
											} ${dark ? 'text-neutral-200' : 'text-neutral-700'}`}>
												{resolveValue(item, cols.contact)}
											</span>
										)}
										{cols.primary && (
											<span className={`text-[10px] truncate ${
												isUnread && !cols.contact ? 'font-bold' : ''
											} ${dark ? 'text-neutral-300' : 'text-neutral-600'}`}>
												{cols.contact ? '— ' : ''}{resolveValue(item, cols.primary)}
											</span>
										)}
									</div>
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

// --- Tabs ---

export function Tabs({ enumValues, compact: _compact }: {
	enumValues?: string[]
	compact?: boolean
}) {
	const values = enumValues ?? []

	if (values.length === 0) return null

	const dark = false
	const shapeClass = 'rounded-full'
	const inactiveClass = dark ? 'bg-neutral-700 text-neutral-400' : 'bg-neutral-100 text-neutral-500'

	return (
		<div className="flex items-center gap-1 w-full">
			{values.map((val, i) => {
				const isActive = i === 0
				const activeClass = isActive
					? (dark ? 'bg-neutral-500 text-white' : 'bg-neutral-700 text-white')
					: inactiveClass

				return (
					<div
						key={val}
						className={`px-2 py-0.5 ${shapeClass} text-[8px] font-medium ${activeClass}`}
					>
						{val.replace(/_/g, ' ')}
					</div>
				)
			})}
		</div>
	)
}

// --- Placeholder ---

function isNavLike(desc: string): boolean {
	const dl = desc.toLowerCase()
	return dl.includes('nav') || dl.includes('sidebar') || dl.includes('menu') ||
		dl.includes('folder') || dl.includes('shortcut')
}

function isSettingsLike(desc: string): boolean {
	const dl = desc.toLowerCase()
	return dl.includes('setting') || dl.includes('config') || dl.includes('preference')
}

const NAV_ITEMS = ['Inbox', 'Starred', 'Sent', 'Drafts', 'Spam', 'Trash']

function NavItems({ compact, dark }: { compact?: boolean; dark?: boolean }) {
	const items = NAV_ITEMS.slice(0, compact ? 4 : 6)
	const accent = '#2563eb'
	const py = 'py-1'

	return (
		<div className="flex flex-col gap-0.5 w-full">
			{items.map((label, i) => {
				const active = i === 0
				return (
					<div
						key={label}
						className={[
							'flex items-center gap-2 px-2 rounded',
							py,
							active ? '' : '',
						].join(' ')}
						style={active ? { backgroundColor: `${accent}15` } : undefined}
					>
						<div
							className={`w-3.5 h-3.5 rounded shrink-0 ${
								active ? '' : dark ? 'bg-neutral-700' : 'bg-neutral-100'
							}`}
							style={active ? { backgroundColor: `${accent}30` } : undefined}
						/>
						<span className={`text-[10px] ${
							active ? 'font-semibold' : dark ? 'text-neutral-400' : 'text-neutral-500'
						}`} style={active ? { color: accent } : undefined}>
							{label}
						</span>
						{active && (
							<span
								className="text-[8px] ml-auto font-medium rounded-full px-1.5"
								style={{ backgroundColor: accent, color: '#fff' }}
							>
								12
							</span>
						)}
					</div>
				)
			})}
		</div>
	)
}

function SettingsItems({ compact, dark }: { compact?: boolean; dark?: boolean }) {
	const items = ['General', 'Notifications', 'Privacy', 'Account'].slice(0, compact ? 3 : 4)
	return (
		<div className="flex flex-col gap-1 w-full">
			{items.map((label, i) => (
				<div key={label} className={`flex items-center gap-2 px-2 py-1.5 ${i === 0 ? (dark ? 'bg-neutral-700' : 'bg-neutral-100') : ''} rounded`}>
					<div className={`w-3 h-3 rounded ${dark ? 'bg-neutral-600' : 'bg-neutral-200'}`} />
					<span className={`text-[10px] ${dark ? 'text-neutral-300' : 'text-neutral-600'}`}>{label}</span>
				</div>
			))}
		</div>
	)
}

function ContentLines({ compact, dark }: { compact?: boolean; dark?: boolean }) {
	const headingClass = dark ? 'bg-neutral-600' : 'bg-neutral-200'
	const lineClass = dark ? 'bg-neutral-700' : 'bg-neutral-100'
	const lineLight = dark ? 'bg-neutral-800' : 'bg-neutral-50'

	return (
		<div className="flex flex-col gap-1.5 w-full">
			<div className={`h-[8px] rounded-sm w-1/2 mb-1 ${headingClass}`} />
			<div className={`h-[5px] rounded-sm w-full ${lineClass}`} />
			<div className={`h-[5px] rounded-sm w-11/12 ${lineClass}`} />
			{!compact && (
				<>
					<div className={`h-[5px] rounded-sm w-4/5 ${lineLight}`} />
					<div className={`h-[5px] rounded-sm w-5/6 ${lineLight}`} />
					<div className={`h-[5px] rounded-sm w-3/5 ${lineLight}`} />
				</>
			)}
		</div>
	)
}

function TableItems({ columns, compact, dark }: { columns: string[]; compact?: boolean; dark?: boolean }) {
	const headerBg = dark ? 'bg-neutral-700' : 'bg-neutral-100'
	const cellBg = dark ? 'bg-neutral-800' : 'bg-neutral-50'
	const borderColor = dark ? 'border-neutral-700' : 'border-neutral-200'
	const rowCount = compact ? 3 : 5

	return (
		<div className="flex flex-col w-full">
			<div className={`flex gap-2 px-2 py-1 border-b ${borderColor} ${headerBg}`}>
				{columns.map(col => (
					<span key={col} className={`text-[8px] font-semibold flex-1 ${dark ? 'text-neutral-400' : 'text-neutral-500'}`}>{col}</span>
				))}
			</div>
			{Array.from({ length: rowCount }, (_, i) => (
				<div key={i} className={`flex gap-2 px-2 py-1.5 ${i < rowCount - 1 ? `border-b ${borderColor}` : ''}`}>
					{columns.map((col, j) => (
						<div key={col} className={`h-[6px] rounded-sm flex-1 ${cellBg}`} style={{ width: `${40 + ((i + j) % 3) * 20}%` }} />
					))}
				</div>
			))}
		</div>
	)
}

function FormItems({ fields, compact, dark }: { fields: string[]; compact?: boolean; dark?: boolean }) {
	const items = fields.slice(0, compact ? 3 : 5)
	const labelColor = dark ? 'text-neutral-400' : 'text-neutral-500'
	const inputBg = dark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-200'

	return (
		<div className="flex flex-col gap-2 w-full">
			{items.map(label => (
				<div key={label} className="flex flex-col gap-0.5">
					<span className={`text-[8px] font-medium ${labelColor}`}>{label}</span>
					<div className={`h-5 rounded border ${inputBg}`} />
				</div>
			))}
		</div>
	)
}

function StackItems({ items, compact, dark }: { items: string[]; compact?: boolean; dark?: boolean }) {
	const shown = items.slice(0, compact ? 3 : 5)
	const cardBg = dark ? 'bg-neutral-800 border-neutral-700' : 'bg-white border-neutral-100'

	return (
		<div className="flex flex-col gap-1 w-full">
			{shown.map((label, i) => (
				<div key={i} className={`flex items-center gap-2 px-2 py-1.5 rounded border ${cardBg}`}>
					<div className={`w-3 h-3 rounded ${dark ? 'bg-neutral-700' : 'bg-neutral-200'}`} />
					<span className={`text-[9px] ${dark ? 'text-neutral-300' : 'text-neutral-600'}`}>{label}</span>
				</div>
			))}
		</div>
	)
}

export function Placeholder({ region, compact }: {
	region: Region
	compact?: boolean
}) {
	const desc = region.description ?? region.name
	const dark = false
	const props = parseComponentProps(region.component_props)

	if (props.columns && Array.isArray(props.columns)) return <TableItems columns={props.columns} compact={compact} dark={dark} />
	if (props.fields && Array.isArray(props.fields)) return <FormItems fields={props.fields} compact={compact} dark={dark} />
	if (props.items && Array.isArray(props.items)) return <StackItems items={props.items} compact={compact} dark={dark} />

	if (isNavLike(desc)) return <NavItems compact={compact} dark={dark} />
	if (isSettingsLike(desc)) return <SettingsItems compact={compact} dark={dark} />
	return <ContentLines compact={compact} dark={dark} />
}
