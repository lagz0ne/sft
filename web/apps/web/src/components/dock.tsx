import { useState, useRef, useEffect } from 'react'
import { ArrowLeft, Monitor, Smartphone, Tablet, Maximize, Play, Layers, ChevronUp } from 'lucide-react'

// --- Segment: a group of switchable items ---

interface SegmentProps {
	items: { id: string; label: string; active: boolean }[]
	onSelect: (id: string) => void
	accent?: string
	maxInline?: number
	label?: string
}

function Segment({ items, onSelect, accent, maxInline = 4, label }: SegmentProps) {
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
	const accentColor = accent ?? 'oklch(0.35 0.02 250)'

	return (
		<div ref={ref} className="relative flex items-center gap-1">
			{label && <span className="text-[8px] text-stone-400 select-none">{label}</span>}
			<div className="flex items-center">
				{inline ? (
					items.map(item => (
						<button key={item.id}
							onClick={() => { onSelect(item.id); setOpen(false) }}
							className={`px-1.5 py-0.5 text-[9px] rounded transition-all duration-150 ${
								item.active
									? 'text-stone-900 font-medium'
									: 'text-stone-400 hover:text-stone-600'
							}`}
							style={item.active ? { color: accentColor } : undefined}
						>
							{item.active && <span className="inline-block w-1 h-1 rounded-full mr-0.5 -translate-y-px" style={{ backgroundColor: accentColor }} />}
							{item.label}
						</button>
					))
				) : (
					<>
						{active && (
							<button className="px-1.5 py-0.5 text-[9px] font-medium rounded transition-all duration-150"
								style={{ color: accentColor }}
								onClick={() => setOpen(!open)}
							>
								<span className="inline-block w-1 h-1 rounded-full mr-0.5 -translate-y-px" style={{ backgroundColor: accentColor }} />
								{active.label}
							</button>
						)}
						<button onClick={() => setOpen(!open)}
							className="text-stone-400 hover:text-stone-500 transition-colors">
							<ChevronUp size={10} />
						</button>
					</>
				)}
			</div>

			{/* Popover */}
			{open && (
				<div className="absolute bottom-full left-0 mb-2 bg-white rounded-lg shadow-xl ring-1 ring-stone-200/60 overflow-hidden z-50 min-w-[120px]"
					style={{ minWidth: ref.current?.offsetWidth }}>
					<div className="max-h-52 overflow-y-auto py-1">
						{items.map(item => (
							<button key={item.id}
								onClick={() => { onSelect(item.id); setOpen(false) }}
								className={`w-full px-3 py-1.5 text-[9px] text-left flex items-center gap-1.5 transition-colors ${
									item.active ? 'font-medium bg-stone-50' : 'text-stone-500 hover:bg-stone-50 hover:text-stone-700'
								}`}
								style={item.active ? { color: accentColor } : undefined}
							>
								{item.active && <span className="w-1 h-1 rounded-full shrink-0" style={{ backgroundColor: accentColor }} />}
								<span>{item.label}</span>
							</button>
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

const sizeIcon = (name: string, active: boolean) => {
	const color = active ? 'oklch(0.35 0.02 250)' : undefined
	const props = { size: 13, strokeWidth: active ? 2 : 1.5, style: { color } }
	switch (name) {
		case 'maximize': return <Maximize {...props} />
		case 'monitor': return <Monitor {...props} />
		case 'tablet': return <Tablet {...props} />
		case 'phone': return <Smartphone {...props} />
		default: return null
	}
}

function ViewportControl({ sizes, activeWidth, onSelect }: {
	sizes: ViewportSize[]
	activeWidth: number | null
	onSelect: (size: ViewportSize) => void
}) {
	return (
		<div className="flex items-center gap-0.5">
			{sizes.map(s => {
				const active = activeWidth === s.width
				return (
					<button key={s.label} onClick={() => onSelect(s)} title={s.label}
						className={`p-0.5 rounded transition-all duration-150 ${
							active ? 'text-stone-800' : 'text-stone-300 hover:text-stone-500'
						}`}
					>
						{s.icon ? sizeIcon(s.icon, active) : <span className="text-[9px]">{s.label}</span>}
					</button>
				)
			})}
		</div>
	)
}

// --- Flow Steps ---

function FlowStrip({ steps, currentIndex, onStep }: {
	steps: { type: string; name: string }[]
	currentIndex: number
	onStep: (index: number) => void
}) {
	const icons: Record<string, string> = { screen: '◻', back: '←', region: '▪', event: '⚡', action: '▶', activate: '●' }
	return (
		<div className="flex items-center gap-0.5 overflow-x-auto max-w-[220px]">
			<button disabled={currentIndex <= 0} onClick={() => onStep(currentIndex - 1)}
				className="text-stone-400 hover:text-stone-600 disabled:opacity-20 shrink-0 p-0.5">
				<ArrowLeft size={10} />
			</button>
			{steps.map((step, i) => {
				const active = i === currentIndex
				return (
					<button key={i} onClick={() => onStep(i)}
						className={`px-1 py-0.5 text-[8px] rounded shrink-0 transition-all duration-150 ${
							active ? 'text-stone-900 font-medium bg-stone-100' : 'text-stone-400 hover:text-stone-600'
						}`}>
						<span className="opacity-50 mr-px">{icons[step.type] ?? '·'}</span>{step.name}
					</button>
				)
			})}
			<button disabled={currentIndex >= steps.length - 1} onClick={() => onStep(currentIndex + 1)}
				className="text-stone-400 hover:text-stone-600 disabled:opacity-20 shrink-0 p-0.5 rotate-180">
				<ArrowLeft size={10} />
			</button>
		</div>
	)
}

// --- Divider ---

function Div() { return <div className="w-px h-3.5 bg-stone-200/60 mx-1" /> }

// --- Dock ---

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
		<div className="fixed bottom-3 left-1/2 -translate-x-1/2 z-40 flex items-center bg-white/90 backdrop-blur-md rounded-xl shadow-[0_2px_20px_rgba(0,0,0,0.08)] ring-1 ring-stone-200/50 px-2 py-1 max-w-[95vw]">
			{/* Back */}
			<button onClick={onBack} className="text-stone-400 hover:text-stone-600 transition-colors p-0.5 mr-1" title="Back">
				<ArrowLeft size={14} strokeWidth={1.5} />
			</button>

			<Div />

			{/* Mode */}
			{hasFlows && (
				<>
					<div className="flex items-center gap-0.5">
						<button onClick={() => mode !== 'screen' && onModeToggle()} title="Screen mode"
							className={`p-0.5 rounded transition-all duration-150 ${
								mode === 'screen' ? 'text-stone-800' : 'text-stone-300 hover:text-stone-500'
							}`}><Layers size={13} strokeWidth={mode === 'screen' ? 2 : 1.5} /></button>
						<button onClick={() => mode !== 'flow' && onModeToggle()} title="Flow mode"
							className={`p-0.5 rounded transition-all duration-150 ${
								mode === 'flow' ? 'text-stone-800' : 'text-stone-300 hover:text-stone-500'
							}`}><Play size={13} strokeWidth={mode === 'flow' ? 2 : 1.5} /></button>
					</div>
					<Div />
				</>
			)}

			{/* Screens */}
			<Segment items={screens} onSelect={onScreen} />

			{/* States / Flow */}
			{flowMode && flowSteps && flowIndex != null && onFlowStep ? (
				<><Div /><FlowStrip steps={flowSteps} currentIndex={flowIndex} onStep={onFlowStep} /></>
			) : states.length > 0 ? (
				<><Div /><Segment items={states} onSelect={onState} accent="oklch(0.55 0.15 160)" /></>
			) : null}

			{/* Viewport */}
			<Div />
			<ViewportControl sizes={viewportSizes} activeWidth={activeViewportWidth} onSelect={onViewportSize} />

			{/* Layout */}
			{layouts.length > 0 && (
				<><Div /><Segment items={layouts} onSelect={onLayout} label="layout" /></>
			)}

			{/* Taste */}
			{tastes.length > 0 && (
				<><Div /><Segment items={tastes} onSelect={onTaste} label="taste" /></>
			)}
		</div>
	)
}

export { PRESET_SIZES }
export type { ViewportSize as ViewportSizeType }
