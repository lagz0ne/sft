import { useState, useRef, useEffect } from 'react'
import { ArrowLeft, Monitor, Smartphone, Tablet, Maximize, Play, Layers, ChevronUp } from 'lucide-react'

// Consistent sizes — everything aligns to a 20px row height
const ICON = 11
const ROW = 'h-5' // 20px — unified height for all interactive elements

// --- Segment ---

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
							<ChevronUp size={8} />
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

// --- Flow ---

function FlowStrip({ steps, currentIndex, onStep }: {
	steps: { type: string; name: string }[]
	currentIndex: number
	onStep: (index: number) => void
}) {
	const ic: Record<string, string> = { screen: '◻', back: '←', region: '▪', event: '⚡', action: '▶', activate: '●' }
	return (
		<div className={`flex items-center overflow-x-auto max-w-[200px] ${ROW}`}>
			<button disabled={currentIndex <= 0} onClick={() => onStep(currentIndex - 1)}
				className={`${ROW} w-4 flex items-center justify-center text-stone-400 hover:text-stone-600 disabled:opacity-20 shrink-0`}>
				<ArrowLeft size={9} />
			</button>
			{steps.map((step, i) => (
				<button key={i} onClick={() => onStep(i)}
					className={`${ROW} px-1 text-[7px] leading-none shrink-0 transition-colors duration-100 ${
						i === currentIndex ? 'text-stone-800 font-semibold' : 'text-stone-400 hover:text-stone-600'
					}`}>
					<span className="opacity-40 mr-px">{ic[step.type] ?? '·'}</span>{step.name}
				</button>
			))}
			<button disabled={currentIndex >= steps.length - 1} onClick={() => onStep(currentIndex + 1)}
				className={`${ROW} w-4 flex items-center justify-center text-stone-400 hover:text-stone-600 disabled:opacity-20 shrink-0 rotate-180`}>
				<ArrowLeft size={9} />
			</button>
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
		<div
			className="fixed bottom-2 left-1/2 -translate-x-1/2 z-40 flex items-center bg-white/92 backdrop-blur-lg rounded-[10px] px-1 max-w-[95vw]"
			style={{ boxShadow: '0 1px 12px rgba(0,0,0,0.06), 0 0 0 0.5px rgba(0,0,0,0.06)' }}
		>
			{/* Back */}
			<button onClick={onBack} title="Back"
				className={`${ROW} w-5 flex items-center justify-center text-stone-400 hover:text-stone-600 transition-colors duration-100`}>
				<ArrowLeft size={ICON} strokeWidth={1.4} />
			</button>

			<Div />

			{/* Mode */}
			{hasFlows && (
				<>
					<div className={`flex items-center ${ROW}`}>
						<button onClick={() => mode !== 'screen' && onModeToggle()} title="Screen"
							className={`${ROW} w-5 flex items-center justify-center transition-colors duration-100 ${
								mode === 'screen' ? 'text-stone-700' : 'text-stone-300 hover:text-stone-500'
							}`}><Layers size={ICON} strokeWidth={mode === 'screen' ? 1.8 : 1.2} /></button>
						<button onClick={() => mode !== 'flow' && onModeToggle()} title="Flow"
							className={`${ROW} w-5 flex items-center justify-center transition-colors duration-100 ${
								mode === 'flow' ? 'text-stone-700' : 'text-stone-300 hover:text-stone-500'
							}`}><Play size={ICON} strokeWidth={mode === 'flow' ? 1.8 : 1.2} /></button>
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
				<><Div /><Segment items={states} onSelect={onState} accent="#34a065" /></>
			) : null}

			{/* Viewport */}
			<Div />
			<ViewportControl sizes={viewportSizes} activeWidth={activeViewportWidth} onSelect={onViewportSize} />

			{/* Layout */}
			{layouts.length > 0 && <><Div /><Segment items={layouts} onSelect={onLayout} /></>}

			{/* Taste */}
			{tastes.length > 0 && <><Div /><Segment items={tastes} onSelect={onTaste} /></>}
		</div>
	)
}

export { PRESET_SIZES }
export type { ViewportSize as ViewportSizeType }
