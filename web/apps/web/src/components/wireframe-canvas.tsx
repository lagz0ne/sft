import type { App, Fixture, Region, Screen } from '../lib/types'
import { DataList } from './skins/data-list'
import { Tabs } from './skins/tabs'
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

/** Parse all regions' tags and group by position, respecting active composition */
function groupByPosition(regions: Region[], composition?: string | null): LayoutGroups {
	const groups: LayoutGroups = { banner: [], header: [], sidebar: [], toolbar: [], main: [], aside: [], footer: [], bottomnav: [], overlay: [], modal: [], drawer: [], split: [] }
	for (const r of regions) {
		const { position } = parseLayout(r.tags, composition)
		groups[position].push(r)
	}
	return groups
}

function sizeToCol(regions: Region[], fallback: string, composition?: string | null): string {
	for (const r of regions) {
		const { modifier } = parseLayout(r.tags, composition)
		if (modifier) return modifierToCol(modifier, fallback)
	}
	return fallback
}

function buildGridTemplate(g: LayoutGroups, composition?: string | null): React.CSSProperties {
	const hasSidebar = g.sidebar.length > 0
	const hasBanner = g.banner.length > 0
	const hasHeader = g.header.length > 0
	const hasToolbar = g.toolbar.length > 0
	const hasAside = g.aside.length > 0
	const hasFooter = g.footer.length > 0
	const hasBottomnav = g.bottomnav.length > 0

	const cols: string[] = []
	const areas: string[][] = []

	if (hasSidebar) cols.push(sizeToCol(g.sidebar, '12rem', composition))
	cols.push('1fr')
	if (hasAside) cols.push(sizeToCol(g.aside, 'minmax(12rem, 0.4fr)', composition))

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

// --- Component → wireframe shape mapping ---

// Map any json-render component type to one of 8 primitive wireframe shapes
type WireframeShape = 'input' | 'select' | 'button' | 'image' | 'text' | 'list' | 'card' | 'tabs'

const COMPONENT_SHAPE: Record<string, WireframeShape> = {
	// Input family
	Input: 'input', Textarea: 'input', Slider: 'input',
	// Select family
	Select: 'select', Checkbox: 'select', Radio: 'select', Toggle: 'select',
	// Button family
	Button: 'button', ButtonGroup: 'button',
	// Image family
	Image: 'image', Avatar: 'image',
	// Text family
	Text: 'text', Heading: 'text', Badge: 'text', Alert: 'text',
	// List/table family
	Table: 'list', Stack: 'list',
	// Card/grid family
	Card: 'card', Grid: 'card',
	// Tabs family
	Tabs: 'tabs', Accordion: 'tabs',
	// Feedback → mapped to closest shape
	Progress: 'input', Spinner: 'image', Skeleton: 'text',
	Rating: 'select', Pagination: 'tabs',
}

function resolveShape(componentType: string | undefined): WireframeShape | null {
	if (!componentType) return null
	return COMPONENT_SHAPE[componentType] ?? null
}

// --- Component wireframe renderer ---

function ComponentRenderer({ component, componentProps, region, screen, fixtureData, compact, componentSet }: {
	component: string
	componentProps?: string
	region: Region
	screen: Screen
	fixtureData?: Record<string, any> | null
	compact?: boolean
	componentSet?: string
}) {
	const shape = resolveShape(component)
	const props = componentProps ? JSON.parse(componentProps) : {}
	const skinCtx = { skin: 'placeholder' as const, fields: {} }
	const skinProps = { region, context: skinCtx, fixtureData, screenName: screen.name, compact, componentSet }

	switch (shape) {
		case 'input': return <InputShape label={props.label} placeholder={props.placeholder} type={props.type} componentSet={componentSet} componentType={component} />
		case 'select': return <SelectShape label={props.label} options={props.options ?? props.items} componentSet={componentSet} componentType={component} />
		case 'button': return <ButtonShape label={props.label ?? component} variant={props.variant} componentSet={componentSet} componentType={component} />
		case 'image': return <ImageShape aspect={props.aspect} alt={props.alt} componentSet={componentSet} componentType={component} />
		case 'text': return <TextShape content={props.content} level={props.level} componentSet={componentSet} componentType={component} />
		case 'list': return <DataList {...skinProps} />
		case 'card': return <Placeholder {...skinProps} />
		case 'tabs': return <Tabs {...skinProps} />
		default: return <Placeholder {...skinProps} />
	}
}

// --- Wireframe primitive shapes ---
// Each renders a recognizable wireframe representation of a json-render component.
// Props enhance the shape with real labels, placeholders, and variants.

// Shared: component type badge
function TypeBadge({ type, dark }: { type: string; dark?: boolean }) {
	return (
		<span className={`absolute top-0 right-0 translate-x-0.5 -translate-y-1/2 text-[6px] px-1 py-px rounded-full leading-none font-mono ${
			dark ? 'bg-stone-700 text-stone-400 ring-1 ring-stone-600' : 'bg-stone-100 text-stone-500 ring-1 ring-stone-200'
		}`}>{type}</span>
	)
}

function InputShape({ label, placeholder, type, componentSet, componentType }: {
	label?: string; placeholder?: string; type?: string; componentSet?: string; componentType?: string
}) {
	const dark = componentSet === "compact" || componentSet === "styled"
	const isTextarea = type === 'textarea' || componentType === 'Textarea'
	const isSlider = componentType === 'Slider'
	const border = dark ? 'border-stone-600' : 'border-stone-300/80'
	const bg = dark ? 'bg-stone-800/50' : 'bg-white'

	if (isSlider) {
		return (
			<div className="flex flex-col gap-1 w-full relative">
				{label && <div className={`text-[8px] font-medium ${dark ? 'text-stone-400' : 'text-stone-500'}`}>{label}</div>}
				<div className="flex items-center gap-1.5 h-4">
					<div className={`flex-1 h-[3px] rounded-full ${dark ? 'bg-stone-700' : 'bg-stone-200'} relative`}>
						<div className="absolute left-0 top-0 h-full w-2/5 rounded-full" style={{ backgroundColor: dark ? '#666' : '#999' }} />
						<div className="absolute top-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full bg-white ring-1 ring-stone-300 shadow-sm" style={{ left: '40%' }} />
					</div>
				</div>
			</div>
		)
	}

	return (
		<div className="flex flex-col gap-1 w-full relative">
			{componentType && <TypeBadge type={componentType} dark={dark} />}
			{label && <div className={`text-[8px] font-medium ${dark ? 'text-stone-400' : 'text-stone-500'}`}>{label}</div>}
			<div className={`${isTextarea ? 'min-h-[3rem]' : 'h-7'} rounded-md border ${border} ${bg} flex items-start px-2 ${isTextarea ? 'pt-1.5' : 'items-center'}`}
				style={{ boxShadow: dark ? 'none' : 'inset 0 1px 2px rgba(0,0,0,0.04)' }}>
				{placeholder ? (
					<span className={`text-[9px] ${dark ? 'text-stone-600' : 'text-stone-400'}`}>{placeholder}</span>
				) : (
					<div className={`w-px h-3 ${dark ? 'bg-stone-500' : 'bg-stone-400'} animate-pulse`} />
				)}
			</div>
		</div>
	)
}

function SelectShape({ label, options, componentSet, componentType }: {
	label?: string; options?: string[]; componentSet?: string; componentType?: string
}) {
	const dark = componentSet === "compact" || componentSet === "styled"
	const border = dark ? 'border-stone-600' : 'border-stone-300/80'
	const bg = dark ? 'bg-stone-800/50' : 'bg-white'
	const isToggle = componentType === 'Toggle' || componentType === 'Checkbox'

	if (isToggle) {
		return (
			<div className="flex items-center gap-2 relative">
				{componentType && <TypeBadge type={componentType} dark={dark} />}
				<div className={`w-7 h-4 rounded-full ${dark ? 'bg-stone-600' : 'bg-stone-200'} relative`}>
					<div className="absolute top-0.5 left-0.5 w-3 h-3 rounded-full bg-white shadow-sm" />
				</div>
				{label && <span className={`text-[9px] ${dark ? 'text-stone-400' : 'text-stone-600'}`}>{label}</span>}
			</div>
		)
	}

	return (
		<div className="flex flex-col gap-1 w-full relative">
			{componentType && <TypeBadge type={componentType} dark={dark} />}
			{label && <div className={`text-[8px] font-medium ${dark ? 'text-stone-400' : 'text-stone-500'}`}>{label}</div>}
			<div className={`h-7 rounded-md border ${border} ${bg} flex items-center justify-between px-2`}
				style={{ boxShadow: dark ? 'none' : 'inset 0 1px 2px rgba(0,0,0,0.04)' }}>
				<span className={`text-[9px] ${dark ? 'text-stone-500' : 'text-stone-500'}`}>
					{options?.[0] ?? 'Select...'}
				</span>
				<svg className={`w-3 h-3 ${dark ? 'text-stone-500' : 'text-stone-400'}`} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
					<path d="M4 6l4 4 4-4" />
				</svg>
			</div>
		</div>
	)
}

function ButtonShape({ label, variant, componentSet, componentType }: {
	label?: string; variant?: string; componentSet?: string; componentType?: string
}) {
	const dark = componentSet === "compact" || componentSet === "styled"
	const accent = undefined
	const isPrimary = variant === 'primary' || variant === 'default' || !variant
	const isGhost = variant === 'ghost' || variant === 'outline'
	const isDestructive = variant === 'destructive'

	let className = 'inline-flex items-center justify-center h-7 px-3.5 rounded-md text-[9px] font-medium tracking-wide transition-colors relative'

	if (isDestructive) {
		className += ' bg-red-500/90 text-white'
	} else if (isGhost) {
		className += dark
			? ' border border-stone-600 text-stone-300'
			: ' border border-stone-300/80 text-stone-600'
	} else if (isPrimary) {
		if (!accent) className += dark ? ' bg-stone-100 text-stone-900' : ' bg-stone-800 text-stone-50'
	} else {
		className += dark ? ' bg-stone-700 text-stone-200' : ' bg-stone-100 text-stone-700'
	}

	return (
		<div className={className}
			style={isPrimary && accent && !isDestructive ? {
				backgroundColor: accent,
				color: '#fff',
				boxShadow: `0 1px 3px ${accent}40`,
			} : undefined}
		>
			{componentType && <TypeBadge type={componentType} dark={dark} />}
			{label ?? 'Button'}
		</div>
	)
}

function ImageShape({ aspect, alt, componentSet, componentType }: {
	aspect?: string; alt?: string; componentSet?: string; componentType?: string
}) {
	const dark = componentSet === "compact" || componentSet === "styled"
	const aspectClass = aspect === 'square' ? 'aspect-square max-h-24'
		: aspect === 'video' ? 'aspect-video'
		: 'h-16'

	return (
		<div className={`${aspectClass} w-full rounded-md overflow-hidden relative ${dark ? 'bg-stone-800' : 'bg-stone-50'}`}
			style={{
				backgroundImage: dark
					? 'linear-gradient(45deg, #292524 25%, transparent 25%, transparent 75%, #292524 75%), linear-gradient(45deg, #292524 25%, transparent 25%, transparent 75%, #292524 75%)'
					: 'linear-gradient(45deg, #f5f5f4 25%, transparent 25%, transparent 75%, #f5f5f4 75%), linear-gradient(45deg, #f5f5f4 25%, transparent 25%, transparent 75%, #f5f5f4 75%)',
				backgroundSize: '8px 8px',
				backgroundPosition: '0 0, 4px 4px',
			}}
		>
			{componentType && <TypeBadge type={componentType} dark={dark} />}
			<div className="absolute inset-0 flex flex-col items-center justify-center gap-1">
				<svg className={`w-5 h-5 ${dark ? 'text-stone-600' : 'text-stone-300'}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.2">
					<rect x="3" y="3" width="18" height="18" rx="2" />
					<circle cx="8.5" cy="8.5" r="1.5" />
					<path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
				</svg>
				{alt && <span className={`text-[7px] ${dark ? 'text-stone-600' : 'text-stone-400'}`}>{alt}</span>}
			</div>
		</div>
	)
}

function TextShape({ content, level, componentSet, componentType }: {
	content?: string; level?: number; componentSet?: string; componentType?: string
}) {
	const dark = componentSet === "compact" || componentSet === "styled"
	if (content) {
		const isHeading = (level && level <= 3) || componentType === 'Heading'
		return (
			<div className={`relative ${isHeading ? 'text-[12px] font-semibold tracking-tight' : 'text-[9px] leading-relaxed'} ${dark ? 'text-stone-300' : 'text-stone-700'}`}>
				{componentType && <TypeBadge type={componentType} dark={dark} />}
				{content}
			</div>
		)
	}

	const bar = dark ? 'bg-stone-600' : 'bg-stone-300/70'
	const line = dark ? 'bg-stone-700' : 'bg-stone-200/80'
	return (
		<div className="flex flex-col gap-1.5 w-full relative">
			{componentType && <TypeBadge type={componentType} dark={dark} />}
			<div className={`h-2.5 rounded w-3/5 ${bar}`} />
			<div className={`h-[5px] rounded-sm w-full ${line}`} />
			<div className={`h-[5px] rounded-sm w-11/12 ${line}`} />
			<div className={`h-[5px] rounded-sm w-4/5 ${line}`} />
		</div>
	)
}

// --- "Set component" prompt ---

function UnboundPrompt({ name, componentSet }: { name: string; componentSet?: string }) {
	const dark = componentSet === "compact" || componentSet === "styled"
	return (
		<div className={`flex flex-col items-center justify-center py-3 rounded-md border border-dashed ${
			dark ? 'border-stone-700' : 'border-stone-300/60'
		}`}>
			<div className={`text-[9px] font-mono ${dark ? 'text-stone-600' : 'text-stone-400'}`}>
				sft component {name} <span className={dark ? 'text-stone-500' : 'text-stone-300'}>Type</span>
			</div>
		</div>
	)
}

// --- Skin dispatcher (reads region.component, falls back to tag, then prompt) ---

function SkinRenderer({ region, screen, fixtureData, compact, componentSet }: {
	region: Region
	screen: Screen
	fixtureData?: Record<string, any> | null
	compact?: boolean
	componentSet?: string
}) {
	// Priority 1: component binding
	if (region.component) {
		return <ComponentRenderer
			component={region.component}
			componentProps={region.component_props}
			region={region}
			screen={screen}
			fixtureData={fixtureData}
			compact={compact}
			componentSet={componentSet}
		/>
	}

	// Priority 2: no component → prompt to set
	return <UnboundPrompt name={region.name} componentSet={componentSet} />
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
	componentSet?: string
	composition?: string | null
}

export function WireframeCanvas({ screen, currentState, appRegions, fixtures, activeRegion, activeEvent, app, componentSet, composition }: WireframeCanvasProps) {
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
	const groups = groupByPosition(allRegions, composition)
	const gridStyle = buildGridTemplate(groups, composition)

	// App-level regions bypass state_regions filter (they're always visible)
	const appRegionNames = new Set((appRegions ?? []).map(r => r.name))
	const regionProps = { visibleRegions, fixtureData, activeRegion, activeEvent, app, screen, componentSet, composition }
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
										style={{ flex: modifierToFlex(parseLayout(r.tags, composition).modifier) }}
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
	componentSet?: string
	composition?: string | null
	style?: React.CSSProperties
}

function WireframeRegion({ region, depth, visibleRegions, fixtureData, activeRegion, activeEvent, compact, isOverlay, app, screen, componentSet, composition, style }: WireframeRegionProps) {
	const hidden = visibleRegions != null && !visibleRegions.includes(region.name)
	const isActive = activeRegion === region.name
	const layout = parseLayout(region.tags, composition)
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

			{/* Component content */}
			<SkinRenderer
					region={region}
					screen={screen}
					fixtureData={fixtureData}
					compact={compact}
					componentSet={componentSet}
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
							componentSet={componentSet}
						/>
					))}
				</div>
			)}
		</div>
	)
}
