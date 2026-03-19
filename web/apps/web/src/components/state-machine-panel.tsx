import { useState, useEffect, useCallback, useMemo } from "react";
import { motion, AnimatePresence } from "motion/react";
import { X } from "lucide-react";
import { useSpecContext } from "../context/spec-context";
import type { Region, Transition } from "../lib/types";

interface StateMachinePanelProps {
	regionName: string;
	onClose: () => void;
}

const triggerColors = [
	{ bg: "bg-emerald-50", text: "text-emerald-700", border: "border-emerald-200" },
	{ bg: "bg-blue-50", text: "text-blue-700", border: "border-blue-200" },
	{ bg: "bg-orange-50", text: "text-orange-700", border: "border-orange-200" },
	{ bg: "bg-purple-50", text: "text-purple-700", border: "border-purple-200" },
];

function findRegion(regions: Region[] | undefined, name: string): Region | null {
	if (!regions) return null;
	for (const r of regions) {
		if (r.name === name) return r;
		const found = findRegion(r.regions, name);
		if (found) return found;
	}
	return null;
}

export function StateMachinePanel({ regionName, onClose }: StateMachinePanelProps) {
	const { spec } = useSpecContext();

	const region = useMemo(() => {
		if (!spec) return null;
		for (const screen of spec.screens) {
			const found = findRegion(screen.regions, regionName);
			if (found) return found;
		}
		return null;
	}, [spec, regionName]);

	const transitions = region?.transitions ?? [];
	const events = region?.events ?? [];

	const initialState = transitions[0]?.from_state || "initial";

	const [currentState, setCurrentState] = useState(initialState);
	const [history, setHistory] = useState<Array<{ from: string; trigger: string; to: string }>>([]);

	// Reset when region changes
	useEffect(() => {
		const first = transitions[0]?.from_state || "initial";
		setCurrentState(first);
		setHistory([]);
	}, [regionName]); // eslint-disable-line react-hooks/exhaustive-deps

	// Keyboard: Escape → close
	useEffect(() => {
		function handleKey(e: KeyboardEvent) {
			if (e.key === "Escape") onClose();
		}
		window.addEventListener("keydown", handleKey);
		return () => window.removeEventListener("keydown", handleKey);
	}, [onClose]);

	const allStates = useMemo(() => {
		const states = new Set<string>();
		for (const t of transitions) {
			if (t.from_state) states.add(t.from_state);
			if (t.to_state) states.add(t.to_state);
		}
		return Array.from(states);
	}, [transitions]);

	const handleTrigger = useCallback(
		(event: string) => {
			const match = transitions.find(
				(t) => t.on_event === event && (!t.from_state || t.from_state === currentState),
			);
			if (!match) return;

			const to = match.to_state || currentState;
			setHistory((prev) => [{ from: currentState, trigger: event, to }, ...prev].slice(0, 10));
			setCurrentState(to);
		},
		[transitions, currentState],
	);

	const isEnabled = useCallback(
		(event: string) =>
			transitions.some(
				(t: Transition) =>
					t.on_event === event && (!t.from_state || t.from_state === currentState),
			),
		[transitions, currentState],
	);

	return (
		<div className="h-full flex flex-col border-l bg-neutral-50">
			{/* Header */}
			<div className="px-4 py-3 border-b flex items-center justify-between">
				<div>
					<div className="font-mono text-[10px] text-foreground/30 uppercase tracking-[0.15em]">
						State Machine
					</div>
					<div className="font-mono text-[14px] font-medium tracking-tight">{regionName}</div>
				</div>
				<button onClick={onClose} className="cursor-pointer text-foreground/40 hover:text-foreground transition-colors">
					<X className="w-5 h-5" />
				</button>
			</div>

			{/* Current state */}
			<div className="px-4 py-2">
				<span className="font-mono text-[12px] bg-emerald-50 text-emerald-600 border border-emerald-100 rounded-full px-3 py-1">
					● {currentState}
				</span>
			</div>

			{/* State pills */}
			<div className="px-4 py-2 flex flex-wrap gap-2">
				{allStates.map((state) => (
					<motion.div
						layout
						key={state === currentState ? `active-${currentState}` : state}
						initial={state === currentState ? { scale: 1.2 } : false}
						animate={{ scale: 1 }}
						className={`px-3 py-1 rounded-full font-mono text-[11px] border ${
							state === currentState
								? "border-blue-400 bg-blue-50 text-blue-600 shadow-sm"
								: "border-neutral-150 bg-white text-foreground/40"
						}`}
					>
						{state}
					</motion.div>
				))}
			</div>

			{/* Trigger buttons */}
			<div className="px-4 py-3 flex flex-wrap gap-2">
				{events.map((event, i) => {
					const color = triggerColors[i % triggerColors.length];
					const enabled = isEnabled(event);
					return (
						<motion.button
							key={event}
							whileTap={enabled ? { scale: 0.95 } : undefined}
							disabled={!enabled}
							onClick={() => handleTrigger(event)}
							className={`px-3 py-1.5 rounded-full font-mono text-[11px] font-medium border cursor-pointer shadow-sm hover:shadow-md transition-shadow ${color.bg} ${color.text} ${color.border} ${!enabled ? "opacity-40 cursor-not-allowed shadow-none hover:shadow-none" : ""}`}
						>
							▶ {event}
						</motion.button>
					);
				})}
			</div>

			{/* History */}
			<div className="flex-1 overflow-y-auto px-4 py-2">
				<div className="font-mono text-[10px] text-foreground/25 uppercase tracking-[0.15em] mb-2">Transition history</div>
				<div className="bg-white rounded-lg p-2">
					<AnimatePresence>
						{history.map((entry, i) => (
							<motion.div
								key={`${entry.from}-${entry.trigger}-${entry.to}-${history.length - i}`}
								initial={{ opacity: 0, y: -10 }}
								animate={{ opacity: 1, y: 0 }}
								className="font-mono text-[11px] text-foreground/60 py-0.5"
							>
								{entry.from} →{entry.trigger}→ {entry.to}
							</motion.div>
						))}
					</AnimatePresence>
				</div>
			</div>
		</div>
	);
}
