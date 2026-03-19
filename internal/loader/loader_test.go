package loader

import (
	"bytes"
	"database/sql"
	"encoding/json"
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

const testDataModelYAML = `app:
  name: test_app
  description: test

  data:
    email:
      subject: string
      sender: contact
      read: boolean
    contact:
      name: string
      email: string

  context:
    current_user: contact
    permissions: permission[]

  screens:
    - name: inbox
      description: email list
      context:
        emails: email[]
        selected: email[]
      regions:
        - name: email_list
          description: list of emails
          ambient:
            emails: data(inbox, .emails)
          events: [select_email]
        - name: unread_badge
          description: badge
          ambient:
            count: "data(inbox, .emails[?read==false] | length)"
        - name: search_bar
          description: search
          data:
            query: string
            suggestions: string[]
`

func TestDataTypeImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testDataModelYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM data_types").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 data types, got %d", count)
	}

	// Verify email type
	var fieldsJSON string
	err := s.DB.QueryRow("SELECT fields FROM data_types WHERE name = 'email'").Scan(&fieldsJSON)
	if err != nil {
		t.Fatalf("query email data type: %v", err)
	}
	var fields map[string]string
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		t.Fatalf("unmarshal email fields: %v", err)
	}
	if fields["subject"] != "string" {
		t.Errorf("email.subject = %q, want string", fields["subject"])
	}
	if fields["sender"] != "contact" {
		t.Errorf("email.sender = %q, want contact", fields["sender"])
	}
	if fields["read"] != "boolean" {
		t.Errorf("email.read = %q, want boolean", fields["read"])
	}

	// Verify contact type
	err = s.DB.QueryRow("SELECT fields FROM data_types WHERE name = 'contact'").Scan(&fieldsJSON)
	if err != nil {
		t.Fatalf("query contact data type: %v", err)
	}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		t.Fatalf("unmarshal contact fields: %v", err)
	}
	if fields["name"] != "string" || fields["email"] != "string" {
		t.Errorf("contact fields: %v", fields)
	}
}

func TestContextImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testDataModelYAML)

	// App-level context
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM contexts WHERE owner_type = 'app'").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 app context fields, got %d", count)
	}

	var fieldType string
	err := s.DB.QueryRow("SELECT field_type FROM contexts WHERE owner_type = 'app' AND field_name = 'current_user'").Scan(&fieldType)
	if err != nil {
		t.Fatalf("query app context current_user: %v", err)
	}
	if fieldType != "contact" {
		t.Errorf("current_user type = %q, want contact", fieldType)
	}

	err = s.DB.QueryRow("SELECT field_type FROM contexts WHERE owner_type = 'app' AND field_name = 'permissions'").Scan(&fieldType)
	if err != nil {
		t.Fatalf("query app context permissions: %v", err)
	}
	if fieldType != "permission[]" {
		t.Errorf("permissions type = %q, want permission[]", fieldType)
	}

	// Screen-level context
	s.DB.QueryRow("SELECT COUNT(*) FROM contexts WHERE owner_type = 'screen'").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 screen context fields, got %d", count)
	}

	err = s.DB.QueryRow("SELECT field_type FROM contexts WHERE owner_type = 'screen' AND field_name = 'emails'").Scan(&fieldType)
	if err != nil {
		t.Fatalf("query screen context emails: %v", err)
	}
	if fieldType != "email[]" {
		t.Errorf("emails type = %q, want email[]", fieldType)
	}
}

func TestAmbientImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testDataModelYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM ambient_refs").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 ambient refs, got %d", count)
	}

	// Verify email_list ambient ref
	var regionID int64
	s.DB.QueryRow("SELECT id FROM regions WHERE name = 'email_list'").Scan(&regionID)

	var source, query string
	err := s.DB.QueryRow("SELECT source, query FROM ambient_refs WHERE region_id = ? AND local_name = 'emails'", regionID).Scan(&source, &query)
	if err != nil {
		t.Fatalf("query ambient ref emails: %v", err)
	}
	if source != "inbox" {
		t.Errorf("emails source = %q, want inbox", source)
	}
	if query != ".emails" {
		t.Errorf("emails query = %q, want .emails", query)
	}

	// Verify unread_badge ambient ref with complex query
	s.DB.QueryRow("SELECT id FROM regions WHERE name = 'unread_badge'").Scan(&regionID)
	err = s.DB.QueryRow("SELECT source, query FROM ambient_refs WHERE region_id = ? AND local_name = 'count'", regionID).Scan(&source, &query)
	if err != nil {
		t.Fatalf("query ambient ref count: %v", err)
	}
	if source != "inbox" {
		t.Errorf("count source = %q, want inbox", source)
	}
	if query != ".emails[?read==false] | length" {
		t.Errorf("count query = %q, want .emails[?read==false] | length", query)
	}
}

func TestRegionDataImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testDataModelYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM region_data").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 region data fields, got %d", count)
	}

	var regionID int64
	s.DB.QueryRow("SELECT id FROM regions WHERE name = 'search_bar'").Scan(&regionID)

	var fieldType string
	err := s.DB.QueryRow("SELECT field_type FROM region_data WHERE region_id = ? AND field_name = 'query'", regionID).Scan(&fieldType)
	if err != nil {
		t.Fatalf("query region data 'query': %v", err)
	}
	if fieldType != "string" {
		t.Errorf("query field_type = %q, want string", fieldType)
	}

	err = s.DB.QueryRow("SELECT field_type FROM region_data WHERE region_id = ? AND field_name = 'suggestions'", regionID).Scan(&fieldType)
	if err != nil {
		t.Fatalf("query region data 'suggestions': %v", err)
	}
	if fieldType != "string[]" {
		t.Errorf("suggestions field_type = %q, want string[]", fieldType)
	}
}

func TestDataModelRoundTrip(t *testing.T) {
	s1 := mustStore(t)
	importYAML(t, s1, testDataModelYAML)
	spec1 := loadSpec(t, s1)

	// Verify show loaded the data
	if len(spec1.App.DataTypes) != 2 {
		t.Errorf("data types: got %d, want 2", len(spec1.App.DataTypes))
	}
	if len(spec1.App.Context) != 2 {
		t.Errorf("app context: got %d, want 2", len(spec1.App.Context))
	}

	// Export
	var buf bytes.Buffer
	Export(spec1, &buf)
	exported := buf.String()

	if !strings.Contains(exported, "data:") {
		t.Error("export missing data: block")
	}
	if !strings.Contains(exported, "context:") {
		t.Error("export missing context: block")
	}
	if !strings.Contains(exported, "ambient:") {
		t.Error("export missing ambient: block")
	}

	// Re-import
	s2 := mustStore(t)
	importYAML(t, s2, buf.String())

	// Verify data survived
	var count int
	s2.DB.QueryRow("SELECT COUNT(*) FROM data_types").Scan(&count)
	if count != 2 {
		t.Errorf("round-trip data_types: got %d, want 2", count)
	}
	s2.DB.QueryRow("SELECT COUNT(*) FROM contexts").Scan(&count)
	if count != 4 { // 2 app + 2 screen
		t.Errorf("round-trip contexts: got %d, want 4", count)
	}
	s2.DB.QueryRow("SELECT COUNT(*) FROM ambient_refs").Scan(&count)
	if count != 2 {
		t.Errorf("round-trip ambient_refs: got %d, want 2", count)
	}
}

const testFixtureYAML = `app:
  name: test_app
  description: test
  screens:
    - name: inbox
      description: email list
      regions:
        - name: email_list
          description: list
          events: [select_email]
      state_machine:
        start:
          fixture: inbox_full
          on:
            select_email: selecting
        selecting:
          fixture: inbox_selecting
          on:
            escape: start
        empty:
          fixture: inbox_empty
  fixtures:
    inbox_full:
      inbox:
        emails:
          - { subject: "Welcome", read: false }
        selected: []
    inbox_empty:
      inbox:
        emails: []
        selected: []
    inbox_selecting:
      extends: inbox_full
      inbox:
        selected:
          - { subject: "Welcome" }
`

func TestFixtureImport(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testFixtureYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM fixtures").Scan(&count)
	if count != 3 {
		t.Errorf("fixtures: got %d, want 3", count)
	}

	// Check extends
	var extends sql.NullString
	s.DB.QueryRow("SELECT extends FROM fixtures WHERE name = 'inbox_selecting'").Scan(&extends)
	if !extends.Valid || extends.String != "inbox_full" {
		t.Errorf("extends = %v, want inbox_full", extends)
	}

	// Check data is valid JSON
	var data string
	s.DB.QueryRow("SELECT data FROM fixtures WHERE name = 'inbox_full'").Scan(&data)
	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Errorf("fixture data is not valid JSON: %v", err)
	}
}

func TestStateFixtureBinding(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testFixtureYAML)

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM state_fixtures").Scan(&count)
	if count != 3 {
		t.Errorf("state_fixtures: got %d, want 3", count)
	}

	var fixtureName string
	s.DB.QueryRow("SELECT fixture_name FROM state_fixtures WHERE state_name = 'start'").Scan(&fixtureName)
	if fixtureName != "inbox_full" {
		t.Errorf("start fixture = %q, want inbox_full", fixtureName)
	}

	s.DB.QueryRow("SELECT fixture_name FROM state_fixtures WHERE state_name = 'selecting'").Scan(&fixtureName)
	if fixtureName != "inbox_selecting" {
		t.Errorf("selecting fixture = %q, want inbox_selecting", fixtureName)
	}

	s.DB.QueryRow("SELECT fixture_name FROM state_fixtures WHERE state_name = 'empty'").Scan(&fixtureName)
	if fixtureName != "inbox_empty" {
		t.Errorf("empty fixture = %q, want inbox_empty", fixtureName)
	}
}

func TestFixtureShow(t *testing.T) {
	s := mustStore(t)
	importYAML(t, s, testFixtureYAML)
	spec := loadSpec(t, s)

	if len(spec.Fixtures) != 3 {
		t.Fatalf("fixtures: got %d, want 3", len(spec.Fixtures))
	}

	// Check state fixtures loaded on screen
	inbox := spec.Screens[0]
	if len(inbox.StateFixtures) != 3 {
		t.Errorf("screen state_fixtures: got %d, want 3", len(inbox.StateFixtures))
	}
	if inbox.StateFixtures["start"] != "inbox_full" {
		t.Errorf("start fixture = %q, want inbox_full", inbox.StateFixtures["start"])
	}
}

func TestFixtureRoundTrip(t *testing.T) {
	s1 := mustStore(t)
	importYAML(t, s1, testFixtureYAML)
	spec1 := loadSpec(t, s1)

	if len(spec1.Fixtures) != 3 {
		t.Fatalf("fixtures: got %d, want 3", len(spec1.Fixtures))
	}

	var buf bytes.Buffer
	Export(spec1, &buf)
	exported := buf.String()

	if !strings.Contains(exported, "fixtures:") {
		t.Error("export missing fixtures:")
	}
	if !strings.Contains(exported, "inbox_full:") {
		t.Error("export missing inbox_full fixture")
	}
	if !strings.Contains(exported, "extends: inbox_full") {
		t.Error("export missing extends: inbox_full")
	}

	// Verify fixture: appears in state_machine export
	if !strings.Contains(exported, "fixture: inbox_full") {
		t.Error("export missing fixture: inbox_full in state_machine")
	}

	// Re-import
	s2 := mustStore(t)
	importYAML(t, s2, buf.String())
	var count int
	s2.DB.QueryRow("SELECT COUNT(*) FROM fixtures").Scan(&count)
	if count != 3 {
		t.Errorf("round-trip fixtures: got %d, want 3", count)
	}

	// Verify state fixtures survive round-trip
	s2.DB.QueryRow("SELECT COUNT(*) FROM state_fixtures").Scan(&count)
	if count != 3 {
		t.Errorf("round-trip state_fixtures: got %d, want 3", count)
	}

	var fixtureName string
	s2.DB.QueryRow("SELECT fixture_name FROM state_fixtures WHERE state_name = 'start'").Scan(&fixtureName)
	if fixtureName != "inbox_full" {
		t.Errorf("round-trip start fixture = %q, want inbox_full", fixtureName)
	}
}

func TestStateTemplateImport(t *testing.T) {
	yamlWithTemplate := `app:
  name: TemplateApp
  description: Test state templates
  state_templates:
    crud_loadable:
      start:
        on: { load: loading }
      loading:
        on: { load_success: loaded, load_error: error }
      loaded:
        on: { edit: editing }
      editing:
        on: { save: saving, cancel: loaded }
      saving:
        on: { save_success: loaded, save_error: editing }
      error:
        on: { retry: loading }
  screens:
    - name: Dashboard
      description: Main dashboard
`
	s := mustStore(t)
	importYAML(t, s, yamlWithTemplate)

	// Verify the template was stored
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM state_templates").Scan(&count)
	if count != 1 {
		t.Fatalf("state_templates: got %d, want 1", count)
	}

	var name, definition string
	s.DB.QueryRow("SELECT name, definition FROM state_templates").Scan(&name, &definition)
	if name != "crud_loadable" {
		t.Errorf("template name = %q, want crud_loadable", name)
	}
	// Verify definition is valid JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(definition), &parsed); err != nil {
		t.Errorf("template definition is not valid JSON: %v", err)
	}
}

func TestStateTemplateExtends(t *testing.T) {
	yamlWithExtends := `app:
  name: ExtendsApp
  description: Test extends
  state_templates:
    crud_loadable:
      start:
        on: { load: loading }
      loading:
        on: { load_success: loaded, load_error: error }
      loaded:
        on: { edit: editing }
      editing:
        on: { save: saving, cancel: loaded }
      saving:
        on: { save_success: loaded, save_error: editing }
      error:
        on: { retry: loading }
  screens:
    - name: order_detail
      description: Order detail screen
      state_machine:
        extends: crud_loadable
        loaded:
          on:
            delete: { action: "navigate(order_list)" }
`
	s := mustStore(t)

	// Need order_list screen to exist for navigate validation
	importYAML(t, s, yamlWithExtends)

	spec := loadSpec(t, s)

	if len(spec.Screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(spec.Screens))
	}
	screen := spec.Screens[0]

	// Template has: start→loading, loading→loaded, loading→error,
	// loaded→editing, editing→saving, editing→loaded, saving→loaded, saving→editing,
	// error→loading
	// Override adds: loaded→delete (navigate(order_list))
	// The override merged with the base loaded state should have: edit→editing AND delete→navigate(order_list)

	// Build a lookup
	type transKey struct {
		from, event string
	}
	found := map[transKey]string{}
	for _, tr := range screen.Transitions {
		key := transKey{tr.FromState, tr.OnEvent}
		if tr.ToState != "" {
			found[key] = tr.ToState
		} else {
			found[key] = tr.Action
		}
	}

	// Check template-provided transitions exist
	checks := []struct {
		from, event, expected string
	}{
		{"start", "load", "loading"},
		{"loading", "load_success", "loaded"},
		{"loading", "load_error", "error"},
		{"loaded", "edit", "editing"},             // from template
		{"loaded", "delete", "navigate(order_list)"}, // from override
		{"editing", "save", "saving"},
		{"editing", "cancel", "loaded"},
		{"saving", "save_success", "loaded"},
		{"saving", "save_error", "editing"},
		{"error", "retry", "loading"},
	}
	for _, c := range checks {
		key := transKey{c.from, c.event}
		val, ok := found[key]
		if !ok {
			t.Errorf("missing transition: %s/%s", c.from, c.event)
			continue
		}
		if val != c.expected {
			t.Errorf("transition %s/%s = %q, want %q", c.from, c.event, val, c.expected)
		}
	}

	if len(screen.Transitions) != 10 {
		t.Errorf("expected 10 transitions, got %d", len(screen.Transitions))
		for _, tr := range screen.Transitions {
			t.Logf("  %s/%s → %s %s", tr.FromState, tr.OnEvent, tr.ToState, tr.Action)
		}
	}
}

func TestStateTemplateOverrideEvent(t *testing.T) {
	yamlOverride := `app:
  name: OverrideApp
  description: Test override
  state_templates:
    simple:
      idle:
        on: { click: active }
      active:
        on: { reset: idle }
  screens:
    - name: Custom
      description: Custom screen
      state_machine:
        extends: simple
        idle:
          on:
            click: custom_active
`
	s := mustStore(t)
	importYAML(t, s, yamlOverride)
	spec := loadSpec(t, s)

	screen := spec.Screens[0]
	// The override should change idle/click from "active" to "custom_active"
	for _, tr := range screen.Transitions {
		if tr.FromState == "idle" && tr.OnEvent == "click" {
			if tr.ToState != "custom_active" {
				t.Errorf("idle/click ToState = %q, want custom_active", tr.ToState)
			}
			return
		}
	}
	t.Error("missing idle/click transition")
}

func TestParseDataRef(t *testing.T) {
	tests := []struct {
		input      string
		wantSource string
		wantQuery  string
		wantErr    bool
	}{
		{"data(inbox, .emails)", "inbox", ".emails", false},
		{"data(app, .permissions)", "app", ".permissions", false},
		{"data(inbox, .emails[?read==false] | length)", "inbox", ".emails[?read==false] | length", false},
		{"invalid", "", "", true},
		{"data(missing_separator)", "", "", true},
		{"notdata(x, y)", "", "", true},
	}
	for _, tt := range tests {
		source, query, err := parseDataRef(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDataRef(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if source != tt.wantSource {
			t.Errorf("parseDataRef(%q): source=%q, want %q", tt.input, source, tt.wantSource)
		}
		if query != tt.wantQuery {
			t.Errorf("parseDataRef(%q): query=%q, want %q", tt.input, query, tt.wantQuery)
		}
	}
}
