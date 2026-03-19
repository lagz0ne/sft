# C3 Structural Index
<!-- hash: sha256:0cfea410521f83555f46efb8597d586a4bc61d796f8cf6e14ef488fbd401afd8 -->

## adr-00000000-c3-adoption — C3 Architecture Documentation Adoption (adr)
blocks: Goal ✓

## adr-20260317-sft-view-nats-backbone — sft view — NATS messaging + SQLite query engine (adr)
blocks: Goal ✓

## adr-20260318-frontend-react-migration — Replace vanilla JS frontend with React + TanStack Router via better-t-stack (adr)
blocks: Goal ✓

## c3-0 — SFT (context)
reverse deps: adr-00000000-c3-adoption, c3-1
blocks: Abstract Constraints ✓, Containers ✓, Goal ✓

## c3-1 — cli (container)
context: c3-0
reverse deps: c3-101, c3-102, c3-103, c3-110, c3-111, c3-112, c3-113, c3-114, c3-115, c3-116, c3-117
constraints from: c3-0
blocks: Complexity Assessment ✓, Components ✓, Goal ✓, Responsibilities ✓

## c3-101 — model (component)
container: c3-1 | context: c3-0
reverse deps: ref-event-model
files: internal/model/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-102 — store (component)
container: c3-1 | context: c3-0
reverse deps: adr-20260317-sft-view-nats-backbone, ref-entity-resolution, ref-sqlite-persistence
files: internal/store/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-103 — format (component)
container: c3-1 | context: c3-0
files: internal/format/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ○

## c3-110 — loader (component)
container: c3-1 | context: c3-0
reverse deps: ref-yaml-format
files: internal/loader/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-111 — show (component)
container: c3-1 | context: c3-0
reverse deps: ref-entity-resolution
files: internal/show/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-112 — query (component)
container: c3-1 | context: c3-0
reverse deps: adr-20260317-sft-view-nats-backbone, ref-sqlite-persistence
files: internal/query/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-113 — validator (component)
container: c3-1 | context: c3-0
reverse deps: ref-event-model, ref-sqlite-persistence
files: internal/validator/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-114 — diff (component)
container: c3-1 | context: c3-0
files: internal/diff/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ○

## c3-115 — render (component)
container: c3-1 | context: c3-0
reverse deps: adr-20260317-sft-view-nats-backbone
files: internal/render/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ○

## c3-116 — flow (component)
container: c3-1 | context: c3-0
reverse deps: ref-entity-resolution
files: internal/flow/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ✓

## c3-117 — entrypoint (component)
container: c3-1 | context: c3-0
reverse deps: adr-20260317-sft-view-nats-backbone, adr-20260318-frontend-react-migration
files: cmd/sft/**
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ✓, Related Refs ○

## ref-entity-resolution — Entity Resolution (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## ref-event-model — Event Bubbling Model (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## ref-sqlite-persistence — SQLite Persistence (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## ref-yaml-format — YAML Spec Format (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## File Map
cmd/sft/** → c3-117
internal/diff/** → c3-114
internal/flow/** → c3-116
internal/format/** → c3-103
internal/loader/** → c3-110
internal/model/** → c3-101
internal/query/** → c3-112
internal/render/** → c3-115
internal/show/** → c3-111
internal/store/** → c3-102
internal/validator/** → c3-113

## Ref Map
ref-entity-resolution | scope: c3-102, c3-111, c3-116
ref-event-model | scope: c3-101, c3-113
ref-sqlite-persistence | scope: c3-102, c3-112, c3-113
ref-yaml-format | scope: c3-110
