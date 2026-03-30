import type { SkinProps } from './types'
import { parseComponentProps } from '../../lib/component-props'

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

export function Placeholder({ region, compact }: SkinProps) {
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
