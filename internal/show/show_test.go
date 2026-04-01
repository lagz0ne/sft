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

func TestRefs(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	sc1 := addScreen(t, s, app.ID, "inbox", "email inbox")
	addScreen(t, s, app.ID, "settings", "app settings")
	addRegion(t, s, app.ID, "screen", sc1.ID, "email_list", "list of emails")

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

func TestLoad_Entities(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	if err := s.InsertEntity(&model.Entity{
		AppID: app.ID, Name: "User", Type: "model",
		Data: `{"id": "number", "name": "string"}`,
	}); err != nil {
		t.Fatalf("insert entity: %v", err)
	}
	if err := s.InsertEntity(&model.Entity{
		AppID: app.ID, Name: "Product", Type: "enum",
		Data: `["basic", "premium"]`,
	}); err != nil {
		t.Fatalf("insert entity: %v", err)
	}

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(spec.Entities))
	}

	// Sorted by name: Product, User
	if spec.Entities[0].Name != "Product" {
		t.Errorf("entities[0].Name = %q, want Product", spec.Entities[0].Name)
	}
	if spec.Entities[0].Type != "enum" {
		t.Errorf("entities[0].Type = %q, want enum", spec.Entities[0].Type)
	}
	if _, ok := spec.Entities[0].Data.([]any); !ok {
		t.Errorf("entities[0].Data type = %T, want []any", spec.Entities[0].Data)
	}

	if spec.Entities[1].Name != "User" {
		t.Errorf("entities[1].Name = %q, want User", spec.Entities[1].Name)
	}
	if spec.Entities[1].Type != "model" {
		t.Errorf("entities[1].Type = %q, want model", spec.Entities[1].Type)
	}
	if m, ok := spec.Entities[1].Data.(map[string]any); !ok {
		t.Errorf("entities[1].Data type = %T, want map[string]any", spec.Entities[1].Data)
	} else if m["id"] != "number" {
		t.Errorf("entities[1].Data[id] = %v, want number", m["id"])
	}
}

func TestLoad_Experiments(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	if err := s.InsertExperiment(&model.Experiment{
		AppID:       app.ID,
		Name:        "dark-mode",
		Description: "Test dark theme",
		Scope:       "Settings",
		Overlay:     `{"theme": "dark"}`,
		Status:      "active",
	}); err != nil {
		t.Fatalf("insert experiment: %v", err)
	}

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Experiments) != 1 {
		t.Fatalf("expected 1 experiment, got %d", len(spec.Experiments))
	}

	exp := spec.Experiments[0]
	if exp.Name != "dark-mode" {
		t.Errorf("Name = %q, want dark-mode", exp.Name)
	}
	if exp.Description != "Test dark theme" {
		t.Errorf("Description = %q, want 'Test dark theme'", exp.Description)
	}
	if exp.Scope != "Settings" {
		t.Errorf("Scope = %q, want Settings", exp.Scope)
	}
	if exp.Status != "active" {
		t.Errorf("Status = %q, want active", exp.Status)
	}
	if m, ok := exp.Overlay.(map[string]any); !ok {
		t.Errorf("Overlay type = %T, want map[string]any", exp.Overlay)
	} else if m["theme"] != "dark" {
		t.Errorf("Overlay[theme] = %v, want dark", m["theme"])
	}
}

func TestLoad_Catalog(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	if err := s.InsertComponentSchema(&model.ComponentSchema{
		AppID:    app.ID,
		Name:     "author-badge",
		Props:    `{"name":"string"}`,
		Template: "<Badge>{name}</Badge>",
	}); err != nil {
		t.Fatalf("insert author-badge: %v", err)
	}
	if err := s.InsertComponentSchema(&model.ComponentSchema{
		AppID:    app.ID,
		Name:     "recipe-card",
		Props:    `{"title":"string","image":"string"}`,
		Template: "<Card>{title}</Card>",
	}); err != nil {
		t.Fatalf("insert recipe-card: %v", err)
	}

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Catalog) != 2 {
		t.Fatalf("expected 2 catalog entries, got %d", len(spec.Catalog))
	}
	if spec.Catalog[0].Name != "author-badge" {
		t.Fatalf("catalog[0].Name = %q, want author-badge", spec.Catalog[0].Name)
	}
	if spec.Catalog[0].Template != "<Badge>{name}</Badge>" {
		t.Fatalf("catalog[0].Template = %q, want badge template", spec.Catalog[0].Template)
	}
	if spec.Catalog[1].Name != "recipe-card" {
		t.Fatalf("catalog[1].Name = %q, want recipe-card", spec.Catalog[1].Name)
	}
	if spec.Catalog[1].Props["title"] != "string" || spec.Catalog[1].Props["image"] != "string" {
		t.Fatalf("catalog[1].Props = %#v", spec.Catalog[1].Props)
	}
}

func TestLoad_ScreenEntry(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	addScreen(t, s, app.ID, "Home", "Landing page")
	addScreen(t, s, app.ID, "Settings", "User settings")

	if err := s.SetEntryScreen(app.ID, "Home"); err != nil {
		t.Fatalf("set entry: %v", err)
	}

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(spec.Screens) != 2 {
		t.Fatalf("expected 2 screens, got %d", len(spec.Screens))
	}

	if !spec.Screens[0].Entry {
		t.Error("Home should be entry screen")
	}
	if spec.Screens[1].Entry {
		t.Error("Settings should not be entry screen")
	}
}

func TestLoad_EmptyEntitiesAndExperiments(t *testing.T) {
	s := mustStore(t)
	seedApp(t, s)

	spec, err := Load(s.DB, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if spec.Entities != nil {
		t.Errorf("expected nil entities, got %v", spec.Entities)
	}
	if spec.Experiments != nil {
		t.Errorf("expected nil experiments, got %v", spec.Experiments)
	}
}

func TestLoad_MalformedEntityDataReturnsError(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	s.DB.Exec(`INSERT INTO entities(app_id, name, type, data) VALUES(?, 'Bad', 'model', '{broken')`, app.ID)
	_, err := Load(s.DB, nil)
	if err == nil {
		t.Fatal("expected error for malformed entity data JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got: %v", err)
	}
}

func TestLoad_MalformedExperimentOverlayReturnsError(t *testing.T) {
	s := mustStore(t)
	app := seedApp(t, s)

	s.DB.Exec(`INSERT INTO experiments(app_id, name, description, scope, overlay, status) VALUES(?, 'Bad', '', '', '{broken', 'active')`, app.ID)
	_, err := Load(s.DB, nil)
	if err == nil {
		t.Fatal("expected error for malformed experiment overlay JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got: %v", err)
	}
}
