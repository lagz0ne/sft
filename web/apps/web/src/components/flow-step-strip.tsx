import type { FlowStep } from '../lib/types'

const typeIndicator: Record<string, string> = {
  screen: '◻',
  back: '←',
  region: '▪',
  event: '⚡',
  action: '▶',
  activate: '●',
}

export function FlowStepStrip({ steps, currentIndex, onStepClick }: {
  steps: FlowStep[]
  currentIndex: number
  onStepClick: (index: number) => void
}) {
  return (
    <div className="flex items-center gap-1 overflow-x-auto p-3 border border-neutral-200 rounded-lg bg-white mb-6">
      {steps.map((step, i) => {
        const active = i === currentIndex
        const indicator = typeIndicator[step.type] ?? '·'
        return (
          <div key={i} className="flex items-center shrink-0">
            {i > 0 && <span className="text-neutral-300 mx-1">→</span>}
            <button
              onClick={() => onStepClick(i)}
              className={`px-3 py-1 rounded-full text-sm transition-colors ${
                active
                  ? 'bg-neutral-900 text-white font-semibold'
                  : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
              }`}
              title={`${step.type}: ${step.name}`}
            >
              <span className="mr-1 opacity-60">{indicator}</span>
              {step.name}
            </button>
          </div>
        )
      })}
    </div>
  )
}
