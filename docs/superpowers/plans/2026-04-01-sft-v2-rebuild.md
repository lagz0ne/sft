# SFT v2 Clean Rebuild — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild SFT as a composition engine — new schema with entity pool, experiments, entry screen, complete diff, all validator rules, clone. Drop frontend.

**Architecture:** Clean rebuild of all Go packages (model, store, loader, show, diff, validator, CLI). Existing schema patterns preserved (polymorphic owner, JSON columns, UNIQUE constraints). New: `entities` table with `$name` load-time resolution, `experiments` table with view-time overlay, `component_schemas` table, `entry` column on screens, `fn` field on validator rules.

**Tech Stack:** Go 1.25, modernc.org/sqlite, gopkg.in/yaml.v3. TDD with `go test`. No frontend.

---

## Dependency Graph & Parallelism

```
Phase 1 (sequential):  schema → model → store
Phase 2 (parallel):    loader | show | validator | diff
Phase 3 (sequential):  CLI → clone → experiments
```

Tasks 1-3 are sequential (each depends on the previous). Tasks 4-7 can run in parallel (all depend on store but not each other). Tasks 8-10 wire everything together.

## File Structure

```
cmd/sft/main.go                    — CLI command routing (REWRITE)
internal/model/model.go            — Go structs (REWRITE — add Entity, Experiment, ComponentSchema)
internal/store/schema.sql           — DDL (REWRITE — add entities, experiments, component_schemas, entry col)
internal/store/store.go             — Open, migrations, CRUD (REWRITE)
internal/store/clone.go             — NEW: deep-copy logic
internal/loader/loader.go           — YAML parsing + $name resolution (REWRITE)
internal/loader/statemachine.go     — State machine parsing (KEEP, minor updates)
internal/loader/resolve.go          — NEW: entity $name reference resolver
internal/show/show.go               — Spec tree assembly (REWRITE — add entities, experiments)
internal/diff/diff.go               — Compare function (REWRITE — all entity types)
internal/diff/diff_test.go          — NEW: comprehensive test suite
internal/validator/validator.go     — Rules + Validate (REWRITE — fn field, new rules)
internal/validator/validator_test.go — NEW: test suite for new rules
internal/format/format.go           — Output formatting (KEEP as-is)
internal/query/query.go             — Named queries (MINOR updates)
```

---

## Task 1: Schema + Model

**Files:**
- Rewrite: `internal/store/schema.sql`
- Rewrite: `internal/model/model.go`
- Test: `internal/model/model_test.go` (optional — structs are plain data)

- [ ] **Step 1: Write the new schema.sql**

Start from the existing schema. Add three new tables and one new column:

```sql
-- Add to existing screens table:
-- entry INTEGER NOT NULL DEFAULT 0

-- NEW TABLE: entities
CREATE TABLE IF NOT EXISTS entities (
  id      INTEGER PRIMARY KEY,
  app_id  INTEGER NOT NULL REFERENCES apps(id),
  name    TEXT NOT NULL,
  type    TEXT NOT NULL,
  data    TEXT NOT NULL DEFAULT '{}',
  UNIQUE(app_id, name)
);

-- NEW TABLE: experiments
CREATE TABLE IF NOT EXISTS experiments (
  id          INTEGER PRIMARY KEY,
  app_id      INTEGER NOT NULL REFERENCES apps(id),
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  scope       TEXT NOT NULL,
  overlay     TEXT NOT NULL DEFAULT '{}',
  status      TEXT NOT NULL DEFAULT 'active'
              CHECK(status IN ('active','committed','discarded')),
  UNIQUE(app_id, name)
);

-- NEW TABLE: component_schemas
CREATE TABLE IF NOT EXISTS component_schemas (
  id      INTEGER PRIMARY KEY,
  app_id  INTEGER NOT NULL REFERENCES apps(id),
  name    TEXT NOT NULL,
  props   TEXT NOT NULL DEFAULT '{}',
  UNIQUE(app_id, name)
);
```

Keep ALL existing tables, constraints, views, and indexes. The full schema is the existing `schema.sql` with these additions.

- [ ] **Step 2: Add new model structs**

Add to `internal/model/model.go`:

```go
type Entity struct {
	ID    int64
	AppID int64
	Name  string
	Type  string // references data_types.name
	Data  string // JSON
}

type Experiment struct {
	ID          int64
	AppID       int64
	Name        string
	Description string
	Scope       string // "screen.region" dot notation
	Overlay     string // JSON
	Status      string // active|committed|discarded
}

type ComponentSchema struct {
	ID    int64
	AppID int64
	Name  string
	Props string // JSON: {"propName": "type", ...}
}
```

Add `Entry bool` field to the existing `Screen` struct.

- [ ] **Step 3: Verify schema loads**

Write a minimal test that opens an in-memory DB with the new schema:

```go
// internal/store/schema_test.go
func TestSchemaCreation(t *testing.T) {
	s := store.OpenMemory(t)
	defer s.Close()
	// If schema.sql has errors, OpenMemory panics
}
```

Run: `go test ./internal/store/ -run TestSchemaCreation -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/store/schema.sql internal/model/model.go internal/store/schema_test.go
git commit -m "feat: v2 schema — entities, experiments, component_schemas tables + entry column"
```

---

## Task 2: Store — Core CRUD

**Files:**
- Rewrite: `internal/store/store.go`
- Test: `internal/store/store_test.go`

This task adds CRUD methods for the 3 new tables + entry screen. Keep ALL existing store methods.

- [ ] **Step 1: Write failing tests for entity CRUD**

```go
// internal/store/store_test.go
func TestEntityCRUD(t *testing.T) {
	s := store.OpenMemory(t)
	appID := seedApp(t, s) // helper: inserts app, returns ID

	// Insert
	e := &model.Entity{AppID: appID, Name: "sarah", Type: "contact", Data: `{"name":"Sarah Chen","email":"sarah@co.com"}`}
	err := s.InsertEntity(e)
	require.NoError(t, err)
	require.NotZero(t, e.ID)

	// Get
	got, err := s.GetEntity(appID, "sarah")
	require.NoError(t, err)
	require.Equal(t, "contact", got.Type)

	// List
	all, err := s.ListEntities(appID)
	require.NoError(t, err)
	require.Len(t, all, 1)

	// Delete
	err = s.DeleteEntity(appID, "sarah")
	require.NoError(t, err)
	all, _ = s.ListEntities(appID)
	require.Len(t, all, 0)
}
```

- [ ] **Step 2: Implement entity CRUD**

```go
func (s *Store) InsertEntity(e *model.Entity) error {
	r, err := s.db().Exec(
		"INSERT INTO entities (app_id, name, type, data) VALUES (?, ?, ?, ?)",
		e.AppID, e.Name, e.Type, e.Data,
	)
	if err != nil { return err }
	e.ID, _ = r.LastInsertId()
	return nil
}

func (s *Store) GetEntity(appID int64, name string) (*model.Entity, error) {
	e := &model.Entity{}
	err := s.db().QueryRow(
		"SELECT id, app_id, name, type, data FROM entities WHERE app_id = ? AND name = ?",
		appID, name,
	).Scan(&e.ID, &e.AppID, &e.Name, &e.Type, &e.Data)
	return e, err
}

func (s *Store) ListEntities(appID int64) ([]model.Entity, error) {
	rows, err := s.db().Query("SELECT id, app_id, name, type, data FROM entities WHERE app_id = ?", appID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []model.Entity
	for rows.Next() {
		var e model.Entity
		if err := rows.Scan(&e.ID, &e.AppID, &e.Name, &e.Type, &e.Data); err != nil { return nil, err }
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) DeleteEntity(appID int64, name string) error {
	_, err := s.db().Exec("DELETE FROM entities WHERE app_id = ? AND name = ?", appID, name)
	return err
}
```

- [ ] **Step 3: Run entity tests**

Run: `go test ./internal/store/ -run TestEntityCRUD -v`
Expected: PASS

- [ ] **Step 4: Write failing tests for experiment CRUD**

```go
func TestExperimentCRUD(t *testing.T) {
	s := store.OpenMemory(t)
	appID := seedApp(t, s)

	exp := &model.Experiment{
		AppID: appID, Name: "compact_sidebar",
		Description: "Narrow sidebar", Scope: "app.main_nav",
		Overlay: `{"delivery":{"classes":["w-12"]}}`, Status: "active",
	}
	err := s.InsertExperiment(exp)
	require.NoError(t, err)

	got, err := s.GetExperiment(appID, "compact_sidebar")
	require.NoError(t, err)
	require.Equal(t, "app.main_nav", got.Scope)

	// Commit
	err = s.SetExperimentStatus(appID, "compact_sidebar", "committed")
	require.NoError(t, err)
	got, _ = s.GetExperiment(appID, "compact_sidebar")
	require.Equal(t, "committed", got.Status)

	// List
	all, err := s.ListExperiments(appID)
	require.NoError(t, err)
	require.Len(t, all, 1)
}
```

- [ ] **Step 5: Implement experiment CRUD**

Follow same pattern as entity CRUD. Methods: `InsertExperiment`, `GetExperiment`, `ListExperiments`, `SetExperimentStatus`, `DeleteExperiment`.

- [ ] **Step 6: Write failing tests for entry screen**

```go
func TestEntryScreen(t *testing.T) {
	s := store.OpenMemory(t)
	appID := seedApp(t, s)

	s.InsertScreen(&model.Screen{AppID: appID, Name: "inbox", Description: "Email list"})
	s.InsertScreen(&model.Screen{AppID: appID, Name: "settings", Description: "Config"})

	// Set entry
	err := s.SetEntryScreen(appID, "inbox")
	require.NoError(t, err)

	// Get entry
	entry, err := s.GetEntryScreen(appID)
	require.NoError(t, err)
	require.Equal(t, "inbox", entry)

	// Clear and set another
	err = s.SetEntryScreen(appID, "settings")
	require.NoError(t, err)
	entry, _ = s.GetEntryScreen(appID)
	require.Equal(t, "settings", entry)
}
```

- [ ] **Step 7: Implement entry screen methods**

```go
func (s *Store) SetEntryScreen(appID int64, name string) error {
	tx, _ := s.db().(*sql.DB).Begin()
	tx.Exec("UPDATE screens SET entry = 0 WHERE app_id = ?", appID)
	tx.Exec("UPDATE screens SET entry = 1 WHERE app_id = ? AND name = ?", appID, name)
	return tx.Commit()
}

func (s *Store) GetEntryScreen(appID int64) (string, error) {
	var name string
	err := s.db().QueryRow("SELECT name FROM screens WHERE app_id = ? AND entry = 1", appID).Scan(&name)
	return name, err
}
```

- [ ] **Step 8: Write + implement component schema CRUD**

Same pattern. Methods: `InsertComponentSchema`, `GetComponentSchema`, `ListComponentSchemas`.

- [ ] **Step 9: Run all store tests**

Run: `go test ./internal/store/ -v`
Expected: ALL PASS

- [ ] **Step 10: Commit**

```bash
git add internal/store/
git commit -m "feat: store CRUD for entities, experiments, component_schemas, entry screen"
```

---

## Task 3: Loader — Entity Resolution

**Files:**
- Rewrite: `internal/loader/loader.go` (add entities parsing)
- Create: `internal/loader/resolve.go` (entity $name resolver)
- Test: `internal/loader/resolve_test.go`
- Test: `internal/loader/loader_test.go`

- [ ] **Step 1: Write failing tests for $name resolution**

```go
// internal/loader/resolve_test.go
func TestResolveEntityRefs(t *testing.T) {
	pool := map[string]any{
		"sarah": map[string]any{"name": "Sarah Chen", "email": "sarah@co.com"},
		"deal":  map[string]any{"name": "Acme", "rep": "$sarah"},
	}

	// Simple scalar ref
	input := "$sarah"
	result, err := resolveValue(input, pool, nil)
	require.NoError(t, err)
	require.Equal(t, pool["sarah"], result)

	// Nested ref (recursive)
	input2 := "$deal"
	result2, err := resolveValue(input2, pool, nil)
	require.NoError(t, err)
	m := result2.(map[string]any)
	require.Equal(t, "Sarah Chen", m["rep"].(map[string]any)["name"])

	// Array with refs
	input3 := []any{"$sarah", "$deal"}
	result3, err := resolveValue(input3, pool, nil)
	require.NoError(t, err)
	arr := result3.([]any)
	require.Len(t, arr, 2)

	// Cycle detection
	cyclicPool := map[string]any{
		"a": map[string]any{"ref": "$b"},
		"b": map[string]any{"ref": "$a"},
	}
	_, err = resolveValue("$a", cyclicPool, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cycle")

	// Literal dollar string (not a ref)
	literal := "price is $50"
	result4, err := resolveValue(literal, pool, nil)
	require.NoError(t, err)
	require.Equal(t, "price is $50", result4) // not resolved — $50 not in pool
}
```

- [ ] **Step 2: Implement resolve.go**

```go
// internal/loader/resolve.go
package loader

import "fmt"

// resolveValue recursively resolves $name references in arbitrary data.
// $name is recognized only as a standalone string value matching a pool key.
// Strings like "price is $50" where "$50" is not a pool key pass through unchanged.
func resolveValue(v any, pool map[string]any, seen map[string]bool) (any, error) {
	if seen == nil { seen = make(map[string]bool) }
	switch val := v.(type) {
	case string:
		if len(val) > 1 && val[0] == '$' {
			name := val[1:]
			if _, ok := pool[name]; !ok {
				return val, nil // not a ref — literal string
			}
			if seen[name] {
				return nil, fmt.Errorf("entity reference cycle detected: $%s", name)
			}
			seen[name] = true
			resolved, err := resolveValue(pool[name], pool, seen)
			delete(seen, name)
			return resolved, err
		}
		return val, nil
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			r, err := resolveValue(v2, pool, seen)
			if err != nil { return nil, err }
			out[k] = r
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			r, err := resolveValue(v2, pool, seen)
			if err != nil { return nil, err }
			out[i] = r
		}
		return out, nil
	default:
		return val, nil
	}
}
```

- [ ] **Step 3: Run resolver tests**

Run: `go test ./internal/loader/ -run TestResolveEntityRefs -v`
Expected: PASS

- [ ] **Step 4: Write failing test for YAML with entities**

```go
// internal/loader/loader_test.go
func TestLoadWithEntities(t *testing.T) {
	yaml := `
app:
  name: test
  description: Test app
  data:
    contact:
      name: string
      email: string
  entities:
    sarah: { _type: contact, name: "Sarah Chen", email: "sarah@co.com" }
  fixtures:
    full:
      screen1:
        person: $sarah
  screens:
    - name: screen1
      description: Test screen
      entry: true
      context:
        person: contact
      state_machine:
        default:
          fixture: full
`
	s := store.OpenMemory(t)
	err := loader.LoadFromString(s, yaml)
	require.NoError(t, err)

	// Verify entity stored
	appID, _ := s.ResolveApp()
	entities, _ := s.ListEntities(appID)
	require.Len(t, entities, 1)
	require.Equal(t, "sarah", entities[0].Name)

	// Verify fixture has resolved data (not $sarah)
	fixtures, _ := s.ListFixtures(appID)
	require.Len(t, fixtures, 1)
	require.NotContains(t, fixtures[0].Data, "$sarah")
	require.Contains(t, fixtures[0].Data, "Sarah Chen")

	// Verify entry screen
	entry, _ := s.GetEntryScreen(appID)
	require.Equal(t, "screen1", entry)
}
```

- [ ] **Step 5: Implement entity loading in loader.go**

Add `Entities yaml.Node` field to `yamlApp` struct. In `Load()`:
1. Parse `entities:` block into `map[string]any` (each with `_type` key)
2. Store each entity via `s.InsertEntity()`
3. Build entity pool map
4. When processing fixtures, call `resolveValue()` on fixture data before `json.Marshal`
5. When processing screens, check for `entry: true` and call `s.SetEntryScreen()`

- [ ] **Step 6: Run full loader test**

Run: `go test ./internal/loader/ -run TestLoadWithEntities -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/loader/
git commit -m "feat: entity pool with \$name load-time resolution + entry screen loading"
```

---

## Task 4: Show — Spec Tree with Entities + Experiments

**Files:**
- Rewrite: `internal/show/show.go`
- Test: `internal/show/show_test.go`

- [ ] **Step 1: Add new fields to show types**

```go
// In Spec:
type Spec struct {
	App         App
	Screens     []Screen
	Fixtures    []Fixture
	Layouts     map[string][]string
	Entities    []Entity       // NEW
	Experiments []Experiment   // NEW
}

type Entity struct {
	Name string
	Type string
	Data any
}

type Experiment struct {
	Name        string
	Description string
	Scope       string
	Overlay     any
	Status      string
}
```

- [ ] **Step 2: Load entities and experiments in show.Load()**

Add queries to fetch from `entities` and `experiments` tables. Append to `Spec.Entities` and `Spec.Experiments`.

- [ ] **Step 3: Add `Entry` field to show.Screen**

```go
type Screen struct {
	// ... existing fields
	Entry bool // NEW
}
```

Load from DB: `SELECT ... entry FROM screens`.

- [ ] **Step 4: Write test**

```go
func TestShowLoadEntities(t *testing.T) {
	s := store.OpenMemory(t)
	// seed app + entities + experiments via store methods
	spec, err := show.Load(s.DB(), nil)
	require.NoError(t, err)
	require.Len(t, spec.Entities, 1)
	require.Len(t, spec.Experiments, 1)
}
```

- [ ] **Step 5: Run and verify**

Run: `go test ./internal/show/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/show/
git commit -m "feat: show.Load includes entities, experiments, entry screen"
```

---

## Task 5: Diff — Complete Coverage with Tests

**Files:**
- Rewrite: `internal/diff/diff.go`
- Create: `internal/diff/diff_test.go`

- [ ] **Step 1: Write test harness for existing entity types**

```go
// internal/diff/diff_test.go
func TestDiffScreens(t *testing.T) {
	base := &show.Spec{Screens: []show.Screen{{Name: "inbox", Description: "Email"}}}
	target := &show.Spec{Screens: []show.Screen{{Name: "inbox", Description: "Updated email"}}}
	changes := diff.Compare(base, target)
	require.Len(t, changes, 1)
	require.Equal(t, "~", changes[0].Op)
	require.Equal(t, "screen", changes[0].Entity)
}

func TestDiffScreenAdded(t *testing.T) {
	base := &show.Spec{}
	target := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	changes := diff.Compare(base, target)
	require.Len(t, changes, 1)
	require.Equal(t, "+", changes[0].Op)
}

func TestDiffScreenRemoved(t *testing.T) {
	base := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	target := &show.Spec{}
	changes := diff.Compare(base, target)
	require.Len(t, changes, 1)
	require.Equal(t, "-", changes[0].Op)
}
```

- [ ] **Step 2: Run tests against existing diff code**

Run: `go test ./internal/diff/ -v`
Expected: PASS (validates existing behavior before extending)

- [ ] **Step 3: Write tests for new entity types**

```go
func TestDiffDataTypes(t *testing.T) {
	base := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{"email": {"subject": "string"}}}}
	target := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{"email": {"subject": "string", "body": "string"}}}}
	changes := diff.Compare(base, target)
	require.NotEmpty(t, changes)
	found := false
	for _, c := range changes { if c.Entity == "data_type" { found = true } }
	require.True(t, found)
}

func TestDiffFixtures(t *testing.T) {
	base := &show.Spec{Fixtures: []show.Fixture{{Name: "f1", Data: map[string]any{"a": 1}}}}
	target := &show.Spec{Fixtures: []show.Fixture{{Name: "f1", Data: map[string]any{"a": 2}}}}
	changes := diff.Compare(base, target)
	require.NotEmpty(t, changes)
}

func TestDiffEntities(t *testing.T) {
	base := &show.Spec{Entities: []show.Entity{{Name: "sarah", Type: "contact"}}}
	target := &show.Spec{Entities: []show.Entity{{Name: "sarah", Type: "user"}}}
	changes := diff.Compare(base, target)
	require.NotEmpty(t, changes)
}

func TestDiffExperiments(t *testing.T) {
	base := &show.Spec{}
	target := &show.Spec{Experiments: []show.Experiment{{Name: "dark_mode", Scope: "app"}}}
	changes := diff.Compare(base, target)
	require.NotEmpty(t, changes)
}

func TestDiffDeliveryClasses(t *testing.T) {
	base := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", DeliveryClasses: []string{"p-4"}}}}}}
	target := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", DeliveryClasses: []string{"p-8"}}}}}}
	changes := diff.Compare(base, target)
	require.NotEmpty(t, changes)
}
```

- [ ] **Step 4: Implement new diff functions**

Add to `diff.go` — top-level diff functions:
- `diffDataTypes(cur, tgt map[string]map[string]string) []Change`
- `diffEnums(cur, tgt map[string][]string) []Change`
- `diffContexts(cur, tgt map[string]string, parent string) []Change`
- `diffFixtures(cur, tgt []show.Fixture) []Change`
- `diffEntities(cur, tgt []show.Entity) []Change`
- `diffExperiments(cur, tgt []show.Experiment) []Change`
- `diffLayouts(cur, tgt map[string][]string) []Change`

Update `diffRegions` to also compare these region-level fields:
- `DeliveryClasses []string` — slice diff (set comparison)
- `DiscoveryLayout []string` — slice diff
- `Ambient map[string]string` — map diff by local name
- `RegionData map[string]string` — map diff by field name

Update `diffScreens` to also compare these screen-level fields:
- `StateFixtures map[string]string` — map diff by state name
- `StateRegions map[string][]string` — map diff by state, value is set diff
- `Context map[string]string` — map diff by field name
- `Entry bool` — equality check

Each follows the existing map-diff pattern: build map by name, iterate for +/-/~.

- [ ] **Step 5: Wire new diff functions into Compare()**

```go
func Compare(current, target *show.Spec) []Change {
	var changes []Change
	// existing: app desc, screens, app regions
	changes = append(changes, diffDataTypes(current.App.DataTypes, target.App.DataTypes)...)
	changes = append(changes, diffEnums(current.App.Enums, target.App.Enums)...)
	changes = append(changes, diffContexts(current.App.Context, target.App.Context, "app")...)
	changes = append(changes, diffFixtures(current.Fixtures, target.Fixtures)...)
	changes = append(changes, diffEntities(current.Entities, target.Entities)...)
	changes = append(changes, diffExperiments(current.Experiments, target.Experiments)...)
	changes = append(changes, diffLayouts(current.Layouts, target.Layouts)...)
	// existing: diffScreens (which calls diffRegions recursively)
	return changes
}
```

- [ ] **Step 6: Run all diff tests**

Run: `go test ./internal/diff/ -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/diff/
git commit -m "feat: complete diff — all entity types with test coverage"
```

---

## Task 6: Validator — All Rules + fn Field

**Files:**
- Rewrite: `internal/validator/validator.go`
- Create: `internal/validator/validator_test.go`

- [ ] **Step 1: Add fn field to rule struct**

```go
type rule struct {
	id       string
	severity Severity
	query    string
	fn       func(*sql.DB) ([]Finding, error) // NEW: Go-level rules
	format   func(*sql.Rows) ([]Finding, error)
}
```

- [ ] **Step 2: Update Validate() dispatch**

```go
func Validate(db *sql.DB) ([]Finding, error) {
	var all []Finding
	for _, r := range rules {
		var findings []Finding
		var err error
		if r.fn != nil {
			findings, err = r.fn(db)
		} else {
			rows, qerr := db.Query(r.query)
			if qerr != nil { return nil, qerr }
			findings, err = r.format(rows)
			rows.Close()
		}
		if err != nil { return nil, err }
		for i := range findings {
			findings[i].Rule = r.id
			findings[i].Severity = r.severity
		}
		all = append(all, findings...)
	}
	return all, nil
}
```

- [ ] **Step 3: Write failing tests for new rules**

```go
func TestEntryScreenMissing(t *testing.T) {
	db := seedDB(t, `
		app: { name: test, description: test }
		screens:
		  - name: inbox
		    description: Email
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "entry-screen-missing")
	require.NotNil(t, found)
}

func TestFixtureExtendsCycle(t *testing.T) {
	db := seedDB(t, `
		app:
		  name: test
		  description: test
		  fixtures:
		    a:
		      extends: b
		      screen1: {}
		    b:
		      extends: a
		      screen1: {}
		  screens:
		    - name: screen1
		      description: test
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "fixture-extends-cycle")
	require.NotNil(t, found)
}

func TestLeafRegionNoContent(t *testing.T) {
	db := seedDB(t, `
		app:
		  name: test
		  description: test
		  screens:
		    - name: s1
		      description: test
		      regions:
		        - name: empty_leaf
		          description: Has no component and no children
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "leaf-region-no-content")
	require.NotNil(t, found)
}

func TestScreenUnreachable(t *testing.T) {
	db := seedDB(t, `
		app:
		  name: test
		  description: test
		  screens:
		    - name: home
		      entry: true
		      description: Home
		    - name: orphan
		      description: Nobody navigates here
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "screen-unreachable")
	require.NotNil(t, found)
	require.Contains(t, found.Message, "orphan")
}
```

- [ ] **Step 4: Implement new SQL rules**

```go
// entry-screen-missing (warning)
{id: "entry-screen-missing", severity: Warning,
 query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM screens WHERE entry = 1)",
 format: func(rows *sql.Rows) ([]Finding, error) {
   if rows.Next() { return []Finding{{Message: "no entry screen defined"}}, nil }
   return nil, nil
 }},

// entry-screen-multiple (error)
{id: "entry-screen-multiple", severity: Error,
 query: "SELECT name FROM screens WHERE entry = 1",
 format: func(rows *sql.Rows) ([]Finding, error) {
   var names []string
   for rows.Next() { var n string; rows.Scan(&n); names = append(names, n) }
   if len(names) > 1 { return []Finding{{Message: fmt.Sprintf("multiple entry screens: %s", strings.Join(names, ", "))}}, nil }
   return nil, nil
 }},

// leaf-region-no-content (warning)
{id: "leaf-region-no-content", severity: Warning,
 query: `SELECT r.name, ` + ownerCase + ` AS parent
   FROM regions r
   WHERE r.id NOT IN (SELECT r2.parent_id FROM regions r2 WHERE r2.parent_type = 'region')
   AND r.id NOT IN (SELECT c.entity_id FROM components c WHERE c.entity_type = 'region')`,
 format: ...},

// unreferenced-data-type (warning)
{id: "unreferenced-data-type", severity: Warning,
 query: `SELECT dt.name FROM data_types dt
   WHERE dt.name NOT IN (
     SELECT REPLACE(REPLACE(c.field_type, '?', ''), '[]', '') FROM contexts c
     UNION SELECT REPLACE(REPLACE(rd.field_type, '?', ''), '[]', '') FROM region_data rd
     UNION SELECT REPLACE(REPLACE(e.annotation, '?', ''), '[]', '') FROM events e WHERE e.annotation IS NOT NULL
   )`,
 format: ...},

// state-region-no-fixture (warning)
// state-without-fixture (warning, screen-level only)
```

- [ ] **Step 5: Implement Go-level rules**

```go
// fixture-extends-cycle (error)
{id: "fixture-extends-cycle", severity: Error,
 fn: func(db *sql.DB) ([]Finding, error) {
   rows, _ := db.Query("SELECT name, extends FROM fixtures WHERE extends != ''")
   defer rows.Close()
   graph := map[string]string{}
   for rows.Next() {
     var name, ext string
     rows.Scan(&name, &ext)
     graph[name] = ext
   }
   var findings []Finding
   for name := range graph {
     visited := map[string]bool{}
     cur := name
     for cur != "" {
       if visited[cur] {
         findings = append(findings, Finding{Message: fmt.Sprintf("fixture extends cycle: %s", cur)})
         break
       }
       visited[cur] = true
       cur = graph[cur]
     }
   }
   return findings, nil
 }},

// screen-unreachable (Go BFS)
{id: "screen-unreachable", severity: Warning,
 fn: func(db *sql.DB) ([]Finding, error) {
   // 1. Find entry screen
   // 2. Parse navigate() targets from transitions
   // 3. BFS from entry following navigate edges
   // 4. Report screens not in visited set
   // ... implementation
 }},
```

- [ ] **Step 6: Run all validator tests**

Run: `go test ./internal/validator/ -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/validator/
git commit -m "feat: validator v2 — fn field, entry screen, reachability, cycle detection, leaf content, unreferenced types"
```

---

## Task 7: CLI — New Commands

**Files:**
- Rewrite: `cmd/sft/main.go`

- [ ] **Step 1: Add entry screen CLI**

In `runSet()`, add handling for `--entry` flag:
```go
case "screen":
	if flagSet.Bool("entry", false, "mark as entry screen") {
		s.SetEntryScreen(appID, screenName)
	}
```

- [ ] **Step 2: Add experiment commands**

```go
case "experiment", "exp":
	switch sub {
	case "create": runExperimentCreate(s, rest)
	case "apply":  runExperimentApply(s, rest)
	case "commit": runExperimentCommit(s, rest)
	case "discard": runExperimentDiscard(s, rest)
	case "list":   runExperimentList(s, rest)
	}
```

Each handler: parse args, call store method, format output.

- [ ] **Step 3: Add clone command**

```go
case "clone":
	runClone(s, rest) // dispatches to store.CloneScreen or store.CloneRegion
```

- [ ] **Step 4: Integration test — full CLI flow**

```bash
# Test script
./sft init examples/gmail.sft.yaml
./sft set screen inbox --entry
./sft validate
./sft show --json | jq '.screens[0].entry'
./sft clone screen inbox inbox_v2
./sft experiment create dark_nav --scope app.main_nav --description "Dark sidebar"
./sft experiment list
```

- [ ] **Step 5: Commit**

```bash
git add cmd/sft/
git commit -m "feat: CLI — entry screen, experiment commands, clone"
```

---

## Task 8: Store — Clone (Deep Copy)

**Files:**
- Create: `internal/store/clone.go`
- Test: `internal/store/clone_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestCloneScreen(t *testing.T) {
	s := store.OpenMemory(t)
	loadFixture(t, s, "examples/gmail.sft.yaml")

	err := s.CloneScreen("inbox", "inbox_v2")
	require.NoError(t, err)

	// Original untouched
	origID, _ := s.ResolveScreen("inbox")
	require.NotZero(t, origID)

	// Clone exists with different ID
	cloneID, _ := s.ResolveScreen("inbox_v2")
	require.NotZero(t, cloneID)
	require.NotEqual(t, origID, cloneID)

	// Clone has same number of child regions
	origRegions := countRegions(t, s, "screen", origID)
	cloneRegions := countRegions(t, s, "screen", cloneID)
	require.Equal(t, origRegions, cloneRegions)

	// Clone has same number of transitions
	origTrans := countTransitions(t, s, "screen", origID)
	cloneTrans := countTransitions(t, s, "screen", cloneID)
	require.Equal(t, origTrans, cloneTrans)
}
```

- [ ] **Step 2: Implement CloneScreen**

```go
// internal/store/clone.go
func (s *Store) CloneScreen(srcName, dstName string) error {
	s.BeginTx()
	defer s.RollbackTx()

	srcID, err := s.ResolveScreen(srcName)
	if err != nil { return err }

	// 1. Copy screen row with new name
	// 2. Recursively copy all child regions (collectDescendantRegions pattern)
	// 3. For each region: copy events, transitions, tags, components, ambient_refs, region_data
	// 4. Copy state_fixtures and state_regions (fixture names shared, not copied)
	// 5. Remap parent IDs in the new tree

	return s.CommitTx()
}
```

The key: build an `oldID → newID` map as you insert cloned rows. Use it to set `parent_id` on child regions and `owner_id` on transitions/events.

- [ ] **Step 3: Run test**

Run: `go test ./internal/store/ -run TestCloneScreen -v`
Expected: PASS

- [ ] **Step 4: Implement CloneRegion (same pattern, different root)**

- [ ] **Step 5: Commit**

```bash
git add internal/store/clone.go internal/store/clone_test.go
git commit -m "feat: clone — deep-copy screen/region with ID remapping"
```

---

## Task 9: Experiments — Apply + Commit

**Files:**
- Create: `internal/show/experiment.go`
- Test: `internal/show/experiment_test.go`

- [ ] **Step 1: Write failing test for experiment overlay application**

```go
func TestApplyExperiment(t *testing.T) {
	spec := &show.Spec{
		Screens: []show.Screen{{
			Name: "dash",
			Regions: []show.Region{{
				Name: "kpi", DeliveryClasses: []string{"grid", "grid-cols-4"},
			}},
		}},
		Experiments: []show.Experiment{{
			Name: "compact", Scope: "dash.kpi", Status: "active",
			Overlay: map[string]any{"delivery": map[string]any{"classes": []any{"flex", "gap-6"}}},
		}},
	}

	applied, err := show.ApplyExperiment(spec, "compact")
	require.NoError(t, err)
	require.Equal(t, []string{"flex", "gap-6"}, applied.Screens[0].Regions[0].DeliveryClasses)

	// Original unchanged
	require.Equal(t, []string{"grid", "grid-cols-4"}, spec.Screens[0].Regions[0].DeliveryClasses)
}
```

- [ ] **Step 2: Implement ApplyExperiment**

```go
// internal/show/experiment.go
func ApplyExperiment(spec *Spec, name string) (*Spec, error) {
	// 1. Find experiment by name
	// 2. Parse scope (e.g., "dash.kpi" → screen "dash", region "kpi")
	// 3. Deep-copy the spec (to avoid mutating original)
	// 4. Find target region in the copy
	// 5. Shallow-merge overlay keys onto target (delivery, props, component, etc.)
	// 6. Return modified copy
}
```

- [ ] **Step 3: Run test**

Run: `go test ./internal/show/ -run TestApplyExperiment -v`
Expected: PASS

- [ ] **Step 4: Implement CommitExperiment in store**

```go
func (s *Store) CommitExperiment(appID int64, name string) error {
	// 1. Load experiment
	// 2. Parse scope → find target region ID
	// 3. Apply overlay values to actual region columns (delivery_classes, etc.)
	// 4. Set experiment status to "committed"
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/show/experiment.go internal/show/experiment_test.go
git commit -m "feat: experiment apply (view-time overlay) + commit (merge to base)"
```

---

## Task 10: Component Schemas + Fixture Validation

**Files:**
- Extend: `internal/validator/validator.go` (add Go-level rules)
- Test: `internal/validator/validator_test.go`

- [ ] **Step 1: Write failing test for component prop validation**

```go
func TestComponentPropValidation(t *testing.T) {
	db := seedDB(t, `
		app:
		  name: test
		  description: test
		  components:
		    button:
		      label: string
		      variant: string?
		  screens:
		    - name: s1
		      entry: true
		      description: test
		      regions:
		        - name: btn
		          description: test
		          component: button
		          props: '{"label": "Click", "unknown_prop": "bad"}'
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "component-prop-unknown")
	require.NotNil(t, found)
}
```

- [ ] **Step 2: Implement component prop validation rule**

Go-level rule: query `components` + `component_schemas`, parse both JSON, check that every prop key in the component exists in the schema.

- [ ] **Step 3: Write failing test for fixture key validation**

```go
func TestFixtureKeysMismatch(t *testing.T) {
	db := seedDB(t, `
		app:
		  name: test
		  description: test
		  fixtures:
		    f1:
		      s1:
		        wrong_key: "data"
		  screens:
		    - name: s1
		      entry: true
		      description: test
		      context:
		        correct_key: string
		      state_machine:
		        default:
		          fixture: f1
	`)
	findings, _ := validator.Validate(db)
	found := findRule(findings, "fixture-keys-mismatch")
	require.NotNil(t, found)
}
```

- [ ] **Step 4: Implement fixture key validation rule**

Go-level rule: for each state_fixture binding, load the fixture data JSON, load the owner screen's context fields, check that fixture top-level keys under the screen name match context field names.

- [ ] **Step 5: Implement remaining Phase 2 validator rules**

Three more Go-level rules from the spec:

```go
// entity-ref-unresolved (error) — $name in fixture data that doesn't resolve
// Only fires if entities: block exists. Checks fixture JSON for $-prefixed strings
// that don't match any entity name.

// entity-type-mismatch (warning) — entity _type doesn't match a declared data type
// Query entities table, check each type against data_types.name

// experiment-scope-invalid (error) — experiment scope doesn't resolve to existing screen/region
// Parse scope "screen.region" dot notation, verify screen exists, verify region exists under screen
```

- [ ] **Step 6: Run all validator tests**

Run: `go test ./internal/validator/ -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/validator/
git commit -m "feat: component prop validation + fixture key mismatch + entity/experiment validators"
```

---

## Task 11: Drop Frontend + Final Integration

**Files:**
- Delete: `web/` directory
- Modify: `web/embed.go` (remove or stub)

- [ ] **Step 1: Remove web/ directory**

```bash
rm -rf web/
```

- [ ] **Step 2: Stub embed.go or remove view command**

Either remove the `view` command from the CLI or stub `embed.go` to return an error ("viewer not available — rebuild in progress").

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: ALL PASS

- [ ] **Step 4: Build binary**

```bash
go build -o ./sft ./cmd/sft
```

- [ ] **Step 5: Integration test with golden examples**

```bash
rm -rf .sft && ./sft init examples/gmail.sft.yaml
./sft validate
./sft diff examples/gmail.sft.yaml  # should show no changes
./sft set screen inbox --entry
./sft validate  # entry-screen-missing should be gone
./sft clone screen inbox inbox_v2
./sft show --json | jq '.screens | length'  # should be 6 (5 + clone)
./sft experiment create dark --scope app.main_nav --description "dark nav"
./sft experiment list --json
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: SFT v2 — clean rebuild complete"
```

---

## Parallelism Guide

```
SEQUENTIAL: Task 1 → Task 2 → Task 3 (schema → store → loader)

PARALLEL after Task 3:
  Agent A: Task 4 (show)
  Agent B: Task 5 (diff + tests)
  Agent C: Task 6 (validator + tests)

SEQUENTIAL after parallel group:
  Task 7 (CLI) — needs show + diff + validator
  Task 8 (clone) — needs store
  Task 9 (experiments) — needs show + store

LAST:
  Task 10 (component schemas + fixture validation) — needs validator + store
  Task 11 (drop frontend + integration) — needs everything
```

Maximum parallelism: **3 agents** working simultaneously on Tasks 4, 5, 6.
