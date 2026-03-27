import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useState, useMemo } from 'react'
import { WireframeCanvas } from '../components/wireframe-canvas'
import { Dock, PRESET_SIZES, type ViewportSizeType } from '../components/dock'
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
	const navigate = useNavigate()
	const [currentScreen, setCurrentScreen] = useState<string | null>(null)
	const [currentState, setCurrentState] = useState<string | null>(null)
	const [mode, setMode] = useState<'screen' | 'flow'>('screen')
	const [currentFlow, _setCurrentFlow] = useState<string | null>(null)
	const [stepIndex, setStepIndex] = useState(0)
	const [currentTaste, setCurrentTaste] = useState<string | null>(null)
	const [currentComposition, setCurrentComposition] = useState<string | null>(null)
	const [viewportWidth, setViewportWidth] = useState<number | null>(null)

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

	// Build viewport sizes — merge presets with discovered compositions
	const viewportSizes = useMemo(() => {
		const sizes = [...PRESET_SIZES]
		// Link discovered compositions to viewport sizes if they match common breakpoint names
		for (const comp of compositions) {
			const existing = sizes.find(s => s.composition === comp)
			if (!existing) {
				// Auto-link common names to widths
				const autoWidth = /^mobile$/i.test(comp) ? 375
					: /^tablet$/i.test(comp) ? 768
					: /^desktop$/i.test(comp) ? 1280
					: null
				if (autoWidth && !sizes.find(s => s.width === autoWidth && s.composition)) {
					// Replace the plain preset with the composition-linked one
					const idx = sizes.findIndex(s => s.width === autoWidth)
					if (idx >= 0) {
						sizes[idx] = { label: comp, width: autoWidth, composition: comp }
					} else {
						sizes.push({ label: comp, width: autoWidth, composition: comp })
					}
				}
			}
		}
		return sizes
	}, [compositions])

	const switchScreen = (name: string) => { setCurrentScreen(name); setCurrentState(null) }
	const goToStep = (i: number) => { setStepIndex(Math.max(0, Math.min(i, steps.length - 1))) }

	const handleViewportSize = (size: ViewportSizeType) => {
		setViewportWidth(size.width)
		// If the viewport size is linked to a composition, switch to it
		if (size.composition !== undefined) {
			setCurrentComposition(size.composition)
		}
	}

	return (
		<div className="h-full bg-neutral-100 relative flex items-start justify-center overflow-auto">
			{/* Wireframe container — constrained by viewport width */}
			<div
				className="bg-neutral-50 min-h-full transition-all duration-300 shadow-sm"
				style={{
					width: viewportWidth ? `${viewportWidth}px` : '100%',
					maxWidth: '100%',
				}}
			>
				<div className="h-full p-3 pb-12">
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
			</div>

			{/* Bottom dock */}
			<Dock
				screens={spec.screens.map(s => ({
					id: s.name, label: s.name, active: effectiveScreen?.name === s.name,
				}))}
				onScreen={id => { if (mode === 'flow') setMode('screen'); switchScreen(id) }}
				states={states.map(s => ({ id: s, label: s, active: effectiveState === s }))}
				onState={id => setCurrentState(id)}
				layouts={[
					{ id: '__default', label: 'default', active: currentComposition === null },
					...compositions.map(c => ({ id: c, label: c, active: currentComposition === c })),
				].filter((_, __, arr) => arr.length > 1)}
				onLayout={id => setCurrentComposition(id === '__default' ? null : id)}
				tastes={(spec.tastes ?? []).map(t => ({
					id: t.name, label: t.name, active: currentTaste === t.name,
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
				onBack={() => navigate({ to: '/' })}
				viewportSizes={viewportSizes}
				activeViewportWidth={viewportWidth}
				onViewportSize={handleViewportSize}
			/>
		</div>
	)
}
