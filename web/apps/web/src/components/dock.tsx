import { useState, useRef, useEffect } from 'react'

// --- Dock Segment ---

interface DockSegmentProps {
	items: { id: string; label: string; active: boolean }[]
	onSelect: (id: string) => void
	accent?: string
	maxInline?: number
}

function DockSegment({ items, onSelect, accent, maxInline = 4 }: DockSegmentProps) {
	const [popoverOpen, setPopoverOpen] = useState(false)
	const segmentRef = useRef<HTMLDivElement>(null)

	useEffect(() => {
		if (!popoverOpen) return
		const handler = (e: MouseEvent) => {
			if (segmentRef.current && !segmentRef.current.contains(e.target as Node)) setPopoverOpen(false)
		}
		document.addEventListener('mousedown', handler)
		return () => document.removeEventListener('mousedown', handler)
	}, [popoverOpen])

	if (items.length === 0) return null

	const activeItem = items.find(i => i.active)
	const showInline = items.length <= maxInline
	const inlineItems = showInline ? items : items.slice(0, 3)
	const overflowCount = showInline ? 0 : items.length - 3
	const activeBg = accent ?? '#333'
	const activeStyle = { backgroundColor: activeBg, color: '#fff' }

	return (
		<div ref={segmentRef} className="relative">
			<div className="flex gap-px bg-neutral-100 rounded p-px">
				{showInline ? (
					inlineItems.map(item => (
						<button
							key={item.id}
							onClick={() => { onSelect(item.id); setPopoverOpen(false) }}
							className={`px-1.5 py-0.5 text-[8px] rounded-sm transition-colors ${
								item.active ? 'font-semibold' : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-200'
							}`}
							style={item.active ? activeStyle : undefined}
						>
							{item.label}
						</button>
					))
				) : (
					<>
						{activeItem && (
							<button
								className="px-1.5 py-0.5 text-[8px] rounded-sm font-semibold"
								style={activeStyle}
								onClick={() => setPopoverOpen(!popoverOpen)}
							>
								{activeItem.label}
							</button>
						)}
						<button
							onClick={() => setPopoverOpen(!popoverOpen)}
							className="px-1 py-0.5 text-[7px] text-neutral-400 hover:text-neutral-600 rounded-sm hover:bg-neutral-200"
						>
							+{overflowCount}
						</button>
					</>
				)}
			</div>

			{popoverOpen && (
				<div
					className="absolute bottom-full left-0 right-0 mb-1 bg-white rounded-md shadow-lg border border-neutral-200 overflow-hidden z-50"
					style={{ minWidth: segmentRef.current?.offsetWidth }}
				>
					<div className="max-h-48 overflow-y-auto py-0.5">
						{items.map(item => (
							<button
								key={item.id}
								onClick={() => { onSelect(item.id); setPopoverOpen(false) }}
								className={`w-full px-2 py-1 text-[8px] text-left flex items-center gap-1 transition-colors ${
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
	width: number | null  // null = full width
	composition: string | null  // linked layout composition
}

const PRESET_SIZES: ViewportSize[] = [
	{ label: 'Full', width: null, composition: null },
	{ label: '1280', width: 1280, composition: null },
	{ label: '768', width: 768, composition: null },
	{ label: '375', width: 375, composition: null },
]

function ViewportControl({ sizes, activeWidth, onSelect }: {
	sizes: ViewportSize[]
	activeWidth: number | null
	onSelect: (size: ViewportSize) => void
}) {
	return (
		<div className="flex gap-px bg-neutral-100 rounded p-px">
			{sizes.map(s => (
				<button
					key={s.label}
					onClick={() => onSelect(s)}
					className={`px-1.5 py-0.5 text-[8px] rounded-sm transition-colors ${
						activeWidth === s.width ? 'bg-neutral-700 text-white font-semibold' : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-200'
					}`}
				>
					{s.label}
				</button>
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
		<div className="flex items-center gap-px bg-neutral-100 rounded p-px overflow-x-auto max-w-xs">
			<button disabled={currentIndex <= 0} onClick={() => onStep(currentIndex - 1)}
				className="px-0.5 py-0.5 text-[8px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0">◀</button>
			{steps.map((step, i) => (
				<button key={i} onClick={() => onStep(i)}
					className={`px-1 py-0.5 text-[7px] rounded-sm shrink-0 transition-colors ${
						i === currentIndex ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
					}`}>
					<span className="opacity-60 mr-px">{icons[step.type] ?? '·'}</span>{step.name}
				</button>
			))}
			<button disabled={currentIndex >= steps.length - 1} onClick={() => onStep(currentIndex + 1)}
				className="px-0.5 py-0.5 text-[8px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0">▶</button>
		</div>
	)
}

// --- Separator ---

function Sep() {
	return <div className="w-px h-3 bg-neutral-200" />
}

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
	// Viewport
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
		<div className="fixed bottom-2 left-1/2 -translate-x-1/2 z-40 flex flex-wrap items-center justify-center gap-1 bg-white/95 backdrop-blur-sm rounded-xl shadow-md border border-neutral-200 px-1.5 py-1 max-w-[95vw]">
			{/* Back */}
			<button onClick={onBack} className="text-[8px] text-neutral-400 hover:text-neutral-600 px-1">←</button>

			<Sep />

			{/* Mode toggle */}
			{hasFlows && (
				<>
					<div className="flex gap-px bg-neutral-100 rounded p-px">
						<button onClick={() => mode !== 'screen' && onModeToggle()}
							className={`px-1.5 py-0.5 text-[8px] rounded-sm transition-colors ${
								mode === 'screen' ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
							}`}>Screen</button>
						<button onClick={() => mode !== 'flow' && onModeToggle()}
							className={`px-1.5 py-0.5 text-[8px] rounded-sm transition-colors ${
								mode === 'flow' ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
							}`}>Flow</button>
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

			{/* Viewport size */}
			<Sep />
			<ViewportControl sizes={viewportSizes} activeWidth={activeViewportWidth} onSelect={onViewportSize} />

			{/* Layout (only if exist) */}
			{layouts.length > 0 && (
				<>
					<Sep />
					<DockSegment items={layouts} onSelect={onLayout} />
				</>
			)}

			{/* Taste (only if exist) */}
			{tastes.length > 0 && (
				<>
					<Sep />
					<DockSegment items={tastes} onSelect={onTaste} />
				</>
			)}
		</div>
	)
}

export { PRESET_SIZES }
export type { ViewportSize as ViewportSizeType }
