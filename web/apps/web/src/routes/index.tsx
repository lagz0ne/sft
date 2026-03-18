import { Link, createFileRoute } from "@tanstack/react-router";
import { useSpecContext } from "../context/spec-context";

export const Route = createFileRoute("/")({
	component: OverviewPage,
});

function OverviewPage() {
	const { spec, loading, error } = useSpecContext();

	if (loading) {
		return (
			<div className="flex h-[60vh] items-center justify-center text-muted-foreground">
				Connecting…
			</div>
		);
	}

	if (error || !spec) {
		return (
			<div className="px-10 py-20 text-center">
				<h2 className="mb-2 text-lg font-bold text-destructive">Connection failed</h2>
				<pre className="mb-2 inline-block rounded bg-muted px-4 py-3 text-left font-mono text-sm text-muted-foreground">
					{error?.message ?? "Unknown error"}
				</pre>
				<p className="text-sm text-muted-foreground">
					Is <code className="font-mono">sft view</code> running?
				</p>
			</div>
		);
	}

	let regionCount = 0;
	let eventCount = 0;
	const countRegions = (regions?: { events?: string[]; regions?: any[] }[]) => {
		for (const r of regions ?? []) {
			regionCount++;
			eventCount += r.events?.length ?? 0;
			countRegions(r.regions);
		}
	};
	spec.screens.forEach((s) => countRegions(s.regions));

	return (
		<>
			<div className="mb-8">
				<h1 className="mb-2 font-serif text-2xl leading-tight tracking-tight">{spec.app.name}</h1>
				{spec.app.description && (
					<p className="max-w-[65ch] leading-relaxed text-muted-foreground">
						{spec.app.description}
					</p>
				)}
				<div className="mt-4 flex gap-5 text-sm text-muted-foreground">
					<span>
						<strong className="font-semibold text-foreground">{spec.screens.length}</strong> screens
					</span>
					<span>
						<strong className="font-semibold text-foreground">{regionCount}</strong> regions
					</span>
					<span>
						<strong className="font-semibold text-foreground">{eventCount}</strong> events
					</span>
					<span>
						<strong className="font-semibold text-foreground">{spec.flows?.length ?? 0}</strong>{" "}
						flows
					</span>
				</div>
			</div>

			{/* Screen cards grid */}
			<div className="mb-10 grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4">
				{spec.screens.map((screen) => (
					<Link
						key={screen.name}
						to="/screens/$name"
						params={{ name: screen.name }}
						className="group overflow-hidden rounded-lg border border-border bg-background transition-all hover:border-foreground/20 hover:shadow-md"
					>
						{screen.attachments?.length ? (
							<img
								src={`/a/${encodeURIComponent(screen.name)}/${encodeURIComponent(screen.attachments[0])}`}
								alt={screen.name}
								loading="lazy"
								className="aspect-[16/10] w-full bg-muted object-cover"
							/>
						) : (
							<div className="flex aspect-[16/10] w-full items-center justify-center bg-muted text-lg font-semibold tracking-wide text-muted-foreground">
								{screen.name.substring(0, 2)}
							</div>
						)}
						<div className="p-3.5">
							<h3 className="text-base font-semibold leading-snug">{screen.name}</h3>
							{screen.description && (
								<p className="mt-1 line-clamp-2 text-sm leading-normal text-muted-foreground">
									{screen.description}
								</p>
							)}
							<div className="mt-2.5 flex gap-3.5 font-mono text-xs text-muted-foreground">
								<span>{countR(screen.regions)} regions</span>
								<span>{countE(screen.regions)} events</span>
							</div>
						</div>
					</Link>
				))}
			</div>

			{/* Flows section */}
			{spec.flows && spec.flows.length > 0 && (
				<>
					<div className="mb-3.5 mt-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
						Flows
					</div>
					<div className="mb-6 space-y-2">
						{spec.flows.map((flow) => (
							<Link
								key={flow.name}
								to="/flows/$name"
								params={{ name: flow.name }}
								className="block rounded-lg border border-border bg-background p-4 transition-colors hover:border-foreground/20"
							>
								<h3 className="text-base font-semibold leading-snug">{flow.name}</h3>
								{flow.description && (
									<p className="mt-1 text-sm leading-normal text-muted-foreground">
										{flow.description}
									</p>
								)}
								<div className="mt-1.5 font-mono text-xs leading-snug text-muted-foreground">
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

function countR(regions?: { regions?: any[] }[]): number {
	let n = 0;
	for (const r of regions ?? []) {
		n++;
		n += countR(r.regions);
	}
	return n;
}

function countE(regions?: { events?: string[]; regions?: any[] }[]): number {
	let n = 0;
	for (const r of regions ?? []) {
		n += r.events?.length ?? 0;
		n += countE(r.regions);
	}
	return n;
}
