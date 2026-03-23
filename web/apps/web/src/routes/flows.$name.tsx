import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { FlowStepStrip } from '../components/flow-step-strip'
import { RegionList } from '../components/region-list'
import { StateMachineStrip } from '../components/state-machine-strip'
import { useState } from 'react'
import type { FlowStep, Screen } from '../lib/types'

export const Route = createFileRoute('/flows/$name')({
  component: FlowDetail,
})

/** Walk backwards from `index` to find the most recent screen-type step */
function findScreenForStep(steps: FlowStep[], index: number): string | null {
  for (let i = index; i >= 0; i--) {
    if (steps[i].type === 'screen') return steps[i].name
  }
  return null
}

function FlowDetail() {
  const { name } = Route.useParams()
  const { spec } = useSpecContext()

  const flow = spec?.flows?.find(f => f.name === name)
  if (!flow) return <div className="p-8 text-neutral-400">Flow not found: {name}</div>

  const steps = flow.steps ?? []
  const [stepIndex, setStepIndex] = useState(0)
  const [currentState, setCurrentState] = useState<string | null>(null)

  const currentStep = steps[stepIndex] ?? null
  const screenName = findScreenForStep(steps, stepIndex)
  const screen: Screen | undefined = screenName
    ? spec?.screens?.find(s => s.name === screenName)
    : undefined

  const effectiveState = currentState ?? screen?.states?.[0] ?? null
  const visibleRegions = effectiveState && screen?.state_regions
    ? screen.state_regions[effectiveState]
    : null

  const goTo = (i: number) => {
    setStepIndex(i)
    setCurrentState(null) // reset state when navigating steps
  }

  return (
    <div className="max-w-3xl mx-auto p-8">
      <Link to="/" className="text-sm text-neutral-400 hover:text-neutral-600 mb-4 block">← back</Link>

      <h1 className="text-2xl font-bold mb-1">{flow.name}</h1>
      {flow.description && <p className="text-neutral-500 mb-6">{flow.description}</p>}

      {/* Step strip */}
      {steps.length > 0 && (
        <FlowStepStrip steps={steps} currentIndex={stepIndex} onStepClick={goTo} />
      )}

      {/* Navigation */}
      <div className="flex items-center gap-2 mb-6">
        <button
          disabled={stepIndex <= 0}
          onClick={() => goTo(stepIndex - 1)}
          className="px-3 py-1.5 text-sm rounded border border-neutral-200 bg-white hover:bg-neutral-50 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          ← Back
        </button>
        <span className="text-sm text-neutral-400 flex-1 text-center">
          Step {stepIndex + 1} of {steps.length}
          {currentStep && (
            <span className="ml-2 text-neutral-300">
              ({currentStep.type}: {currentStep.name})
            </span>
          )}
        </span>
        <button
          disabled={stepIndex >= steps.length - 1}
          onClick={() => goTo(stepIndex + 1)}
          className="px-3 py-1.5 text-sm rounded border border-neutral-200 bg-white hover:bg-neutral-50 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          Next →
        </button>
      </div>

      {/* Current screen detail */}
      {screen && (
        <div className="border border-neutral-200 rounded-lg bg-white p-4">
          <div className="font-semibold text-lg mb-1">{screen.name}</div>
          {screen.description && <p className="text-sm text-neutral-500 mb-4">{screen.description}</p>}

          {screen.states && screen.states.length > 0 && effectiveState && (
            <StateMachineStrip
              states={screen.states}
              transitions={screen.transitions ?? []}
              currentState={effectiveState}
              onStateChange={setCurrentState}
            />
          )}

          {screen.regions && screen.regions.length > 0 && (
            <div>
              <div className="text-xs uppercase tracking-wider text-neutral-400 mb-2">Regions</div>
              <RegionList regions={screen.regions} visibleRegions={visibleRegions} />
            </div>
          )}
        </div>
      )}

      {!screen && currentStep && (
        <div className="border border-dashed border-neutral-200 rounded-lg p-6 text-center text-neutral-400 text-sm">
          No screen context at this step
        </div>
      )}
    </div>
  )
}
