package show

import (
	"strings"
	"testing"

	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/store"
)

// --- test helpers ---

func mustStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seedApp(t *testing.T, s *store.Store) *model.App {
	t.Helper()
	a := &model.App{Name: "TestApp", Description: "test"}
	if err := s.InsertApp(a); err != nil {
		t.Fatalf("InsertApp: %v", err)
	}
	return a
}

func addScreen(t *testing.T, s *store.Store, appID int64, name, desc string) *model.Screen {
	t.Helper()
	sc := &model.Screen{AppID: appID, Name: name, Description: desc}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatalf("InsertScreen: %v", err)
	}
	return sc
}

func addRegion(t *testing.T, s *store.Store, appID int64, parentType string, parentID int64, name, desc string) *model.Region {
	t.Helper()
	r := &model.Region{AppID: appID, ParentType: parentType, ParentID: parentID, Name: name, Description: desc}
	if err := s.InsertRegion(r); err != nil {
		t.Fatalf("InsertRegion: %v", err)
	}
	return r
}

func addTransition(t *testing.T, s *store.Store, ownerType string, ownerID int64, onEvent, from, to, action string) {
	t.Helper()
	tr := &model.Transition{OwnerType: ownerType, OwnerID: ownerID, OnEvent: onEvent, FromState: from, ToState: to, Action: action}
	if err := s.InsertTransition(tr); err != nil {
		t.Fatalf("InsertTransition: %v", err)
	}
}

// --- tests ---

func TestScreenStates(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	sc := addScreen(t, s, app.ID, "Inbox", "email inbox")
	addTransition(t, s, "screen", sc.ID, "load", "empty", "loading", "")
	addTransition(t, s, "screen", sc.ID, "loaded", "loading", "ready", "")
	addTransition(t, s, "screen", sc.ID, "error", "loading", "error", "")

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(spec.Screens))
	}

	got := spec.Screens[0].States
	want := []string{"empty", "loading", "ready", "error"}
	if len(got) != len(want) {
		t.Fatalf("states: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("states[%d]: got %q, want %q", i, got[i], want[i])
		}
	}

	// First state must be the initial state (first from_state)
	if got[0] != "empty" {
		t.Errorf("initial state: got %q, want %q", got[0], "empty")
	}
}

func TestRegionStates(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	sc := addScreen(t, s, app.ID, "Inbox", "email inbox")
	r := addRegion(t, s, app.ID, "screen", sc.ID, "EmailList", "list of emails")
	addTransition(t, s, "region", r.ID, "select", "idle", "selected", "")
	addTransition(t, s, "region", r.ID, "deselect", "selected", "idle", "")

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	region := spec.Screens[0].Regions[0]
	got := region.States
	want := []string{"idle", "selected"}
	if len(got) != len(want) {
		t.Fatalf("states: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("states[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNoTransitionsNoStates(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	addScreen(t, s, app.ID, "Empty", "no transitions")

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if spec.Screens[0].States != nil {
		t.Errorf("expected nil states, got %v", spec.Screens[0].States)
	}
}

func TestSelfTransitionDot(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	sc := addScreen(t, s, app.ID, "S1", "screen")

	// "." means self-transition — should be excluded from states list
	addTransition(t, s, "screen", sc.ID, "refresh", "ready", ".", "")
	addTransition(t, s, "screen", sc.ID, "load", "idle", "ready", "")

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := spec.Screens[0].States
	// "." should not appear; order: ready (first from_state), idle, then ready already seen
	want := []string{"ready", "idle"}
	if len(got) != len(want) {
		t.Fatalf("states: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("states[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestActionOnlyTransitions(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	sc := addScreen(t, s, app.ID, "Nav", "navigation")

	// Transitions with no from/to states (action-only)
	addTransition(t, s, "screen", sc.ID, "tap_settings", "", "", "navigate(Settings)")

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if spec.Screens[0].States != nil {
		t.Errorf("expected nil states for action-only transitions, got %v", spec.Screens[0].States)
	}
}

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

func TestRefs(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	sc1 := addScreen(t, s, app.ID, "inbox", "email inbox")
	addScreen(t, s, app.ID, "settings", "app settings")
	addRegion(t, s, app.ID, "screen", sc1.ID, "email_list", "list of emails")

	// Add a flow via raw SQL
	s.DB.Exec(`INSERT INTO flows(app_id, name, sequence) VALUES(?, 'read_flow', 'inbox → settings')`, app.ID)

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Screen refs
	if spec.Screens[0].Ref != "@s1" {
		t.Fatalf("expected @s1, got %s", spec.Screens[0].Ref)
	}
	if spec.Screens[1].Ref != "@s2" {
		t.Fatalf("expected @s2, got %s", spec.Screens[1].Ref)
	}
	// Screen IDs
	if spec.Screens[0].ID != 1 {
		t.Fatalf("expected ID 1, got %d", spec.Screens[0].ID)
	}
	if spec.Screens[1].ID != 2 {
		t.Fatalf("expected ID 2, got %d", spec.Screens[1].ID)
	}

	// Region refs
	inbox := spec.Screens[0]
	if len(inbox.Regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(inbox.Regions))
	}
	if inbox.Regions[0].Ref != "@r1" {
		t.Fatalf("expected @r1, got %s", inbox.Regions[0].Ref)
	}
	if inbox.Regions[0].ID != 1 {
		t.Fatalf("expected region ID 1, got %d", inbox.Regions[0].ID)
	}

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
}

func TestLoadTastes(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	s.DB.Exec(`INSERT INTO tastes (app_id, name, tokens) VALUES (?, 'dark', '{"bg":"#000","fg":"#fff"}')`, app.ID)
	s.DB.Exec(`INSERT INTO tastes (app_id, name, tokens) VALUES (?, 'light', '{"bg":"#fff","fg":"#000"}')`, app.ID)

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Tastes) != 2 {
		t.Fatalf("expected 2 tastes, got %d", len(spec.Tastes))
	}
	// ordered by name: dark, light
	if spec.Tastes[0].Name != "dark" {
		t.Errorf("expected first taste 'dark', got %q", spec.Tastes[0].Name)
	}
	if spec.Tastes[1].Name != "light" {
		t.Errorf("expected second taste 'light', got %q", spec.Tastes[1].Name)
	}
	if spec.Tastes[0].Tokens["bg"] != "#000" {
		t.Errorf("unexpected token: %v", spec.Tastes[0].Tokens)
	}
}

// --- malformed JSON error propagation tests ---

func TestMalformedDataTypeFieldsReturnsError(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	addScreen(t, s, app.ID, "S1", "screen")

	s.DB.Exec(`INSERT INTO data_types(app_id, name, fields) VALUES(?, 'Email', '{"subject":"string","body":"string"}')`, app.ID)
	if _, err := Load(s.DB, nil); err != nil {
		t.Fatalf("valid data type should not error: %v", err)
	}

	s.DB.Exec(`UPDATE data_types SET fields = '{bad' WHERE name = 'Email'`)
	_, err := Load(s.DB, nil)
	if err == nil {
		t.Fatal("expected error for malformed data_type fields JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got: %v", err)
	}
}

func TestMalformedEnumValuesReturnsError(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	addScreen(t, s, app.ID, "S1", "screen")

	s.DB.Exec(`INSERT INTO enums(app_id, name, "values") VALUES(?, 'Status', '["active","inactive"]')`, app.ID)
	if _, err := Load(s.DB, nil); err != nil {
		t.Fatalf("valid enum should not error: %v", err)
	}

	s.DB.Exec(`UPDATE enums SET "values" = '[broken' WHERE name = 'Status'`)
	_, err := Load(s.DB, nil)
	if err == nil {
		t.Fatal("expected error for malformed enum values JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got: %v", err)
	}
}

func TestMalformedFixtureDataReturnsError(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)
	addScreen(t, s, app.ID, "S1", "screen")

	s.DB.Exec(`INSERT INTO fixtures(app_id, name, data) VALUES(?, 'default', '{"key":"value"}')`, app.ID)
	if _, err := Load(s.DB, nil); err != nil {
		t.Fatalf("valid fixture should not error: %v", err)
	}

	s.DB.Exec(`UPDATE fixtures SET data = '{{bad' WHERE name = 'default'`)
	_, err := Load(s.DB, nil)
	if err == nil {
		t.Fatal("expected error for malformed fixture data JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got: %v", err)
	}
}

func TestDeriveStatesDedup(t *testing.T) {
	// Direct unit test of deriveStates
	transitions := []Transition{
		{OnEvent: "e1", FromState: "A", ToState: "B"},
		{OnEvent: "e2", FromState: "B", ToState: "C"},
		{OnEvent: "e3", FromState: "A", ToState: "C"},
	}
	got := deriveStates(transitions)
	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
