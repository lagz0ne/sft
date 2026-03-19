# State Machine Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `state_machine:` map format to SFT's YAML import/export with dual-format support, 4 new validation rules, and migrate all 6 example specs.

**Architecture:** The loader gets a new YAML parser for `state_machine:` maps that converts to the same `[]model.Transition` tuples the DB already stores. No schema changes. No model changes. The validator gets 4 new SQL rules. Export produces the new format. The 6 example YAML files are converted to the new format with `lower_snake` naming.

**Tech Stack:** Go, SQLite, gopkg.in/yaml.v3 (yaml.Node for dynamic parsing)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/loader/statemachine.go` | Create | Parse `state_machine:` yaml.Node → `[]model.Transition` |
| `internal/loader/statemachine_test.go` | Create | Unit tests for state machine parsing |
| `internal/loader/loader.go` | Modify | Add `StateMachine yaml.Node` to yamlScreen/yamlRegion, dual-format dispatch |
| `internal/loader/loader_test.go` | Modify | Add integration tests for new format import + round-trip |
| `internal/validator/validator.go` | Modify | Add 4 new rules: undeclared_state, dead_end, guard_ambiguity, cycle_in_emit |
| `internal/validator/validator_test.go` | Create | Test each new validation rule |
| `examples/gmail.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |
| `examples/linear.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |
| `examples/bank.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |
| `examples/stripe.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |
| `examples/shopify.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |
| `examples/docs.sft.yaml` | Rewrite | Convert to `state_machine:` + `lower_snake` |

---

## Chunk 1: State Machine Parser

### Task 1: Parse `state_machine:` yaml.Node into transitions

**Files:**
- Create: `internal/loader/statemachine.go`
- Create: `internal/loader/statemachine_test.go`

The parser reads a `yaml.Node` (MappingNode) where keys are state names and values are state definitions containing an `on:` map of transitions. It produces `[]model.Transition` — the same tuples the DB already stores.

**Value forms to handle:**
- String target: `check_email: selecting` → `{OnEvent: "check_email", FromState: "start", ToState: "selecting"}`
- Stay: `check_email: .` → `{OnEvent: "check_email", FromState: "start", ToState: "start"}`
- Action shorthand: `select_email: navigate(thread_view)` → `{OnEvent: "select_email", FromState: "start", Action: "navigate(thread_view)"}`
- Object: `send_reply: { to: start, action: emit(reply_sent) }` → `{OnEvent: "send_reply", FromState: "expanded", ToState: "start", Action: "emit(reply_sent)"}`
- Guarded array: `submit: [{ guard: "valid", to: saving }, ...]` → multiple transitions with same OnEvent+FromState, guard stored in Action as `guard(valid)`
- Terminal state: `empty: {}` → no transitions, state name recorded for validation

- [ ] **Step 1: Write the failing test for simple string target**

```go
// internal/loader/statemachine_test.go
package loader

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func parseNode(t *testing.T, input string) yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(input), &n); err != nil {
		t.Fatal(err)
	}
	// yaml.Unmarshal wraps in a document node
	return *n.Content[0]
}

func TestParseStateMachine_SimpleTarget(t *testing.T) {
	node := parseNode(t, `
start:
  on:
    check_email: selecting
selecting:
  on:
    escape: start
`)
	transitions, states, err := ParseStateMachine(node)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[0] != "start" {
		t.Errorf("first state = %q, want start", states[0])
	}
	if len(transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(transitions))
	}
	tr := transitions[0]
	if tr.OnEvent != "check_email" || tr.FromState != "start" || tr.ToState != "selecting" {
		t.Errorf("transition 0: %+v", tr)
	}
	tr = transitions[1]
	if tr.OnEvent != "escape" || tr.FromState != "selecting" || tr.ToState != "start" {
		t.Errorf("transition 1: %+v", tr)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/loader/ -run TestParseStateMachine_SimpleTarget -v`
Expected: FAIL — `ParseStateMachine` undefined

- [ ] **Step 3: Implement minimal ParseStateMachine**

```go
// internal/loader/statemachine.go
package loader

import (
	"fmt"
	"strings"

	"github.com/lagz0ne/sft/internal/model"
	"gopkg.in/yaml.v3"
)

// ParseStateMachine converts a state_machine: yaml.Node (MappingNode) into
// transitions and an ordered list of state names.
// The first state in the returned list is the initial state.
func ParseStateMachine(node yaml.Node) ([]model.Transition, []string, error) {
	if node.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("state_machine: expected mapping, got %d", node.Kind)
	}

	var transitions []model.Transition
	var states []string

	// Iterate state entries: key=state_name, value=state_def
	for i := 0; i < len(node.Content); i += 2 {
		stateName := node.Content[i].Value
		stateDef := node.Content[i+1]
		states = append(states, stateName)

		// Empty state (terminal): `empty: {}`
		if stateDef.Kind == yaml.MappingNode && len(stateDef.Content) == 0 {
			continue
		}
		if stateDef.Kind == yaml.ScalarNode && stateDef.Value == "" {
			continue
		}

		// Look for `on:` key in the state definition
		onNode := findKey(stateDef, "on")
		if onNode == nil {
			continue
		}

		// Parse each transition in `on:`
		trs, err := parseOnBlock(stateName, onNode)
		if err != nil {
			return nil, nil, fmt.Errorf("state %s: %w", stateName, err)
		}
		transitions = append(transitions, trs...)
	}

	return transitions, states, nil
}

func findKey(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func parseOnBlock(fromState string, node *yaml.Node) ([]model.Transition, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("on: expected mapping")
	}

	var transitions []model.Transition
	for i := 0; i < len(node.Content); i += 2 {
		event := node.Content[i].Value
		value := node.Content[i+1]

		trs, err := parseTransitionValue(fromState, event, value)
		if err != nil {
			return nil, fmt.Errorf("event %s: %w", event, err)
		}
		transitions = append(transitions, trs...)
	}
	return transitions, nil
}

func parseTransitionValue(fromState, event string, value *yaml.Node) ([]model.Transition, error) {
	switch value.Kind {
	case yaml.ScalarNode:
		return parseScalarTransition(fromState, event, value.Value)

	case yaml.MappingNode:
		return parseObjectTransition(fromState, event, value)

	case yaml.SequenceNode:
		// Guarded: array of objects
		var all []model.Transition
		for _, item := range value.Content {
			trs, err := parseGuardedTransition(fromState, event, item)
			if err != nil {
				return nil, err
			}
			all = append(all, trs...)
		}
		return all, nil

	default:
		return nil, fmt.Errorf("unexpected node kind %d", value.Kind)
	}
}

func parseScalarTransition(fromState, event, value string) ([]model.Transition, error) {
	t := model.Transition{OnEvent: event, FromState: fromState}

	switch {
	case value == ".":
		// Stay in current state
		t.ToState = fromState
	case isAction(value):
		// Action shorthand: navigate(...) or emit(...)
		t.Action = value
	default:
		// Target state name
		t.ToState = value
	}

	return []model.Transition{t}, nil
}

func parseObjectTransition(fromState, event string, node *yaml.Node) ([]model.Transition, error) {
	t := model.Transition{OnEvent: event, FromState: fromState}

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1].Value
		switch key {
		case "to":
			if val == "." {
				t.ToState = fromState
			} else {
				t.ToState = val
			}
		case "action":
			t.Action = val
		case "guard":
			// Store guard in action field as guard(description)
			if t.Action != "" {
				t.Action = fmt.Sprintf("guard(%s), %s", val, t.Action)
			} else {
				t.Action = fmt.Sprintf("guard(%s)", val)
			}
		}
	}
	return []model.Transition{t}, nil
}

func parseGuardedTransition(fromState, event string, node *yaml.Node) ([]model.Transition, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("guarded transition: expected mapping")
	}
	return parseObjectTransition(fromState, event, node)
}

func isAction(s string) bool {
	return strings.HasPrefix(s, "navigate(") || strings.HasPrefix(s, "emit(")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/loader/ -run TestParseStateMachine_SimpleTarget -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loader/statemachine.go internal/loader/statemachine_test.go
git commit -m "feat: add state_machine parser — simple string targets"
```

---

### Task 2: Test and handle all value forms

**Files:**
- Modify: `internal/loader/statemachine_test.go`

- [ ] **Step 1: Write tests for stay, action, object, guarded, terminal**

```go
func TestParseStateMachine_Stay(t *testing.T) {
	node := parseNode(t, `
selecting:
  on:
    check_email: .
`)
	transitions, _, _ := ParseStateMachine(node)
	if transitions[0].ToState != "selecting" {
		t.Errorf("stay: ToState = %q, want selecting", transitions[0].ToState)
	}
}

func TestParseStateMachine_ActionOnly(t *testing.T) {
	node := parseNode(t, `
start:
  on:
    select_email: navigate(thread_view)
`)
	transitions, _, _ := ParseStateMachine(node)
	tr := transitions[0]
	if tr.Action != "navigate(thread_view)" {
		t.Errorf("action = %q", tr.Action)
	}
	if tr.ToState != "" {
		t.Errorf("to_state should be empty, got %q", tr.ToState)
	}
}

func TestParseStateMachine_ObjectForm(t *testing.T) {
	node := parseNode(t, `
expanded:
  on:
    send_reply: { to: start, action: "emit(reply_sent)" }
`)
	transitions, _, _ := ParseStateMachine(node)
	tr := transitions[0]
	if tr.ToState != "start" || tr.Action != "emit(reply_sent)" {
		t.Errorf("object: %+v", tr)
	}
}

func TestParseStateMachine_Guarded(t *testing.T) {
	node := parseNode(t, `
start:
  on:
    submit:
      - { guard: "valid", to: saving }
      - { guard: "invalid", to: . }
`)
	transitions, _, _ := ParseStateMachine(node)
	if len(transitions) != 2 {
		t.Fatalf("expected 2 guarded transitions, got %d", len(transitions))
	}
	if transitions[0].ToState != "saving" {
		t.Errorf("guard 1: to = %q", transitions[0].ToState)
	}
	if transitions[1].ToState != "start" {
		t.Errorf("guard 2: to = %q, want start (stay)", transitions[1].ToState)
	}
	if !strings.Contains(transitions[0].Action, "guard(valid)") {
		t.Errorf("guard 1: action = %q", transitions[0].Action)
	}
}

func TestParseStateMachine_Terminal(t *testing.T) {
	node := parseNode(t, `
start:
  on:
    done: end
end: {}
`)
	transitions, states, _ := ParseStateMachine(node)
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[1] != "end" {
		t.Errorf("second state = %q, want end", states[1])
	}
}

func TestParseStateMachine_InitialState(t *testing.T) {
	node := parseNode(t, `
browsing:
  on:
    click: selecting
selecting:
  on:
    escape: browsing
`)
	_, states, _ := ParseStateMachine(node)
	if states[0] != "browsing" {
		t.Errorf("initial state = %q, want browsing", states[0])
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/loader/ -run TestParseStateMachine -v`
Expected: ALL PASS

- [ ] **Step 3: Fix any failures, then commit**

```bash
git add internal/loader/statemachine_test.go
git commit -m "test: cover all state_machine value forms"
```

---

### Task 3: Wire state_machine parser into the loader (dual-format)

**Files:**
- Modify: `internal/loader/loader.go`
- Modify: `internal/loader/loader_test.go`

- [ ] **Step 1: Add StateMachine field to yamlScreen and yamlRegion**

In `loader.go`, add the `yaml.Node` field to both structs:

```go
type yamlScreen struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Tags        []string         `yaml:"tags,omitempty"`
	Component   string           `yaml:"component,omitempty"`
	Props       string           `yaml:"props,omitempty"`
	OnActions   string           `yaml:"on_actions,omitempty"`
	Visible     string           `yaml:"visible,omitempty"`
	Regions     []yamlRegion     `yaml:"regions,omitempty"`
	States      []yamlTransition `yaml:"states,omitempty"`
	StateMachine *yaml.Node      `yaml:"state_machine,omitempty"`
}

type yamlRegion struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Tags        []string         `yaml:"tags,omitempty"`
	Component   string           `yaml:"component,omitempty"`
	Props       string           `yaml:"props,omitempty"`
	OnActions   string           `yaml:"on_actions,omitempty"`
	Visible     string           `yaml:"visible,omitempty"`
	Events      []string         `yaml:"events,omitempty"`
	Regions     []yamlRegion     `yaml:"regions,omitempty"`
	States      []yamlTransition `yaml:"states,omitempty"`
	StateMachine *yaml.Node      `yaml:"state_machine,omitempty"`
}
```

- [ ] **Step 2: Add dual-format dispatch in Load() and insertRegion()**

In `Load()`, replace the screen transition insertion block (lines 140-147) with:

```go
		// Transitions: dual-format (state_machine: map OR states: list)
		if err := insertTransitions(s, "screen", screen.ID, sc.States, sc.StateMachine, sc.Name); err != nil {
			return err
		}
```

In `insertRegion()`, replace the region transition insertion block (lines 191-198) with:

```go
	if err := insertTransitions(s, "region", region.ID, r.States, r.StateMachine, r.Name); err != nil {
		return err
	}
```

Add the helper:

```go
func insertTransitions(s *store.Store, ownerType string, ownerID int64, legacy []yamlTransition, sm *yaml.Node, ownerName string) error {
	if len(legacy) > 0 && sm != nil {
		return fmt.Errorf("%s %s: cannot have both states: and state_machine:", ownerType, ownerName)
	}

	if sm != nil {
		transitions, _, err := ParseStateMachine(*sm)
		if err != nil {
			return fmt.Errorf("state_machine in %s %s: %w", ownerType, ownerName, err)
		}
		for _, tr := range transitions {
			tr.OwnerType = ownerType
			tr.OwnerID = ownerID
			if err := s.InsertTransition(&tr); err != nil {
				return fmt.Errorf("transition on %s in %s %s: %w", tr.OnEvent, ownerType, ownerName, err)
			}
		}
		return nil
	}

	for _, t := range legacy {
		if err := s.InsertTransition(&model.Transition{
			OwnerType: ownerType, OwnerID: ownerID,
			OnEvent: t.On, FromState: t.From, ToState: t.To, Action: t.Action,
		}); err != nil {
			return fmt.Errorf("transition on %s in %s %s: %w", t.On, ownerType, ownerName, err)
		}
	}
	return nil
}
```

- [ ] **Step 3: Write integration test for new format import**

```go
// Add to loader_test.go
const testStateMachineYAML = `app:
  name: TestApp
  description: A test application
  screens:
    - name: Home
      description: Landing page
      regions:
        - name: Hero
          description: Hero section
          events: [cta_click]
      state_machine:
        start:
          on:
            cta_click: active
        active:
          on:
            reset: start
`

func TestStateMachineImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testStateMachineYAML)

	// Check transitions were created
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM transitions WHERE owner_type = 'screen'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 transitions, got %d", count)
	}

	// Check from_state is set correctly
	var from string
	s.DB.QueryRow("SELECT from_state FROM transitions WHERE on_event = 'cta_click'").Scan(&from)
	if from != "start" {
		t.Errorf("from_state = %q, want start", from)
	}
}

func TestDualFormatError(t *testing.T) {
	yaml := `app:
  name: Bad
  description: Both formats
  screens:
    - name: Home
      description: Landing
      states:
        - on: click
          from: idle
      state_machine:
        start:
          on:
            click: active
`
	s := mustStore(t)
	tmp := t.TempDir() + "/bad.sft.yaml"
	os.WriteFile(tmp, []byte(yaml), 0o644)
	err := Load(s, tmp)
	if err == nil {
		t.Error("expected error for dual-format, got nil")
	}
}
```

- [ ] **Step 4: Run all loader tests**

Run: `go test ./internal/loader/ -v`
Expected: ALL PASS (including existing tests — legacy format still works)

- [ ] **Step 5: Commit**

```bash
git add internal/loader/loader.go internal/loader/loader_test.go
git commit -m "feat: wire state_machine parser into loader with dual-format support"
```

---

### Task 4: Update export to produce new format

**Files:**
- Modify: `internal/loader/loader.go`
- Modify: `internal/loader/loader_test.go`

The export should produce `state_machine:` format. Transitions grouped by `from_state`, each state's transitions under an `on:` key.

- [ ] **Step 1: Write test for new format export**

```go
func TestStateMachineExportFormat(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testStateMachineYAML)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatal(err)
	}
	exported := buf.String()

	// Should contain state_machine: not states:
	if strings.Contains(exported, "states:") {
		t.Error("export should use state_machine:, not states:")
	}
	if !strings.Contains(exported, "state_machine:") {
		t.Error("export missing state_machine: key")
	}
	if !strings.Contains(exported, "start:") {
		t.Error("export missing start state")
	}
}
```

- [ ] **Step 2: Implement export with state_machine format**

Replace `exportTransitions` in `loader.go` with a function that groups transitions by `from_state` and produces the map format. Update `exportScreens` and `exportRegions` to call it.

The export YAML structure for state machines needs to use `yaml.Node` for ordered map output:

```go
func exportStateMachine(transitions []show.Transition) *yaml.Node {
	if len(transitions) == 0 {
		return nil
	}

	// Group transitions by from_state
	type stateEntry struct {
		name        string
		transitions []show.Transition
	}
	var ordered []stateEntry
	seen := map[string]int{}

	for _, t := range transitions {
		from := t.FromState
		if from == "" {
			from = "start"
		}
		if idx, ok := seen[from]; ok {
			ordered[idx].transitions = append(ordered[idx].transitions, t)
		} else {
			seen[from] = len(ordered)
			ordered = append(ordered, stateEntry{name: from, transitions: []show.Transition{t}})
		}
	}

	// Also collect target states that aren't source states (terminal states)
	for _, t := range transitions {
		if t.ToState != "" && t.ToState != "." {
			if _, ok := seen[t.ToState]; !ok {
				seen[t.ToState] = len(ordered)
				ordered = append(ordered, stateEntry{name: t.ToState})
			}
		}
	}

	// Build yaml.Node (MappingNode)
	sm := &yaml.Node{Kind: yaml.MappingNode}
	for _, entry := range ordered {
		// State name key
		sm.Content = append(sm.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: entry.name})

		if len(entry.transitions) == 0 {
			// Terminal state: empty mapping
			sm.Content = append(sm.Content, &yaml.Node{Kind: yaml.MappingNode})
			continue
		}

		// State definition with on: block
		stateDef := &yaml.Node{Kind: yaml.MappingNode}
		onKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "on"}
		onMap := &yaml.Node{Kind: yaml.MappingNode}

		for _, tr := range entry.transitions {
			onMap.Content = append(onMap.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: tr.OnEvent})
			onMap.Content = append(onMap.Content, transitionValueNode(tr, entry.name))
		}

		stateDef.Content = append(stateDef.Content, onKey, onMap)
		sm.Content = append(sm.Content, stateDef)
	}
	return sm
}

func transitionValueNode(t show.Transition, currentState string) *yaml.Node {
	hasTo := t.ToState != ""
	hasAction := t.Action != ""

	// Strip guard prefix from action for export
	action := t.Action
	guardStr := ""
	if strings.HasPrefix(action, "guard(") {
		// Parse guard(desc), rest
		idx := strings.Index(action, ")")
		if idx > 0 {
			guardStr = action[6:idx]
			action = strings.TrimPrefix(action[idx+1:], ", ")
		}
	}

	// Simple cases: scalar value
	if !hasAction && hasTo && guardStr == "" {
		val := t.ToState
		if val == currentState {
			val = "."
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: val}
	}
	if hasAction && !hasTo && guardStr == "" {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: action}
	}

	// Object form: { to: x, action: y, guard: z }
	obj := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
	if guardStr != "" {
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "guard"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: guardStr})
	}
	if hasTo {
		val := t.ToState
		if val == currentState {
			val = "."
		}
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "to"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val})
	}
	if action != "" {
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "action"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: action})
	}
	return obj
}
```

Update `yamlScreen` and `yamlRegion` export structs to include a `StateMachine *yaml.Node` field, and update `exportScreens`/`exportRegions` to call `exportStateMachine` instead of `exportTransitions`.

- [ ] **Step 3: Run all tests including round-trip**

Run: `go test ./internal/loader/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/loader/loader.go
git commit -m "feat: export produces state_machine format"
```

---

## Chunk 2: Validation Rules

### Task 5: Add `undeclared_state` validation rule

**Files:**
- Modify: `internal/validator/validator.go`
- Create: `internal/validator/validator_test.go`

This rule checks: every `to_state` in transitions must also appear as a `from_state` somewhere in the same owner's transitions (meaning it's a declared state). The initial state (first from_state by rowid) is exempt.

- [ ] **Step 1: Write the failing test**

```go
// internal/validator/validator_test.go
package validator

import (
	"database/sql"
	"testing"

	"github.com/lagz0ne/sft/internal/store"
	"github.com/lagz0ne/sft/internal/model"
)

func setupTestDB(t *testing.T) (*store.Store, *sql.DB) {
	t.Helper()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, s.DB
}

func TestUndeclaredState(t *testing.T) {
	s, db := setupTestDB(t)
	app := &model.App{Name: "test", Description: "test"}
	s.InsertApp(app)
	screen := &model.Screen{AppID: app.ID, Name: "home", Description: "home"}
	s.InsertScreen(screen)

	// Transition to "nonexistent" state — which is never a from_state
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "click", FromState: "start", ToState: "nonexistent",
	})

	findings, err := Validate(db)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if f.Rule == "undeclared-state" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected undeclared-state finding")
	}
}

func TestUndeclaredState_ValidSpec(t *testing.T) {
	s, db := setupTestDB(t)
	app := &model.App{Name: "test", Description: "test"}
	s.InsertApp(app)
	screen := &model.Screen{AppID: app.ID, Name: "home", Description: "home"}
	s.InsertScreen(screen)

	// Both states appear as from_state — valid
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "click", FromState: "start", ToState: "active",
	})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "reset", FromState: "active", ToState: "start",
	})

	findings, _ := Validate(db)
	for _, f := range findings {
		if f.Rule == "undeclared-state" {
			t.Errorf("unexpected undeclared-state finding: %s", f.Message)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/validator/ -run TestUndeclaredState -v`
Expected: FAIL — no `undeclared-state` rule exists

- [ ] **Step 3: Add the rule to validator.go**

Add to the `rules` slice:

```go
	{
		id:       "undeclared-state",
		severity: Error,
		query: `SELECT DISTINCT t1.to_state, ` + ownerCase + ` AS owner_name
		        FROM transitions t1
		        WHERE t1.to_state IS NOT NULL
		          AND t1.to_state NOT IN (
		            SELECT t2.from_state FROM transitions t2
		            WHERE t2.owner_type = t1.owner_type AND t2.owner_id = t1.owner_id
		              AND t2.from_state IS NOT NULL
		          )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var state string
				var owner sql.NullString
				if err := rows.Scan(&state, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "undeclared-state",
					Severity: Error,
					Message:  fmt.Sprintf("state %q in %s is not declared (no transitions from it)", state, ns(owner)),
				})
			}
			return findings, nil
		},
	},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/validator/ -run TestUndeclaredState -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/validator/validator.go internal/validator/validator_test.go
git commit -m "feat: add undeclared_state validation rule"
```

---

### Task 6: Add `dead_end`, `guard_ambiguity`, `cycle_in_emit` rules

**Files:**
- Modify: `internal/validator/validator.go`
- Modify: `internal/validator/validator_test.go`

- [ ] **Step 1: Write tests for all three rules**

```go
func TestDeadEnd(t *testing.T) {
	s, db := setupTestDB(t)
	app := &model.App{Name: "test", Description: "test"}
	s.InsertApp(app)
	screen := &model.Screen{AppID: app.ID, Name: "home", Description: "home"}
	s.InsertScreen(screen)

	// "stuck" has incoming but no outgoing
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "break", FromState: "start", ToState: "stuck",
	})

	findings, _ := Validate(db)
	found := false
	for _, f := range findings {
		if f.Rule == "dead-end" {
			found = true
		}
	}
	if !found {
		t.Error("expected dead-end finding for state 'stuck'")
	}
}

func TestGuardAmbiguity(t *testing.T) {
	s, db := setupTestDB(t)
	app := &model.App{Name: "test", Description: "test"}
	s.InsertApp(app)
	screen := &model.Screen{AppID: app.ID, Name: "home", Description: "home"}
	s.InsertScreen(screen)

	// Same event+state, no guard distinction
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "submit", FromState: "start", ToState: "a",
	})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screen.ID,
		OnEvent: "submit", FromState: "start", ToState: "b",
	})

	findings, _ := Validate(db)
	found := false
	for _, f := range findings {
		if f.Rule == "guard-ambiguity" {
			found = true
		}
	}
	if !found {
		t.Error("expected guard-ambiguity finding")
	}
}
```

- [ ] **Step 2: Add rules to validator.go**

```go
	// dead-end: state with incoming transitions but no outgoing
	{
		id:       "dead-end",
		severity: Warning,
		query: `SELECT DISTINCT t1.to_state, ` + ownerCase + ` AS owner_name
		        FROM transitions t1
		        WHERE t1.to_state IS NOT NULL
		          AND t1.to_state NOT IN (
		            SELECT t2.from_state FROM transitions t2
		            WHERE t2.owner_type = t1.owner_type AND t2.owner_id = t1.owner_id
		              AND t2.from_state IS NOT NULL
		          )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var state string
				var owner sql.NullString
				if err := rows.Scan(&state, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "dead-end",
					Severity: Warning,
					Message:  fmt.Sprintf("state %q in %s has no outgoing transitions (terminal)", state, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// guard-ambiguity: same event+from_state appears 2+ times without guard distinction
	{
		id:       "guard-ambiguity",
		severity: Warning,
		query: `SELECT ` + ownerCase + ` AS owner_name, t.on_event, t.from_state, COUNT(*) AS cnt
		        FROM transitions t
		        WHERE t.from_state IS NOT NULL
		          AND (t.action IS NULL OR t.action NOT LIKE 'guard(%)')
		        GROUP BY t.owner_type, t.owner_id, t.on_event, t.from_state
		        HAVING cnt > 1`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var owner sql.NullString
				var event string
				var fromState sql.NullString
				var cnt int
				if err := rows.Scan(&owner, &event, &fromState, &cnt); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "guard-ambiguity",
					Severity: Warning,
					Message:  fmt.Sprintf("%dx %q from %q in %s without guard distinction", cnt, event, ns(fromState), ns(owner)),
				})
			}
			return findings, nil
		},
	},
```

Note: `cycle_in_emit` requires graph traversal that's expensive in pure SQL. For Phase 1, skip it (it's marked low priority in the spec). Add a TODO comment.

- [ ] **Step 3: Run all validator tests**

Run: `go test ./internal/validator/ -v`
Expected: ALL PASS

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/validator/validator.go internal/validator/validator_test.go
git commit -m "feat: add dead_end and guard_ambiguity validation rules"
```

---

## Chunk 3: Example Migration

### Task 7: Convert gmail.sft.yaml to new format

**Files:**
- Rewrite: `examples/gmail.sft.yaml`

Convert all PascalCase/kebab-case to `lower_snake`. Convert `states:` lists to `state_machine:` maps. Keep all behavioral content identical.

- [ ] **Step 1: Write the converted file**

Read the current `examples/gmail.sft.yaml` and convert:
- Screen/region names: `Inbox` → `inbox`, `EmailList` → `email_list`
- Event names: `select-email` → `select_email`, `check-email` → `check_email`
- Tag names: `preview-pane-enabled` → `preview_pane_enabled`
- `states:` blocks → `state_machine:` maps with `start` as initial state name
- Flow sequences: update entity names to `lower_snake`

- [ ] **Step 2: Test import of converted file**

```bash
rm -f /tmp/test-gmail.db && sft --db /tmp/test-gmail.db import examples/gmail.sft.yaml
sft --db /tmp/test-gmail.db validate
sft --db /tmp/test-gmail.db show
```

Expected: import succeeds, validate shows no errors (warnings OK for ambient events like `escape`), show displays the spec.

- [ ] **Step 3: Commit**

```bash
git add examples/gmail.sft.yaml
git commit -m "migrate: convert gmail.sft.yaml to state_machine format + lower_snake"
```

---

### Task 8: Convert remaining 5 example specs

**Files:**
- Rewrite: `examples/linear.sft.yaml`
- Rewrite: `examples/bank.sft.yaml`
- Rewrite: `examples/stripe.sft.yaml`
- Rewrite: `examples/shopify.sft.yaml`
- Rewrite: `examples/docs.sft.yaml`

- [ ] **Step 1: Convert each file following the same pattern as gmail**

For each file:
- All names to `lower_snake`
- All `states:` to `state_machine:` maps
- First `from:` state becomes `start` (or keep the original name if meaningful)
- Events in `state_machine:` `on:` blocks use the appropriate value form
- Flows updated with `lower_snake` entity names

- [ ] **Step 2: Test import of each converted file**

```bash
for f in examples/*.sft.yaml; do
  db="/tmp/test-$(basename "$f" .sft.yaml).db"
  rm -f "$db"
  sft --db "$db" import "$f"
  echo "=== $f ==="
  sft --db "$db" validate
done
```

Expected: all 6 import successfully, validate shows no errors.

- [ ] **Step 3: Commit all**

```bash
git add examples/*.sft.yaml
git commit -m "migrate: convert all 6 example specs to state_machine + lower_snake"
```

---

### Task 9: Update loader test fixtures

**Files:**
- Modify: `internal/loader/loader_test.go`

- [ ] **Step 1: Add a test that imports the new-format Gmail spec**

```go
func TestGmailExampleImport(t *testing.T) {
	s := mustStore(t)
	if err := Load(s, "../../examples/gmail.sft.yaml"); err != nil {
		t.Fatalf("Gmail import: %v", err)
	}

	spec := loadSpec(t, s)
	if spec.App.Name != "gmail" {
		t.Errorf("app name = %q", spec.App.Name)
	}
	if len(spec.Screens) < 3 {
		t.Errorf("expected 3+ screens, got %d", len(spec.Screens))
	}

	// Check state machine transitions exist
	inbox := spec.Screens[0]
	if len(inbox.Transitions) < 4 {
		t.Errorf("inbox: expected 4+ transitions, got %d", len(inbox.Transitions))
	}
}
```

- [ ] **Step 2: Ensure existing legacy format tests still pass**

Run: `go test ./internal/loader/ -v`

The `testYAML` constant uses the old format with `states:` — it must still work (dual-format).

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/loader/loader_test.go
git commit -m "test: add gmail example import test, verify dual-format support"
```

---

### Task 10: Final verification

- [ ] **Step 1: Build**

```bash
go build ./cmd/sft
```

- [ ] **Step 2: Import + validate each example**

```bash
for f in examples/*.sft.yaml; do
  db="/tmp/verify-$(basename "$f" .sft.yaml).db"
  rm -f "$db"
  ./sft --db "$db" import "$f" && echo "OK: $f" || echo "FAIL: $f"
  ./sft --db "$db" validate
done
```

- [ ] **Step 3: Round-trip test — import, export, re-import**

```bash
rm -f /tmp/rt.db && ./sft --db /tmp/rt.db import examples/gmail.sft.yaml
./sft --db /tmp/rt.db export /tmp/gmail-rt.yaml
rm -f /tmp/rt2.db && ./sft --db /tmp/rt2.db import /tmp/gmail-rt.yaml
./sft --db /tmp/rt2.db show
```

- [ ] **Step 4: Full test suite**

```bash
go test ./...
```

Expected: ALL PASS

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat: state machine phase 1 complete — format, validation, migration"
```
