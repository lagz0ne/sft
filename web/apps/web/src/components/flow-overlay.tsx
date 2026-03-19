import { useEffect, useMemo } from "react";
import { motion } from "motion/react";
import { ArrowLeft, X } from "lucide-react";
import { useSpecContext } from "../context/spec-context";
import { useViewContext } from "../context/view-context";
import type { Region, Screen } from "../lib/types";

interface FlowOverlayProps {
	flowName: string;
}

function cleanStep(s: string): string {
	return s
		.replace(/\[.*?\]\s*/, "")
		.replace(/\(.*?\)/, "")
		.trim();
}

function buildRegionScreenMap(screens: Screen[]): Map<string, string> {
	const map = new Map<string, string>();

	function walkRegions(regions: Region[], screenName: string) {
		for (const region of regions) {
			map.set(region.name, screenName);
			if (region.regions) {
				walkRegions(region.regions, screenName);
			}
		}
	}

	for (const screen of screens) {
		if (screen.regions) {
			walkRegions(screen.regions, screen.name);
		}
	}

	return map;
}

export function FlowOverlay({ flowName }: FlowOverlayProps) {
	const { spec } = useSpecContext();
	const { openScreen, goBack, goCanvas } = useViewContext();

	const flow = spec?.flows?.find((f) => f.name === flowName);

	const screenNames = useMemo(() => {
		if (!spec) return new Set<string>();
		return new Set(spec.screens.map((s) => s.name));
	}, [spec]);

	const regionScreenMap = useMemo(() => {
		if (!spec) return new Map<string, string>();
		return buildRegionScreenMap(spec.screens);
	}, [spec]);

	const steps = useMemo(() => {
		if (!flow) return [];
		return flow.sequence.split(/\s*→\s*/).map((raw) => {
			const name = cleanStep(raw);
			const isScreen = screenNames.has(name);
			const isRegion = regionScreenMap.has(name);
			const screenName = isScreen
				? name
				: isRegion
					? regionScreenMap.get(name)!
					: undefined;
			return { raw, name, isScreen, isRegion, screenName };
		});
	}, [flow, screenNames, regionScreenMap]);

	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === "Escape") goBack();
		}
		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [goBack]);

	if (!flow) {
		return (
			<motion.div
				className="fixed inset-0 z-50 flex flex-col bg-background"
				initial={{ y: "100%" }}
				animate={{ y: 0 }}
				exit={{ y: "100%" }}
				transition={{ type: "spring", damping: 30, stiffness: 300 }}
			>
				<div className="flex items-center gap-3 bg-orange-600 px-6 py-3 text-white">
					<button type="button" onClick={goBack} className="cursor-pointer text-white/40 hover:text-white transition-colors">
						<ArrowLeft className="h-5 w-5" />
					</button>
					<span className="font-serif text-lg tracking-tight">Flow not found</span>
				</div>
				<div className="flex flex-1 items-center justify-center">
					<div className="text-center">
						<p className="text-muted-foreground">
							Flow "{flowName}" was not found in this spec.
						</p>
						<button
							type="button"
							onClick={goBack}
							className="mt-4 cursor-pointer text-sm text-blue-600 hover:underline"
						>
							Go back
						</button>
					</div>
				</div>
			</motion.div>
		);
	}

	return (
		<motion.div
			className="fixed inset-0 z-50 flex flex-col bg-background"
			initial={{ y: "100%" }}
			animate={{ y: 0 }}
			exit={{ y: "100%" }}
			transition={{ type: "spring", damping: 30, stiffness: 300 }}
		>
			{/* Header */}
			<div className="flex items-center justify-between bg-orange-600 px-6 py-3 text-white">
				<div className="flex items-center gap-3">
					<button type="button" onClick={goBack} className="cursor-pointer text-white/40 hover:text-white transition-colors">
						<ArrowLeft className="h-5 w-5" />
					</button>
					<span className="font-serif text-lg tracking-tight">{flow.name}</span>
					{flow.description && (
						<span className="text-[13px] text-white/50">{flow.description}</span>
					)}
				</div>
				<div className="flex items-center gap-4">
					{flow.on_event && (
						<span className="font-mono text-[11px] text-white/40 bg-white/10 rounded-full px-2.5 py-0.5">
							triggered by: {flow.on_event}
						</span>
					)}
					<button type="button" onClick={goCanvas} className="cursor-pointer text-white/40 hover:text-white transition-colors">
						<X className="h-5 w-5" />
					</button>
				</div>
			</div>

			{/* Body */}
			<div className="flex-1 overflow-y-auto px-8 py-6">
				<div className="mx-auto max-w-lg">
					{steps.map((step, i) => (
						<div key={i}>
							<div className="flex items-start">
								{/* Circle badge */}
								<div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-neutral-950 text-[11px] font-bold text-white font-mono">
									{i + 1}
								</div>

								{/* Card */}
								<div
									className={`ml-3 flex flex-1 items-center justify-between bg-white shadow-sm rounded-lg px-4 py-3 transition-shadow ${
										step.screenName
											? "cursor-pointer hover:shadow-md"
											: "opacity-60"
									}`}
									onClick={() => {
										if (step.screenName) {
											openScreen(
												step.screenName,
												step.isRegion ? step.name : undefined,
											);
										}
									}}
								>
									<div>
										<div className="font-mono text-[13px] font-medium tracking-tight">{step.name}</div>
										{step.screenName && (
											<div className="font-sans text-[11px] text-foreground/40">
												in {step.screenName}
											</div>
										)}
									</div>
									{i < steps.length - 1 && step.screenName && (
										<span className="font-mono text-foreground/20">&rarr;</span>
									)}
								</div>
							</div>

							{/* Connector line */}
							{i < steps.length - 1 && (
								<div className="w-px h-3 bg-neutral-150 ml-[14px]" />
							)}
						</div>
					))}
				</div>
			</div>
		</motion.div>
	);
}
