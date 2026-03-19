package loader

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/lagz0ne/sft/internal/show"
	"github.com/lagz0ne/sft/internal/store"
)

func mustStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

const testYAML = `app:
  name: TestApp
  description: A test application
  regions:
    - name: GlobalNav
      description: Top-level navigation
      events: [nav-click]
  screens:
    - name: Home
      description: Landing page
      tags: [landing]
      regions:
        - name: Hero
          description: Hero section
          events: [cta-click]
          tags: [above-fold]
      states:
        - on: cta-click
          from: idle
          to: active
    - name: Detail
      description: Detail view
  flows:
    - name: Landing
      description: User lands and clicks CTA
      on: page-load
      sequence: "Home → Detail → [Back] → Home(H)"
`

func importYAML(t *testing.T, s *store.Store, yaml string) {
	t.Helper()
	tmp := t.TempDir() + "/test.sft.yaml"
	if err := os.WriteFile(tmp, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Load(s, tmp); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func loadSpec(t *testing.T, s *store.Store) *show.Spec {
	t.Helper()
	spec, err := show.Load(s.DB, s)
	if err != nil {
		t.Fatalf("show.Load: %v", err)
	}
	return spec
}

func TestRoundTrip(t *testing.T) {
	s1 := mustStore(t)
	importYAML(t, s1, testYAML)
	spec1 := loadSpec(t, s1)

	// Export
	var buf bytes.Buffer
	if err := Export(spec1, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Re-import
	s2 := mustStore(t)
	tmp := t.TempDir() + "/rt.sft.yaml"
	os.WriteFile(tmp, buf.Bytes(), 0o644)
	if err := Load(s2, tmp); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	spec2 := loadSpec(t, s2)

	// Compare
	if spec1.App.Name != spec2.App.Name {
		t.Errorf("app name: %q vs %q", spec1.App.Name, spec2.App.Name)
	}
	if len(spec1.Screens) != len(spec2.Screens) {
		t.Fatalf("screens: %d vs %d", len(spec1.Screens), len(spec2.Screens))
	}
	for i, sc := range spec1.Screens {
		sc2 := spec2.Screens[i]
		if sc.Name != sc2.Name {
			t.Errorf("screen %d name: %q vs %q", i, sc.Name, sc2.Name)
		}
		if sc.Description != sc2.Description {
			t.Errorf("screen %s desc mismatch", sc.Name)
		}
		if len(sc.Tags) != len(sc2.Tags) {
			t.Errorf("screen %s tags: %d vs %d", sc.Name, len(sc.Tags), len(sc2.Tags))
		}
		if len(sc.Regions) != len(sc2.Regions) {
			t.Errorf("screen %s regions: %d vs %d", sc.Name, len(sc.Regions), len(sc2.Regions))
		}
		if len(sc.Transitions) != len(sc2.Transitions) {
			t.Errorf("screen %s transitions: %d vs %d", sc.Name, len(sc.Transitions), len(sc2.Transitions))
		}
	}
	if len(spec1.Flows) != len(spec2.Flows) {
		t.Fatalf("flows: %d vs %d", len(spec1.Flows), len(spec2.Flows))
	}
	for i, f := range spec1.Flows {
		f2 := spec2.Flows[i]
		if f.Name != f2.Name || f.Sequence != f2.Sequence || f.OnEvent != f2.OnEvent {
			t.Errorf("flow %d mismatch: %+v vs %+v", i, f, f2)
		}
	}
	if len(spec1.App.Regions) != len(spec2.App.Regions) {
		t.Errorf("app regions: %d vs %d", len(spec1.App.Regions), len(spec2.App.Regions))
	}
}

func TestComponentRoundTrip(t *testing.T) {
	s1 := mustStore(t)
	importYAML(t, s1, testYAML)

	if err := s1.SetComponent("Home", "Dashboard", `{"layout":"grid"}`, "onClick", "auth"); err != nil {
		t.Fatal(err)
	}

	spec1 := loadSpec(t, s1)
	if spec1.Screens[0].Component != "Dashboard" {
		t.Fatalf("expected component Dashboard, got %q", spec1.Screens[0].Component)
	}

	// Export
	var buf bytes.Buffer
	if err := Export(spec1, &buf); err != nil {
		t.Fatal(err)
	}
	exported := buf.String()

	if !strings.Contains(exported, "component: Dashboard") {
		t.Error("exported YAML missing component field")
	}

	// Re-import and verify component survives
	s2 := mustStore(t)
	tmp := t.TempDir() + "/comp.sft.yaml"
	os.WriteFile(tmp, []byte(exported), 0o644)
	if err := Load(s2, tmp); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	c := s2.GetComponentByName("Home")
	if c == nil {
		t.Fatal("component lost after round-trip")
	}
	if c.Component != "Dashboard" {
		t.Errorf("component = %q, want Dashboard", c.Component)
	}
	if c.Props != `{"layout":"grid"}` {
		t.Errorf("props = %q, want {\"layout\":\"grid\"}", c.Props)
	}
	if c.OnActions != "onClick" {
		t.Errorf("on_actions = %q, want onClick", c.OnActions)
	}
	if c.Visible != "auth" {
		t.Errorf("visible = %q, want auth", c.Visible)
	}
}

func TestComponentInYAMLImport(t *testing.T) {
	yamlWithComponent := `app:
  name: CompApp
  description: Test component import
  screens:
    - name: Dashboard
      description: Main dashboard
      component: DataGrid
      props: '{"cols":5}'
      on_actions: handleClick
      visible: admin
      regions:
        - name: Sidebar
          description: Side panel
          component: NavPanel
          props: '{"collapsed":true}'
`
	s := mustStore(t)
	importYAML(t, s, yamlWithComponent)

	c := s.GetComponentByName("Dashboard")
	if c == nil {
		t.Fatal("Dashboard component not imported")
	}
	if c.Component != "DataGrid" {
		t.Errorf("component = %q, want DataGrid", c.Component)
	}
	if c.Props != `{"cols":5}` {
		t.Errorf("props = %q", c.Props)
	}
	if c.OnActions != "handleClick" {
		t.Errorf("on_actions = %q", c.OnActions)
	}

	cr := s.GetComponentByName("Sidebar")
	if cr == nil {
		t.Fatal("Sidebar component not imported")
	}
	if cr.Component != "NavPanel" {
		t.Errorf("region component = %q, want NavPanel", cr.Component)
	}
}

const testStateMachineYAML = `app:
  name: SMApp
  description: State machine test
  screens:
    - name: Login
      description: Login screen
      state_machine:
        idle:
          on:
            SUBMIT: loading
        loading:
          on:
            SUCCESS: authenticated
            FAILURE: idle
        authenticated:
`

func TestStateMachineExportFormat(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testStateMachineYAML)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	exported := buf.String()

	if strings.Contains(exported, "states:") {
		t.Errorf("export should use state_machine:, not states:\nexported:\n%s", exported)
	}
	if !strings.Contains(exported, "state_machine:") {
		t.Errorf("export missing state_machine:\nexported:\n%s", exported)
	}
}

func TestStateMachineRoundTrip(t *testing.T) {
	s1 := mustStore(t)
	importYAML(t, s1, testStateMachineYAML)
	spec1 := loadSpec(t, s1)

	var buf bytes.Buffer
	if err := Export(spec1, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	s2 := mustStore(t)
	importYAML(t, s2, buf.String())
	spec2 := loadSpec(t, s2)

	// Compare transition counts
	if len(spec1.Screens) != len(spec2.Screens) {
		t.Fatalf("screen count: %d vs %d", len(spec1.Screens), len(spec2.Screens))
	}
	for i, sc := range spec1.Screens {
		sc2 := spec2.Screens[i]
		if len(sc.Transitions) != len(sc2.Transitions) {
			t.Errorf("screen %s: transitions %d vs %d", sc.Name, len(sc.Transitions), len(sc2.Transitions))
		}
		// Verify each transition matches.
		for j, tr := range sc.Transitions {
			tr2 := sc2.Transitions[j]
			if tr.OnEvent != tr2.OnEvent || tr.FromState != tr2.FromState || tr.ToState != tr2.ToState || tr.Action != tr2.Action {
				t.Errorf("screen %s transition %d: %+v vs %+v", sc.Name, j, tr, tr2)
			}
		}
	}
}

func TestStateMachineExportTerminalStates(t *testing.T) {
	// Verify that terminal states (appear as to but never as from) are included.
	s := mustStore(t)
	importYAML(t, s, testStateMachineYAML)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	exported := buf.String()

	// "authenticated" is a terminal state — it should appear in the output.
	if !strings.Contains(exported, "authenticated:") {
		t.Errorf("terminal state 'authenticated' missing from export:\n%s", exported)
	}
}

func TestStateMachineExportStayDot(t *testing.T) {
	// When to == from, export should use "." notation.
	yamlStay := `app:
  name: StayApp
  description: Stay test
  screens:
    - name: Editor
      description: Editor screen
      state_machine:
        idle:
          on:
            REFRESH: .
            SUBMIT: loading
        loading:
          on:
            DONE: idle
`
	s := mustStore(t)
	importYAML(t, s, yamlStay)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	exported := buf.String()

	if !strings.Contains(exported, "REFRESH: \".\"") && !strings.Contains(exported, "REFRESH: .") {
		t.Errorf("stay transition should export as '.'\nexported:\n%s", exported)
	}
}

func TestStateMachineExportWithAction(t *testing.T) {
	// to + action should produce flow-style object.
	yamlAction := `app:
  name: ActionApp
  description: Action test
  screens:
    - name: Inbox
      description: Inbox screen
      state_machine:
        start:
          on:
            select_email: {to: viewing, action: "navigate(thread_view)"}
        viewing:
`
	s := mustStore(t)
	importYAML(t, s, yamlAction)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	exported := buf.String()

	// Should contain both to and action in object form.
	if !strings.Contains(exported, "to:") || !strings.Contains(exported, "action:") {
		t.Errorf("to+action transition should be object form:\n%s", exported)
	}
}

func TestLegacyExportProducesStateMachine(t *testing.T) {
	// Importing legacy states: format should export as state_machine: format.
	s := mustStore(t)
	importYAML(t, s, testYAML)
	spec := loadSpec(t, s)

	var buf bytes.Buffer
	if err := Export(spec, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	exported := buf.String()

	if strings.Contains(exported, "  states:") {
		t.Errorf("legacy import should export as state_machine:, not states:\n%s", exported)
	}
	// Home screen has a transition, so state_machine: should appear.
	if !strings.Contains(exported, "state_machine:") {
		t.Errorf("export missing state_machine: for Home screen\n%s", exported)
	}
}

func TestStateMachineImport(t *testing.T) {
	yamlSM := `app:
  name: SMApp
  description: State machine test
  screens:
    - name: Login
      description: Login screen
      state_machine:
        idle:
          on:
            SUBMIT: loading
        loading:
          on:
            SUCCESS: authenticated
            FAILURE: idle
        authenticated:
`
	s := mustStore(t)
	importYAML(t, s, yamlSM)
	spec := loadSpec(t, s)

	if len(spec.Screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(spec.Screens))
	}
	login := spec.Screens[0]
	if len(login.Transitions) != 3 {
		t.Fatalf("expected 3 transitions, got %d", len(login.Transitions))
	}

	// Verify specific transitions
	found := map[string]bool{}
	for _, tr := range login.Transitions {
		key := tr.FromState + "/" + tr.OnEvent + "/" + tr.ToState
		found[key] = true
	}
	for _, want := range []string{"idle/SUBMIT/loading", "loading/SUCCESS/authenticated", "loading/FAILURE/idle"} {
		if !found[want] {
			t.Errorf("missing transition: %s", want)
		}
	}
}

func TestStateMachineRegionImport(t *testing.T) {
	yamlSM := `app:
  name: SMRegionApp
  description: State machine in region
  screens:
    - name: Dashboard
      description: Main screen
      regions:
        - name: Sidebar
          description: Side panel
          state_machine:
            collapsed:
              on:
                TOGGLE: expanded
            expanded:
              on:
                TOGGLE: collapsed
`
	s := mustStore(t)
	importYAML(t, s, yamlSM)
	spec := loadSpec(t, s)

	if len(spec.Screens[0].Regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(spec.Screens[0].Regions))
	}
	sidebar := spec.Screens[0].Regions[0]
	if len(sidebar.Transitions) != 2 {
		t.Fatalf("expected 2 transitions on region, got %d", len(sidebar.Transitions))
	}
}

func TestStateMachineAndStatesConflict(t *testing.T) {
	yamlConflict := `app:
  name: ConflictApp
  description: Both formats
  screens:
    - name: Broken
      description: Has both
      states:
        - on: click
          from: a
          to: b
      state_machine:
        a:
          on:
            click: b
`
	s := mustStore(t)
	tmp := t.TempDir() + "/conflict.sft.yaml"
	if err := os.WriteFile(tmp, []byte(yamlConflict), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Load(s, tmp)
	if err == nil {
		t.Fatal("expected error when both states and state_machine are present")
	}
	if !strings.Contains(err.Error(), "cannot specify both states and state_machine") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLegacyStatesFormatStillWorks(t *testing.T) {
	// This test verifies backward compatibility with the existing states format
	s := mustStore(t)
	importYAML(t, s, testYAML)
	spec := loadSpec(t, s)

	// The testYAML has Home screen with 1 transition (cta-click: idle -> active)
	home := spec.Screens[0]
	if home.Name != "Home" {
		t.Fatalf("expected Home screen, got %s", home.Name)
	}
	if len(home.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(home.Transitions))
	}
	tr := home.Transitions[0]
	if tr.OnEvent != "cta-click" || tr.FromState != "idle" || tr.ToState != "active" {
		t.Errorf("unexpected transition: %+v", tr)
	}
}

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
