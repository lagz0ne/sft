export function setupSidebar(spec, onNavigate) {
  // Header
  const header = document.getElementById("sidebar-header");
  header.innerHTML = `<h1>${spec.app.name}</h1><p>${spec.app.description}</p>`;

  // Nav
  const nav = document.getElementById("sidebar-nav");
  nav.innerHTML = "";

  // Screens section
  const screensSection = document.createElement("div");
  screensSection.className = "nav-section";
  screensSection.innerHTML = '<div class="nav-section-title">Screens</div>';

  // Overview item
  const overviewBtn = makeNavItem("Overview", null, () => onNavigate({ type: "overview" }));
  overviewBtn.classList.add("active");
  screensSection.appendChild(overviewBtn);

  for (const screen of spec.screens) {
    const hasAtt = screen.attachments?.length > 0;
    const btn = makeNavItem(screen.name, hasAtt, () => onNavigate({ type: "screen", name: screen.name }));
    screensSection.appendChild(btn);
  }
  nav.appendChild(screensSection);

  // Flows section
  if (spec.flows?.length) {
    const flowsSection = document.createElement("div");
    flowsSection.className = "nav-section";
    flowsSection.innerHTML = '<div class="nav-section-title">Flows</div>';
    for (const flow of spec.flows) {
      const btn = makeNavItem(flow.name, false, () => onNavigate({ type: "flow", name: flow.name }));
      flowsSection.appendChild(btn);
    }
    nav.appendChild(flowsSection);
  }
}

export function setActiveNav(label) {
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.classList.toggle("active", item.dataset.label === label);
  });
}

function makeNavItem(label, hasDot, onClick) {
  const btn = document.createElement("button");
  btn.className = "nav-item";
  btn.dataset.label = label;

  const text = document.createElement("span");
  text.className = "nav-label";
  text.textContent = label;
  btn.appendChild(text);

  if (hasDot) {
    const dot = document.createElement("span");
    dot.className = "dot";
    btn.appendChild(dot);
  }

  btn.addEventListener("click", onClick);
  return btn;
}
