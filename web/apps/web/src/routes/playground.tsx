import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useState, useEffect, useMemo } from 'react'
import { WireframeCanvas } from '../components/wireframe-canvas'
import { CommandBar, ContextBadge, type CommandItem } from '../components/command-bar'
import { discoverCompositions } from '../lib/layout-tags'
import type { FlowStep, TasteTokens } from '../lib/types'

export const Route = createFileRoute('/playground')({
	component: PlaygroundPage,
})

function findScreenForStep(steps: FlowStep[], index: number): string | null {
	for (let i = index; i >= 0; i--) {
		if (steps[i].type === 'screen') return steps[i].name
	}
	return null
}

function PlaygroundPage() {
	const { spec, loading } = useSpecContext()
	const [currentScreen, setCurrentScreen] = useState<string | null>(null)
	const [currentState, setCurrentState] = useState<string | null>(null)
	const [mode, setMode] = useState<'screen' | 'flow'>('screen')
	const [currentFlow, setCurrentFlow] = useState<string | null>(null)
	const [stepIndex, setStepIndex] = useState(0)
	const [currentTaste, setCurrentTaste] = useState<string | null>(null)
	const [currentComposition, setCurrentComposition] = useState<string | null>(null)
	const [commandBarOpen, setCommandBarOpen] = useState(false)

	// Global keyboard shortcut: / to open command bar
	useEffect(() => {
		const handler = (e: KeyboardEvent) => {
			if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
				const target = e.target as HTMLElement
				if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') return
				e.preventDefault()
				setCommandBarOpen(true)
			}
		}
		window.addEventListener('keydown', handler)
		return () => window.removeEventListener('keydown', handler)
	}, [])

	if (loading || !spec) return <div className="h-full flex items-center justify-center text-neutral-400">Loading...</div>

	// Screen mode
	const screen = spec.screens.find(s => s.name === currentScreen) ?? spec.screens[0]
	const states = screen?.states ?? []
	const activeState = currentState ?? states[0] ?? null

	// Flow mode
	const flow = currentFlow ? spec.flows?.find(f => f.name === currentFlow) : spec.flows?.[0]
	const steps = flow?.steps ?? []
	const currentStep = steps[stepIndex] ?? null

	const flowScreenName = steps.length > 0 ? findScreenForStep(steps, stepIndex) : null
	const flowScreen = flowScreenName ? spec.screens.find(s => s.name === flowScreenName) : null

	const effectiveScreen = mode === 'flow' && flowScreen ? flowScreen : screen
	const effectiveState = mode === 'flow' ? (flowScreen?.states?.[0] ?? null) : activeState

	const activeTaste: TasteTokens = currentTaste
		? spec.tastes?.find(t => t.name === currentTaste)?.tokens ?? {}
		: {}

	const activeRegion = mode === 'flow' && currentStep?.type === 'region' ? currentStep.name : null
	const activeEvent = mode === 'flow' && currentStep?.type === 'event' ? currentStep.name : null

	// Discover compositions
	const allRegions = useMemo(() =>
		[...(spec.app.regions ?? []), ...(effectiveScreen?.regions ?? [])],
		[spec.app.regions, effectiveScreen?.regions]
	)
	const compositions = useMemo(() => discoverCompositions(allRegions), [allRegions])

	const switchScreen = (name: string) => { setCurrentScreen(name); setCurrentState(null) }
	const goToStep = (i: number) => { setStepIndex(Math.max(0, Math.min(i, steps.length - 1))) }

	// Build command items for the command bar
	const commandItems: CommandItem[] = useMemo(() => {
		const items: CommandItem[] = []

		// Screens
		for (const s of spec.screens) {
			items.push({
				id: `screen:${s.name}`,
				label: s.name,
				group: 'Screen',
				active: effectiveScreen?.name === s.name,
				onSelect: () => { setMode('screen'); switchScreen(s.name) },
			})
		}

		// States (for current screen)
		if (effectiveScreen?.states) {
			for (const s of effectiveScreen.states) {
				items.push({
					id: `state:${s}`,
					label: s,
					group: 'State',
					active: effectiveState === s,
					onSelect: () => setCurrentState(s),
				})
			}
		}

		// Compositions
		if (compositions.length > 0) {
			items.push({
				id: 'layout:default',
				label: 'default',
				group: 'Layout',
				active: currentComposition === null,
				onSelect: () => setCurrentComposition(null),
			})
			for (const c of compositions) {
				items.push({
					id: `layout:${c}`,
					label: c,
					group: 'Layout',
					active: currentComposition === c,
					onSelect: () => setCurrentComposition(c),
				})
			}
		}

		// Tastes
		if (spec.tastes) {
			for (const t of spec.tastes) {
				items.push({
					id: `taste:${t.name}`,
					label: t.name,
					group: 'Taste',
					active: currentTaste === t.name,
					onSelect: () => setCurrentTaste(currentTaste === t.name ? null : t.name),
				})
			}
		}

		// Flows
		if (spec.flows) {
			for (const f of spec.flows) {
				items.push({
					id: `flow:${f.name}`,
					label: f.name,
					group: 'Flow',
					active: mode === 'flow' && flow?.name === f.name,
					onSelect: () => { setMode('flow'); setCurrentFlow(f.name); setStepIndex(0) },
				})
			}
		}

		return items
	}, [spec, effectiveScreen, effectiveState, currentComposition, currentTaste, mode, flow, compositions])

	return (
		<div className="h-full bg-neutral-50 relative">
			{/* Context badge — tiny floating pill showing current state */}
			<ContextBadge
				screen={effectiveScreen?.name ?? ''}
				state={effectiveState}
				layout={currentComposition}
				taste={currentTaste}
				onOpen={() => setCommandBarOpen(true)}
			/>

			{/* Command bar — press / to open */}
			<CommandBar
				items={commandItems}
				open={commandBarOpen}
				onClose={() => setCommandBarOpen(false)}
			/>

			{/* Flow step strip — shown at top when in flow mode */}
			{mode === 'flow' && steps.length > 0 && (
				<div className="fixed top-12 left-1/2 -translate-x-1/2 z-30 flex items-center gap-1 bg-white/90 backdrop-blur rounded-full shadow-md border border-neutral-200 px-3 py-1.5">
					<button
						disabled={stepIndex <= 0}
						onClick={() => goToStep(stepIndex - 1)}
						className="text-xs text-neutral-400 hover:text-neutral-600 disabled:opacity-30 px-1"
					>◀</button>

					<div className="flex items-center gap-0.5 overflow-x-auto max-w-lg">
						{steps.map((step, i) => {
							const active = i === stepIndex
							const icons: Record<string, string> = { screen: '◻', back: '←', region: '▪', event: '⚡', action: '▶', activate: '●' }
							return (
								<div key={i} className="flex items-center shrink-0">
									{i > 0 && <span className="text-neutral-300 mx-0.5 text-[10px]">→</span>}
									<button
										onClick={() => goToStep(i)}
										className={`px-2 py-0.5 text-[10px] rounded-full transition-colors ${
											active
												? 'bg-neutral-800 text-white font-semibold'
												: 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200'
										}`}
									>
										<span className="mr-0.5 opacity-60">{icons[step.type] ?? '·'}</span>
										{step.name}
									</button>
								</div>
							)
						})}
					</div>

					<button
						disabled={stepIndex >= steps.length - 1}
						onClick={() => goToStep(stepIndex + 1)}
						className="text-xs text-neutral-400 hover:text-neutral-600 disabled:opacity-30 px-1"
					>▶</button>

					<span className="text-[10px] text-neutral-400 ml-1">
						{stepIndex + 1}/{steps.length}
					</span>
				</div>
			)}

			{/* Wireframe canvas — fills entire viewport */}
			<div className="h-full p-4 pt-12 overflow-auto">
				{effectiveScreen && (
					<div className="h-full">
						<WireframeCanvas
							screen={effectiveScreen}
							currentState={effectiveState}
							appRegions={spec.app.regions}
							fixtures={spec.fixtures}
							activeRegion={activeRegion}
							activeEvent={activeEvent}
							app={spec.app}
							taste={activeTaste}
							composition={currentComposition}
						/>
					</div>
				)}
			</div>

			{/* Back link */}
			<Link to="/" className="fixed bottom-3 left-3 z-40 text-neutral-400 hover:text-neutral-600 text-xs bg-white/80 backdrop-blur rounded-full px-2 py-1 shadow-sm border border-neutral-200">
				← back
			</Link>
		</div>
	)
}
