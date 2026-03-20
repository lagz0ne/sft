import type { Region } from '../lib/types'

function eventLabel(event: string): string {
  const name = event.split('(')[0]
  return name.replace(/_/g, ' ').replace(/^\w/, c => c.toUpperCase())
}

export function RegionList({ regions, visibleRegions }: {
  regions: Region[]
  visibleRegions?: string[] | null
}) {
  return (
    <div className="flex flex-col gap-1">
      {regions.map(region => {
        const hidden = visibleRegions && !visibleRegions.includes(region.name)
        return (
          <div key={region.name}
            className={`rounded-lg border border-neutral-200 bg-white p-3 ${hidden ? 'opacity-30' : ''}`}>
            <div className="font-semibold text-sm">{region.name}</div>
            {region.description && <div className="text-xs text-neutral-500 mt-0.5">{region.description}</div>}
            {region.events && region.events.length > 0 && (
              <div className="mt-1.5 flex flex-col gap-0.5">
                {region.events.map(event => (
                  <div key={event} className="text-xs text-neutral-400">{eventLabel(event)}</div>
                ))}
              </div>
            )}
            {region.regions && region.regions.length > 0 && (
              <div className="mt-2 ml-3 border-l border-neutral-100 pl-3">
                <RegionList regions={region.regions} visibleRegions={visibleRegions} />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
