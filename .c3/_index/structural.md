# C3 Structural Index
<!-- hash: sha256:ccd44b544dabdb7e3f3f9041be4250ab86f903a08e779b87565d540d7950fa43 -->

## adr-00000000-c3-adoption — C3 Architecture Documentation Adoption (adr)
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
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-102 — store (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-103 — format (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-110 — loader (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-111 — show (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-112 — query (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-113 — validator (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-114 — diff (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-115 — render (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-116 — flow (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## c3-117 — entrypoint (component)
container: c3-1 | context: c3-0
constraints from: c3-0, c3-1
blocks: Container Connection ✓, Dependencies ✓, Goal ○, Related Refs ○

## ref-entity-resolution — entity-resolution (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## ref-sqlite-persistence — sqlite-persistence (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## ref-yaml-format — yaml-format (ref)
blocks: Choice ✓, Goal ✓, How ✓, Why ✓

## Ref Map
ref-entity-resolution
ref-sqlite-persistence
ref-yaml-format
