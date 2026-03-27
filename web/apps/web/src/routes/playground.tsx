import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useState, useMemo } from 'react'
import { WireframeCanvas } from '../components/wireframe-canvas'
import { Dock } from '../components/dock'
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
	const [currentFlow, _setCurrentFlow] = useState<string | null>(null)
	const [stepIndex, setStepIndex] = useState(0)
	const [currentTaste, setCurrentTaste] = useState<string | null>(null)
	const [currentComposition, setCurrentComposition] = useState<string | null>(null)

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

	return (
		<div className="h-full bg-neutral-50 relative">
			{/* Wireframe canvas — fills viewport, padded for dock */}
			<div className="h-full p-4 pb-16 overflow-auto">
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

			{/* Bottom dock */}
			<Dock
				screens={spec.screens.map(s => ({
					id: s.name,
					label: s.name,
					active: effectiveScreen?.name === s.name,
				}))}
				onScreen={id => {
					if (mode === 'flow') setMode('screen')
					switchScreen(id)
				}}

				states={states.map(s => ({
					id: s,
					label: s,
					active: effectiveState === s,
				}))}
				onState={id => setCurrentState(id)}

				layouts={[
					{ id: '__default', label: 'default', active: currentComposition === null },
					...compositions.map(c => ({ id: c, label: c, active: currentComposition === c })),
				].filter((_, __, arr) => arr.length > 1)}
				onLayout={id => setCurrentComposition(id === '__default' ? null : id)}

				tastes={(spec.tastes ?? []).map(t => ({
					id: t.name,
					label: t.name,
					active: currentTaste === t.name,
				}))}
				onTaste={id => setCurrentTaste(currentTaste === id ? null : id)}

				mode={mode}
				onModeToggle={() => {
					if (mode === 'screen') { setMode('flow'); setStepIndex(0) }
					else setMode('screen')
				}}
				hasFlows={(spec.flows?.length ?? 0) > 0}

				flowMode={mode === 'flow'}
				flowSteps={steps}
				flowIndex={stepIndex}
				onFlowStep={goToStep}
			/>

			{/* Back link */}
			<Link to="/" className="fixed top-3 left-3 z-40 text-neutral-400 hover:text-neutral-600 text-xs bg-white/80 backdrop-blur rounded-full px-2 py-1 shadow-sm border border-neutral-200">
				← back
			</Link>
		</div>
	)
}
