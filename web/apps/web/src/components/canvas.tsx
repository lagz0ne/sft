import { useState, useMemo, useCallback } from "react";
import {
	ReactFlow,
	Background,
	type Node,
	type Edge,
	type NodeProps,
	type EdgeProps,
	Handle,
	Position,
	BaseEdge,
	getBezierPath,
	EdgeLabelRenderer,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { cn } from "@sft-web/ui/lib/utils";
import { useSpecContext } from "../context/spec-context";
import { useViewContext } from "../context/view-context";
import type { Screen, Region, Flow } from "../lib/types";
import Loader from "./loader";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function countRegions(regions?: Region[]): number {
	if (!regions) return 0;
	return regions.reduce((sum, r) => sum + 1 + countRegions(r.regions), 0);
}

function countEvents(regions?: Region[]): number {
	if (!regions) return 0;
	return regions.reduce(
		(sum, r) => sum + (r.events?.length ?? 0) + countEvents(r.regions),
		0,
	);
}

interface ScreenEdge {
	from: string;
	to: string;
	flowName: string;
}

function parseFlowEdges(flows: Flow[], screenNames: Set<string>): ScreenEdge[] {
	const edges: ScreenEdge[] = [];
	const seen = new Set<string>();

	for (const flow of flows) {
		const steps = flow.sequence.split(/\s*→\s*/).map((s) =>
			s
				.replace(/\[.*?\]\s*/, "")
				.replace(/\(.*?\)/, "")
				.trim(),
		);

		let lastScreen: string | null = null;
		for (const step of steps) {
			if (screenNames.has(step)) {
				if (lastScreen && lastScreen !== step) {
					const key = `${lastScreen}→${step}→${flow.name}`;
					if (!seen.has(key)) {
						seen.add(key);
						edges.push({ from: lastScreen, to: step, flowName: flow.name });
					}
				}
				lastScreen = step;
			}
		}
	}

	return edges;
}

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

const H_GAP = 260;
const V_GAP = 140;

function layoutNodes(
	screens: Screen[],
	flowEdges: ScreenEdge[],
): Map<string, { x: number; y: number }> {
	const positions = new Map<string, { x: number; y: number }>();

	if (flowEdges.length === 0) {
		// Grid fallback: 3 columns
		const cols = 3;
		screens.forEach((s, i) => {
			const col = i % cols;
			const row = Math.floor(i / cols);
			positions.set(s.name, { x: col * H_GAP, y: row * V_GAP });
		});
		return positions;
	}

	// Build adjacency for BFS
	const incoming = new Map<string, Set<string>>();
	const outgoing = new Map<string, Set<string>>();
	const allNames = new Set(screens.map((s) => s.name));

	for (const name of allNames) {
		incoming.set(name, new Set());
		outgoing.set(name, new Set());
	}

	for (const e of flowEdges) {
		outgoing.get(e.from)?.add(e.to);
		incoming.get(e.to)?.add(e.from);
	}

	// Find roots (no incoming)
	const roots = [...allNames].filter((n) => incoming.get(n)!.size === 0);
	if (roots.length === 0) roots.push(screens[0].name); // cycle fallback

	// BFS to assign layers
	const layers = new Map<string, number>();
	const queue: string[] = [...roots];
	for (const r of roots) layers.set(r, 0);

	while (queue.length > 0) {
		const current = queue.shift()!;
		const currentLayer = layers.get(current)!;
		for (const next of outgoing.get(current) ?? []) {
			if (!layers.has(next) || layers.get(next)! < currentLayer + 1) {
				layers.set(next, currentLayer + 1);
				queue.push(next);
			}
		}
	}

	// Assign unvisited screens to layer 0
	for (const s of screens) {
		if (!layers.has(s.name)) layers.set(s.name, 0);
	}

	// Group by layer
	const layerGroups = new Map<number, string[]>();
	for (const [name, layer] of layers) {
		if (!layerGroups.has(layer)) layerGroups.set(layer, []);
		layerGroups.get(layer)!.push(name);
	}

	// Find tallest layer for vertical centering
	let maxLayerSize = 0;
	for (const names of layerGroups.values()) {
		if (names.length > maxLayerSize) maxLayerSize = names.length;
	}
	const totalMaxHeight = (maxLayerSize - 1) * V_GAP;

	// Position: horizontal layers (left-to-right), centered vertically
	for (const [layer, names] of layerGroups) {
		const totalHeight = (names.length - 1) * V_GAP;
		const startY = (totalMaxHeight - totalHeight) / 2;
		names.forEach((name, idx) => {
			positions.set(name, { x: layer * H_GAP, y: startY + idx * V_GAP });
		});
	}

	return positions;
}

// ---------------------------------------------------------------------------
// Reachability for blast radius
// ---------------------------------------------------------------------------

function findReachable(
	startId: string,
	flowEdges: ScreenEdge[],
): Set<string> {
	const reachable = new Set<string>();
	const queue = [startId];
	reachable.add(startId);

	while (queue.length > 0) {
		const current = queue.shift()!;
		for (const e of flowEdges) {
			if (e.from === current && !reachable.has(e.to)) {
				reachable.add(e.to);
				queue.push(e.to);
			}
		}
	}

	return reachable;
}

// ---------------------------------------------------------------------------
// Custom Node
// ---------------------------------------------------------------------------

type ScreenNodeData = {
	screenName: string;
	regionCount: number;
	hovered: boolean;
	dimmed: boolean;
	blastRadius: boolean;
};

function ScreenNode({ data }: NodeProps<Node<ScreenNodeData>>) {
	return (
		<div
			className={cn(
				"w-[160px] cursor-pointer rounded-lg border border-border bg-white px-3 py-2 shadow-sm transition-all duration-200",
				data.hovered && "shadow-md border-foreground/30 scale-[1.02]",
				data.dimmed && "opacity-20",
				data.blastRadius &&
					"bg-red-50/80 border-red-300 shadow-[0_0_12px_rgba(239,68,68,0.15)]",
			)}
		>
			<Handle type="target" position={Position.Left} className="!invisible" />
			<div className="flex items-start justify-between gap-1">
				<div className="truncate font-mono text-[13px] font-medium tracking-tight">
					{data.screenName}
				</div>
				<span className="shrink-0 text-foreground/20 text-[11px] leading-[18px]">
					◎
				</span>
			</div>
			{data.regionCount > 0 && (
				<div className="font-mono text-[11px] text-muted-foreground">
					<span className="text-foreground/20 mr-1">·</span>
					{data.regionCount} region{data.regionCount !== 1 ? "s" : ""}
				</div>
			)}
			<Handle type="source" position={Position.Right} className="!invisible" />
		</div>
	);
}

// ---------------------------------------------------------------------------
// Custom Edge
// ---------------------------------------------------------------------------

type FlowEdgeData = {
	flowName: string;
	dimmed: boolean;
};

function FlowEdge({
	id,
	sourceX,
	sourceY,
	targetX,
	targetY,
	sourcePosition,
	targetPosition,
	data,
}: EdgeProps<Edge<FlowEdgeData>>) {
	const [hovered, setHovered] = useState(false);
	const [edgePath, labelX, labelY] = getBezierPath({
		sourceX,
		sourceY,
		sourcePosition,
		targetX,
		targetY,
		targetPosition,
	});

	return (
		<>
			<BaseEdge
				id={id}
				path={edgePath}
				style={{
					stroke: data?.dimmed
						? "rgba(196,196,196,0.2)"
						: hovered
							? "#888"
							: "#c4c4c4",
					strokeWidth: 1.5,
					strokeDasharray: hovered ? "6 3" : "none",
					animation: hovered ? "dash 0.5s linear infinite" : "none",
				}}
			/>
			<EdgeLabelRenderer>
				<button
					type="button"
					className={cn(
						"nodrag nopan absolute cursor-pointer rounded-full bg-white/90 backdrop-blur-sm px-2 py-0.5 font-mono text-[11px] tracking-tight text-muted-foreground shadow-sm transition-all hover:text-foreground",
						hovered && "underline decoration-foreground/30 underline-offset-2",
						data?.dimmed && "opacity-20",
					)}
					style={{
						transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
						pointerEvents: "all",
					}}
					onMouseEnter={() => setHovered(true)}
					onMouseLeave={() => setHovered(false)}
				>
					{data?.flowName}
				</button>
			</EdgeLabelRenderer>
		</>
	);
}

// ---------------------------------------------------------------------------
// Stable type registries (outside component to avoid re-render warnings)
// ---------------------------------------------------------------------------

const nodeTypes = { screen: ScreenNode };
const edgeTypes = { flow: FlowEdge };

// ---------------------------------------------------------------------------
// Canvas Component
// ---------------------------------------------------------------------------

export default function Canvas() {
	const { spec, loading, error } = useSpecContext();
	const { openScreen, openFlow } = useViewContext();

	const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
	const [blastRadiusNodeId, setBlastRadiusNodeId] = useState<string | null>(
		null,
	);

	// Derive flow edges
	const flowEdges = useMemo(() => {
		if (!spec) return [];
		const screenNames = new Set(spec.screens.map((s) => s.name));
		return parseFlowEdges(spec.flows ?? [], screenNames);
	}, [spec]);

	// Blast radius set
	const blastRadiusSet = useMemo(() => {
		if (!blastRadiusNodeId) return new Set<string>();
		return findReachable(blastRadiusNodeId, flowEdges);
	}, [blastRadiusNodeId, flowEdges]);

	// Connected nodes/edges for hover highlighting
	const connectedMap = useMemo(() => {
		const map = new Map<string, Set<string>>();
		for (const e of flowEdges) {
			if (!map.has(e.from)) map.set(e.from, new Set());
			if (!map.has(e.to)) map.set(e.to, new Set());
			map.get(e.from)!.add(e.to);
			map.get(e.to)!.add(e.from);
		}
		return map;
	}, [flowEdges]);

	const connectedEdgeIds = useMemo(() => {
		if (!hoveredNodeId) return new Set<string>();
		const ids = new Set<string>();
		flowEdges.forEach((e, i) => {
			if (e.from === hoveredNodeId || e.to === hoveredNodeId) {
				ids.add(`e-${i}`);
			}
		});
		return ids;
	}, [hoveredNodeId, flowEdges]);

	// Build React Flow nodes & edges
	const { nodes, edges } = useMemo(() => {
		if (!spec) return { nodes: [] as Node[], edges: [] as Edge[] };

		const positions = layoutNodes(spec.screens, flowEdges);

		const connectedToHovered = hoveredNodeId
			? connectedMap.get(hoveredNodeId) ?? new Set<string>()
			: null;

		const rfNodes: Node[] = spec.screens.map((screen) => {
			const pos = positions.get(screen.name) ?? { x: 0, y: 0 };
			const isHovered = hoveredNodeId === screen.name;
			const dimmed =
				hoveredNodeId !== null &&
				!isHovered &&
				!(connectedToHovered?.has(screen.name) ?? false);

			return {
				id: screen.name,
				type: "screen",
				position: pos,
				data: {
					screenName: screen.name,
					regionCount: countRegions(screen.regions),
					hovered: isHovered,
					dimmed: blastRadiusNodeId ? !blastRadiusSet.has(screen.name) : dimmed,
					blastRadius:
						blastRadiusNodeId !== null &&
						blastRadiusSet.has(screen.name) &&
						screen.name !== blastRadiusNodeId,
				},
			};
		});

		const rfEdges: Edge[] = flowEdges.map((fe, i) => {
			const edgeId = `e-${i}`;
			const dimmedByHover =
				hoveredNodeId !== null && !connectedEdgeIds.has(edgeId);
			const dimmedByBlast =
				blastRadiusNodeId !== null &&
				(!blastRadiusSet.has(fe.from) || !blastRadiusSet.has(fe.to));

			return {
				id: edgeId,
				source: fe.from,
				target: fe.to,
				type: "flow",
				data: {
					flowName: fe.flowName,
					dimmed: blastRadiusNodeId ? dimmedByBlast : dimmedByHover,
				},
			};
		});

		return { nodes: rfNodes, edges: rfEdges };
	}, [
		spec,
		flowEdges,
		hoveredNodeId,
		connectedMap,
		connectedEdgeIds,
		blastRadiusNodeId,
		blastRadiusSet,
	]);

	// Handlers
	const handleNodeClick = useCallback(
		(_: React.MouseEvent, node: Node) => {
			openScreen(node.id);
		},
		[openScreen],
	);

	const handleEdgeClick = useCallback(
		(_: React.MouseEvent, edge: Edge) => {
			const flowName = (edge.data as FlowEdgeData | undefined)?.flowName;
			if (flowName) openFlow(flowName);
		},
		[openFlow],
	);

	const handleNodeMouseEnter = useCallback(
		(_: React.MouseEvent, node: Node) => {
			setHoveredNodeId(node.id);
		},
		[],
	);

	const handleNodeMouseLeave = useCallback(() => {
		setHoveredNodeId(null);
	}, []);

	const handleNodeContextMenu = useCallback(
		(e: React.MouseEvent, node: Node) => {
			e.preventDefault();
			setBlastRadiusNodeId((prev) => (prev === node.id ? null : node.id));
		},
		[],
	);

	const handlePaneClick = useCallback(() => {
		setBlastRadiusNodeId(null);
	}, []);

	// Loading / Error states
	if (loading)
		return (
			<div className="flex h-full w-full items-center justify-center">
				<Loader />
			</div>
		);

	if (error) {
		return (
			<div className="flex h-full items-center justify-center">
				<p className="text-sm text-destructive">{error.message}</p>
			</div>
		);
	}

	if (!spec) return null;

	const totalEvents = spec.screens.reduce(
		(sum, s) => sum + countEvents(s.regions),
		0,
	);

	return (
		<div className="relative h-full w-full">
			{/* React Flow Canvas */}
			<ReactFlow
				nodes={nodes}
				edges={edges}
				nodeTypes={nodeTypes}
				edgeTypes={edgeTypes}
				onNodeClick={handleNodeClick}
				onEdgeClick={handleEdgeClick}
				onNodeMouseEnter={handleNodeMouseEnter}
				onNodeMouseLeave={handleNodeMouseLeave}
				onNodeContextMenu={handleNodeContextMenu}
				onPaneClick={handlePaneClick}
				fitView
				nodesDraggable={false}
				nodesConnectable={false}
				proOptions={{ hideAttribution: true }}
			>
				<Background variant={"dots" as any} gap={20} size={1} color="#e8e8e8" />
			</ReactFlow>

			{/* App header overlay */}
			<div className="pointer-events-none absolute top-4 left-4 z-10">
				<div className="pointer-events-auto bg-white/80 backdrop-blur-sm rounded-lg px-4 py-3">
					<h1 className="font-serif text-xl tracking-tight">{spec.app.name}</h1>
					{spec.app.description && (
						<p className="text-sm text-foreground/50 max-w-md leading-relaxed">
							{spec.app.description}
						</p>
					)}
					<p className="mt-1 font-mono text-[11px] text-foreground/40 tracking-wide uppercase">
						{spec.screens.length} screen
						{spec.screens.length !== 1 ? "s" : ""}
						{" \u00b7 "}
						{spec.flows?.length ?? 0} flow
						{(spec.flows?.length ?? 0) !== 1 ? "s" : ""}
						{" \u00b7 "}
						{totalEvents} event{totalEvents !== 1 ? "s" : ""}
					</p>
				</div>
			</div>

			{/* Interaction hints */}
			<div className="pointer-events-none absolute right-4 bottom-4 z-10">
				<p className="font-mono text-[10px] tracking-widest uppercase text-foreground/25 bg-white/60 backdrop-blur-sm rounded-full px-3 py-1">
					click · hover · right-click
				</p>
			</div>

			{/* Animated dash keyframes */}
			<style>{`
				@keyframes dash {
					to { stroke-dashoffset: -9; }
				}
			`}</style>
		</div>
	);
}
