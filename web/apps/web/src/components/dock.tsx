import { useState, useRef, useEffect } from 'react'
import { Monitor, Smartphone, Tablet, Maximize, Layers, ChevronDown, Search, X } from 'lucide-react'

const ICON = 11
const ROW = 'h-5'

// --- Picker ---
// Popover panel for selecting from large lists (screens)

interface PickerItem { id: string; label: string; active: boolean }

function Picker({ items, onSelect, accent }: {
	items: PickerItem[]
	onSelect: (id: string) => void
	accent?: string
}) {
	const [open, setOpen] = useState(false)
	const [query, setQuery] = useState('')
	const ref = useRef<HTMLDivElement>(null)
	const inputRef = useRef<HTMLInputElement>(null)

	useEffect(() => {
		if (!open) return
		const h = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false) }
		document.addEventListener('mousedown', h)
		return () => document.removeEventListener('mousedown', h)
	}, [open])

	useEffect(() => {
		if (open) {
			setQuery('')
			requestAnimationFrame(() => inputRef.current?.focus())
		}
	}, [open])

	if (items.length === 0) return null

	const active = items.find(i => i.active)
	const ac = accent ?? '#48484a'
	const showSearch = items.length > 6
	const filtered = query
		? items.filter(i => i.label.toLowerCase().includes(query.toLowerCase()))
		: items

	return (
		<div ref={ref} className="relative">
			<button
				onClick={() => setOpen(!open)}
				className={`${ROW} flex items-center gap-0.5 px-1.5 text-[9px] leading-none font-medium transition-colors duration-100`}
				style={{ color: ac }}
			>
				<span className="truncate max-w-[120px]">{active?.label ?? 'Select'}</span>
				<ChevronDown size={7} className={`opacity-50 transition-transform duration-150 shrink-0 ${open ? 'rotate-180' : ''}`} />
			</button>

			{open && (
				<div
					className="absolute bottom-full left-0 mb-2 bg-white rounded-lg overflow-hidden z-50"
					style={{
						minWidth: 180,
						maxWidth: 280,
						boxShadow: '0 8px 32px rgba(0,0,0,0.08), 0 0 0 0.5px rgba(0,0,0,0.06)',
					}}
				>
					{showSearch && (
						<div className="flex items-center gap-1.5 px-2.5 h-8 border-b border-stone-100">
							<Search size={10} className="text-stone-300 shrink-0" />
							<input
								ref={inputRef}
								value={query}
								onChange={e => setQuery(e.target.value)}
								placeholder="Search…"
								className="flex-1 text-[10px] bg-transparent outline-none placeholder:text-stone-300"
							/>
							{query && (
								<button onClick={() => setQuery('')} className="text-stone-300 hover:text-stone-500 shrink-0">
									<X size={8} />
								</button>
							)}
						</div>
					)}
					<div className="max-h-60 overflow-y-auto py-0.5 overscroll-contain">
						{filtered.length === 0 ? (
							<div className="px-2.5 py-3 text-[9px] text-stone-300 text-center">No matches</div>
						) : (
							filtered.map(item => (
								<button
									key={item.id}
									onClick={() => { onSelect(item.id); setOpen(false) }}
									className={`w-full h-7 px-2.5 text-[10px] text-left flex items-center gap-2 transition-colors duration-100 ${
										item.active ? 'font-semibold text-stone-800' : 'text-stone-500 hover:bg-stone-50 hover:text-stone-700'
									}`}
								>
									<span
										className={`w-1 h-1 rounded-full shrink-0 transition-opacity ${item.active ? 'opacity-100' : 'opacity-0'}`}
										style={{ backgroundColor: ac }}
									/>
									<span className="truncate">{item.label}</span>
								</button>
							))
						)}
					</div>
				</div>
			)}
		</div>
	)
}

// --- Segment (kept for small lists: states, layouts, component sets) ---

interface SegmentProps {
	items: { id: string; label: string; active: boolean }[]
	onSelect: (id: string) => void
	accent?: string
	maxInline?: number
}

function Segment({ items, onSelect, accent, maxInline = 4 }: SegmentProps) {
	const [open, setOpen] = useState(false)
	const ref = useRef<HTMLDivElement>(null)

	useEffect(() => {
		if (!open) return
		const h = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false) }
		document.addEventListener('mousedown', h)
		return () => document.removeEventListener('mousedown', h)
	}, [open])

	if (items.length === 0) return null

	const active = items.find(i => i.active)
	const inline = items.length <= maxInline
	const ac = accent ?? '#48484a'

	return (
		<div ref={ref} className="relative">
			<div className={`flex items-center ${ROW}`}>
				{inline ? (
					items.map(item => (
						<button key={item.id}
							onClick={() => { onSelect(item.id); setOpen(false) }}
							className={`${ROW} px-1.5 text-[8px] leading-none transition-colors duration-100 ${
								item.active ? 'font-semibold' : 'text-stone-400 hover:text-stone-600'
							}`}
							style={item.active ? { color: ac } : undefined}
						>{item.label}</button>
					))
				) : (
					<>
						{active && (
							<button
								className={`${ROW} px-1.5 text-[8px] leading-none font-semibold transition-colors duration-100`}
								style={{ color: ac }}
								onClick={() => setOpen(!open)}
							>{active.label}</button>
						)}
						<button onClick={() => setOpen(!open)}
							className={`${ROW} flex items-center text-stone-400 hover:text-stone-500 transition-colors duration-100`}>
							<ChevronDown size={7} className={`transition-transform duration-150 ${open ? 'rotate-180' : ''}`} />
						</button>
					</>
				)}
			</div>

			{open && (
				<div className="absolute bottom-full left-0 mb-1.5 bg-white rounded-lg overflow-hidden z-50"
					style={{
						minWidth: Math.max(ref.current?.offsetWidth ?? 0, 100),
						boxShadow: '0 4px 24px rgba(0,0,0,0.1), 0 0 0 1px rgba(0,0,0,0.04)',
					}}>
					<div className="max-h-52 overflow-y-auto py-0.5">
						{items.map(item => (
							<button key={item.id}
								onClick={() => { onSelect(item.id); setOpen(false) }}
								className={`w-full h-6 px-2.5 text-[8px] text-left flex items-center transition-colors duration-100 ${
									item.active ? 'font-semibold' : 'text-stone-500 hover:bg-stone-50'
								}`}
								style={item.active ? { color: ac } : undefined}
							>{item.label}</button>
						))}
					</div>
				</div>
			)}
		</div>
	)
}

// --- Viewport ---

export interface ViewportSize {
	label: string
	width: number | null
	composition: string | null
	icon?: 'monitor' | 'tablet' | 'phone' | 'maximize'
}

const PRESET_SIZES: ViewportSize[] = [
	{ label: 'Full', width: null, composition: null, icon: 'maximize' },
	{ label: '1280', width: 1280, composition: null, icon: 'monitor' },
	{ label: '768', width: 768, composition: null, icon: 'tablet' },
	{ label: '375', width: 375, composition: null, icon: 'phone' },
]

function ViewportControl({ sizes, activeWidth, onSelect }: {
	sizes: ViewportSize[]
	activeWidth: number | null
	onSelect: (size: ViewportSize) => void
}) {
	const icons: Record<string, typeof Monitor> = { maximize: Maximize, monitor: Monitor, tablet: Tablet, phone: Smartphone }
	return (
		<div className={`flex items-center ${ROW}`}>
			{sizes.map(s => {
				const active = activeWidth === s.width
				const Icon = s.icon ? icons[s.icon] : null
				return (
					<button key={s.label} onClick={() => onSelect(s)} title={s.label}
						className={`${ROW} w-5 flex items-center justify-center transition-colors duration-100 ${
							active ? 'text-stone-700' : 'text-stone-300 hover:text-stone-500'
						}`}
					>{Icon ? <Icon size={ICON} strokeWidth={active ? 1.8 : 1.2} /> : <span className="text-[7px]">{s.label}</span>}</button>
				)
			})}
		</div>
	)
}

// --- Divider ---

function Div() { return <div className="w-px h-3 bg-stone-200/50 mx-0.5 shrink-0" /> }

// --- Dock ---

export interface DockProps {
	screens: { id: string; label: string; active: boolean }[]
	onScreen: (id: string) => void
	states: { id: string; label: string; active: boolean }[]
	onState: (id: string) => void
	layouts: { id: string; label: string; active: boolean }[]
	onLayout: (id: string) => void
	componentSets: { id: string; label: string; active: boolean }[]
	onComponentSet: (id: string) => void
	viewportSizes: ViewportSize[]
	activeViewportWidth: number | null
	onViewportSize: (size: ViewportSize) => void
}

export function Dock({
	screens, onScreen,
	states, onState,
	layouts, onLayout,
	componentSets, onComponentSet,
	viewportSizes, activeViewportWidth, onViewportSize,
}: DockProps) {
	return (
		<div
			className="fixed bottom-2 left-1/2 -translate-x-1/2 z-40 bg-white/92 backdrop-blur-lg rounded-[10px] max-w-[95vw]"
			style={{ boxShadow: '0 1px 12px rgba(0,0,0,0.06), 0 0 0 0.5px rgba(0,0,0,0.06)' }}
		>
			{/* Controls row */}
			<div className="flex items-center px-1">
				{/* Screen mode icon */}
				<div className={`flex items-center ${ROW}`}>
					<div className={`${ROW} w-5 flex items-center justify-center text-stone-700`}>
						<Layers size={ICON} strokeWidth={1.8} />
					</div>
				</div>
				<Div />

				{/* Screen picker + states */}
				<Picker items={screens} onSelect={onScreen} />
				{states.length > 0 && <><Div /><Segment items={states} onSelect={onState} accent="#34a065" /></>}

				{/* Viewport */}
				<Div />
				<ViewportControl sizes={viewportSizes} activeWidth={activeViewportWidth} onSelect={onViewportSize} />

				{/* Layout */}
				{layouts.length > 0 && <><Div /><Segment items={layouts} onSelect={onLayout} /></>}

				{/* Component Set */}
				{componentSets.length > 0 && <><Div /><Segment items={componentSets} onSelect={onComponentSet} /></>}
			</div>
		</div>
	)
}

export { PRESET_SIZES }
export type { ViewportSize as ViewportSizeType }
