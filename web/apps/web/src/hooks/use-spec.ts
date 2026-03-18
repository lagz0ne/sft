import { useCallback, useEffect, useRef, useState } from "react";
import { connectNats, requestSpec, requestRender, subscribeChanges } from "../lib/nats";
import type { Spec, RenderSpec } from "../lib/types";

interface SpecState {
	spec: Spec | null;
	renderSpec: RenderSpec | null;
	error: Error | null;
	loading: boolean;
}

export function useSpec(): SpecState & { refresh: () => Promise<void> } {
	const [state, setState] = useState<SpecState>({
		spec: null,
		renderSpec: null,
		error: null,
		loading: true,
	});
	const connectedRef = useRef(false);

	const fetchData = useCallback(async () => {
		try {
			const [spec, renderSpec] = await Promise.all([requestSpec(), requestRender()]);
			setState({ spec, renderSpec, error: null, loading: false });
		} catch (err) {
			setState((prev: SpecState) => ({
				...prev,
				error: err instanceof Error ? err : new Error(String(err)),
				loading: false,
			}));
		}
	}, []);

	useEffect(() => {
		if (connectedRef.current) return;
		connectedRef.current = true;

		connectNats()
			.then(() => {
				fetchData();
				subscribeChanges(() => {
					fetchData();
				});
			})
			.catch((err: unknown) => {
				setState({
					spec: null,
					renderSpec: null,
					error: err instanceof Error ? err : new Error(String(err)),
					loading: false,
				});
			});
	}, [fetchData]);

	return { ...state, refresh: fetchData };
}
