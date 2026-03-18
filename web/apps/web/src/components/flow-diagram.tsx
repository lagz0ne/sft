import { useEffect, useRef, useState } from "react";

let vizInstance: any = null;

async function getViz() {
	if (vizInstance) return vizInstance;
	const { instance } = await import("@viz-js/viz");
	vizInstance = await instance();
	return vizInstance;
}

function sequenceToDot(name: string, sequence: string): string {
	const steps = sequence.split(/\s*→\s*/);
	const edges: string[] = [];
	for (let i = 0; i < steps.length - 1; i++) {
		const from = cleanStep(steps[i]);
		const to = cleanStep(steps[i + 1]);
		const label = extractLabel(steps[i + 1]);
		if (label) {
			edges.push(`"${from}" -> "${to}" [label="${label}"]`);
		} else {
			edges.push(`"${from}" -> "${to}"`);
		}
	}
	return `digraph "${name}" {
  bgcolor="transparent"
  rankdir=LR
  node [shape=box style="rounded,filled" fillcolor="#f7f7f8" color="#e3e5e8" fontcolor="#1a1d23" fontname="DM Sans" fontsize=11]
  edge [color="#b0b4bc" fontcolor="#4b5060" fontname="DM Sans" fontsize=10 arrowsize=0.5]
  ${edges.join("\n  ")}
}`;
}

function cleanStep(s: string): string {
	return s
		.replace(/\[.*?\]\s*/, "")
		.replace(/\(.*?\)/, "")
		.trim();
}

function extractLabel(s: string): string | null {
	const m = s.match(/\[(.*?)\]/);
	return m ? m[1] : null;
}

interface FlowDiagramProps {
	name: string;
	sequence: string;
}

export function FlowDiagram({ name, sequence }: FlowDiagramProps) {
	const containerRef = useRef<HTMLDivElement>(null);
	const [error, setError] = useState(false);

	useEffect(() => {
		let cancelled = false;
		setError(false);

		getViz()
			.then((viz) => {
				if (cancelled || !containerRef.current) return;
				const dot = sequenceToDot(name, sequence);
				const svg = viz.renderSVGElement(dot);
				svg.style.width = "100%";
				svg.style.height = "auto";
				containerRef.current.innerHTML = "";
				containerRef.current.appendChild(svg);
			})
			.catch(() => {
				if (!cancelled) setError(true);
			});

		return () => {
			cancelled = true;
		};
	}, [name, sequence]);

	if (error) {
		return (
			<div className="rounded-lg border border-border bg-background p-5 font-mono text-sm text-muted-foreground">
				{sequence}
			</div>
		);
	}

	return <div ref={containerRef} className="rounded-lg border border-border bg-background p-5" />;
}
