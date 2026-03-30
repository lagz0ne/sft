# Remove Flows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the entire flow subsystem so screens + state machines are the sole behavioral model.

**Architecture:** Surgical deletion across 7 Go packages, 1 CLI entrypoint, 4 web source files, and all YAML examples. The `internal/flow/` package is deleted entirely. DB schema drops both flow tables and adds cleanup DDL for existing databases. No new code is written — this is pure removal.

**Tech Stack:** Go, SQLite, TypeScript/React (TanStack Router)

---

### Task 1: Delete `internal/flow/` package

**Files:**
- Delete: `internal/flow/parse.go`
- Delete: `internal/flow/parse_test.go`

- [ ] **Step 1: Delete the flow package directory**

```bash
rm -rf internal/flow/
```

- [ ] **Step 2: Verify deletion**

```bash
ls internal/flow/ 2>&1
```

Expected: `No such file or directory`

- [ ] **Step 3: Commit**

```bash
git add -A internal/flow/
git commit -m "chore: delete internal/flow package (flow parser)"
```

---

### Task 2: Remove flow structs from model

**Files:**
- Modify: `internal/model/model.go:52-70`

- [ ] **Step 1: Remove Flow and FlowStep structs**

In `internal/model/model.go`, delete lines 52–70 (the `Flow` and `FlowStep` structs plus the blank line between them):

```go
// DELETE this block:
type Flow struct {
	ID          int64  `json:"id"`
	AppID       int64  `json:"app_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OnEvent     string `json:"on_event,omitempty"`
	Sequence    string `json:"sequence"`
}

type FlowStep struct {
	ID       int64  `json:"id"`
	FlowID   int64  `json:"flow_id"`
	Position int    `json:"position"`
	Raw      string `json:"raw"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	History  int    `json:"history"`
	Data     string `json:"data,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/model/
```

Expected: success (model has no internal deps on these types)

- [ ] **Step 3: Commit**

```bash
git add internal/model/model.go
git commit -m "chore: remove Flow and FlowStep model structs"
```

---

### Task 3: Remove flow tables from DB schema and add cleanup DDL

**Files:**
- Modify: `internal/store/schema.sql:53-71` (flows + flow_steps tables)
- Modify: `internal/store/schema.sql:192` (idx_flow_steps_type_name index)

- [ ] **Step 1: Remove the `flows` and `flow_steps` CREATE TABLE statements**

In `internal/store/schema.sql`, delete lines 53–71:

```sql
-- DELETE this block:
CREATE TABLE IF NOT EXISTS flows (
  id          INTEGER PRIMARY KEY,
  app_id      INTEGER NOT NULL REFERENCES apps(id),
  name        TEXT NOT NULL UNIQUE,
  description TEXT,
  on_event    TEXT,
  sequence    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS flow_steps (
  id       INTEGER PRIMARY KEY,
  flow_id  INTEGER NOT NULL REFERENCES flows(id),
  position INTEGER NOT NULL,
  raw      TEXT NOT NULL,
  type     TEXT NOT NULL CHECK(type IN ('screen','region','event','back','action','activate')),
  name     TEXT NOT NULL,
  history  INTEGER NOT NULL DEFAULT 0,
  data     TEXT
);
```

- [ ] **Step 2: Remove the flow_steps index**

Delete the line (was line 192 before, will have shifted):

```sql
-- DELETE this line:
CREATE INDEX IF NOT EXISTS idx_flow_steps_type_name ON flow_steps(type, name);
```

- [ ] **Step 3: Add cleanup DDL at the end of the schema**

Append before the views section (before `-- Cross-cutting views`):

```sql
-- Cleanup: remove legacy flow tables from existing databases
DROP TABLE IF EXISTS flow_steps;
DROP TABLE IF EXISTS flows;
```

- [ ] **Step 4: Verify schema loads**

```bash
go test ./internal/store/ -run TestOpen -count=1 -v
```

Expected: PASS (store opens with updated schema)

- [ ] **Step 5: Commit**

```bash
git add internal/store/schema.sql
git commit -m "chore: drop flow tables from schema, add cleanup DDL"
```

---

### Task 4: Remove flow store methods and flow import

**Files:**
- Modify: `internal/store/store.go:13` (remove flow import)
- Modify: `internal/store/store.go:380-413` (InsertFlow, IsEvent stays, InsertFlowStep)
- Modify: `internal/store/store.go:1015-1024` (DeleteFlow)
- Modify: `internal/store/store.go:1236-1246` (RenameFlow)

- [ ] **Step 1: Remove the `flow` package import**

In `internal/store/store.go`, remove line 13:

```go
// DELETE this line:
	"github.com/lagz0ne/sft/internal/flow"
```

- [ ] **Step 2: Remove InsertFlow method**

Delete the `InsertFlow` method (lines 380–396):

```go
// DELETE this block:
func (s *Store) InsertFlow(f *model.Flow) error {
	res, err := s.db().Exec("INSERT INTO flows (app_id, name, description, on_event, sequence) VALUES (?, ?, ?, ?, ?)",
		f.AppID, f.Name, f.Description, f.OnEvent, f.Sequence)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()

	// Parse sequence into flow_steps
	steps := flow.ParseSequence(f.Sequence, f.ID, s)
	for i := range steps {
		if err := s.InsertFlowStep(&steps[i]); err != nil {
			return fmt.Errorf("flow step %d: %w", i+1, err)
		}
	}
	return nil
}
```

- [ ] **Step 3: Remove InsertFlowStep method**

Delete the `InsertFlowStep` method (lines 405–413):

```go
// DELETE this block:
func (s *Store) InsertFlowStep(fs *model.FlowStep) error {
	res, err := s.db().Exec("INSERT INTO flow_steps (flow_id, position, raw, type, name, history, data) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fs.FlowID, fs.Position, fs.Raw, fs.Type, fs.Name, fs.History, fs.Data)
	if err != nil {
		return err
	}
	fs.ID, _ = res.LastInsertId()
	return nil
}
```

Note: `IsEvent` (lines 398–403) stays — it's used independently.

- [ ] **Step 4: Remove DeleteFlow method**

Delete the `DeleteFlow` method (lines 1015–1024):

```go
// DELETE this block:
func (s *Store) DeleteFlow(name string) error {
	var flowID int64
	err := s.db().QueryRow("SELECT id FROM flows WHERE name = ?", name).Scan(&flowID)
	if err != nil {
		return fmt.Errorf("flow %q not found", name)
	}
	s.db().Exec("DELETE FROM flow_steps WHERE flow_id = ?", flowID)
	s.db().Exec("DELETE FROM flows WHERE id = ?", flowID)
	return nil
}
```

- [ ] **Step 5: Remove RenameFlow method**

Delete the `RenameFlow` method (lines 1236–1246):

```go
// DELETE this block:
func (s *Store) RenameFlow(old, newName string) error {
	res, err := s.db().Exec("UPDATE flows SET name = ? WHERE name = ?", newName, old)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("flow %q not found", old)
	}
	return nil
}
```

- [ ] **Step 6: Verify compilation**

```bash
go build ./internal/store/
```

Expected: success

- [ ] **Step 7: Commit**

```bash
git add internal/store/store.go
git commit -m "chore: remove flow store methods (InsertFlow, DeleteFlow, RenameFlow)"
```

---

### Task 5: Remove flow store test

**Files:**
- Modify: `internal/store/store_test.go:115-141`

- [ ] **Step 1: Remove TestInsertFlowPopulatesSteps**

Delete the `TestInsertFlowPopulatesSteps` function (lines 115–141):

```go
// DELETE this block:
func TestInsertFlowPopulatesSteps(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Home", Description: "h"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	f := &model.Flow{AppID: a.ID, Name: "TestFlow", Sequence: "Home → action → Home(H)"}
	if err := s.InsertFlow(f); err != nil {
		t.Fatal(err)
	}

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM flow_steps WHERE flow_id = ?", f.ID).Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 flow steps, got %d", count)
	}

	// First step should be classified as "screen"
	var stepType string
	s.DB.QueryRow("SELECT type FROM flow_steps WHERE flow_id = ? AND position = 1", f.ID).Scan(&stepType)
	if stepType != "screen" {
		t.Errorf("step 1 type = %q, want screen", stepType)
	}
}
```

- [ ] **Step 2: Run store tests**

```bash
go test ./internal/store/ -v -count=1
```

Expected: PASS (all remaining tests pass)

- [ ] **Step 3: Commit**

```bash
git add internal/store/store_test.go
git commit -m "chore: remove flow store test"
```

---

### Task 6: Remove flow from show package

**Files:**
- Modify: `internal/show/show.go:18` (Flows field on Spec)
- Modify: `internal/show/show.go:89-105` (FlowStep, Flow types)
- Modify: `internal/show/show.go:170-189` (flow loading in Load())
- Modify: `internal/show/show.go:506-523` (loadFlowSteps function)
- Modify: `internal/show/show.go:580-594` (flow rendering in Render())

- [ ] **Step 1: Remove Flows field from Spec struct**

In `internal/show/show.go` line 18, delete:

```go
// DELETE this line:
	Flows    []Flow                 `json:"flows,omitempty"`
```

- [ ] **Step 2: Remove FlowStep and Flow type definitions**

Delete lines 89–105:

```go
// DELETE this block:
type FlowStep struct {
	Position int    `json:"position"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	History  int    `json:"history,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Flow struct {
	ID          int64      `json:"id"`
	Ref         string     `json:"ref"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	OnEvent     string     `json:"on_event,omitempty"`
	Sequence    string     `json:"sequence"`
	Steps       []FlowStep `json:"steps,omitempty"`
}
```

- [ ] **Step 3: Remove flow loading block from Load()**

Delete lines 170–189 (the `// Flows` section):

```go
// DELETE this block:
	// Flows
	frows, err := db.Query("SELECT id, name, description, on_event, sequence FROM flows ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	for frows.Next() {
		var f Flow
		var flowID int64
		var desc, onEvent sql.NullString
		if err := frows.Scan(&flowID, &f.Name, &desc, &onEvent, &f.Sequence); err != nil {
			return nil, fmt.Errorf("scan flow: %w", err)
		}
		f.ID = flowID
		f.Ref = fmt.Sprintf("@f%d", flowID)
		f.Description = desc.String
		f.OnEvent = onEvent.String
		f.Steps, _ = loadFlowSteps(db, flowID)
		spec.Flows = append(spec.Flows, f)
	}
```

- [ ] **Step 4: Remove loadFlowSteps function**

Delete lines 506–523:

```go
// DELETE this block:
func loadFlowSteps(db *sql.DB, flowID int64) ([]FlowStep, error) {
	rows, err := db.Query(`SELECT position, type, name, history, data FROM flow_steps WHERE flow_id = ? ORDER BY position`, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var steps []FlowStep
	for rows.Next() {
		var s FlowStep
		var data sql.NullString
		if err := rows.Scan(&s.Position, &s.Type, &s.Name, &s.History, &data); err != nil {
			return nil, fmt.Errorf("scan flow_step: %w", err)
		}
		s.Data = data.String
		steps = append(steps, s)
	}
	return steps, nil
}
```

- [ ] **Step 5: Remove flow rendering from Render()**

Delete lines 580–594:

```go
// DELETE this block:
	if len(spec.Flows) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "flows:\n")
		for _, f := range spec.Flows {
			fmt.Fprintf(w, "  %s %s", f.Ref, f.Name)
			if f.OnEvent != "" {
				fmt.Fprintf(w, " (on %s)", f.OnEvent)
			}
			fmt.Fprintln(w)
			if f.Description != "" {
				fmt.Fprintf(w, "    %s\n", f.Description)
			}
			fmt.Fprintf(w, "    %s\n", f.Sequence)
		}
	}
```

- [ ] **Step 6: Remove unused `fmt` import if needed**

Check if `fmt` is still used (it will be — Render still uses it). Also check if `database/sql` import's `sql.NullString` is still used (it will be for other loading functions).

```bash
go build ./internal/show/
```

Expected: success

- [ ] **Step 7: Commit**

```bash
git add internal/show/show.go
git commit -m "chore: remove flow types and loading from show package"
```

---

### Task 7: Remove flow show tests

**Files:**
- Modify: `internal/show/show_test.go:182-207` (TestFlowSteps)
- Modify: `internal/show/show_test.go:217-218,252-261` (flow refs in TestRefs)

- [ ] **Step 1: Remove TestFlowSteps**

Delete the entire `TestFlowSteps` function (lines 182–207):

```go
// DELETE this block:
func TestFlowSteps(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	addScreen(t, s, app.ID, "inbox", "email inbox")

	// Insert flow and steps via raw SQL (no store helper for flows)
	s.DB.Exec(`INSERT INTO flows(app_id, name, sequence) VALUES(?, 'read', 'inbox → thread → inbox')`, app.ID)
	s.DB.Exec(`INSERT INTO flow_steps(flow_id, position, raw, type, name, history, data) VALUES(1, 1, 'inbox', 'screen', 'inbox', 0, '')`)
	s.DB.Exec(`INSERT INTO flow_steps(flow_id, position, raw, type, name, history, data) VALUES(1, 2, 'thread', 'screen', 'thread', 0, NULL)`)
	s.DB.Exec(`INSERT INTO flow_steps(flow_id, position, raw, type, name, history, data) VALUES(1, 3, 'inbox', 'screen', 'inbox', 1, '')`)

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow := spec.Flows[0]
	if len(flow.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(flow.Steps))
	}
	if flow.Steps[0].Type != "screen" || flow.Steps[0].Name != "inbox" {
		t.Fatalf("unexpected first step: %+v", flow.Steps[0])
	}
	if flow.Steps[2].History != 1 {
		t.Fatal("expected history=1 on step 3")
	}
}
```

- [ ] **Step 2: Remove flow SQL and assertions from TestRefs**

In `TestRefs` (starts at line 209), remove the flow SQL insert (line 217-218) and the flow ref assertions (lines 252-261):

```go
// DELETE this line (~217-218):
	// Add a flow via raw SQL
	s.DB.Exec(`INSERT INTO flows(app_id, name, sequence) VALUES(?, 'read_flow', 'inbox → settings')`, app.ID)

// DELETE this block (~252-261):
	// Flow refs
	if len(spec.Flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(spec.Flows))
	}
	if spec.Flows[0].Ref != "@f1" {
		t.Fatalf("expected @f1, got %s", spec.Flows[0].Ref)
	}
	if spec.Flows[0].ID != 1 {
		t.Fatalf("expected flow ID 1, got %d", spec.Flows[0].ID)
	}
```

- [ ] **Step 3: Run show tests**

```bash
go test ./internal/show/ -v -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/show/show_test.go
git commit -m "chore: remove flow show tests"
```

---

### Task 8: Remove flow query

**Files:**
- Modify: `internal/query/query.go:12` (flows named query)
- Modify: `internal/query/query.go:30` (flows in error message)
- Modify: `internal/query/query.go:51-56` (Steps function)

- [ ] **Step 1: Remove "flows" from namedQueries**

Delete line 12:

```go
// DELETE this line:
	"flows":    "SELECT id, name, description, on_event, sequence FROM flows",
```

- [ ] **Step 2: Remove "flows" from error message**

On line 30, update the error message to remove "flows":

```go
// BEFORE:
return nil, fmt.Errorf("unknown query %q (available: screens, events, flows, tags, regions, types, enums, fixtures, contexts, attachments, layouts, or raw SELECT)", input)

// AFTER:
return nil, fmt.Errorf("unknown query %q (available: screens, events, tags, regions, types, enums, fixtures, contexts, attachments, layouts, or raw SELECT)", input)
```

- [ ] **Step 3: Remove Steps function**

Delete lines 51–56:

```go
// DELETE this block:
// Steps returns parsed flow steps for a given flow name.
func Steps(db *sql.DB, flowName string) ([]map[string]any, error) {
	return execQuery(db, `SELECT fs.position, fs.raw, fs.type, fs.name, fs.history, fs.data
		FROM flow_steps fs JOIN flows f ON f.id = fs.flow_id
		WHERE f.name = ? ORDER BY fs.position`, flowName)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/query/
```

Expected: success

- [ ] **Step 5: Commit**

```bash
git add internal/query/query.go
git commit -m "chore: remove flow query and Steps function"
```

---

### Task 9: Remove flow diagram generator

**Files:**
- Modify: `internal/diagram/diagram.go:165-233` (Flow function)

- [ ] **Step 1: Remove the Flow function**

Delete lines 165–233:

```go
// DELETE this block:
// Flow generates a Mermaid flowchart for a named flow.
func Flow(db *sql.DB, name string) (string, error) {
	rows, err := db.Query(`SELECT fs.position, fs.type, fs.name, fs.data, fs.history
		FROM flow_steps fs JOIN flows f ON f.id = fs.flow_id
		WHERE f.name = ? ORDER BY fs.position`, name)
	if err != nil {
		return "", fmt.Errorf("diagram flow: %w", err)
	}
	defer rows.Close()

	type step struct {
		pos     int
		typ     string
		name    string
		data    string
		history bool
	}
	var steps []step
	for rows.Next() {
		var s step
		var data sql.NullString
		var hist int
		if err := rows.Scan(&s.pos, &s.typ, &s.name, &data, &hist); err != nil {
			return "", err
		}
		s.data = data.String
		s.history = hist > 0
		steps = append(steps, s)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(steps) == 0 {
		return "", fmt.Errorf("no flow found for %q", name)
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")

	for i, s := range steps {
		id := fmt.Sprintf("s%d", i)
		label := s.name
		if s.data != "" {
			label += "\\n{" + s.data + "}"
		}

		switch s.typ {
		case "screen":
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		case "region":
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		case "event":
			fmt.Fprintf(&b, "  %s{{%s}}\n", id, label)
		case "back":
			fmt.Fprintf(&b, "  %s>Back]\n", id)
		case "action", "activate":
			fmt.Fprintf(&b, "  %s(%s)\n", id, label)
		default:
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		}

		if i > 0 {
			prev := fmt.Sprintf("s%d", i-1)
			fmt.Fprintf(&b, "  %s --> %s\n", prev, id)
		}
	}

	return b.String(), nil
}
```

- [ ] **Step 2: Check for unused imports after removal**

The `Flow` function uses `database/sql`, `fmt`, `strings`. Check if `database/sql` is still needed by remaining functions (`States`, `Nav`). Both use `*sql.DB` — so all imports stay.

```bash
go build ./internal/diagram/
```

Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/diagram/diagram.go
git commit -m "chore: remove flow diagram generator"
```

---

### Task 10: Remove flow from loader (import, export, flat format)

**Files:**
- Modify: `internal/loader/loader.go:23` (yamlFile.Flows field)
- Modify: `internal/loader/loader.go:36` (yamlApp.Flows field)
- Modify: `internal/loader/loader.go:52` (yamlScreen.flows field)
- Modify: `internal/loader/loader.go:107-112` (yamlFlow struct)
- Modify: `internal/loader/loader.go:130,136-137` (flat format flow decoding)
- Modify: `internal/loader/loader.go:266-279` (flow insertion in Load)
- Modify: `internal/loader/loader.go:699-705` (screen-level flows parsing)
- Modify: `internal/loader/loader.go:832-874` (decodeFlatFlows, decodeFlatFlowsMapping)
- Modify: `internal/loader/loader.go:1032-1046` (ExportFlat flows section)
- Modify: `internal/loader/loader.go:1113` (Export flows reference)
- Modify: `internal/loader/loader.go:1492-1506` (exportFlows function)

- [ ] **Step 1: Remove yamlFlow struct**

Delete lines 107–112:

```go
// DELETE this block:
type yamlFlow struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	On          string `yaml:"on,omitempty"`
	Sequence    string `yaml:"sequence"`
}
```

- [ ] **Step 2: Remove Flows fields from yamlFile, yamlApp, yamlScreen**

In yamlFile (line 23), delete:
```go
// DELETE:
	Flows   yaml.Node           `yaml:"flows,omitempty"`
```

In yamlApp (line 36), delete:
```go
// DELETE:
	Flows          []yamlFlow                   `yaml:"flows,omitempty"`
```

In yamlScreen (line 52), delete:
```go
// DELETE:
	flows        []yamlFlow        // unexported: screen-level flows (flat format only)
```

- [ ] **Step 3: Remove flat format flow decoding in Load()**

At line 130 (scalar app handling), remove lines 136–137:
```go
// DELETE:
		if err := decodeFlatFlows(&f.Flows, &app); err != nil {
			return fmt.Errorf("decode flows in %s: %w", path, err)
		}
```

- [ ] **Step 4: Remove screen-level flows collection and flow insertion**

Delete lines 266–279:
```go
// DELETE this block:
	// Collect screen-level flows (flat format puts flows inside screens)
	for _, sc := range app.Screens {
		app.Flows = append(app.Flows, sc.flows...)
	}

	// Flows
	for _, fl := range app.Flows {
		if err := s.InsertFlow(&model.Flow{
			AppID: a.ID, Name: fl.Name, Description: fl.Description,
			OnEvent: fl.On, Sequence: fl.Sequence,
		}); err != nil {
			return fmt.Errorf("flow %s: %w", fl.Name, err)
		}
	}
```

- [ ] **Step 5: Remove screen-level flows case in flat screen parser**

In the flat screen parser (~line 699), delete the `"flows"` case:
```go
// DELETE this block:
		case "flows":
			flows, err := decodeFlatFlowsMapping(val)
			if err != nil {
				return sc, fmt.Errorf("flows: %w", err)
			}
			// Screen-level flows get merged into app-level flows
			sc.flows = flows
```

- [ ] **Step 6: Remove decodeFlatFlows and decodeFlatFlowsMapping functions**

Delete lines 832–874:
```go
// DELETE this block:
// decodeFlatFlows parses top-level flows as a mapping node.
func decodeFlatFlows(node *yaml.Node, app *yamlApp) error {
	if node == nil || node.Kind == 0 {
		return nil
	}
	if node.Kind == yaml.MappingNode {
		flows, err := decodeFlatFlowsMapping(node)
		if err != nil {
			return err
		}
		app.Flows = append(app.Flows, flows...)
	}
	return nil
}

// decodeFlatFlowsMapping parses a flow mapping (name → {description, sequence}).
func decodeFlatFlowsMapping(node *yaml.Node) ([]yamlFlow, error) {
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}
	var flows []yamlFlow
	for i := 0; i < len(node.Content)-1; i += 2 {
		flowName := node.Content[i].Value
		flowDef := node.Content[i+1]
		f := yamlFlow{Name: flowName}
		if flowDef.Kind == yaml.MappingNode {
			for j := 0; j < len(flowDef.Content)-1; j += 2 {
				k := flowDef.Content[j].Value
				v := flowDef.Content[j+1]
				switch k {
				case "description":
					f.Description = v.Value
				case "sequence":
					f.Sequence = v.Value
				case "on":
					f.On = v.Value
				}
			}
		}
		flows = append(flows, f)
	}
	return flows, nil
}
```

- [ ] **Step 7: Remove flows from ExportFlat**

Delete lines 1032–1046:
```go
// DELETE this block:
	if len(spec.Flows) > 0 {
		flowsNode := seqNode()
		for _, f := range spec.Flows {
			item := mapNode()
			appendPair(item, "name", f.Name)
			if f.Description != "" {
				appendPair(item, "description", f.Description)
			}
			if f.OnEvent != "" {
				appendPair(item, "on", f.OnEvent)
			}
			appendPair(item, "sequence", f.Sequence)
			flowsNode.Content = append(flowsNode.Content, item)
		}
		appendKey(root, "flows", flowsNode)
	}
```

- [ ] **Step 8: Remove flows from Export**

On line 1113, delete:
```go
// DELETE:
		Flows:       exportFlows(spec.Flows),
```

- [ ] **Step 9: Remove exportFlows function**

Delete lines 1492–1506:
```go
// DELETE this block:
func exportFlows(flows []show.Flow) []yamlFlow {
	if len(flows) == 0 {
		return nil
	}
	var out []yamlFlow
	for _, f := range flows {
		out = append(out, yamlFlow{
			Name:        f.Name,
			Description: f.Description,
			On:          f.OnEvent,
			Sequence:    f.Sequence,
		})
	}
	return out
}
```

- [ ] **Step 10: Verify compilation**

```bash
go build ./internal/loader/
```

Expected: success

- [ ] **Step 11: Commit**

```bash
git add internal/loader/loader.go
git commit -m "chore: remove flow import, export, and flat format support from loader"
```

---

### Task 11: Remove flow from loader tests

**Files:**
- Modify: `internal/loader/loader_test.go:47-51` (testYAML flows section)
- Modify: `internal/loader/loader_test.go:119-127` (flow round-trip assertions)
- Modify: `internal/loader/loader_test.go:519-535` (TestFlowStepsAfterImport)
- Modify: `internal/loader/loader_test.go:1281-1284` (golden gmail flow check)
- Modify: `internal/loader/loader_test.go:1348-1349` (golden round-trip flow check)
- Modify: `internal/loader/loader_test.go:1639-1641,1662-1663` (TestExportFlat flows)

- [ ] **Step 1: Remove flows from testYAML constant**

Remove the flows section from the testYAML string (lines 47–51):
```yaml
# DELETE these lines from the YAML string:
  flows:
    - name: Landing
      description: User lands and clicks CTA
      on: page-load
      sequence: "Home → Detail → [Back] → Home(H)"
```

- [ ] **Step 2: Remove flow round-trip assertions**

Delete the flow comparison block from the round-trip test (lines 119–127):
```go
// DELETE this block:
	if len(spec1.Flows) != len(spec2.Flows) {
		t.Fatalf("flows: %d vs %d", len(spec1.Flows), len(spec2.Flows))
	}
	for i, f := range spec1.Flows {
		f2 := spec2.Flows[i]
		if f.Name != f2.Name || f.Sequence != f2.Sequence || f.OnEvent != f2.OnEvent {
			t.Errorf("flow %d mismatch: %+v vs %+v", i, f, f2)
		}
	}
```

- [ ] **Step 3: Remove TestFlowStepsAfterImport**

Delete the entire function (lines 519–535):
```go
// DELETE this block:
func TestFlowStepsAfterImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM flow_steps").Scan(&count)
	if count == 0 {
		t.Error("expected flow steps to be populated after import")
	}

	var flowID int64
	s.DB.QueryRow("SELECT id FROM flows WHERE name = 'Landing'").Scan(&flowID)
	s.DB.QueryRow("SELECT COUNT(*) FROM flow_steps WHERE flow_id = ?", flowID).Scan(&count)
	if count != 4 {
		t.Errorf("Landing flow: expected 4 steps, got %d", count)
	}
}
```

- [ ] **Step 4: Remove golden gmail flow check**

Delete lines 1281–1284:
```go
// DELETE this block:
	// Check flows
	if len(spec.Flows) < 3 {
		t.Errorf("flows = %d, want >= 3", len(spec.Flows))
	}
```

- [ ] **Step 5: Remove golden round-trip flow check**

Delete lines 1348–1349:
```go
// DELETE this block:
	if len(spec1.Flows) != len(spec2.Flows) {
		t.Fatalf("flows: %d vs %d", len(spec1.Flows), len(spec2.Flows))
	}
```

- [ ] **Step 6: Remove flows from TestExportFlat**

Remove the flows YAML from the test constant (~lines 1639–1641):
```yaml
# DELETE these lines from the YAML string:
  flows:
    - name: Landing
      sequence: "Home → Home"
```

Remove the flows assertion (~lines 1662–1663):
```go
// DELETE this block:
	if !strings.Contains(out, "flows:") {
		t.Errorf("expected flows: key, got:\n%s", out)
	}
```

- [ ] **Step 7: Run loader tests**

```bash
go test ./internal/loader/ -v -count=1
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/loader/loader_test.go
git commit -m "chore: remove flow test data and assertions from loader tests"
```

---

### Task 12: Remove flow CLI commands

**Files:**
- Modify: `cmd/sft/main.go` (multiple locations)

- [ ] **Step 1: Remove "flow" from usage text**

Update the usage string to remove flow references. Specifically:

Line 105: Remove "flows" from description:
```go
// BEFORE:
const usage = `sft — behavioral spec tool for UI screens, regions, events, flows, and components.
// AFTER:
const usage = `sft — behavioral spec tool for UI screens, regions, events, and components.
```

Line 113: Remove "flows" from query list:
```go
// BEFORE:
  sft query screens              # list screens, regions, events, flows
// AFTER:
  sft query screens              # list screens, regions, events
```

Line 119: Remove "flows" and "steps <flow>" from query type list:
```go
// BEFORE:
  query  <type>                    screens | regions | events | flows | tags | types | enums | fixtures | contexts | attachments | steps <flow>
// AFTER:
  query  <type>                    screens | regions | events | tags | types | enums | fixtures | contexts | attachments
```

Line 132: Remove "add flow" usage:
```go
// DELETE:
  add flow <name> <sequence> [--description <d>] [--on <event>]
```

Line 142: Remove "flow" from rename usage:
```go
// BEFORE:
  rename <screen|region|flow|type|enum|fixture> <old> <new> [--in <parent>]
// AFTER:
  rename <screen|region|type|enum|fixture> <old> <new> [--in <parent>]
```

Line 176: Remove "diagram flow" usage:
```go
// DELETE:
  diagram flow <name>             flow sequence
```

- [ ] **Step 2: Remove "flow" from query dispatch**

Line 206: Remove "flows" from die message:
```go
// BEFORE:
die("usage: sft query <screens|events|states|flows|tags|regions|types|enums|fixtures|contexts|attachments|layouts|SELECT ...>")
// AFTER:
die("usage: sft query <screens|events|states|tags|regions|types|enums|fixtures|contexts|attachments|layouts|SELECT ...>")
```

Lines 219–225: Remove the `"steps"` query case that calls `query.Steps()`:
```go
// DELETE this block:
		if name == "steps" {
			die("usage: sft query steps <flow-name>")
		}
```

Line 225: Remove "flows" from the name check (if it's part of a validation list). Update the validation to exclude "flows":
```go
// Check the line that validates query names and remove "flows" from the list
```

- [ ] **Step 3: Remove "add flow" case**

Delete the `"flow"` case in the add command switch (lines 348–361):
```go
// DELETE this block:
	case "flow":
		if len(args) < 3 {
			die("usage: sft add flow <name> <sequence> [--description <d>] [--on <event>]")
		}
		f := &model.Flow{
			AppID:       appID,
			Name:        args[1],
			Sequence:    args[2],
			Description: flagVal(args, "--description"),
			OnEvent:     flagVal(args, "--on"),
		}
		must(s.InsertFlow(f))
		ok("flow %s", f.Name)
```

Update the error at end of add switch (line 455) to remove "flow":
```go
// BEFORE:
die("unknown entity %q (use: app, screen, region, event, transition, tag, flow, type, enum, context, field, ambient, fixture, state-fixture, state-region)", entity)
// AFTER:
die("unknown entity %q (use: app, screen, region, event, transition, tag, type, enum, context, field, ambient, fixture, state-fixture, state-region)", entity)
```

- [ ] **Step 4: Remove "rename flow" case**

Delete the `"flow"` case in the rename switch (lines 639–640):
```go
// DELETE:
	case "flow":
		must(s.RenameFlow(old, newName))
```

Update the error (line 648) to remove "flow":
```go
// BEFORE:
die("rename supports: screen, region, flow, type, enum, fixture")
// AFTER:
die("rename supports: screen, region, type, enum, fixture")
```

- [ ] **Step 5: Remove "rm flow" case**

Delete the `"flow"` case in the rm switch (lines 723–725):
```go
// DELETE:
	case "flow":
		must(s.DeleteFlow(name))
		ok("deleted flow %s", name)
```

Update the error (line 800) to remove "flow":
```go
// BEFORE:
die("rm supports: screen, region, event, transition, tag, flow, type, enum, context, field, ambient, fixture, state-fixture, state-region")
// AFTER:
die("rm supports: screen, region, event, transition, tag, type, enum, context, field, ambient, fixture, state-fixture, state-region")
```

- [ ] **Step 6: Remove "diagram flow" case**

Delete the `"flow"` case in the diagram switch (lines 1072–1074):
```go
// DELETE:
	case "flow":
		need(args, 2, "sft diagram flow <name>")
		out, err = diagram.Flow(s.DB, args[1])
```

Update the error (line 1076) to remove "flow":
```go
// BEFORE:
die("unknown diagram type %q (available: states, nav, flow)", args[0])
// AFTER:
die("unknown diagram type %q (available: states, nav)", args[0])
```

Also update the usage check (line 1062):
```go
// BEFORE:
die("usage: sft diagram <states <name> | nav | flow <name>>")
// AFTER:
die("usage: sft diagram <states <name> | nav>")
```

- [ ] **Step 7: Remove unused imports if any**

Check if `model` or `diagram` imports are still needed (they are — used elsewhere). Check if `query` import is still needed (yes — `query.Run` and `query.States` are still used).

```bash
go build ./cmd/sft/
```

Expected: success

- [ ] **Step 8: Commit**

```bash
git add cmd/sft/main.go
git commit -m "chore: remove all flow CLI commands (add, query, rename, rm, diagram)"
```

---

### Task 13: Remove flows from YAML examples

**Files:**
- Modify: `examples/gmail.sft.yaml` (~lines 234-247)
- Modify: `examples/spotify.sft.yaml` (~lines 221-232)
- Modify: `examples/stripe.sft.yaml` (~lines 416-442)
- Modify: `examples/linear.sft.yaml` (~lines 216-228)
- Modify: any other example files with flows sections

Also check golden examples:
- Modify: `examples/golden/*.sft.yaml` if they contain flows

- [ ] **Step 1: Remove flows sections from all example YAML files**

For each example file, find and delete the entire `flows:` section (key + all child items). The section looks like:

```yaml
  flows:
    - name: ...
      description: ...
      sequence: "..."
```

Use grep to find all files:
```bash
grep -rn "flows:" examples/
```

Remove the flows section from each file found.

- [ ] **Step 2: Verify examples still load**

```bash
for f in examples/*.sft.yaml examples/golden/*.sft.yaml; do
  echo "Loading $f..."
  go run ./cmd/sft/ import "$f" 2>&1 && echo "OK" || echo "FAIL: $f"
  rm -f .sft/db  # clean up for next
done
```

Expected: all OK

- [ ] **Step 3: Commit**

```bash
git add examples/
git commit -m "chore: remove flows sections from all example YAML specs"
```

---

### Task 14: Remove flow from web frontend

**Files:**
- Delete: `web/apps/web/src/routes/flows.$name.tsx`
- Delete: `web/apps/web/src/components/flow-step-strip.tsx`
- Modify: `web/apps/web/src/lib/types.ts:51-65,76` (FlowStep, Flow interfaces, flows on Spec)
- Modify: `web/apps/web/src/routes/screens.$name.tsx:18-20,64-72` (flowsThrough references)
- Modify: `web/apps/web/src/routes/playground.tsx` (flow mode, flow references)
- Modify: `web/apps/web/src/components/dock.tsx` (FlowRail, StepNav, flow props)
- Regenerate: `web/apps/web/src/routeTree.gen.ts` (auto-generated by TanStack Router)

- [ ] **Step 1: Delete flow route and component files**

```bash
rm web/apps/web/src/routes/flows.\$name.tsx
rm web/apps/web/src/components/flow-step-strip.tsx
```

- [ ] **Step 2: Remove FlowStep and Flow interfaces from types.ts**

In `web/apps/web/src/lib/types.ts`, delete the FlowStep interface (lines 51–57), Flow interface (lines 59–65), and the `flows?` field from Spec (line 76):

```typescript
// DELETE FlowStep interface:
export interface FlowStep {
  position: number
  type: 'screen' | 'region' | 'event' | 'back' | 'action' | 'activate'
  name: string
  history?: number
  data?: string
}

// DELETE Flow interface:
export interface Flow {
  name: string
  description?: string
  on_event?: string
  sequence: string
  steps?: FlowStep[]
}

// DELETE from Spec interface:
  flows?: Flow[]
```

- [ ] **Step 3: Remove flow references from screens.$name.tsx**

Remove the `flowsThrough` computation and its rendering block:

```typescript
// DELETE:
const flowsThrough = spec?.flows?.filter(f =>
  f.steps?.some(s => s.type === 'screen' && s.name === name)
) ?? []

// DELETE the "Flows through this screen" section in the JSX
```

- [ ] **Step 4: Remove flow mode from playground.tsx**

This is the most involved web change. Remove:
- `FlowStep` import
- `mode` and `flow` from search params type
- `findScreenForStep` function
- All `flow`/`flowMode`/`flowScreen`/`flowSteps` variables
- Flow-related dock props (`flowMode`, `flowSteps`, `flowIndex`, `onFlowStep`, `flows`, `onFlow`, `hasFlows`)
- Mode toggle logic

The playground should only have screen mode. Simplify the search params to remove `mode`, `flow`, `step`.

- [ ] **Step 5: Remove flow mode from dock.tsx**

Remove:
- `FlowRail` component
- `StepNav` component (if flow-specific)
- Flow-related props from `Dock` component (`flowMode`, `flowSteps`, `flowIndex`, `onFlowStep`, `flows`, `onFlow`, `mode`, `onModeToggle`, `hasFlows`)
- Mode toggle buttons (screen/flow toggle)
- All conditional rendering based on `isFlow`

The dock should only show screen picker and state controls.

- [ ] **Step 6: Remove flow from index redirect**

In `web/apps/web/src/routes/index.tsx` line 4, remove `flow`, `step`, and `mode` from the redirect search params:

```typescript
// BEFORE:
component: () => <Navigate to="/playground" search={{ screen: '', state: '', mode: 'screen', flow: '', step: 0, set: 'wireframe', layout: '', width: 0 }} />,
// AFTER:
component: () => <Navigate to="/playground" search={{ screen: '', state: '', set: 'wireframe', layout: '', width: 0 }} />,
```

- [ ] **Step 7: Regenerate route tree**

```bash
cd web/apps/web && npx tsr generate
```

This will regenerate `routeTree.gen.ts` without the flows route.

- [ ] **Step 8: Verify frontend builds**

```bash
cd web/apps/web && npm run build
```

Expected: success with no errors

- [ ] **Step 9: Commit**

```bash
git add web/
git commit -m "chore: remove flow UI (route, components, dock mode, types)"
```

---

### Task 15: Full verification

- [ ] **Step 1: Run all Go tests**

```bash
go test ./... -count=1
```

Expected: all PASS

- [ ] **Step 2: Run vet**

```bash
go vet ./...
```

Expected: clean

- [ ] **Step 3: Build CLI**

```bash
go build ./cmd/sft/
```

Expected: success

- [ ] **Step 4: Build frontend**

```bash
cd web/apps/web && npm run build
```

Expected: success

- [ ] **Step 5: Verify example import**

```bash
./sft import examples/gmail.sft.yaml && ./sft show
```

Expected: spec loads, no flow section in output

- [ ] **Step 6: Commit (if any fixups needed)**

```bash
git add -A
git commit -m "fix: address any remaining flow references"
```
