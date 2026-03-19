import { createContext, useCallback, useContext, useState, type ReactNode } from "react";
import type { ViewState, PanelState } from "../lib/types";

interface ViewContextValue {
	view: ViewState;
	panel: PanelState;
	openScreen: (screenName: string, highlightRegion?: string) => void;
	openFlow: (flowName: string) => void;
	openStateMachine: (regionName: string) => void;
	closePanel: () => void;
	goBack: () => void;
	goCanvas: () => void;
}

const ViewContext = createContext<ViewContextValue | null>(null);

export function ViewProvider({ children }: { children: ReactNode }) {
	const [view, setView] = useState<ViewState>({ type: "canvas" });
	const [panel, setPanel] = useState<PanelState>({ type: "none" });
	const [_history, setHistory] = useState<ViewState[]>([]);

	const openScreen = useCallback(
		(screenName: string, highlightRegion?: string) => {
			setHistory((h) => [...h, view]);
			setView({ type: "screen", screenName, highlightRegion });
			setPanel({ type: "none" });
		},
		[view],
	);

	const openFlow = useCallback(
		(flowName: string) => {
			setHistory((h) => [...h, view]);
			setView({ type: "flow", flowName });
			setPanel({ type: "none" });
		},
		[view],
	);

	const openStateMachine = useCallback((regionName: string) => {
		setPanel({ type: "state-machine", regionName });
	}, []);

	const closePanel = useCallback(() => {
		setPanel({ type: "none" });
	}, []);

	const goBack = useCallback(() => {
		setHistory((h) => {
			const prev = h[h.length - 1];
			if (prev) {
				setView(prev);
				setPanel({ type: "none" });
				return h.slice(0, -1);
			}
			setView({ type: "canvas" });
			setPanel({ type: "none" });
			return [];
		});
	}, []);

	const goCanvas = useCallback(() => {
		setView({ type: "canvas" });
		setPanel({ type: "none" });
		setHistory([]);
	}, []);

	return (
		<ViewContext.Provider
			value={{ view, panel, openScreen, openFlow, openStateMachine, closePanel, goBack, goCanvas }}
		>
			{children}
		</ViewContext.Provider>
	);
}

export function useViewContext(): ViewContextValue {
	const ctx = useContext(ViewContext);
	if (!ctx) throw new Error("useViewContext must be used within ViewProvider");
	return ctx;
}
