import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { RegionList } from '../components/region-list'
import { StateMachineStrip } from '../components/state-machine-strip'
import { useState } from 'react'

export const Route = createFileRoute('/screens/$name')({
  component: ScreenDetail,
})

function ScreenDetail() {
  const { name } = Route.useParams()
  const { spec } = useSpecContext()

  const screen = spec?.screens?.find(s => s.name === name)
  if (!screen) return <div className="p-8 text-neutral-400">Screen not found: {name}</div>

  const flowsThrough = spec?.flows?.filter(f =>
    f.steps?.some(s => s.type === 'screen' && s.name === name)
    || f.sequence?.includes(name)
  ) ?? []

  // State management — initial state, visible regions
  const [currentState, setCurrentState] = useState<string | null>(null)
  const effectiveState = currentState ?? screen.states?.[0] ?? null
  const visibleRegions = effectiveState && screen.state_regions
    ? screen.state_regions[effectiveState]
    : null

  const mockup = screen.attachments?.[0] ?? null

  return (
    <div className="max-w-3xl mx-auto p-8">
      <Link to="/" className="text-sm text-neutral-400 hover:text-neutral-600 mb-4 block">← back</Link>

      <h1 className="text-2xl font-bold mb-1">{screen.name}</h1>
      {screen.description && <p className="text-neutral-500 mb-6">{screen.description}</p>}

      {screen.states && screen.states.length > 0 && effectiveState && (
        <StateMachineStrip
          states={screen.states}
          transitions={screen.transitions ?? []}
          currentState={effectiveState}
          onStateChange={setCurrentState}
        />
      )}

      {mockup && (
        <div className="mb-6">
          <div className="text-xs uppercase tracking-wider text-neutral-400 mb-2">Look</div>
          <div className="border border-neutral-200 rounded-lg overflow-hidden bg-white">
            <img src={`/a/${screen.name}/${mockup}`} alt={mockup} className="w-full" />
          </div>
        </div>
      )}

      {screen.regions && screen.regions.length > 0 && (
        <div className="mb-6">
          <div className="text-xs uppercase tracking-wider text-neutral-400 mb-2">What's on this screen</div>
          <RegionList regions={screen.regions} visibleRegions={visibleRegions} />
        </div>
      )}

      {flowsThrough.length > 0 && (
        <div>
          <div className="text-xs uppercase tracking-wider text-neutral-400 mb-2">Flows through this screen</div>
          <div className="flex flex-col gap-1">
            {flowsThrough.map(flow => (
              <Link key={flow.name} to="/flows/$name" params={{ name: flow.name }}
                className="px-3 py-2 border border-neutral-200 rounded-lg bg-white text-sm hover:border-neutral-300 flex items-center">
                <span className="font-medium flex-1">{flow.name}</span>
                <span className="text-neutral-300">→</span>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
