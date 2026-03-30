import { useState, useEffect, useRef, useCallback } from 'react'

export interface CommandItem {
	id: string
	label: string
	group: string
	active?: boolean
	onSelect: () => void
}

interface CommandBarProps {
	items: CommandItem[]
	open: boolean
	onClose: () => void
}

export function CommandBar({ items, open, onClose }: CommandBarProps) {
	const [query, setQuery] = useState('')
	const [selectedIndex, setSelectedIndex] = useState(0)
	const inputRef = useRef<HTMLInputElement>(null)

	// Filter items by query
	const filtered = query
		? items.filter(item =>
			item.label.toLowerCase().includes(query.toLowerCase()) ||
			item.group.toLowerCase().includes(query.toLowerCase())
		)
		: items

	// Group filtered items
	const groups = new Map<string, CommandItem[]>()
	for (const item of filtered) {
		if (!groups.has(item.group)) groups.set(item.group, [])
		groups.get(item.group)!.push(item)
	}

	// Flat list for keyboard nav
	const flatList = [...filtered]

	// Reset on open
	useEffect(() => {
		if (open) {
			setQuery('')
			setSelectedIndex(0)
			setTimeout(() => inputRef.current?.focus(), 50)
		}
	}, [open])

	// Clamp selected index
	useEffect(() => {
		if (selectedIndex >= flatList.length) setSelectedIndex(Math.max(0, flatList.length - 1))
	}, [flatList.length, selectedIndex])

	const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
		switch (e.key) {
			case 'ArrowDown':
				e.preventDefault()
				setSelectedIndex(i => Math.min(i + 1, flatList.length - 1))
				break
			case 'ArrowUp':
				e.preventDefault()
				setSelectedIndex(i => Math.max(i - 1, 0))
				break
			case 'Enter':
				e.preventDefault()
				if (flatList[selectedIndex]) {
					flatList[selectedIndex].onSelect()
					onClose()
				}
				break
			case 'Escape':
				e.preventDefault()
				onClose()
				break
		}
	}, [flatList, selectedIndex, onClose])

	if (!open) return null

	return (
		<div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]" onClick={onClose}>
			<div
				className="w-full max-w-md bg-white rounded-xl shadow-2xl border border-neutral-200 overflow-hidden"
				onClick={e => e.stopPropagation()}
			>
				{/* Search input */}
				<div className="px-3 py-2 border-b border-neutral-100">
					<input
						ref={inputRef}
						type="text"
						value={query}
						onChange={e => { setQuery(e.target.value); setSelectedIndex(0) }}
						onKeyDown={handleKeyDown}
						placeholder="Switch screen, layout, state..."
						className="w-full text-sm outline-none text-neutral-800 placeholder:text-neutral-400"
					/>
				</div>

				{/* Results */}
				<div className="max-h-[40vh] overflow-y-auto py-1">
					{[...groups.entries()].map(([group, groupItems]) => (
						<div key={group}>
							<div className="px-3 py-1 text-[10px] uppercase tracking-wider text-neutral-400 font-medium">
								{group}
							</div>
							{groupItems.map(item => {
								const idx = flatList.indexOf(item)
								const isSelected = idx === selectedIndex
								return (
									<button
										key={item.id}
										onClick={() => { item.onSelect(); onClose() }}
										onMouseEnter={() => setSelectedIndex(idx)}
										className={`w-full px-3 py-1.5 text-left text-sm flex items-center gap-2 ${
											isSelected ? 'bg-neutral-100' : ''
										}`}
									>
										{item.active && (
											<span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
										)}
										<span className={item.active ? 'font-medium text-neutral-900' : 'text-neutral-600'}>
											{item.label}
										</span>
									</button>
								)
							})}
						</div>
					))}

					{flatList.length === 0 && (
						<div className="px-3 py-4 text-sm text-neutral-400 text-center">
							No matches
						</div>
					)}
				</div>

				{/* Footer hint */}
				<div className="px-3 py-1.5 border-t border-neutral-100 text-[10px] text-neutral-400 flex gap-3">
					<span>↑↓ navigate</span>
					<span>↵ select</span>
					<span>esc close</span>
				</div>
			</div>
		</div>
	)
}

/** Tiny floating badge showing current context — click to open command bar */
export function ContextBadge({ screen, state, layout, onOpen }: {
	screen: string
	state?: string | null
	layout?: string | null
	onOpen: () => void
}) {
	return (
		<button
			onClick={onOpen}
			className="fixed top-3 left-1/2 -translate-x-1/2 z-40 flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/90 backdrop-blur shadow-md border border-neutral-200 text-[11px] text-neutral-600 hover:shadow-lg transition-shadow"
		>
			<span className="font-semibold text-neutral-900">{screen}</span>
			{state && (
				<>
					<span className="text-neutral-300">/</span>
					<span className="text-emerald-600">{state}</span>
				</>
			)}
			{layout && (
				<>
					<span className="text-neutral-300">/</span>
					<span>{layout}</span>
				</>
			)}
			<span className="text-neutral-400 ml-1">/</span>
		</button>
	)
}
