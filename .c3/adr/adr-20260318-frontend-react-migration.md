---
id: adr-20260318-frontend-react-migration
title: Replace vanilla JS frontend with React + TanStack Router via better-t-stack
type: adr
status: accepted
date: 2026-03-18
affects: [c3-117]
---

# frontend-react-migration

## Goal

Replace the hand-rolled vanilla JS frontend (`web/`) with a proper React 19 + TypeScript + Vite stack scaffolded from better-t-stack. Enables component-based UI development, type-safe routing, shadcn/ui primitives, and rapid iteration for future features.

## Work Breakdown

1. Archive old `web/` source (keep as reference)
2. Move better-t-stack scaffold into `web/`
3. Customize:
   - Remove unused `packages/env` (no env vars needed)
   - Light theme default (per user preference)
   - Configure Vite proxy for Go backend (`/nats`, `/a/`)
   - Add `nats.ws` dependency
4. Create core infrastructure:
   - `lib/nats.ts` — typed NATS connection
   - `lib/types.ts` — TypeScript types for spec data
   - `hooks/use-spec.ts` — reactive spec data via NATS
5. Create route structure: `/` (overview), `/screens/$name`, `/flows/$name`
6. Port layout (sidebar shell) and views
7. Update Go server to serve from `dist/` for production builds
8. Verify build + dev workflow

## Risks

- Go server path changes — must update `findWebDir()` and static file serving
- NATS WS proxy path must remain `/nats` for backward compat
- Dev workflow: two processes (Vite dev + Go backend) vs single process
