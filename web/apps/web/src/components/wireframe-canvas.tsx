import type { App, Fixture, Region, Screen, TasteTokens } from '../lib/types'
import { selectSkin } from '../lib/skin-selector'
import { DataList } from './skins/data-list'
import { FormLayout } from './skins/form-layout'
import { Tabs } from './skins/tabs'
import { DetailCard } from './skins/detail-card'
import { ActionBar } from './skins/action-bar'
import { ActionButton } from './skins/action-button'
import { SearchInput } from './skins/search-input'
import { Placeholder } from './skins/placeholder'

// --- Layout roles from tags ---

type LayoutRole = 'sidebar' | 'toolbar' | 'footer' | 'aside' | 'overlay' | 'main'

const LAYOUT_TAGS = new Set<LayoutRole>(['sidebar', 'toolbar', 'footer', 'aside', 'overlay'])

function getRole(region: Region): LayoutRole {
	if (!region.tags) return 'main'
	for (const tag of region.tags) {
		if (LAYOUT_TAGS.has(tag as LayoutRole)) return tag as LayoutRole
	}
	return 'main'
}

interface LayoutGroups {
	sidebar: Region[]
	toolbar: Region[]
	main: Region[]
	aside: Region[]
	footer: Region[]
	overlay: Region[]
}

function groupByRole(regions: Region[]): LayoutGroups {
	const groups: LayoutGroups = { sidebar: [], toolbar: [], main: [], aside: [], footer: [], overlay: [] }
	for (const r of regions) {
		groups[getRole(r)].push(r)
	}
	return groups
}

function buildGridTemplate(g: LayoutGroups): React.CSSProperties {
	const hasSidebar = g.sidebar.length > 0
	const hasToolbar = g.toolbar.length > 0
	const hasAside = g.aside.length > 0
	const hasFooter = g.footer.length > 0

	const cols: string[] = []
	const areas: string[][] = []

	if (hasSidebar) cols.push('12rem')
	cols.push('1fr')
	if (hasAside) cols.push('minmax(12rem, 0.4fr)')

	const colCount = cols.length

	if (hasToolbar) {
		areas.push(Array(colCount).fill('toolbar'))
	}

	const mainRow: string[] = []
	if (hasSidebar) mainRow.push('sidebar')
	mainRow.push('main')
	if (hasAside) mainRow.push('aside')
	areas.push(mainRow)

	if (hasFooter) {
		areas.push(Array(colCount).fill('footer'))
	}

	const rows: string[] = []
	if (hasToolbar) rows.push('auto')
	rows.push('1fr')
	if (hasFooter) rows.push('auto')

	return {
		display: 'grid',
		gridTemplateColumns: cols.join(' '),
		gridTemplateRows: rows.join(' '),
		gridTemplateAreas: areas.map(row => `"${row.join(' ')}"`).join(' '),
		gap: '6px',
		height: '100%',
	}
}

// --- Fixture resolution ---

function resolveFixture(name: string, fixtures: Fixture[]): Record<string, any> | null {
	const f = fixtures.find(fx => fx.name === name)
	if (!f) return null
	if (f.extends) {
		const base = resolveFixture(f.extends, fixtures)
		if (base) return deepMerge(base, f.data)
	}
	return f.data
}

function deepMerge(base: Record<string, any>, overlay: Record<string, any>): Record<string, any> {
	const result = { ...base }
	for (const [k, v] of Object.entries(overlay)) {
		if (v && typeof v === 'object' && !Array.isArray(v) && result[k] && typeof result[k] === 'object' && !Array.isArray(result[k])) {
			result[k] = deepMerge(result[k], v)
		} else {
			result[k] = v
		}
	}
	return result
}

// --- Skin dispatcher ---

function SkinRenderer({ region, app, screen, fixtureData, compact, taste }: {
	region: Region
	app: App
	screen: Screen
	fixtureData?: Record<string, any> | null
	compact?: boolean
	taste?: TasteTokens
}) {
	const ctx = selectSkin(region, app, screen)
	const props = { region, context: ctx, fixtureData, screenName: screen.name, compact, taste }

	switch (ctx.skin) {
		case 'data-list': return <DataList {...props} />
		case 'form': return <FormLayout {...props} />
		case 'tabs': return <Tabs {...props} />
		case 'detail-card': return <DetailCard {...props} />
		case 'action-bar': return <ActionBar {...props} />
		case 'action-button': return <ActionButton {...props} />
		case 'search-input': return <SearchInput {...props} />
		case 'placeholder': return <Placeholder {...props} />
	}
}

// --- Canvas ---

interface WireframeCanvasProps {
	screen: Screen
	currentState: string | null
	appRegions?: Region[]
	fixtures?: Fixture[]
	activeRegion?: string | null
	activeEvent?: string | null
	app: App
	taste?: TasteTokens
}

export function WireframeCanvas({ screen, currentState, appRegions, fixtures, activeRegion, activeEvent, app, taste }: WireframeCanvasProps) {
	const visibleRegions = currentState && screen.state_regions
		? screen.state_regions[currentState]
		: null

	const fixtureName = currentState && screen.state_fixtures
		? screen.state_fixtures[currentState]
		: null
	const fixtureData = fixtureName && fixtures
		? resolveFixture(fixtureName, fixtures)
		: null

	// Combine app + screen regions, then group by layout role
	const allRegions = [...(appRegions ?? []), ...(screen.regions ?? [])]
	const groups = groupByRole(allRegions)
	const gridStyle = buildGridTemplate(groups)

	const regionProps = { visibleRegions, fixtureData, activeRegion, activeEvent, app, screen, taste }

	return (
		<div style={{ position: 'relative', height: '100%' }}>
			<div style={gridStyle}>
				{groups.toolbar.length > 0 && (
					<div style={{ gridArea: 'toolbar' }} className="flex flex-col gap-1.5">
						{groups.toolbar.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} />
						))}
					</div>
				)}

				{groups.sidebar.length > 0 && (
					<div style={{ gridArea: 'sidebar' }} className="flex flex-col gap-1.5">
						{groups.sidebar.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} compact />
						))}
					</div>
				)}

				<div style={{ gridArea: 'main' }} className="flex flex-col gap-1.5 min-h-0 overflow-auto">
					{groups.main.length > 0 ? (
						groups.main.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} />
						))
					) : (
						<div className="flex-1 flex items-center justify-center border border-dashed border-neutral-200 rounded-lg text-neutral-300 text-sm">
							No main regions
						</div>
					)}
				</div>

				{groups.aside.length > 0 && (
					<div style={{ gridArea: 'aside' }} className="flex flex-col gap-1.5">
						{groups.aside.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} />
						))}
					</div>
				)}

				{groups.footer.length > 0 && (
					<div style={{ gridArea: 'footer' }} className="flex flex-col gap-1.5">
						{groups.footer.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} />
						))}
					</div>
				)}
			</div>

			{/* Overlays float above the grid */}
			{groups.overlay.length > 0 && (
				<div className="absolute inset-0 pointer-events-none flex items-end justify-center p-4">
					<div className="pointer-events-auto flex flex-col gap-1.5 w-full max-w-lg">
						{groups.overlay.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...regionProps} isOverlay />
						))}
					</div>
				</div>
			)}
		</div>
	)
}

// --- Region ---

interface WireframeRegionProps {
	region: Region
	depth: number
	visibleRegions: string[] | null
	fixtureData?: Record<string, any> | null
	activeRegion?: string | null
	activeEvent?: string | null
	compact?: boolean
	isOverlay?: boolean
	app: App
	screen: Screen
	taste?: TasteTokens
}

function WireframeRegion({ region, depth, visibleRegions, fixtureData, activeRegion, activeEvent, compact, isOverlay, app, screen, taste }: WireframeRegionProps) {
	const hidden = visibleRegions != null && !visibleRegions.includes(region.name)
	const isActive = activeRegion === region.name
	const hasOwnStateMachine = region.states && region.states.length > 0
	const hasFixtureContent = fixtureData && (fixtureData[region.name] != null)
	const hasChildren = region.regions && region.regions.length > 0
	const role = getRole(region)

	// Check if region's own state machine hides it
	const ownStateHidden = hasOwnStateMachine && region.states![0] !== undefined
		&& region.tags?.includes('overlay')
		&& region.states![0] === 'hidden'

	const effectiveHidden = hidden || (ownStateHidden && !isActive)

	return (
		<div
			className={[
				'rounded-lg border-2 transition-all duration-300 flex flex-col',
				effectiveHidden ? 'opacity-15 border-dashed border-neutral-200' : 'border-neutral-200',
				isActive ? 'ring-2 ring-blue-400 ring-offset-1 border-blue-300' : '',
				hasFixtureContent && !effectiveHidden ? 'bg-amber-50/50 border-amber-200' : '',
				isOverlay && !effectiveHidden ? 'shadow-lg bg-white border-violet-300' : '',
				compact ? 'p-2' : 'p-3',
				depth === 0 ? 'flex-1 min-h-[48px]' : 'min-h-[36px]',
			].join(' ')}
		>
			{/* Region header */}
			<div className="flex items-center gap-1.5 mb-1">
				<span className={`font-semibold ${compact ? 'text-xs' : 'text-sm'} text-neutral-700`}>
					{region.name}
				</span>
				{role !== 'main' && (
					<span className="text-[9px] bg-blue-50 text-blue-500 px-1 py-0.5 rounded">
						{role}
					</span>
				)}
				{hasOwnStateMachine && (
					<span className="text-[9px] bg-violet-100 text-violet-600 px-1 py-0.5 rounded">
						FSM
					</span>
				)}
				{hasFixtureContent && !effectiveHidden && (
					<span className="text-[9px] bg-amber-100 text-amber-600 px-1 py-0.5 rounded">
						data
					</span>
				)}
				{effectiveHidden && (
					<span className="text-[9px] text-neutral-400 italic">hidden</span>
				)}
			</div>

			{/* Skin content */}
			{!effectiveHidden && (
				<SkinRenderer
					region={region}
					app={app}
					screen={screen}
					fixtureData={fixtureData}
					compact={compact}
					taste={taste}
				/>
			)}

			{/* Nested regions */}
			{hasChildren && (
				<div className={`flex gap-1.5 mt-auto ${
					region.regions!.length <= 2 ? 'flex-row' : 'flex-col'
				}`}>
					{region.regions!.map(child => (
						<WireframeRegion
							key={child.name}
							region={child}
							depth={depth + 1}
							visibleRegions={visibleRegions}
							fixtureData={fixtureData}
							activeRegion={activeRegion}
							activeEvent={activeEvent}
							compact={compact}
							app={app}
							screen={screen}
							taste={taste}
						/>
					))}
				</div>
			)}
		</div>
	)
}
