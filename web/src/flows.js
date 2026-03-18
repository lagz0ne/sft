let vizInstance = null;

async function getViz() {
  if (vizInstance) return vizInstance;
  const { instance } = await import("@viz-js/viz");
  vizInstance = await instance();
  return vizInstance;
}

export async function renderFlowDiagram(name, sequence) {
  const viz = await getViz();
  const dot = sequenceToDot(name, sequence);
  return viz.renderSVGElement(dot);
}

function sequenceToDot(name, sequence) {
  const steps = sequence.split(/\s*→\s*/);
  const edges = [];
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

function cleanStep(s) {
  return s.replace(/\[.*?\]\s*/, "").replace(/\(.*?\)/, "").trim();
}

function extractLabel(s) {
  const m = s.match(/\[(.*?)\]/);
  return m ? m[1] : null;
}
