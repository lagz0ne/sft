import { createFileRoute } from "@tanstack/react-router";
import { useSpecContext } from "../../context/spec-context";
import { FlowDiagram } from "../../components/flow-diagram";

export const Route = createFileRoute("/flows/$name")({
	component: FlowPage,
});

function FlowPage() {
	const { name } = Route.useParams();
	const flowName = decodeURIComponent(name);
	const { spec } = useSpecContext();

	if (!spec) return null;

	const flow = spec.flows?.find((f) => f.name === flowName);
	if (!flow) {
		return <p className="text-muted-foreground">Flow &quot;{flowName}&quot; not found</p>;
	}

	// Extract screen names from sequence
	const steps = flow.sequence.split(/\s*→\s*/).map((s) =>
		s
			.replace(/\[.*?\]\s*/, "")
			.replace(/\(.*?\)/, "")
			.trim(),
	);
	const relatedScreens = spec.screens.filter((s) => steps.includes(s.name));

	return (
		<>
			<div className="mb-6">
				<h1 className="mb-2 font-serif text-2xl leading-tight tracking-tight">{flow.name}</h1>
				{flow.description && (
					<p className="max-w-[65ch] leading-relaxed text-muted-foreground">{flow.description}</p>
				)}
			</div>

			{/* Sequence text */}
			<div className="mb-6 rounded border border-border bg-background px-4 py-3 font-mono text-sm leading-normal text-muted-foreground break-words">
				{flow.sequence}
			</div>

			{/* Diagram */}
			<div className="mb-5">
				<FlowDiagram name={flow.name} sequence={flow.sequence} />
			</div>

			{/* Related screens */}
			{relatedScreens.length > 0 && (
				<>
					<div className="mb-3.5 mt-5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
						Screens in this flow
					</div>
					<div className="space-y-2.5">
						{relatedScreens.map((screen) => (
							<div key={screen.name} className="rounded-lg border border-border bg-background p-4">
								<div className="text-base font-semibold leading-snug">{screen.name}</div>
								{screen.description && (
									<p className="mt-1 max-w-[65ch] leading-relaxed text-muted-foreground">
										{screen.description}
									</p>
								)}
							</div>
						))}
					</div>
				</>
			)}
		</>
	);
}
