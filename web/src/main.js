import { connectNats, requestSpec, requestRender, subscribeChanges } from "./nats.js";
import { renderOverview, renderScreen, renderFlowView } from "./renderer.js";
import { setupSidebar, setActiveNav } from "./nav.js";

let spec, renderSpec;
let currentView = { type: "overview" };

async function init() {
  const content = document.getElementById("main-content");
  content.innerHTML = '<div class="loading-state">Connecting\u2026</div>';

  try {
    await connectNats();
    [spec, renderSpec] = await Promise.all([requestSpec(), requestRender()]);

    setupSidebar(spec, navigate);
    navigate({ type: "overview" });

    // Live reload: re-fetch and re-render when spec changes
    subscribeChanges(async () => {
      [spec, renderSpec] = await Promise.all([requestSpec(), requestRender()]);
      setupSidebar(spec, navigate);
      navigate(currentView);
    });
  } catch (err) {
    content.innerHTML = `<div class="error-state">
      <h2>Connection failed</h2>
      <pre>${err.message}</pre>
      <p>Is <code>sft view</code> running?</p>
    </div>`;
  }
}

function navigate(view) {
  currentView = view;
  const content = document.getElementById("main-content");
  const main = document.getElementById("main");
  main.scrollTop = 0;

  switch (view.type) {
    case "overview":
      setActiveNav("Overview");
      renderOverview(content, spec, renderSpec, navigate);
      break;
    case "screen":
      setActiveNav(view.name);
      renderScreen(content, view.name, spec, renderSpec, showLightbox);
      break;
    case "flow":
      setActiveNav(view.name);
      renderFlowView(content, view.name, spec, showLightbox);
      break;
  }
}

function showLightbox(src, caption) {
  const lb = document.getElementById("lightbox");
  lb.querySelector("img").src = src;
  lb.querySelector(".lb-caption").textContent = caption;
  lb.hidden = false;
}

document.getElementById("lightbox").addEventListener("click", () => {
  document.getElementById("lightbox").hidden = true;
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") document.getElementById("lightbox").hidden = true;
});

init();
