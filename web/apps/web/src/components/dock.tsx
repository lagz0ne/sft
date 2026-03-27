import { useState, useRef, useEffect } from 'react'

// --- Dock Segment ---

interface DockSegmentProps {
	items: { id: string; label: string; active: boolean }[]
	onSelect: (id: string) => void
	accent?: string // color for active state indicator
	maxInline?: number
}

function DockSegment({ items, onSelect, accent, maxInline = 4 }: DockSegmentProps) {
	const [popoverOpen, setPopoverOpen] = useState(false)
	const segmentRef = useRef<HTMLDivElement>(null)

	// Close popover on outside click
	useEffect(() => {
		if (!popoverOpen) return
		const handler = (e: MouseEvent) => {
			if (segmentRef.current && !segmentRef.current.contains(e.target as Node)) {
				setPopoverOpen(false)
			}
		}
		document.addEventListener('mousedown', handler)
		return () => document.removeEventListener('mousedown', handler)
	}, [popoverOpen])

	if (items.length === 0) return null

	const activeItem = items.find(i => i.active)
	const showInline = items.length <= maxInline
	const inlineItems = showInline ? items : items.slice(0, 3)
	const overflowCount = showInline ? 0 : items.length - 3

	const activeBg = accent ?? '#222'
	const activeStyle = { backgroundColor: activeBg, color: '#fff' }

	return (
		<div ref={segmentRef} className="relative">
			{/* Inline pills */}
			<div className="flex gap-0.5 bg-neutral-100 rounded-lg p-0.5">
				{showInline ? (
					// All items inline
					inlineItems.map(item => (
						<button
							key={item.id}
							onClick={() => { onSelect(item.id); setPopoverOpen(false) }}
							className={`px-2 py-1 text-[10px] rounded-md transition-colors ${
								item.active ? 'font-semibold' : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-200'
							}`}
							style={item.active ? activeStyle : undefined}
						>
							{item.label}
						</button>
					))
				) : (
					// Active item + overflow trigger
					<>
						{activeItem && (
							<button
								className="px-2 py-1 text-[10px] rounded-md font-semibold"
								style={activeStyle}
								onClick={() => setPopoverOpen(!popoverOpen)}
							>
								{activeItem.label}
							</button>
						)}
						<button
							onClick={() => setPopoverOpen(!popoverOpen)}
							className="px-1.5 py-1 text-[9px] text-neutral-400 hover:text-neutral-600 rounded-md hover:bg-neutral-200"
						>
							+{overflowCount}
						</button>
					</>
				)}
			</div>

			{/* Upward popover — same width as segment */}
			{popoverOpen && (
				<div
					className="absolute bottom-full left-0 right-0 mb-1.5 bg-white rounded-lg shadow-lg border border-neutral-200 overflow-hidden z-50"
					style={{ minWidth: segmentRef.current?.offsetWidth }}
				>
					<div className="max-h-48 overflow-y-auto py-1">
						{items.map(item => (
							<button
								key={item.id}
								onClick={() => { onSelect(item.id); setPopoverOpen(false) }}
								className={`w-full px-3 py-1.5 text-[10px] text-left flex items-center gap-1.5 transition-colors ${
									item.active ? 'font-semibold' : 'text-neutral-600 hover:bg-neutral-50'
								}`}
								style={item.active ? { color: activeBg } : undefined}
							>
								{item.active && (
									<span className="w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: activeBg }} />
								)}
								<span>{item.label}</span>
							</button>
						))}
					</div>
				</div>
			)}
		</div>
	)
}

// --- Flow Step Strip (replaces State segment in flow mode) ---

interface FlowStripProps {
	steps: { type: string; name: string }[]
	currentIndex: number
	onStep: (index: number) => void
}

function FlowStrip({ steps, currentIndex, onStep }: FlowStripProps) {
	const icons: Record<string, string> = { screen: '◻', back: '←', region: '▪', event: '⚡', action: '▶', activate: '●' }

	return (
		<div className="flex items-center gap-0.5 bg-neutral-100 rounded-lg p-0.5 overflow-x-auto max-w-xs">
			<button
				disabled={currentIndex <= 0}
				onClick={() => onStep(currentIndex - 1)}
				className="px-1 py-1 text-[10px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0"
			>◀</button>
			{steps.map((step, i) => (
				<button
					key={i}
					onClick={() => onStep(i)}
					className={`px-1.5 py-1 text-[9px] rounded-md shrink-0 transition-colors ${
						i === currentIndex
							? 'bg-neutral-800 text-white font-semibold'
							: 'text-neutral-500 hover:bg-neutral-200'
					}`}
				>
					<span className="opacity-60 mr-0.5">{icons[step.type] ?? '·'}</span>
					{step.name}
				</button>
			))}
			<button
				disabled={currentIndex >= steps.length - 1}
				onClick={() => onStep(currentIndex + 1)}
				className="px-1 py-1 text-[10px] text-neutral-400 hover:text-neutral-600 disabled:opacity-30 shrink-0"
			>▶</button>
		</div>
	)
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

	// Flow mode
	flowMode?: boolean
	flowSteps?: { type: string; name: string }[]
	flowIndex?: number
	onFlowStep?: (index: number) => void

	// Mode toggle
	mode: 'screen' | 'flow'
	onModeToggle: () => void
	hasFlows: boolean
}

export function Dock({
	screens, onScreen,
	states, onState,
	layouts, onLayout,
	tastes, onTaste,
	flowMode, flowSteps, flowIndex, onFlowStep,
	mode, onModeToggle, hasFlows,
}: DockProps) {
	return (
		<div className="fixed bottom-3 left-1/2 -translate-x-1/2 z-40 flex flex-wrap items-center justify-center gap-1 bg-white/95 backdrop-blur-sm rounded-2xl shadow-lg border border-neutral-200 px-2 py-1.5 max-w-[95vw]">
			{/* Mode toggle */}
			{hasFlows && (
				<>
					<div className="flex gap-0.5 bg-neutral-100 rounded-lg p-0.5">
						<button
							onClick={() => mode !== 'screen' && onModeToggle()}
							className={`px-2 py-1 text-[10px] rounded-md transition-colors ${
								mode === 'screen' ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
							}`}
						>Screen</button>
						<button
							onClick={() => mode !== 'flow' && onModeToggle()}
							className={`px-2 py-1 text-[10px] rounded-md transition-colors ${
								mode === 'flow' ? 'bg-neutral-800 text-white font-semibold' : 'text-neutral-500 hover:bg-neutral-200'
							}`}
						>Flow</button>
					</div>
					<div className="w-px h-5 bg-neutral-200" />
				</>
			)}

			{/* Screen segment */}
			<DockSegment items={screens} onSelect={onScreen} />

			<div className="w-px h-5 bg-neutral-200" />

			{/* State or Flow strip */}
			{flowMode && flowSteps && flowIndex != null && onFlowStep ? (
				<FlowStrip steps={flowSteps} currentIndex={flowIndex} onStep={onFlowStep} />
			) : (
				<DockSegment items={states} onSelect={onState} accent="#059669" />
			)}

			{/* Layout segment (only if layouts exist) */}
			{layouts.length > 0 && (
				<>
					<div className="w-px h-5 bg-neutral-200" />
					<DockSegment items={layouts} onSelect={onLayout} />
				</>
			)}

			{/* Taste segment (only if tastes exist) */}
			{tastes.length > 0 && (
				<>
					<div className="w-px h-5 bg-neutral-200" />
					<DockSegment items={tastes} onSelect={onTaste} />
				</>
			)}
		</div>
	)
}
