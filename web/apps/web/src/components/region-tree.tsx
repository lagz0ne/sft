import type { Region, RenderSpec } from "../lib/types";

interface RegionTreeProps {
	regions: Region[];
	renderSpec: RenderSpec | null;
	onImageClick?: (src: string, caption: string) => void;
}

export function RegionTree({ regions, renderSpec, onImageClick }: RegionTreeProps) {
	return (
		<div className="space-y-2.5">
			{regions.map((r) => (
				<RegionCard key={r.name} region={r} renderSpec={renderSpec} onImageClick={onImageClick} />
			))}
		</div>
	);
}

function RegionCard({
	region,
	renderSpec,
	onImageClick,
}: {
	region: Region;
	renderSpec: RenderSpec | null;
	onImageClick?: (src: string, caption: string) => void;
}) {
	const renderEl = renderSpec?.elements?.[region.name];
	const compType = renderEl?.type !== "Stack" ? renderEl?.type : undefined;

	return (
		<div className="rounded-lg border border-border bg-background p-4">
			<div className="mb-1.5 flex items-center gap-2.5">
				<span className="text-base font-semibold leading-snug">{region.name}</span>
				{compType && (
					<span className="rounded bg-purple-50 px-2 py-0.5 font-mono text-xs text-purple-700">
						{String(compType)}
					</span>
				)}
			</div>

			{region.description && (
				<p className="mb-3 max-w-[65ch] leading-relaxed text-muted-foreground">
					{region.description}
				</p>
			)}

			{region.events && region.events.length > 0 && (
				<div className="mb-2.5 flex flex-wrap gap-1.5">
					{region.events.map((e) => (
						<span
							key={e}
							className="rounded bg-green-50 px-2.5 py-0.5 font-mono text-xs text-green-700"
						>
							{e}
						</span>
					))}
				</div>
			)}

			{region.transitions && region.transitions.length > 0 && (
				<div className="mb-2.5 space-y-1">
					{region.transitions.map((t, i) => (
						<div
							key={i}
							className="rounded bg-muted px-2.5 py-1.5 font-mono text-sm leading-snug text-muted-foreground"
						>
							<span className="font-medium text-orange-600">{t.on_event}</span>
							{t.from_state && (
								<>
									{" "}
									{t.from_state} → {t.to_state}
								</>
							)}
							{t.action && <> ⇒ {t.action}</>}
						</div>
					))}
				</div>
			)}

			{region.attachments && region.attachments.length > 0 && (
				<div className="mt-2 space-y-1">
					{region.attachments.map((a) => {
						const src = `/a/${encodeURIComponent(region.name)}/${encodeURIComponent(a)}`;
						return (
							<img
								key={a}
								src={src}
								alt={a}
								loading="lazy"
								className="w-full cursor-pointer rounded-lg border border-border transition-shadow hover:shadow-lg"
								onClick={() => onImageClick?.(src, a)}
							/>
						);
					})}
				</div>
			)}

			{region.regions && region.regions.length > 0 && (
				<div className="mt-3 border-l-2 border-border pl-4">
					<RegionTree
						regions={region.regions}
						renderSpec={renderSpec}
						onImageClick={onImageClick}
					/>
				</div>
			)}
		</div>
	);
}
