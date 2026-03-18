import { Link, createFileRoute } from "@tanstack/react-router";
import { useSpecContext } from "../../context/spec-context";
import { RegionTree } from "../../components/region-tree";
import { useLightbox } from "../../components/lightbox";

export const Route = createFileRoute("/screens/$name")({
	component: ScreenPage,
});

function ScreenPage() {
	const { name } = Route.useParams();
	const screenName = decodeURIComponent(name);
	const { spec, renderSpec } = useSpecContext();
	const showLightbox = useLightbox();

	if (!spec) return null;

	const screen = spec.screens.find((s) => s.name === screenName);
	if (!screen) {
		return <p className="text-muted-foreground">Screen &quot;{screenName}&quot; not found</p>;
	}

	const relatedFlows = spec.flows?.filter((f) => f.sequence.includes(screen.name)) ?? [];

	return (
		<>
			{/* Header */}
			<div className="mb-7">
				<h1 className="mb-2.5 font-serif text-2xl leading-tight tracking-tight">{screen.name}</h1>
				{screen.description && (
					<p className="max-w-[65ch] leading-relaxed text-muted-foreground">{screen.description}</p>
				)}
				{screen.tags && screen.tags.length > 0 && (
					<div className="mt-3 flex gap-1.5">
						{screen.tags.map((t) => (
							<span
								key={t}
								className="rounded-full bg-accent px-3 py-0.5 text-xs font-medium text-accent-foreground"
							>
								{t}
							</span>
						))}
					</div>
				)}
			</div>

			{/* Attachments */}
			{screen.attachments && screen.attachments.length > 0 && (
				<div className="mb-8">
					{screen.attachments.map((a) => {
						const src = `/a/${encodeURIComponent(screen.name)}/${encodeURIComponent(a)}`;
						return (
							<div key={a} className="mb-4">
								<img
									src={src}
									alt={a}
									loading="lazy"
									className="w-full cursor-pointer rounded-lg border border-border transition-shadow hover:shadow-lg"
									onClick={() => showLightbox(src, a)}
								/>
								<div className="mt-1.5 font-mono text-xs text-muted-foreground">{a}</div>
							</div>
						);
					})}
				</div>
			)}

			{/* Regions */}
			{screen.regions && screen.regions.length > 0 && (
				<>
					<div className="mb-3.5 mt-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
						Regions
					</div>
					<RegionTree
						regions={screen.regions}
						renderSpec={renderSpec}
						onImageClick={showLightbox}
					/>
				</>
			)}

			{/* Screen-level transitions */}
			{screen.transitions && screen.transitions.length > 0 && (
				<>
					<div className="mb-3.5 mt-5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
						State Transitions
					</div>
					<div className="space-y-1">
						{screen.transitions.map((t, i) => (
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
				</>
			)}

			{/* Related flows */}
			{relatedFlows.length > 0 && (
				<>
					<div className="mb-3.5 mt-5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
						Related Flows
					</div>
					<div className="space-y-2">
						{relatedFlows.map((flow) => (
							<Link
								key={flow.name}
								to="/flows/$name"
								params={{ name: flow.name }}
								className="block rounded-lg border border-border bg-background p-4 transition-colors hover:border-foreground/20"
							>
								<h3 className="text-base font-semibold leading-snug">{flow.name}</h3>
								<div className="mt-1 font-mono text-xs leading-snug text-muted-foreground">
									{flow.sequence}
								</div>
							</Link>
						))}
					</div>
				</>
			)}
		</>
	);
}
