import { useCallback, useEffect, useState } from "react";

interface LightboxState {
	src: string;
	caption: string;
}

let showFn: ((src: string, caption: string) => void) | null = null;

export function useLightbox() {
	return useCallback((src: string, caption: string) => {
		showFn?.(src, caption);
	}, []);
}

export function Lightbox() {
	const [state, setState] = useState<LightboxState | null>(null);

	useEffect(() => {
		showFn = (src, caption) => setState({ src, caption });
		return () => {
			showFn = null;
		};
	}, []);

	useEffect(() => {
		if (!state) return;
		const handler = (e: KeyboardEvent) => {
			if (e.key === "Escape") setState(null);
		};
		document.addEventListener("keydown", handler);
		return () => document.removeEventListener("keydown", handler);
	}, [state]);

	if (!state) return null;

	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center"
			onClick={() => setState(null)}
		>
			<div className="absolute inset-0 bg-black/70 backdrop-blur-sm" />
			<div className="relative flex max-h-[90vh] max-w-[90vw] flex-col items-center gap-2.5">
				<img
					src={state.src}
					alt={state.caption}
					className="max-h-[85vh] max-w-[90vw] rounded-lg object-contain shadow-2xl"
				/>
				<span className="font-mono text-xs text-white/70">{state.caption}</span>
			</div>
		</div>
	);
}
