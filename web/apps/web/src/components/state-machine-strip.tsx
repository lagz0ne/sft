import type { Transition } from '../lib/types'

function eventLabel(event: string): string {
  const name = event.split('(')[0]
  return name.replace(/_/g, ' ').replace(/^\w/, c => c.toUpperCase())
}

export function StateMachineStrip({ states, transitions, currentState, onStateChange }: {
  states: string[]
  transitions: Transition[]
  currentState: string
  onStateChange: (state: string) => void
}) {
  // Group transitions by from_state for quick lookup
  const transitionsByFrom = new Map<string, Transition[]>()
  for (const t of transitions) {
    const from = t.from_state ?? currentState
    if (!transitionsByFrom.has(from)) transitionsByFrom.set(from, [])
    transitionsByFrom.get(from)!.push(t)
  }

  return (
    <div className="mb-6 border border-neutral-200 rounded-lg bg-white overflow-hidden">
      {/* State chips strip */}
      <div className="flex items-center gap-1 p-3 overflow-x-auto">
        {states.map((state, i) => {
          const active = state === currentState
          return (
            <div key={state} className="flex items-center shrink-0">
              {i > 0 && <span className="text-neutral-300 mx-1">→</span>}
              <button
                onClick={() => onStateChange(state)}
                className={`px-3 py-1 rounded-full text-sm transition-colors ${
                  active
                    ? 'bg-neutral-900 text-white font-semibold'
                    : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
                }`}
              >
                {state}
              </button>
            </div>
          )
        })}

        {/* Reset to initial */}
        {currentState !== states[0] && (
          <button
            onClick={() => onStateChange(states[0])}
            className="ml-2 px-2 py-1 text-xs text-neutral-400 hover:text-neutral-600 shrink-0"
          >
            ↺ reset
          </button>
        )}
      </div>

      {/* Transitions from current state */}
      {(() => {
        const fromCurrent = transitionsByFrom.get(currentState) ?? []
        const forward = fromCurrent.filter(t => {
          const to = t.to_state
          if (!to || to === '.' || to === currentState) return false
          if (t.action?.startsWith('navigate(')) return false
          return true
        })
        const selfTransitions = fromCurrent.filter(t =>
          t.to_state === '.' || t.to_state === currentState
        )
        const navigateOuts = fromCurrent.filter(t =>
          t.action?.startsWith('navigate(')
        )

        if (forward.length === 0 && selfTransitions.length === 0 && navigateOuts.length === 0) return null

        return (
          <div className="border-t border-neutral-100 px-3 py-2 text-xs text-neutral-400 flex flex-wrap gap-x-4 gap-y-1">
            {forward.map((t, i) => (
              <button key={i}
                onClick={() => t.to_state && onStateChange(t.to_state)}
                className="hover:text-neutral-600">
                {eventLabel(t.on_event)} → {t.to_state}
              </button>
            ))}
            {selfTransitions.map((t, i) => (
              <span key={`self-${i}`} className="text-neutral-300">
                ↻ {eventLabel(t.on_event)}
              </span>
            ))}
            {navigateOuts.map((t, i) => (
              <span key={`nav-${i}`} className="text-neutral-300">
                ↗ {eventLabel(t.on_event)} → {t.action}
              </span>
            ))}
          </div>
        )
      })()}
    </div>
  )
}
