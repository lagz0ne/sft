/**
 * SFT Layout Tag Vocabulary
 *
 * Tags are Tailwind-like explicit instructions. Three concerns:
 *
 *   POSITION (where)  → sidebar, header, split:wide, mobile:bottomnav
 *   SKIN (what)       → list, form, tabs, detail, metric, card-grid, etc.
 *   VISUAL (how)      → elevated
 *
 * Position format: position[:modifier] or composition:position[:modifier]
 * Skin format: just the skin name as a standalone tag
 * Visual format: just the visual name as a standalone tag
 *
 * Example: tags: [sidebar, list, elevated]
 *   → position=sidebar, skin=list, elevated=true
 */

// --- Vocabulary definition ---

export type Position = 'sidebar' | 'header' | 'toolbar' | 'footer' | 'bottomnav' | 'aside' | 'overlay' | 'modal' | 'drawer' | 'banner' | 'split' | 'main'

type SizeModifier = 'narrow' | 'wide'

/** Which modifiers each position accepts */
const POSITION_MODIFIERS: Record<string, readonly string[]> = {
	sidebar: ['narrow', 'wide'],
	aside: ['narrow', 'wide'],
	split: ['narrow', 'wide'],
	header: [],
	toolbar: [],
	footer: [],
	bottomnav: [],
	banner: [],
	overlay: [],
	modal: [],
	drawer: [],
} as const

/** Visual tags — standalone, can coexist with any position */
const VISUAL_TAGS = new Set(['elevated'])

/** Skin tags removed — component bindings handle what regions render as */
export type SkinTag = 'placeholder'

const SKIN_TAGS = new Set<string>()

/** Check if a string is a known position */
function isPosition(s: string): boolean {
	return s in POSITION_MODIFIERS
}

// --- Parsed result ---

export interface ParsedLayout {
	position: Position
	modifier: SizeModifier | null
	elevated: boolean
}

const DEFAULT_LAYOUT: ParsedLayout = { position: 'main', modifier: null, elevated: false }

// --- Parser ---

/**
 * Parse a region's tags into a structured layout instruction.
 *
 * When composition is provided, looks for composition-prefixed tags first,
 * falls back to unprefixed (default) tags.
 *
 * Tag format disambiguation:
 *   "sidebar:narrow"         → first part IS a position → position:modifier
 *   "mobile:sidebar"         → first part is NOT a position → composition:position
 *   "mobile:sidebar:narrow"  → composition:position:modifier
 *
 * Examples:
 *   parseLayout(["sidebar:narrow", "mobile:bottomnav"])        → sidebar:narrow (default)
 *   parseLayout(["sidebar:narrow", "mobile:bottomnav"], "mobile") → bottomnav
 */
export function parseLayout(tags: string[] | undefined, composition?: string | null): ParsedLayout {
	if (!tags || tags.length === 0) return DEFAULT_LAYOUT

	let position: Position = 'main'
	let modifier: SizeModifier | null = null
	let elevated = false
	let foundComposition = false

	for (const tag of tags) {
		// Visual tags
		if (VISUAL_TAGS.has(tag)) {
			elevated = true
			continue
		}

		// Split tag into parts by colon
		const parts = tag.split(':')

		if (composition) {
			// Looking for composition-prefixed tags: composition:position[:modifier]
			if (parts.length >= 2 && parts[0] === composition && isPosition(parts[1])) {
				position = parts[1] as Position
				modifier = null
				if (parts.length >= 3) {
					const validMods = POSITION_MODIFIERS[parts[1]]
					if (validMods.includes(parts[2])) {
						modifier = parts[2] as SizeModifier
					}
				}
				foundComposition = true
			}
		}

		// Default (unprefixed) tags: position[:modifier]
		if (!foundComposition && parts.length <= 2 && isPosition(parts[0])) {
			position = parts[0] as Position
			modifier = null
			if (parts.length === 2) {
				const validMods = POSITION_MODIFIERS[parts[0]]
				if (validMods.includes(parts[1])) {
					modifier = parts[1] as SizeModifier
				}
			}
		}
	}

	return { position, modifier, elevated }
}

/**
 * Discover all composition names from a set of regions' tags.
 *
 * Scans all tags for prefixes that aren't known positions.
 * "mobile:sidebar" → "mobile" is a composition.
 * "sidebar:narrow" → "sidebar" IS a position, not a composition.
 */
export function discoverCompositions(regions: { tags?: string[] }[]): string[] {
	const compositions = new Set<string>()
	for (const r of regions) {
		for (const tag of r.tags ?? []) {
			const parts = tag.split(':')
			// A composition tag has 2+ parts and the first part is NOT a position
			if (parts.length >= 2 && !isPosition(parts[0]) && !VISUAL_TAGS.has(parts[0]) && !SKIN_TAGS.has(parts[0])) {
				compositions.add(parts[0])
			}
		}
	}
	return [...compositions].sort()
}

/** Grid column size based on parsed modifier */
export function modifierToCol(modifier: SizeModifier | null, fallback: string): string {
	switch (modifier) {
		case 'narrow': return 'minmax(8rem, 10rem)'
		case 'wide': return 'minmax(16rem, 20rem)'
		default: return fallback
	}
}

/** Flex ratio for split regions */
export function modifierToFlex(modifier: SizeModifier | null): string {
	switch (modifier) {
		case 'narrow': return '0.5'
		case 'wide': return '2'
		default: return '1'
	}
}
