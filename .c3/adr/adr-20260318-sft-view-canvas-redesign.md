---
id: adr-20260318-sft-view-canvas-redesign
title: "sft view: canvas + overlay redesign"
type: adr
status: implemented
date: 2026-03-18
affects: []
---

# sft view: canvas + overlay redesign

## Goal

Replace the sidebar + route-based frontend with a canvas home base + overlay architecture, matching the SFT behavioral spec. Canvas shows screen nodes and flow edges with progressive disclosure. Overlays for screen detail (wireframe layout), flow detail (step sequence), and state machine simulation (compact panel).

## What changes

**Remove:** sidebar.tsx, routes/index.tsx, routes/screens/$name.tsx, routes/flows/$name.tsx, flow-diagram.tsx, region-tree.tsx, router.tsx, routeTree.gen.ts

**Keep:** lib/nats.ts, lib/types.ts, hooks/use-spec.ts, context/spec-context.tsx, components/lightbox.tsx, components/loader.tsx, vite.config.ts

**Create:**
- `components/canvas.tsx` — screen nodes + flow edges, pan/zoom, progressive disclosure
- `components/screen-overlay.tsx` — wireframe layout with attachment hero, dashed region zones, trigger badges
- `components/flow-overlay.tsx` — vertical step sequence
- `components/state-machine-panel.tsx` — interactive simulation panel
- `routes/__root.tsx` — full-screen canvas with overlay system (no sidebar, no router)
- `routes/index.tsx` — minimal, just renders canvas

**Extend types.ts:** add flow on_event field, flow steps

## Work Breakdown

1. Update types.ts with missing fields
2. Create canvas.tsx with layout algorithm and progressive disclosure
3. Create screen-overlay.tsx with wireframe regions
4. Create flow-overlay.tsx with step sequence
5. Create state-machine-panel.tsx with interactive simulation
6. Rewrite root layout — remove sidebar/grid, full-screen canvas + overlay layer
7. Remove old files (sidebar, old routes, region-tree, flow-diagram)

## Risks

- Canvas layout algorithm quality — simple layered layout may not look great for all specs
- Pan/zoom without a library — CSS transforms should work for read-only use
- No TanStack Router needed anymore — the app is a single-page canvas with overlays, not routes
