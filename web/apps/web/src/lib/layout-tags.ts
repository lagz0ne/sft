/**
 * SFT Layout Tag Vocabulary
 *
 * Tags are Tailwind-like explicit layout instructions.
 * Format: position or position:modifier
 *
 * Each position defines what modifiers it accepts.
 * One tag = one instruction. No impossible combinations.
 */

// --- Vocabulary definition ---

export type Position = 'sidebar' | 'header' | 'toolbar' | 'footer' | 'bottomnav' | 'aside' | 'overlay' | 'modal' | 'drawer' | 'banner' | 'split' | 'main'

type SizeModifier = 'narrow' | 'wide'

/** Which modifiers each position accepts */
const POSITION_MODIFIERS: Record<string, readonly string[]> = {
	sidebar: ['narrow', 'wide'],
	aside: ['narrow', 'wide'],
	split: ['narrow', 'wide'],
	// Positions with no modifiers — always full width or fixed
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
 * Scans the tag list for:
 * 1. A position tag (with optional :modifier) → grid placement + sizing
 * 2. Visual tags (elevated) → styling
 *
 * Examples:
 *   ["sidebar:narrow", "elevated"]  → { position: "sidebar", modifier: "narrow", elevated: true }
 *   ["toolbar"]                     → { position: "toolbar", modifier: null, elevated: false }
 *   ["split:wide"]                 → { position: "split", modifier: "wide", elevated: false }
 *   ["preview_pane_enabled"]        → { position: "main", modifier: null, elevated: false }
 *   []                              → { position: "main", modifier: null, elevated: false }
 */
export function parseLayout(tags: string[] | undefined): ParsedLayout {
	if (!tags || tags.length === 0) return DEFAULT_LAYOUT

	let position: Position = 'main'
	let modifier: SizeModifier | null = null
	let elevated = false

	for (const tag of tags) {
		// Check visual tags first
		if (VISUAL_TAGS.has(tag)) {
			elevated = true
			continue
		}

		// Parse position:modifier format
		const colonIdx = tag.indexOf(':')
		const pos = colonIdx >= 0 ? tag.slice(0, colonIdx) : tag
		const mod = colonIdx >= 0 ? tag.slice(colonIdx + 1) : null

		// Check if this is a recognized position
		if (pos in POSITION_MODIFIERS) {
			position = pos as Position

			// Validate modifier if present
			if (mod) {
				const validMods = POSITION_MODIFIERS[pos]
				if (validMods.includes(mod)) {
					modifier = mod as SizeModifier
				}
				// Invalid modifier → silently ignore the modifier, keep the position
			}
		}
		// Unrecognized tags (feature flags, domain tags) → skip
	}

	return { position, modifier, elevated }
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
