import { createFileRoute, Link } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useState } from 'react'

export const Route = createFileRoute('/')({
  component: HomePage,
})

function HomePage() {
  const { spec, loading } = useSpecContext()
  const [tab, setTab] = useState<'screens' | 'flows'>('screens')

  if (loading || !spec) return <div className="p-8 text-neutral-400">Loading...</div>

  const appRegions = spec.app?.regions?.filter(r =>
    r.transitions && r.transitions.length > 0
  ) ?? []

  return (
    <div className="max-w-3xl mx-auto p-8">
      <div className="flex items-center gap-3 mb-1">
        <h1 className="text-2xl font-bold">{spec.app?.name || 'Spec'}</h1>
        <Link to="/playground"
          className="px-3 py-1 text-xs font-medium rounded-full bg-neutral-900 text-white hover:bg-neutral-700 transition-colors">
          Playground
        </Link>
      </div>
      {spec.app?.description && (
        <p className="text-neutral-500 mb-6">{spec.app.description}</p>
      )}

      <div className="flex gap-0 border-b-2 border-neutral-200 mb-6">
        <button
          className={`px-5 py-2 font-semibold text-sm ${tab === 'screens' ? 'border-b-2 border-neutral-900 text-neutral-900 -mb-[2px]' : 'text-neutral-400'}`}
          onClick={() => setTab('screens')}
        >Screens</button>
        <button
          className={`px-5 py-2 font-semibold text-sm ${tab === 'flows' ? 'border-b-2 border-neutral-900 text-neutral-900 -mb-[2px]' : 'text-neutral-400'}`}
          onClick={() => setTab('flows')}
        >Flows</button>
      </div>

      {tab === 'screens' && (
        <div className="flex flex-col gap-1">
          {spec.screens?.map(screen => (
            <Link key={screen.name} to="/screens/$name" params={{ name: screen.name }}
              className="flex items-center gap-3 px-4 py-3 rounded-lg border border-neutral-200 hover:border-neutral-300 bg-white">
              <span className="font-semibold flex-1">{screen.name}</span>
              <span className="text-xs text-neutral-400">
                {screen.regions?.length ?? 0} areas
                {screen.states && screen.states.length > 0 && ` · ${screen.states.length} looks`}
              </span>
              <span className="text-neutral-300">→</span>
            </Link>
          ))}
          {appRegions.length > 0 && (
            <>
              <div className="text-xs uppercase tracking-wider text-neutral-400 mt-6 mb-2">App-level</div>
              {appRegions.map(region => (
                <div key={region.name} className="flex items-center gap-3 px-4 py-3 rounded-lg border border-neutral-200 bg-white">
                  <span className="font-semibold flex-1">{region.name}</span>
                  <span className="text-xs text-neutral-400">
                    {region.events?.length ?? 0} events
                    {region.states && region.states.length > 0 && ` · ${region.states.length} states`}
                  </span>
                </div>
              ))}
            </>
          )}
        </div>
      )}

      {tab === 'flows' && (
        <div className="flex flex-col gap-1">
          {spec.flows?.map(flow => (
            <Link key={flow.name} to="/flows/$name" params={{ name: flow.name }}
              className="flex items-center gap-3 px-4 py-3 rounded-lg border border-neutral-200 hover:border-neutral-300 bg-white">
              <span className="font-semibold flex-1">{flow.name}</span>
              <span className="text-xs text-neutral-400 truncate max-w-[50%]">{flow.sequence}</span>
              <span className="text-neutral-300">→</span>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
