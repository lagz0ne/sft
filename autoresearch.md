# Autoresearch: Eliminate tag keywords — min vocab, max comprehension

## Config
- **Benchmark**: keyword count (lower) + subagent comprehension (must pass)
- **Target metric**: `keyword_count` (lower is better)
- **Constraint**: subagent must correctly solve layout tasks with reduced vocabulary
- **Branch**: `autoresearch/eliminate-tag-keywords`
- **Started**: 2026-03-27T16:10:00Z

## Result: 17 → 14 keywords (4/5 comprehension)

### Positions (8)
```
header    sidebar    toolbar    footer
bottomnav    modal    overlay    split
```

### Modifiers (6)
```
narrow    wide    far    fixed    elevated    fill
```

### Absorption map
```
aside       → sidebar:far
banner      → header (just use header)
drawer      → modal:side
fab         → overlay:fixed
elevated    → modifier (was standalone visual keyword)
```

### Key compositions
```
bottomnav:fixed        # pinned bottom nav
header:fixed           # sticky header
sidebar:far:narrow     # narrow right panel
modal:side             # side-sliding drawer
overlay:fixed          # floating widget/FAB
split:fill             # panel that fills remaining space
```

### Known gap
`overlay:fixed` is position-blind — can't distinguish top-right from bottom-right. Region names disambiguate, or add corner modifiers later.
