import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useState } from 'react'
import { WireframeCanvas } from '../components/wireframe-canvas'
import type { FlowStep, TasteTokens } from '../lib/types'

export const Route = createFileRoute('/playground')({
	component: PlaygroundPage,
})

/** Walk backwards from `index` to find the most recent screen-type step */
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

	if (loading || !spec) return <div className="h-full flex items-center justify-center text-neutral-400">Loading...</div>

	// Screen mode
	const screen = spec.screens.find(s => s.name === currentScreen) ?? spec.screens[0]
	const states = screen?.states ?? []
	const activeState = currentState ?? states[0] ?? null

	// Flow mode
	const flow = currentFlow ? spec.flows?.find(f => f.name === currentFlow) : spec.flows?.[0]
	const steps = flow?.steps ?? []
	const currentStep = steps[stepIndex] ?? null

	// In flow mode, derive screen + state from current step
	const flowScreenName = steps.length > 0 ? findScreenForStep(steps, stepIndex) : null
	const flowScreen = flowScreenName ? spec.screens.find(s => s.name === flowScreenName) : null

	const effectiveScreen = mode === 'flow' && flowScreen ? flowScreen : screen
	const effectiveState = mode === 'flow'
		? (flowScreen?.states?.[0] ?? null)
		: activeState

	// Active taste tokens
	const activeTaste: TasteTokens = currentTaste
		? spec.tastes?.find(t => t.name === currentTaste)?.tokens ?? {}
		: {}

	// Active region/event from flow step
	const activeRegion = mode === 'flow' && currentStep?.type === 'region' ? currentStep.name : null
	const activeEvent = mode === 'flow' && currentStep?.type === 'event' ? currentStep.name : null

	// Transitions from current state
	const transitions = effectiveScreen?.transitions ?? []
	const fromCurrent = transitions.filter(t => {
		const from = t.from_state ?? effectiveState
		return from === effectiveState
	})
	const forward = fromCurrent.filter(t => {
		const to = t.to_state
		if (!to || to === '.' || to === effectiveState) return false
		if (t.action?.startsWith('navigate(')) return false
		return true
	})
	const navigateOuts = fromCurrent.filter(t => t.action?.startsWith('navigate('))

	const visibleRegions = effectiveState && effectiveScreen?.state_regions
		? effectiveScreen.state_regions[effectiveState]
		: null

	const switchScreen = (name: string) => {
		setCurrentScreen(name)
		setCurrentState(null)
	}

	const goToStep = (i: number) => {
		setStepIndex(Math.max(0, Math.min(i, steps.length - 1)))
	}

	return (
		<div className="h-full flex flex-col bg-neutral-50">
			{/* Compact toolbar */}
			<div className="bg-white border-b border-neutral-200 px-4 py-2 flex flex-col gap-2 shrink-0">
				{/* Top row: nav + mode toggle */}
				<div className="flex items-center gap-3">
					<Link to="/" className="text-neutral-400 hover:text-neutral-600 text-sm">←</Link>
					<span className="font-bold text-sm text-neutral-900">{spec.app.name}</span>

					{/* Mode toggle */}
					<div className="flex gap-0 border border-neutral-200 rounded-lg overflow-hidden ml-3">
						<button
							onClick={() => setMode('screen')}
							className={`px-3 py-1 text-xs font-medium transition-colors ${
								mode === 'screen' ? 'bg-neutral-900 text-white' : 'bg-white text-neutral-500 hover:bg-neutral-50'
							}`}
						>Screen</button>
						{spec.flows && spec.flows.length > 0 && (
							<button
								onClick={() => { setMode('flow'); setStepIndex(0) }}
								className={`px-3 py-1 text-xs font-medium transition-colors ${
									mode === 'flow' ? 'bg-neutral-900 text-white' : 'bg-white text-neutral-500 hover:bg-neutral-50'
								}`}
							>Flow</button>
						)}
					</div>

					{/* Screen mode: screen tabs */}
					{mode === 'screen' && (
						<>
							<div className="w-px h-4 bg-neutral-200" />
							<div className="flex gap-1 overflow-x-auto">
								{spec.screens.map(s => (
									<button
										key={s.name}
										onClick={() => switchScreen(s.name)}
										className={`px-3 py-1 text-xs rounded-full shrink-0 transition-colors ${
											s.name === effectiveScreen?.name
												? 'bg-neutral-800 text-white font-semibold'
												: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
										}`}
									>
										{s.name}
									</button>
								))}
							</div>
						</>
					)}

					{/* Flow mode: flow selector + step nav */}
					{mode === 'flow' && spec.flows && (
						<>
							<div className="w-px h-4 bg-neutral-200" />
							<select
								value={flow?.name ?? ''}
								onChange={e => { setCurrentFlow(e.target.value); setStepIndex(0) }}
								className="text-xs bg-neutral-100 border border-neutral-200 rounded-lg px-2 py-1"
							>
								{spec.flows.map(f => (
									<option key={f.name} value={f.name}>{f.name}</option>
								))}
							</select>
						</>
					)}

					{/* Taste switcher */}
					{spec.tastes && spec.tastes.length > 0 && (
						<>
							<div className="w-px h-4 bg-neutral-200" />
							<span className="text-[10px] text-neutral-400">taste:</span>
							<div className="flex gap-1">
								{spec.tastes.map(t => (
									<button
										key={t.name}
										onClick={() => setCurrentTaste(t.name === currentTaste ? null : t.name)}
										className={`px-2.5 py-0.5 text-[10px] rounded-full transition-colors ${
											t.name === currentTaste
												? 'bg-neutral-800 text-white font-semibold'
												: 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200'
										}`}
									>
										{t.name}
									</button>
								))}
							</div>
						</>
					)}
				</div>

				{/* Second row: states + transitions (screen mode) */}
				{mode === 'screen' && states.length > 0 && (
					<div className="flex items-center gap-1 -mt-0.5">
						{states.map(s => (
							<button
								key={s}
								onClick={() => setCurrentState(s)}
								className={`px-2.5 py-0.5 text-[11px] rounded-full transition-colors ${
									s === effectiveState
										? 'bg-emerald-600 text-white font-semibold'
										: 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200'
								}`}
							>
								{s === effectiveState && '● '}{s}
							</button>
						))}

						{/* State transitions */}
						{forward.length > 0 && (
							<>
								<div className="w-px h-3 bg-neutral-200 mx-1" />
								{forward.map((t, i) => (
									<button
										key={i}
										onClick={() => t.to_state && setCurrentState(t.to_state)}
										className="text-[10px] text-neutral-400 hover:text-neutral-600"
									>
										{t.on_event.split('(')[0]} → {t.to_state}
									</button>
								))}
							</>
						)}
						{navigateOuts.length > 0 && (
							<>
								{forward.length === 0 && <div className="w-px h-3 bg-neutral-200 mx-1" />}
								{navigateOuts.map((t, i) => (
									<span key={i} className="text-[10px] text-neutral-300">
										↗ {t.on_event.split('(')[0]} → {t.action}
									</span>
								))}
							</>
						)}
					</div>
				)}

				{/* Second row: flow step strip */}
				{mode === 'flow' && steps.length > 0 && (
					<div className="flex items-center gap-1 -mt-0.5">
						<button
							disabled={stepIndex <= 0}
							onClick={() => goToStep(stepIndex - 1)}
							className="text-xs text-neutral-400 hover:text-neutral-600 disabled:opacity-30 px-1"
						>◀</button>

						<div className="flex items-center gap-0.5 overflow-x-auto">
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
			</div>

			{/* Wireframe canvas */}
			<div className="flex-1 p-4 overflow-auto">
				{effectiveScreen && (
					<div className="h-full">
						{/* Screen label */}
						<div className="flex items-center gap-2 mb-3">
							<span className="text-xs font-bold text-neutral-500 uppercase tracking-wider">
								{effectiveScreen.name}
							</span>
							{effectiveState && (
								<span className="text-[10px] bg-emerald-50 text-emerald-600 px-1.5 py-0.5 rounded">
									{effectiveState}
								</span>
							)}
							{visibleRegions && (
								<span className="text-[10px] text-neutral-400">
									{visibleRegions.length} of {effectiveScreen.regions?.length ?? 0} regions visible
								</span>
							)}
							{mode === 'flow' && currentStep && (
								<span className="text-[10px] bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded">
									{currentStep.type}: {currentStep.name}
								</span>
							)}
						</div>

						<WireframeCanvas
							screen={effectiveScreen}
							currentState={effectiveState}
							appRegions={spec.app.regions}
							fixtures={spec.fixtures}
							activeRegion={activeRegion}
							activeEvent={activeEvent}
							app={spec.app}
							taste={activeTaste}
						/>
					</div>
				)}
			</div>
		</div>
	)
}
