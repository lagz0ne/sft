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

// --- Layout system ---

import { parseLayout, modifierToCol, modifierToFlex } from '../lib/layout-tags'

interface LayoutGroups {
	banner: Region[]
	header: Region[]
	sidebar: Region[]
	toolbar: Region[]
	main: Region[]
	aside: Region[]
	footer: Region[]
	bottomnav: Region[]
	overlay: Region[]
	modal: Region[]
	drawer: Region[]
	split: Region[]
}

/** Parse all regions' tags and group by position */
function groupByPosition(regions: Region[]): LayoutGroups {
	const groups: LayoutGroups = { banner: [], header: [], sidebar: [], toolbar: [], main: [], aside: [], footer: [], bottomnav: [], overlay: [], modal: [], drawer: [], split: [] }
	for (const r of regions) {
		const { position } = parseLayout(r.tags)
		groups[position].push(r)
	}
	return groups
}

function sizeToCol(regions: Region[], fallback: string): string {
	for (const r of regions) {
		const { modifier } = parseLayout(r.tags)
		if (modifier) return modifierToCol(modifier, fallback)
	}
	return fallback
}

function buildGridTemplate(g: LayoutGroups): React.CSSProperties {
	const hasSidebar = g.sidebar.length > 0
	const hasBanner = g.banner.length > 0
	const hasHeader = g.header.length > 0
	const hasToolbar = g.toolbar.length > 0
	const hasAside = g.aside.length > 0
	const hasFooter = g.footer.length > 0
	const hasBottomnav = g.bottomnav.length > 0

	const cols: string[] = []
	const areas: string[][] = []

	if (hasSidebar) cols.push(sizeToCol(g.sidebar, '12rem'))
	cols.push('1fr')
	if (hasAside) cols.push(sizeToCol(g.aside, 'minmax(12rem, 0.4fr)'))

	const colCount = cols.length

	// Build rows. Sidebar spans from toolbar through main (skips banner/header which are full-width above it).
	// Row order: banner → header → toolbar → main → footer → bottomnav
	// Sidebar spans: toolbar + main rows (the content area)
	// Aside spans: main row only

	const makeRow = (area: string): string[] => {
		const row: string[] = []
		if (hasSidebar) row.push(area === 'main' || area === 'toolbar' ? 'sidebar' : area)
		row.push(area)
		if (hasAside) row.push(area === 'main' ? 'aside' : area)
		return row
	}

	if (hasBanner) areas.push(Array(colCount).fill('banner'))
	if (hasHeader) areas.push(Array(colCount).fill('header'))
	if (hasToolbar) areas.push(makeRow('toolbar'))
	areas.push(makeRow('main'))
	if (hasFooter) areas.push(Array(colCount).fill('footer'))
	if (hasBottomnav) areas.push(Array(colCount).fill('bottomnav'))

	const rows: string[] = []
	if (hasBanner) rows.push('auto')
	if (hasHeader) rows.push('auto')
	if (hasToolbar) rows.push('auto')
	rows.push('1fr')
	if (hasFooter) rows.push('auto')
	if (hasBottomnav) rows.push('auto')

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

	// Combine app + screen regions, then group by position
	const allRegions = [...(appRegions ?? []), ...(screen.regions ?? [])]
	const groups = groupByPosition(allRegions)
	const gridStyle = buildGridTemplate(groups)

	// App-level regions bypass state_regions filter (they're always visible)
	const appRegionNames = new Set((appRegions ?? []).map(r => r.name))
	const regionProps = { visibleRegions, fixtureData, activeRegion, activeEvent, app, screen, taste }
	const propsFor = (r: Region) => appRegionNames.has(r.name) ? { ...regionProps, visibleRegions: null as string[] | null } : regionProps

	return (
		<div style={{ position: 'relative', height: '100%' }}>
			<div style={gridStyle}>
				{groups.banner.length > 0 && (
					<div style={{ gridArea: 'banner' }} className="flex flex-col gap-1">
						{groups.banner.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} compact />
						))}
					</div>
				)}

				{groups.header.length > 0 && (
					<div style={{ gridArea: 'header' }} className="flex flex-col gap-1.5">
						{groups.header.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
					</div>
				)}

				{groups.toolbar.length > 0 && (
					<div style={{ gridArea: 'toolbar' }} className="flex flex-col gap-1.5">
						{groups.toolbar.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
					</div>
				)}

				{groups.sidebar.length > 0 && (
					<div style={{ gridArea: 'sidebar' }} className="flex flex-col gap-1.5">
						{groups.sidebar.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} compact />
						))}
					</div>
				)}

				<div style={{ gridArea: 'main' }} className="min-h-0 overflow-auto">
					<div className="flex flex-col gap-1.5 h-full">
						{/* Main regions stack vertically */}
						{groups.main.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
						{/* Split regions sit side by side */}
						{groups.split.length > 0 && (
							<div className="flex gap-1.5 flex-1 min-h-0">
								{groups.split.map(r => (
									<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)}
										style={{ flex: modifierToFlex(parseLayout(r.tags).modifier) }}
									/>
								))}
							</div>
						)}
						{groups.main.length === 0 && groups.split.length === 0 && (
							<div className="flex-1 flex items-center justify-center border border-dashed border-neutral-200 rounded-lg text-neutral-300 text-sm">
								No main regions
							</div>
						)}
					</div>
				</div>

				{groups.aside.length > 0 && (
					<div style={{ gridArea: 'aside' }} className="flex flex-col gap-1.5">
						{groups.aside.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
					</div>
				)}

				{groups.footer.length > 0 && (
					<div style={{ gridArea: 'footer' }} className="flex flex-col gap-1.5">
						{groups.footer.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
					</div>
				)}

				{groups.bottomnav.length > 0 && (
					<div style={{ gridArea: 'bottomnav' }} className="flex flex-col gap-1">
						{groups.bottomnav.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} />
						))}
					</div>
				)}
			</div>

			{/* Overlays float above the grid */}
			{/* Overlay: anchored to bottom center */}
			{groups.overlay.length > 0 && (
				<div className="absolute inset-0 pointer-events-none flex items-end justify-center p-4">
					<div className="pointer-events-auto flex flex-col gap-1.5 w-full max-w-lg">
						{groups.overlay.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} isOverlay />
						))}
					</div>
				</div>
			)}

			{/* Modal: centered with backdrop dim */}
			{groups.modal.length > 0 && (
				<div className="absolute inset-0 pointer-events-none flex items-center justify-center p-8">
					<div className="absolute inset-0 bg-neutral-900/10 rounded-lg" />
					<div className="pointer-events-auto flex flex-col gap-1.5 w-full max-w-md relative">
						{groups.modal.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} isOverlay />
						))}
					</div>
				</div>
			)}

			{/* Drawer: attached to right edge */}
			{groups.drawer.length > 0 && (
				<div className="absolute inset-y-0 right-0 pointer-events-none flex items-stretch p-2">
					<div className="pointer-events-auto flex flex-col gap-1.5 w-72">
						{groups.drawer.map(r => (
							<WireframeRegion key={r.name} region={r} depth={0} {...propsFor(r)} isOverlay />
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
	style?: React.CSSProperties
}

function WireframeRegion({ region, depth, visibleRegions, fixtureData, activeRegion, activeEvent, compact, isOverlay, app, screen, taste, style }: WireframeRegionProps) {
	const hidden = visibleRegions != null && !visibleRegions.includes(region.name)
	const isActive = activeRegion === region.name
	const layout = parseLayout(region.tags)
	const hasOwnStateMachine = region.states && region.states.length > 0
	const hasFixtureContent = fixtureData && (fixtureData[region.name] != null)
	const hasChildren = region.regions && region.regions.length > 0

	// Check if region's own state machine hides it (overlay, modal, drawer with hidden initial state)
	const isFloating = layout.position === 'overlay' || layout.position === 'modal' || layout.position === 'drawer'
	const ownStateHidden = hasOwnStateMachine && isFloating
		&& region.states![0] === 'hidden'

	const effectiveHidden = hidden || (ownStateHidden && !isActive)

	// Hidden regions don't render — click a different state to see them
	if (effectiveHidden) return null

	return (
		<div
			style={style}
			className={[
				'transition-all duration-300 flex flex-col',
				layout.elevated ? 'rounded-xl shadow-lg bg-white border border-neutral-100' : 'rounded-lg border-2 border-neutral-200',
				isActive ? 'ring-2 ring-blue-400 ring-offset-1 border-blue-300' : '',
				hasFixtureContent && !layout.elevated ? 'bg-amber-50/50 border-amber-200' : '',
				isOverlay ? 'shadow-lg bg-white border-violet-300' : '',
				compact ? 'p-2' : 'p-3',
				depth === 0 ? 'flex-1 min-h-[48px]' : 'min-h-[36px]',
			].join(' ')}
		>
			{/* Region header — hidden for elevated regions (the skin IS the region) */}
			{!layout.elevated && (
				<div className="flex items-center gap-1.5 mb-1">
					<span className={`font-semibold ${compact ? 'text-xs' : 'text-sm'} text-neutral-700`}>
						{region.name}
					</span>
					{layout.position !== 'main' && (
						<span className="text-[9px] bg-blue-50 text-blue-500 px-1 py-0.5 rounded">
							{layout.position}
						</span>
					)}
					{hasOwnStateMachine && (
						<span className="text-[9px] bg-violet-100 text-violet-600 px-1 py-0.5 rounded">
							FSM
						</span>
					)}
					{hasFixtureContent && (
						<span className="text-[9px] bg-amber-100 text-amber-600 px-1 py-0.5 rounded">
							data
						</span>
					)}
				</div>
			)}

			{/* Skin content */}
			<SkinRenderer
					region={region}
					app={app}
					screen={screen}
					fixtureData={fixtureData}
					compact={compact}
					taste={taste}
			/>

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
