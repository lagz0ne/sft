import { createFileRoute } from "@tanstack/react-router";
import { AnimatePresence } from "motion/react";

import Canvas from "../components/canvas";
import { ScreenOverlay } from "../components/screen-overlay";
import { FlowOverlay } from "../components/flow-overlay";
import { useViewContext } from "../context/view-context";

export const Route = createFileRoute("/")({
	component: CanvasPage,
});

function CanvasPage() {
	const { view } = useViewContext();

	return (
		<div className="h-svh w-svw">
			<Canvas />
			<AnimatePresence>
				{view.type === "screen" && (
					<ScreenOverlay
						key={`screen-${view.screenName}`}
						screenName={view.screenName}
						highlightRegion={view.highlightRegion}
					/>
				)}
				{view.type === "flow" && (
					<FlowOverlay key={`flow-${view.flowName}`} flowName={view.flowName} />
				)}
			</AnimatePresence>
		</div>
	);
}
