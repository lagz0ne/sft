import { createContext, useContext, type ReactNode } from "react";
import { useSpec } from "../hooks/use-spec";
import type { Spec, RenderSpec } from "../lib/types";

interface SpecContextValue {
	spec: Spec | null;
	renderSpec: RenderSpec | null;
	error: Error | null;
	loading: boolean;
	refresh: () => Promise<void>;
}

const SpecContext = createContext<SpecContextValue | null>(null);

export function SpecProvider({ children }: { children: ReactNode }) {
	const value = useSpec();
	return <SpecContext.Provider value={value}>{children}</SpecContext.Provider>;
}

export function useSpecContext(): SpecContextValue {
	const ctx = useContext(SpecContext);
	if (!ctx) throw new Error("useSpecContext must be used within SpecProvider");
	return ctx;
}
