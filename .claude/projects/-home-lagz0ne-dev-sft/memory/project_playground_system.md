---
name: playground-system-design
description: SFT playground architecture — three axes (tags/component/component-set), dock UI, json-render bridge, progressive fidelity
type: project
---

SFT playground is an interactive wireframe viewer for behavioral specs. Three independent axes:

**Tags** = layout only (WHERE). Position tags: sidebar, header, toolbar, footer, bottomnav, aside, overlay, modal, drawer, banner, split. Modifiers via colon: sidebar:narrow. Compositions via prefix: mobile:bottomnav. Visual: elevated.

**Component** = content (WHAT). json-render types + props bound via `sft component region Type --props '{...}'`. Playground renders 8 wireframe shapes (input, select, button, image, text, list, card, tabs). No component = prompt to set.

**Component set** = implementation (HOW). Named rendering implementations: wireframe, styled, compact. Maps to json-render registries in production.

**Why:** Tag was originally overloaded with skin types. Separated because tags, components, and visual treatment are three different user journeys that happen at different project stages. Discovery → tags. Refinement → components. Polish → component sets.

**How to apply:** Tags for layout exploration. `sft component` for declaring what regions are. Component sets for switching rendering style. The dock controls all three. Viewport sizing links to compositions (phone icon → mobile composition).
