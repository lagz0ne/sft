import { useEffect, useMemo } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ArrowLeft, X } from "lucide-react";
import { useSpecContext } from "../context/spec-context";
import { useViewContext } from "../context/view-context";
import { useLightbox } from "./lightbox";
import { StateMachinePanel } from "./state-machine-panel";
import type { Region, Flow } from "../lib/types";

const IMAGE_EXTENSIONS = new Set([".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif", ".bmp"]);

function isImageFile(filename: string): boolean {
	const dot = filename.lastIndexOf(".");
	if (dot === -1) return false;
	return IMAGE_EXTENSIONS.has(filename.slice(dot).toLowerCase());
}

interface ScreenOverlayProps {
	screenName: string;
	highlightRegion?: string;
}

function collectRegionNames(regions: Region[]): string[] {
	const names: string[] = [];
	for (const r of regions) {
		names.push(r.name);
		if (r.regions) names.push(...collectRegionNames(r.regions));
	}
	return names;
}

function findMatchingFlows(
	flows: Flow[],
	screenName: string,
	regionNames: string[],
): Flow[] {
	const allNames = new Set([screenName, ...regionNames]);
	return flows.filter((f) => {
		const steps = f.sequence.split(/\s*→\s*/);
		return steps.some((step) => {
			const cleaned = step
				.replace(/\[.*?\]\s*/, "")
				.replace(/\(.*?\)/, "")
				.trim();
			return allNames.has(cleaned);
		});
	});
}

function RegionZone({
	region,
	screenName,
	highlightRegion,
	openLightbox,
	openStateMachine,
}: {
	region: Region;
	screenName: string;
	highlightRegion?: string;
	openLightbox: (src: string, caption: string) => void;
	openStateMachine: (regionName: string) => void;
}) {
	const isHighlighted = highlightRegion === region.name;
	const stateCount = region.transitions?.length ?? 0;

	return (
		<div
			className={`min-w-[200px] flex-1 rounded-lg border-2 border-dashed p-4 transition-colors ${
				isHighlighted
					? "border-blue-300 bg-blue-50/30 ring-1 ring-blue-200"
					: "border-neutral-200"
			} ${stateCount > 0 ? "cursor-pointer hover:border-neutral-300" : ""}`}
		>
			<div className="font-mono text-[13px] font-medium tracking-tight">{region.name}</div>

			{region.description && (
				<p className="mt-1 text-[12px] text-foreground/50 leading-relaxed">
					{region.description}
				</p>
			)}

			{region.events && region.events.length > 0 && (
				<div className="mt-2 flex flex-wrap gap-1.5">
					{region.events.map((e) => (
						<span
							key={e}
							className="rounded-full bg-emerald-50 px-2 py-0.5 font-mono text-[11px] text-emerald-600 border border-emerald-100"
						>
							{e}
						</span>
					))}
				</div>
			)}

			{stateCount > 0 && (
				<button
					type="button"
					className="mt-2 cursor-pointer font-mono text-[11px] text-blue-500 hover:text-blue-600 transition-colors"
					onClick={() => openStateMachine(region.name)}
				>
					{stateCount} transitions &middot; click to simulate &rarr;
				</button>
			)}

			{region.attachments && region.attachments.length > 0 && (
				<div className="mt-2 flex flex-wrap gap-2">
					{region.attachments.map((a) => {
						const src = `/a/${encodeURIComponent(screenName)}/${encodeURIComponent(a)}`;
						if (isImageFile(a)) {
							return (
								<img
									key={a}
									src={src}
									alt={a}
									loading="lazy"
									className="h-16 w-16 cursor-pointer rounded border border-neutral-200 object-cover"
									onClick={() => openLightbox(src, a)}
								/>
							);
						}
						return (
							<a
								key={a}
								href={src}
								target="_blank"
								className="flex items-center gap-2 rounded-lg border border-neutral-200 bg-neutral-50 px-3 py-2 font-mono text-[12px] text-foreground/60 hover:bg-neutral-100 transition-colors"
							>
								{a}
							</a>
						);
					})}
				</div>
			)}

			{region.regions && region.regions.length > 0 && (
				<div className="mt-3 ml-3 border-l border-neutral-200 pl-3 space-y-3">
					{region.regions.map((nested) => (
						<RegionZone
							key={nested.name}
							region={nested}
							screenName={screenName}
							highlightRegion={highlightRegion}
							openLightbox={openLightbox}
							openStateMachine={openStateMachine}
						/>
					))}
				</div>
			)}
		</div>
	);
}

export function ScreenOverlay({ screenName, highlightRegion }: ScreenOverlayProps) {
	const { spec } = useSpecContext();
	const { openFlow, openStateMachine, closePanel, goBack, goCanvas, panel } =
		useViewContext();
	const openLightbox = useLightbox();

	const screen = spec?.screens.find((s) => s.name === screenName);

	const regionNames = useMemo(
		() => (screen?.regions ? collectRegionNames(screen.regions) : []),
		[screen],
	);

	const matchingFlows = useMemo(
		() => findMatchingFlows(spec?.flows ?? [], screenName, regionNames),
		[spec?.flows, screenName, regionNames],
	);

	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === "Escape") {
				if (panel.type !== "none") {
					closePanel();
				} else {
					goBack();
				}
			}
		}
		document.addEventListener("keydown", handleKeyDown);
		return () => document.removeEventListener("keydown", handleKeyDown);
	}, [panel, closePanel, goBack]);

	if (!screen) return null;

	return (
		<motion.div
			initial={{ y: "100%" }}
			animate={{ y: 0 }}
			exit={{ y: "100%" }}
			transition={{ type: "spring", damping: 30, stiffness: 300 }}
			className="fixed inset-0 z-50 flex flex-col overflow-hidden bg-white"
		>
			{/* Header */}
			<div className="flex items-center justify-between bg-neutral-950 px-6 py-3 text-white">
				<div className="flex items-center gap-3">
					<button
						type="button"
						className="cursor-pointer text-white/40 hover:text-white transition-colors"
						onClick={goBack}
					>
						<ArrowLeft size={18} />
					</button>
					<span className="font-serif text-lg tracking-tight">{screen.name}</span>
					{screen.description && (
						<span className="text-[13px] font-sans text-white/40 line-clamp-1" title={screen.description}>{screen.description}</span>
					)}
				</div>
				<div className="flex items-center gap-2">
					{matchingFlows.map((f) => (
						<button
							key={f.name}
							type="button"
							className="cursor-pointer font-mono text-[11px] bg-white/8 border border-white/10 hover:bg-white/15 rounded-full px-2.5 py-0.5 transition-colors"
							onClick={() => openFlow(f.name)}
						>
							{f.name}
						</button>
					))}
					<button
						type="button"
						className="ml-2 cursor-pointer text-white/40 hover:text-white transition-colors"
						onClick={goCanvas}
					>
						<X size={18} />
					</button>
				</div>
			</div>

			{/* Scrollable body */}
			<div className="flex flex-1 overflow-hidden">
				<div
					className={`flex-1 min-w-0 overflow-y-auto ${
						panel.type === "state-machine" ? "" : "w-full"
					}`}
				>
					{/* Wireframe content */}
					<div className="mx-auto max-w-4xl px-8 py-6 bg-neutral-50/50">
						{/* Attachment hero */}
						{screen.attachments && screen.attachments.length > 0 && (() => {
							const imageAttachments = screen.attachments.filter(isImageFile);
							const nonImageAttachments = screen.attachments.filter((a) => !isImageFile(a));
							const hasImages = imageAttachments.length > 0;

							return (
								<div className="mb-6">
									{hasImages && (
										<>
											<img
												src={`/a/${encodeURIComponent(screenName)}/${encodeURIComponent(imageAttachments[0])}`}
												alt={imageAttachments[0]}
												className="max-h-80 cursor-pointer rounded-lg object-contain"
												onClick={() =>
													openLightbox(
														`/a/${encodeURIComponent(screenName)}/${encodeURIComponent(imageAttachments[0])}`,
														imageAttachments[0],
													)
												}
											/>
											{imageAttachments.length > 1 && (
												<div className="mt-3 flex flex-wrap gap-2">
													{imageAttachments.slice(1).map((a) => {
														const src = `/a/${encodeURIComponent(screenName)}/${encodeURIComponent(a)}`;
														return (
															<img
																key={a}
																src={src}
																alt={a}
																loading="lazy"
																className="h-16 w-16 cursor-pointer rounded border border-neutral-200 object-cover"
																onClick={() => openLightbox(src, a)}
															/>
														);
													})}
												</div>
											)}
										</>
									)}
									{nonImageAttachments.length > 0 && (
										<div className={`flex flex-wrap gap-2 ${hasImages ? "mt-3" : ""}`}>
											{nonImageAttachments.map((a) => {
												const src = `/a/${encodeURIComponent(screenName)}/${encodeURIComponent(a)}`;
												return (
													<a
														key={a}
														href={src}
														target="_blank"
														className="flex items-center gap-2 rounded-lg border border-neutral-200 bg-neutral-50 px-3 py-2 font-mono text-[12px] text-foreground/60 hover:bg-neutral-100 transition-colors"
													>
														{a}
													</a>
												);
											})}
										</div>
									)}
								</div>
							);
						})()}

						{/* Region zones */}
						{screen.regions && screen.regions.length > 0 && (
							<div className="mt-6 flex flex-wrap gap-4">
								{screen.regions.map((r) => (
									<RegionZone
										key={r.name}
										region={r}
										screenName={screenName}
										highlightRegion={highlightRegion}
										openLightbox={openLightbox}
										openStateMachine={openStateMachine}
									/>
								))}
							</div>
						)}
					</div>
				</div>

				{/* State machine panel */}
				<AnimatePresence>
					{panel.type === "state-machine" && (
						<motion.div
							initial={{ width: 0, opacity: 0 }}
							animate={{ width: 400, opacity: 1 }}
							exit={{ width: 0, opacity: 0 }}
							transition={{ type: "spring", damping: 30, stiffness: 300 }}
							className="overflow-hidden border-l border-neutral-200"
						>
							<div className="w-[400px]">
								<StateMachinePanel
									regionName={panel.regionName}
									onClose={closePanel}
								/>
							</div>
						</motion.div>
					)}
				</AnimatePresence>
			</div>
		</motion.div>
	);
}
