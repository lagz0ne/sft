import { renderFlowDiagram } from "./flows.js";

export function renderOverview(container, spec, renderSpec, onNavigate) {
  container.innerHTML = "";

  // Header
  const header = document.createElement("div");
  header.className = "overview-header";

  const h1 = document.createElement("h1");
  h1.textContent = spec.app.name;
  header.appendChild(h1);

  if (spec.app.description) {
    const p = document.createElement("p");
    p.textContent = spec.app.description;
    header.appendChild(p);
  }

  // Stats
  let regionCount = 0, eventCount = 0;
  const countR = (regions) => {
    for (const r of regions || []) {
      regionCount++;
      eventCount += r.events?.length || 0;
      countR(r.regions);
    }
  };
  spec.screens.forEach((s) => countR(s.regions));

  const stats = document.createElement("div");
  stats.className = "overview-stats";
  stats.innerHTML = `
    <span class="overview-stat"><strong>${spec.screens.length}</strong> screens</span>
    <span class="overview-stat"><strong>${regionCount}</strong> regions</span>
    <span class="overview-stat"><strong>${eventCount}</strong> events</span>
    <span class="overview-stat"><strong>${spec.flows?.length || 0}</strong> flows</span>
  `;
  header.appendChild(stats);
  container.appendChild(header);

  // Screen cards grid
  const grid = document.createElement("div");
  grid.className = "screen-grid";

  for (const screen of spec.screens) {
    const card = document.createElement("div");
    card.className = "screen-card";
    card.addEventListener("click", () => onNavigate({ type: "screen", name: screen.name }));

    // Thumbnail
    if (screen.attachments?.length) {
      const img = document.createElement("img");
      img.className = "screen-card-thumb";
      img.src = `/a/${encodeURIComponent(screen.name)}/${encodeURIComponent(screen.attachments[0])}`;
      img.alt = screen.name;
      img.loading = "lazy";
      card.appendChild(img);
    } else {
      const ph = document.createElement("div");
      ph.className = "screen-card-placeholder";
      ph.textContent = screen.name.substring(0, 2);
      card.appendChild(ph);
    }

    // Body
    const body = document.createElement("div");
    body.className = "screen-card-body";

    const h3 = document.createElement("h3");
    h3.textContent = screen.name;
    body.appendChild(h3);

    if (screen.description) {
      const p = document.createElement("p");
      p.textContent = screen.description;
      body.appendChild(p);
    }

    // Meta
    let rCount = 0, eCount = 0;
    const count = (regions) => {
      for (const r of regions || []) {
        rCount++;
        eCount += r.events?.length || 0;
        count(r.regions);
      }
    };
    count(screen.regions);

    const meta = document.createElement("div");
    meta.className = "screen-card-meta";
    meta.innerHTML = `<span>${rCount} regions</span><span>${eCount} events</span>`;
    body.appendChild(meta);

    card.appendChild(body);
    grid.appendChild(card);
  }
  container.appendChild(grid);

  // Flows section
  if (spec.flows?.length) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = "Flows";
    container.appendChild(title);

    const flowList = document.createElement("div");
    flowList.className = "flow-list";
    for (const flow of spec.flows) {
      const card = document.createElement("div");
      card.className = "flow-card";
      card.addEventListener("click", () => onNavigate({ type: "flow", name: flow.name }));
      card.innerHTML = `
        <h3>${flow.name}</h3>
        ${flow.description ? `<p>${flow.description}</p>` : ""}
        <div class="flow-seq">${flow.sequence}</div>
      `;
      flowList.appendChild(card);
    }
    container.appendChild(flowList);
  }
}

export function renderScreen(container, screenName, spec, renderSpec, showLightbox) {
  container.innerHTML = "";
  const screen = spec.screens.find((s) => s.name === screenName);
  if (!screen) {
    container.textContent = `Screen "${screenName}" not found`;
    return;
  }

  // Header
  const header = document.createElement("div");
  header.className = "screen-header";

  const h1 = document.createElement("h1");
  h1.textContent = screen.name;
  header.appendChild(h1);

  if (screen.description) {
    const p = document.createElement("p");
    p.textContent = screen.description;
    header.appendChild(p);
  }

  if (screen.tags?.length) {
    const tags = document.createElement("div");
    tags.className = "screen-tags";
    for (const t of screen.tags) {
      const tag = document.createElement("span");
      tag.className = "tag";
      tag.textContent = t;
      tags.appendChild(tag);
    }
    header.appendChild(tags);
  }
  container.appendChild(header);

  // Attachments (prominent)
  if (screen.attachments?.length) {
    const section = document.createElement("div");
    section.className = "attachments";
    for (const a of screen.attachments) {
      const src = `/a/${encodeURIComponent(screen.name)}/${encodeURIComponent(a)}`;
      const img = document.createElement("img");
      img.className = "attachment-img";
      img.src = src;
      img.alt = a;
      img.loading = "lazy";
      img.addEventListener("click", () => showLightbox(src, a));
      section.appendChild(img);

      const label = document.createElement("div");
      label.className = "attachment-label";
      label.textContent = a;
      section.appendChild(label);
    }
    container.appendChild(section);
  }

  // Regions
  if (screen.regions?.length) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = "Regions";
    container.appendChild(title);
    renderRegions(container, screen.regions, renderSpec, showLightbox);
  }

  // Screen-level transitions
  if (screen.transitions?.length) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = "State Transitions";
    container.appendChild(title);
    renderTransitions(container, screen.transitions);
  }

  // Related flows
  const related = spec.flows?.filter((f) => f.sequence.includes(screen.name)) || [];
  if (related.length) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = "Related Flows";
    container.appendChild(title);
    for (const flow of related) {
      renderFlowCard(container, flow);
    }
  }
}

function renderRegions(container, regions, renderSpec, showLightbox) {
  for (const r of regions) {
    const el = document.createElement("div");
    el.className = "region";

    // Header
    const header = document.createElement("div");
    header.className = "region-header";

    const name = document.createElement("span");
    name.className = "region-name";
    name.textContent = r.name;
    header.appendChild(name);

    // Component type from render spec
    const renderEl = renderSpec?.elements?.[r.name];
    if (renderEl && renderEl.type !== "Stack") {
      const comp = document.createElement("span");
      comp.className = "region-component";
      comp.textContent = renderEl.type;
      header.appendChild(comp);
    }
    el.appendChild(header);

    // Description
    if (r.description) {
      const desc = document.createElement("div");
      desc.className = "region-desc";
      desc.textContent = r.description;
      el.appendChild(desc);
    }

    // Events
    if (r.events?.length) {
      const pills = document.createElement("div");
      pills.className = "event-pills";
      for (const e of r.events) {
        const pill = document.createElement("span");
        pill.className = "event-pill";
        pill.textContent = e;
        pills.appendChild(pill);
      }
      el.appendChild(pills);
    }

    // Transitions
    if (r.transitions?.length) {
      renderTransitions(el, r.transitions);
    }

    // Attachments
    if (r.attachments?.length) {
      for (const a of r.attachments) {
        const src = `/a/${encodeURIComponent(r.name)}/${encodeURIComponent(a)}`;
        const img = document.createElement("img");
        img.className = "attachment-img";
        img.src = src;
        img.alt = a;
        img.style.marginTop = "8px";
        img.addEventListener("click", () => showLightbox(src, a));
        el.appendChild(img);
      }
    }

    // Children
    if (r.regions?.length) {
      const children = document.createElement("div");
      children.className = "region-children";
      renderRegions(children, r.regions, renderSpec, showLightbox);
      el.appendChild(children);
    }

    container.appendChild(el);
  }
}

function renderTransitions(container, transitions) {
  for (const t of transitions) {
    const row = document.createElement("div");
    row.className = "transition-row";
    let html = `<span class="transition-event">${t.on_event}</span>`;
    if (t.from_state) html += ` ${t.from_state} \u2192 ${t.to_state}`;
    if (t.action) html += ` \u21D2 ${t.action}`;
    row.innerHTML = html;
    container.appendChild(row);
  }
}

async function renderFlowCard(container, flow) {
  const card = document.createElement("div");
  card.className = "flow-diagram";
  try {
    const svg = await renderFlowDiagram(flow.name, flow.sequence);
    card.appendChild(svg);
  } catch {
    card.innerHTML = `<div class="flow-sequence-text">${flow.sequence}</div>`;
  }
  container.appendChild(card);
}

export async function renderFlowView(container, flowName, spec, showLightbox) {
  container.innerHTML = "";
  const flow = spec.flows?.find((f) => f.name === flowName);
  if (!flow) {
    container.textContent = `Flow "${flowName}" not found`;
    return;
  }

  const header = document.createElement("div");
  header.className = "flow-header";
  header.innerHTML = `<h1>${flow.name}</h1>`;
  if (flow.description) {
    const p = document.createElement("p");
    p.textContent = flow.description;
    header.appendChild(p);
  }
  container.appendChild(header);

  // Sequence text
  const seqText = document.createElement("div");
  seqText.className = "flow-sequence-text";
  seqText.textContent = flow.sequence;
  container.appendChild(seqText);

  // Diagram
  try {
    const svg = await renderFlowDiagram(flow.name, flow.sequence);
    const diagram = document.createElement("div");
    diagram.className = "flow-diagram";
    diagram.appendChild(svg);
    container.appendChild(diagram);
  } catch { /* fallback already shown as text */ }

  // Related screens
  const steps = flow.sequence.split(/\s*\u2192\s*/).map((s) => s.replace(/\[.*?\]\s*/, "").replace(/\(.*?\)/, "").trim());
  const relatedScreens = spec.screens.filter((s) => steps.includes(s.name));
  if (relatedScreens.length) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = "Screens in this flow";
    container.appendChild(title);

    for (const screen of relatedScreens) {
      const card = document.createElement("div");
      card.className = "region";
      const name = document.createElement("div");
      name.className = "region-name";
      name.textContent = screen.name;
      card.appendChild(name);
      if (screen.description) {
        const desc = document.createElement("div");
        desc.className = "region-desc";
        desc.textContent = screen.description;
        card.appendChild(desc);
      }
      container.appendChild(card);
    }
  }
}
