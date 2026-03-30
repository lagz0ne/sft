import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useSpecContext } from '../context/spec-context'
import { useMemo } from 'react'
import { WireframeCanvas } from '../components/wireframe-canvas'
import { Dock, PRESET_SIZES, type ViewportSizeType } from '../components/dock'
import { discoverCompositions } from '../lib/layout-tags'

type PlaygroundSearch = {
	screen: string
	state: string
	set: string
	layout: string
	width: number
}

export const Route = createFileRoute('/playground')({
	validateSearch: (search: Record<string, unknown>): PlaygroundSearch => ({
		screen: (search.screen as string) || '',
		state: (search.state as string) || '',
		set: (search.set as string) || 'wireframe',
		layout: (search.layout as string) || '',
		width: Number(search.width) || 0,
	}),
	component: PlaygroundPage,
})

const COMPONENT_SETS = [
	{ id: 'wireframe', label: 'wireframe' },
	{ id: 'styled', label: 'styled' },
	{ id: 'compact', label: 'compact' },
]

function PlaygroundPage() {
	const { spec, loading } = useSpecContext()
	const search = Route.useSearch()
	const navigate = useNavigate({ from: Route.fullPath })

	const set = (params: Partial<PlaygroundSearch>) => {
		navigate({ search: (prev: PlaygroundSearch) => ({ ...prev, ...params }), replace: true })
	}

	if (loading || !spec) return <div className="h-full flex items-center justify-center text-neutral-400">Loading...</div>

	const currentScreen = search.screen || spec.screens[0]?.name || ''
	const screen = spec.screens.find(s => s.name === currentScreen) ?? spec.screens[0]
	const states = screen?.states ?? []
	const activeState = search.state || states[0] || null
	const currentComposition = search.layout || null

	const allRegions = useMemo(() =>
		[...(spec.app.regions ?? []), ...(screen?.regions ?? [])],
		[spec.app.regions, screen?.regions]
	)
	const compositions = useMemo(() => {
		const found = discoverCompositions(allRegions)
		if (!found.includes('mobile')) found.push('mobile')
		return found
	}, [allRegions])

	const viewportSizes = useMemo(() => {
		const sizes = [...PRESET_SIZES]
		for (const comp of compositions) {
			const autoWidth = /^mobile$/i.test(comp) ? 375
				: /^tablet$/i.test(comp) ? 768
				: /^desktop$/i.test(comp) ? 1280
				: null
			if (autoWidth) {
				const idx = sizes.findIndex(s => s.width === autoWidth && !s.composition)
				if (idx >= 0) sizes[idx] = { label: comp, width: autoWidth, composition: comp }
			}
		}
		return sizes
	}, [compositions])

	const viewportWidth = useMemo(() => {
		const match = viewportSizes.find(s =>
			(search.layout && s.composition === search.layout) ||
			(search.width && s.width === search.width)
		)
		return match?.width ?? (search.width || null)
	}, [viewportSizes, search.layout, search.width])

	const switchScreen = (name: string) => set({ screen: name, state: '' })

	const handleViewportSize = (size: ViewportSizeType) => {
		set({
			width: size.width ?? 0,
			layout: size.composition ?? '',
		})
	}

	return (
		<div className="h-full bg-neutral-100 relative flex items-start justify-center overflow-auto">
			<div
				className="bg-neutral-50 min-h-full transition-all duration-300 shadow-sm"
				style={{ width: viewportWidth ? `${viewportWidth}px` : '100%', maxWidth: '100%' }}
			>
				<div className="h-full p-3 pb-12">
					{screen && (
						<div className="h-full">
							<WireframeCanvas
								screen={screen}
								currentState={activeState}
								appRegions={spec.app.regions}
								fixtures={spec.fixtures}
								activeRegion={null}
								activeEvent={null}
								app={spec.app}
								componentSet={search.set}
								composition={currentComposition}
							/>
						</div>
					)}
				</div>
			</div>

			<Dock
				screens={spec.screens.map(s => ({
					id: s.name, label: s.name, active: screen?.name === s.name,
				}))}
				onScreen={(id: string) => switchScreen(id)}
				states={states.map(s => ({ id: s, label: s, active: activeState === s }))}
				onState={(id: string) => set({ state: id })}
				layouts={[
					{ id: '__default', label: 'default', active: currentComposition === null },
					...compositions.map(c => ({ id: c, label: c, active: currentComposition === c })),
				].filter((_, __, arr) => arr.length > 1)}
				onLayout={(id: string) => set({ layout: id === '__default' ? '' : id })}
				componentSets={COMPONENT_SETS.map(s => ({ ...s, active: search.set === s.id }))}
				onComponentSet={(id: string) => set({ set: id })}
				viewportSizes={viewportSizes}
				activeViewportWidth={viewportWidth}
				onViewportSize={handleViewportSize}
			/>
		</div>
	)
}
