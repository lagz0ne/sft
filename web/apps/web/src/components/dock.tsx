import { useState, useRef, useEffect } from 'react'
import { ArrowLeft, Monitor, Smartphone, Tablet, Maximize, Layout, Palette, Play, Layers, ChevronUp } from 'lucide-react'

const I = 12 // icon size

// --- Dock Segment ---

interface DockSegmentProps {
	items: { id: string; label: string; active: boolean }[]
	onSelect: (id: string) => void
	accent?: string
	maxInline?: number
	icon?: React.ReactNode
}

function DockSegment({ items, onSelect, accent, maxInline = 4, icon }: DockSegmentProps) {
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
	const activeBg = accent ?? '#333'
	const activeStyle = { backgroundColor: activeBg, color: '#fff' }

	return (
		<div ref={ref} className="relative flex items-center">
			{icon && <span className="text-neutral-400 mr-0.5">{icon}</span>}
			<div className="flex gap-px bg-neutral-100 rounded p-px">
				{inline ? (
					items.map(item => (
						<button key={item.id}
							onClick={() => { onSelect(item.id); setOpen(false) }}
							className={`px-1 py-px text-[7px] rounded-sm transition-colors leading-tight ${
								item.active ? 'font-semibold' : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-200'
							}`}
							style={item.active ? activeStyle : undefined}
						>{item.label}</button>
					))
				) : (
					<>
						{active && (
							<button className="px-1 py-px text-[7px] rounded-sm font-semibold leading-tight"
								style={activeStyle} onClick={() => setOpen(!open)}
							>{active.label}</button>
						)}
						<button onClick={() => setOpen(!open)}
							className="px-0.5 py-px text-neutral-400 hover:text-neutral-600 rounded-sm hover:bg-neutral-200">
							<ChevronUp size={8} />
						</button>
					</>
				)}
			</div>

			{open && (
				<div className="absolute bottom-full left-0 right-0 mb-1 bg-white rounded-md shadow-lg border border-neutral-200 overflow-hidden z-50"
					style={{ minWidth: ref.current?.offsetWidth }}>
					<div className="max-h-48 overflow-y-auto py-px">
						{items.map(item => (
							<button key={item.id}
								onClick={() => { onSelect(item.id); setOpen(false) }}
								className={`w-full px-2 py-0.5 text-[7px] text-left flex items-center gap-1 transition-colors leading-tight ${
									item.active ? 'font-semibold' : 'text-neutral-600 hover:bg-neutral-50'
								}`}
								style={item.active ? { color: activeBg } : undefined}
							>
								{item.active && <span className="w-1 h-1 rounded-full shrink-0" style={{ backgroundColor: activeBg }} />}
								<span>{item.label}</span>
							</button>
						))}
					</div>
				</div>
			)}
		</div>
	)
}

// --- Viewport Size Control ---

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

const SIZE_ICONS: Record<string, React.ReactNode> = {
	maximize: <Maximize size={9} />,
	monitor: <Monitor size={9} />,
	tablet: <Tablet size={9} />,
	phone: <Smartphone size={9} />,
}

function ViewportControl({ sizes, activeWidth, onSelect }: {
	sizes: ViewportSize[]
	activeWidth: number | null
	onSelect: (size: ViewportSize) => void
}) {
	return (
		<div className="flex gap-px bg-neutral-100 rounded p-px">
			{sizes.map(s => (
				<button key={s.label} onClick={() => onSelect(s)}
					title={s.label}
					className={`px-1 py-px rounded-sm transition-colors ${
						activeWidth === s.width ? 'bg-neutral-700 text-white' : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-200'
					}`}
				>{s.icon ? SIZE_ICONS[s.icon] : <span className="text-[7px] leading-tight">{s.label}</span>}</button>
			))}
		</div>
	)
}

// --- Flow Step Strip ---

function FlowStrip({ steps, currentIndex, onStep }: {
	steps: { type: string; name: string }[]
	currentIndex: number
	onStep: (index: number) => void
}) {
	const icons: Record<string, string> = { screen: '◻', back: '←', region: '▪', event: '⚡', action: '▶', activate: '●' }
	return (
		<div className="flex items-center gap-px bg-neutral-100 rounded p-px overflow-x-auto max-w-[200px]">
			<button disabled={currentIndex <= 0} onClick={() => onStep(currentIndex - 1)}
				className="px-0.5 py-px text-[7px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0">◀</button>
			{steps.map((step, i) => (
				<button key={i} onClick={() => onStep(i)}
					className={`px-1 py-px text-[6px] rounded-sm shrink-0 transition-colors leading-tight ${
						i === currentIndex ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
					}`}>
					<span className="opacity-60 mr-px">{icons[step.type] ?? '·'}</span>{step.name}
				</button>
			))}
			<button disabled={currentIndex >= steps.length - 1} onClick={() => onStep(currentIndex + 1)}
				className="px-0.5 py-px text-[7px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0">▶</button>
		</div>
	)
}

function Sep() { return <div className="w-px h-3 bg-neutral-200 mx-px" /> }

// --- Main Dock ---

export interface DockProps {
	screens: { id: string; label: string; active: boolean }[]
	onScreen: (id: string) => void
	states: { id: string; label: string; active: boolean }[]
	onState: (id: string) => void
	layouts: { id: string; label: string; active: boolean }[]
	onLayout: (id: string) => void
	tastes: { id: string; label: string; active: boolean }[]
	onTaste: (id: string) => void
	flowMode?: boolean
	flowSteps?: { type: string; name: string }[]
	flowIndex?: number
	onFlowStep?: (index: number) => void
	mode: 'screen' | 'flow'
	onModeToggle: () => void
	hasFlows: boolean
	onBack: () => void
	viewportSizes: ViewportSize[]
	activeViewportWidth: number | null
	onViewportSize: (size: ViewportSize) => void
}

export function Dock({
	screens, onScreen,
	states, onState,
	layouts, onLayout,
	tastes, onTaste,
	flowMode, flowSteps, flowIndex, onFlowStep,
	mode, onModeToggle, hasFlows,
	onBack,
	viewportSizes, activeViewportWidth, onViewportSize,
}: DockProps) {
	return (
		<div className="fixed bottom-2 left-1/2 -translate-x-1/2 z-40 flex items-center gap-px bg-white/95 backdrop-blur-sm rounded-lg shadow-md border border-neutral-200 px-1 py-0.5 max-w-[95vw]">
			{/* Back */}
			<button onClick={onBack} className="text-neutral-400 hover:text-neutral-600 p-0.5" title="Back">
				<ArrowLeft size={I} />
			</button>
			<Sep />

			{/* Mode */}
			{hasFlows && (
				<>
					<div className="flex gap-px bg-neutral-100 rounded p-px">
						<button onClick={() => mode !== 'screen' && onModeToggle()} title="Screen mode"
							className={`p-0.5 rounded-sm transition-colors ${
								mode === 'screen' ? 'bg-neutral-800 text-white' : 'text-neutral-500 hover:bg-neutral-200'
							}`}><Layers size={10} /></button>
						<button onClick={() => mode !== 'flow' && onModeToggle()} title="Flow mode"
							className={`p-0.5 rounded-sm transition-colors ${
								mode === 'flow' ? 'bg-neutral-800 text-white' : 'text-neutral-500 hover:bg-neutral-200'
							}`}><Play size={10} /></button>
					</div>
					<Sep />
				</>
			)}

			{/* Screen */}
			<DockSegment items={screens} onSelect={onScreen} />
			<Sep />

			{/* State / Flow */}
			{flowMode && flowSteps && flowIndex != null && onFlowStep ? (
				<FlowStrip steps={flowSteps} currentIndex={flowIndex} onStep={onFlowStep} />
			) : (
				states.length > 0 && <DockSegment items={states} onSelect={onState} accent="#059669" />
			)}

			{/* Viewport */}
			<Sep />
			<ViewportControl sizes={viewportSizes} activeWidth={activeViewportWidth} onSelect={onViewportSize} />

			{/* Layout */}
			{layouts.length > 0 && (
				<>
					<Sep />
					<DockSegment items={layouts} onSelect={onLayout} icon={<Layout size={9} />} />
				</>
			)}

			{/* Taste */}
			{tastes.length > 0 && (
				<>
					<Sep />
					<DockSegment items={tastes} onSelect={onTaste} icon={<Palette size={9} />} />
				</>
			)}
		</div>
	)
}

export { PRESET_SIZES }
export type { ViewportSize as ViewportSizeType }
