---
id: adr-20260317-sft-view-nats-backbone
title: sft view — NATS messaging + SQLite query engine
type: adr
status: proposed
date: 2026-03-18
affects: [c3-102, c3-112, c3-115, c3-117]
---

# sft view — NATS messaging + SQLite query engine

## Goal

Add `sft view` command that starts an embedded NATS server with WebSocket exposure alongside an HTTP server for static assets and binary serving. SQLite remains the query engine — enhanced with FTS5 full-text search and BM25 ranking. NATS handles real-time messaging and sync. HTTP handles binary blobs and the SPA shell. Each layer does what it's best at.

## Context

SFT is currently a CLI-only tool. The spec lives in SQLite, output goes to stdout. There is no way to visualize the spec interactively or see live updates as the spec evolves.

Goals:
1. Browser-based live view of the spec (screens, regions, components, attachments, flows)
2. Real-time sync — CLI mutations appear in browser instantly
3. A messaging backbone that CLI extensions and future multi-user scenarios can use
4. Full-text search across all spec entities with relevance ranking

## Decision

Three complementary layers, each handling what it's best at:

| Layer | Role | Handles |
|-------|------|---------|
| **SQLite** | Data + query engine | CRUD, FTS5 search, BM25 ranking, recursive CTEs, JSON queries on component props |
| **NATS** | Real-time messaging | Request/reply for spec data, pub/sub for change sync, bidirectional browser↔server |
| **HTTP** | Binary + static serving | SPA shell, attachment blobs (Content-Type, caching, any size) |

### Architecture

```
┌──────────────────────────────────────────────────┐
│  sft view (single Go binary)                     │
│                                                  │
│  ┌──────────────┐   ┌────────────────────────┐   │
│  │ Embedded NATS │   │ Go NATS Handlers       │   │
│  │ WS :8443     │◄──│ sft.spec / sft.render  │   │
│  │ TCP :4222    │   │ sft.search / sft.query │   │
│  └──────┬───────┘   │ sft.mutation           │   │
│         │           │ sft.changes (pub)      │   │
│         │           └────────┬───────────────┘   │
│  ┌──────▼───────┐            │                   │
│  │ HTTP Server  │   ┌────────▼───────────────┐   │
│  │ / → SPA      │   │ SQLite (.sft/db)       │   │
│  │ /a/* → blobs │   │ WAL mode               │   │
│  └──────────────┘   │ FTS5 index (spec_fts)  │   │
│                     │ JSON functions          │   │
│         ┌───────────┤ Recursive CTEs          │   │
│         │           └────────────────────────┘   │
│  ┌──────▼───────┐                                │
│  │ fsnotify     │                                │
│  │ .sft/db-wal  │                                │
│  └──────────────┘                                │
└──────────────────────────────────────────────────┘
       ▲ WS                ▲ TCP
       │                   │
  ┌────┴────┐        ┌─────┴─────┐
  │ Browser │        │ sft CLI   │
  │ SPA     │        │ (other    │
  │ nats-core│        │ terminal) │
  └─────────┘        └───────────┘
```

### SQLite as query engine — FTS5 + BM25

`modernc.org/sqlite` ships with FTS5 compiled in (confirmed via `PRAGMA compile_options`). This unlocks full-text search with zero new dependencies.

**Unified search index** (external content, zero data duplication):

```sql
CREATE VIRTUAL TABLE spec_fts USING fts5(
  entity_type,       -- 'screen', 'region', 'event', 'flow', 'tag'
  entity_name,       -- name of the entity
  description,       -- description text
  extra,             -- flow sequence, tag value, event name
  tokenize='porter unicode61'
);
```

Porter stemmer means "authenticate" matches "authentication", "authenticated". Unicode61 handles diacritics and case folding.

**Queries this enables:**

| What | How |
|------|-----|
| Search everything for "auth" | `MATCH 'auth*' ORDER BY bm25(spec_fts, 0, 10.0, 5.0, 1.0)` — name-boosted ranking |
| Find events containing "click" | `MATCH 'entity_type : event AND click'` |
| Phrase search in flows | `MATCH 'extra : "cart checkout"'` |
| Boolean combo | `MATCH 'login AND (screen OR region) NOT test'` |
| Prefix + proximity | `MATCH 'NEAR(nav* menu, 5)'` |

**Other SQLite power already available:**

| Feature | Use in SFT |
|---------|-----------|
| `WITH RECURSIVE` CTEs | Region hierarchy traversal (replace Go-side `collectDescendantRegions`) |
| `json_extract()` | Query component props: `WHERE json_extract(props, '$.variant') = 'primary'` |
| `json_each()` | Enumerate all prop keys across components |
| Window functions | Rank entities, compute positional stats |

SQLite is not just persistence — it's the query engine. NATS doesn't replace it; NATS delivers the query results in real time.

### NATS subjects

| Subject | Pattern | Purpose |
|---------|---------|---------|
| `sft.spec` | request/reply | Full spec tree (show.Load) |
| `sft.render` | request/reply | json-render element tree |
| `sft.search` | request/reply | FTS5 full-text search with BM25 ranking |
| `sft.validate` | request/reply | Validation findings |
| `sft.query.<type>` | request/reply | Named queries (screens, events, flows, states, etc.) |
| `sft.mutation` | request/reply | All mutations (JSON payload, server-side dispatch) |
| `sft.changes` | pub/sub | Spec changed notification (version counter) |

Mutation payload: `{"cmd": "add", "entity": "screen", "args": {"name": "...", "description": "..."}}`. Single subject, dispatched server-side.

Search payload: `{"q": "auth*", "type": "screen", "limit": 20}`. Returns ranked results with highlights.

### Attachments — served via HTTP

Attachments are binary blobs that can exceed NATS `max_payload`. Serve via HTTP on the same port as the SPA:

- `GET /a/{entity}/{name}` — serves blob with inferred Content-Type
- Browser uses standard `<img src="/a/Home/mockup.png">`
- Gets caching, range requests, Content-Type for free

NATS handles structured messaging; HTTP handles binary serving.

### Change detection

1. **Mutations via NATS** — handler writes to SQLite, updates FTS index, publishes `sft.changes` with version counter
2. **External CLI writes** — `fsnotify` on `.sft/db-wal` detects changes, handler re-reads and publishes `sft.changes`

Browser on reconnect: compares version counter, re-fetches if stale.

### CLI dual-path

1. CLI checks for running server (read `.sft/view.port`)
2. If found → connect as NATS client, send `sft.mutation`, get reply
3. If not found → direct SQLite (current behavior, unchanged)

Optional and non-breaking. Direct SQLite always works.

### Browser SPA (Vite 8)

Pure TypeScript. No Go WASM. Connects to NATS via `wsconnect()`.

- `sft.render` request/reply → json-render tree → interactive UI
- `sft.search` request/reply → full-text search results with ranking
- `sft.changes` subscription → live updates
- Attachments via HTTP `<img>` / `<object>` tags
- On reconnect: re-subscribe, compare version, re-fetch if stale

Built with Vite 8, output embedded in Go binary via `go:embed`.

### SQLite WAL

Add to `store.Open()`:
```go
db.Exec("PRAGMA journal_mode=WAL")
db.Exec("PRAGMA busy_timeout=5000")
```

Enables concurrent readers (NATS handlers + HTTP server) while CLI writes.

### Network binding

NATS WebSocket binds `0.0.0.0` — multi-user access is a future goal. Security via:
- Token auth: random token generated on startup
- NATS `authorization { token: "..." }` on embedded server
- Browser receives token via initial SPA page load
- CLI clients pass token via `.sft/view.token` file

## Affected Components

| Component | Change |
|-----------|--------|
| **c3-102 (store)** | WAL + busy_timeout. FTS5 index creation in migrations. FTS sync triggers on insert/update/delete. Search query method. |
| **c3-112 (query)** | New `Search()` function wrapping FTS5 MATCH + BM25. Expose via `sft search` CLI command. |
| **c3-115 (render)** | No changes — output consumed by browser via NATS |
| **c3-117 (entrypoint)** | Add `view` and `search` commands. Wire NATS server, handlers, HTTP serving. Optional NATS client path. |
| **NEW: view** | `internal/view` — embedded NATS server, subject handlers, HTTP attachment serving, fsnotify watcher, SPA embedding |
| **NEW: web/** | Vite 8 SPA — json-render interpreter, NATS client, search UI |

## Work Breakdown

### Phase 1: SQLite enhancements
1. Add WAL + busy_timeout to store.Open()
2. Create FTS5 `spec_fts` table in migrations
3. Add triggers to keep FTS index in sync on insert/update/delete
4. Implement `store.Search(query, entityType, limit)` method
5. Add `sft search <term>` CLI command

### Phase 2: Server foundation
6. Create `internal/view` package with embedded NATS server (WS + TCP)
7. Add HTTP server for SPA + attachment serving on same port
8. Implement `sft.spec` and `sft.render` request/reply handlers
9. Implement `sft.search` handler (delegates to store.Search)
10. Add `view` command to entrypoint

### Phase 3: Browser SPA
11. Scaffold Vite 8 SPA with `@nats-io/nats-core` wsconnect
12. Minimal json-render interpreter (Card, Stack → DOM)
13. Search UI with live results
14. Embed built SPA in Go binary via `go:embed`

### Phase 4: Live sync
15. Implement `sft.mutation` handler (dispatch to store methods)
16. Add `sft.changes` publish on every mutation (version counter)
17. Add fsnotify watcher for external CLI writes
18. Browser subscribes to `sft.changes`, re-fetches on update

### Phase 5: CLI integration
19. CLI detects running server (`.sft/view.port` file)
20. Optional NATS client path for mutations
21. CLI mutations route through NATS when server is available

### Phase 6: Rich browser experience
22. Flow visualization
23. Attachment inline preview
24. Screen navigation / composability view
25. Component prop inspector with JSON query

## Risks

| Risk | Mitigation |
|------|------------|
| Binary size +10MB from NATS | Accepted trade-off for messaging capability |
| Port conflicts | Dynamic port with `.sft/view.port` file, fallback range |
| NATS at-most-once loses events | Version counter + re-fetch on reconnect; fsnotify catches external writes |
| Attachment size > max_payload | Served via HTTP, not NATS |
| SQLite concurrent access | WAL mode + busy_timeout |
| FTS index drift | Triggers on source tables keep index in sync; rebuild command as fallback |
| Network exposure (0.0.0.0) | Token auth on NATS; token file for CLI clients |

## Alternatives Considered

### HTTP + SSE (no NATS)
Simpler, zero new dependencies. Rejected because:
- SSE is unidirectional. Browser mutations need separate HTTP POST path.
- No unified protocol — CLI speaks HTTP or direct SQLite, browser speaks SSE + HTTP.
- NATS gives request/reply + pub/sub + bidirectional in one connection.
- NATS subjects are natural extension points for plugins and multi-user.

### Everything over NATS (no HTTP)
Rejected because:
- Attachments exceed max_payload. HTTP serves binary blobs natively.
- SQLite's FTS5/BM25/CTEs are a powerful query engine that NATS KV cannot replicate.
- Each layer does what it's best at: SQLite queries, NATS syncs, HTTP serves files.

### Replace SQLite with JetStream KV
Rejected because:
- SFT schema has 5 foreign keys, 4 SQL views, recursive parent resolution, cross-table uniqueness checks.
- JetStream KV is flat key-value. Every join, constraint, and view would need reimplementation in Go.
- FTS5 is free and already compiled in. JetStream has nothing comparable.
- SQLite gets stronger with FTS5, not weaker.

### Go WASM in browser
Rejected. Transitive SQLite dependency fails under GOOS=js. 2.2MB binary for ~130 lines of transforms. Pure TypeScript SPA is the right choice.
