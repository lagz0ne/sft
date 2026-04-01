package validator

import (
	"testing"

	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/store"
)

func setup(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	app := &model.App{Name: "TestApp", Description: "test app"}
	if err := s.InsertApp(app); err != nil {
		t.Fatalf("InsertApp: %v", err)
	}

	sc := &model.Screen{AppID: app.ID, Name: "Main", Description: "main screen"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatalf("InsertScreen: %v", err)
	}

	return s
}

func findRule(findings []Finding, ruleID string) []Finding {
	var matched []Finding
	for _, f := range findings {
		if f.Rule == ruleID {
			matched = append(matched, f)
		}
	}
	return matched
}

func TestDeadEnd(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// start → stuck, but "stuck" never appears as a from_state
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "click", FromState: "start", ToState: "stuck",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "dead-end")
	if len(matched) == 0 {
		t.Error("expected dead-end finding for state 'stuck'")
	}
}

func TestDeadEnd_NoFalsePositive(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// start → active, active → start — both states have outgoing transitions
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "click", FromState: "start", ToState: "active",
	})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "reset", FromState: "active", ToState: "start",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "dead-end")
	if len(matched) != 0 {
		t.Errorf("expected no dead-end findings, got %d: %v", len(matched), matched)
	}
}

func TestGuardAmbiguity(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Two transitions with same event+from_state, no guards
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "submit", FromState: "start", ToState: "a",
	})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "submit", FromState: "start", ToState: "b",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "guard-ambiguity")
	if len(matched) == 0 {
		t.Error("expected guard-ambiguity finding for submit+start")
	}
}

func TestGuardAmbiguity_WithGuards(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Two transitions with same event+from_state, but both have guard() actions
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "submit", FromState: "start", ToState: "a",
		Action: "guard(isValid)",
	})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "submit", FromState: "start", ToState: "b",
		Action: "guard(isInvalid)",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "guard-ambiguity")
	if len(matched) != 0 {
		t.Errorf("expected no guard-ambiguity findings when guards are present, got %d: %v", len(matched), matched)
	}
}

func TestUndefinedDataType(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Context references "nonexistent" type
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "items", FieldType: "nonexistent[]"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "undefined-data-type")
	if len(matched) == 0 {
		t.Error("expected undefined-data-type finding")
	}
}

func TestUndefinedDataType_ValidPrimitive(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Primitive types should not trigger
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "name", FieldType: "string"})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "tags", FieldType: "string[]"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "undefined-data-type")
	if len(matched) != 0 {
		t.Errorf("unexpected undefined-data-type finding: %v", matched)
	}
}

func TestUndefinedDataType_DefinedType(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Define a custom type, then reference it — should not trigger
	s.InsertDataType(&model.DataType{AppID: appID, Name: "email", Fields: `{"address": "string"}`})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "emails", FieldType: "email[]"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "undefined-data-type")
	if len(matched) != 0 {
		t.Errorf("unexpected undefined-data-type finding for defined type: %v", matched)
	}
}

func TestInvalidAmbientPath(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	region := &model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "widget", Description: "w"}
	s.InsertRegion(region)

	// Source "nonexistent" is not a screen name or "app"
	s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: "items", Source: "nonexistent", Query: ".items"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "invalid-ambient-path")
	if len(matched) == 0 {
		t.Error("expected invalid-ambient-path finding")
	}
}

func TestInvalidAmbientPath_BadQuery(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	region := &model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "widget", Description: "w"}
	s.InsertRegion(region)

	// Valid source but query doesn't start with "."
	s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: "data", Source: "app", Query: "items"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "invalid-ambient-path")
	if len(matched) == 0 {
		t.Error("expected invalid-ambient-path finding for bad query")
	}
}

func TestInvalidAmbientPath_ValidRef(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	region := &model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "widget", Description: "w"}
	s.InsertRegion(region)

	// Valid source "app" and query starts with "."
	s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: "items", Source: "app", Query: ".items"})
	// Valid source is a real screen name
	s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: "data", Source: "Main", Query: ".data"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "invalid-ambient-path")
	if len(matched) != 0 {
		t.Errorf("unexpected invalid-ambient-path finding for valid refs: %v", matched)
	}
}

func TestFixtureNotFound(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Reference a fixture that doesn't exist
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "start", FixtureName: "nonexistent",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "fixture-not-found")
	if len(matched) == 0 {
		t.Error("expected fixture-not-found finding")
	}
}

func TestFixtureNotFound_Valid(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Create a fixture, then reference it — should not trigger
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "my_fixture", Data: `{"key": "val"}`})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "start", FixtureName: "my_fixture",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "fixture-not-found")
	if len(matched) != 0 {
		t.Errorf("unexpected fixture-not-found finding for valid fixture: %v", matched)
	}
}

func TestOrphanFixture(t *testing.T) {
	s := setup(t)
	appID := int64(1)

	// Create a fixture that isn't referenced by any state or extends
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "unused_fixture", Data: `{"key": "val"}`})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "orphan-fixture")
	if len(matched) == 0 {
		t.Error("expected orphan-fixture finding")
	}
}

func TestOrphanFixture_Referenced(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Create a fixture and reference it — should not trigger
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "used_fixture", Data: `{"key": "val"}`})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "start", FixtureName: "used_fixture",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "orphan-fixture")
	if len(matched) != 0 {
		t.Errorf("unexpected orphan-fixture finding for referenced fixture: %v", matched)
	}
}

func TestNavigateWithParams(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Create the target screen so navigate is valid
	target := &model.Screen{AppID: appID, Name: "OrderDetail", Description: "order detail"}
	if err := s.InsertScreen(target); err != nil {
		t.Fatalf("InsertScreen: %v", err)
	}

	// navigate(OrderDetail, { order: data(order_list, .selected) })
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "select_order", FromState: "start",
		Action: "navigate(OrderDetail, { order: data(order_list, .selected) })",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "dangling-navigate")
	if len(matched) != 0 {
		t.Errorf("navigate with params should not trigger dangling-navigate: %v", matched)
	}
}

func TestNavigateWithParams_DanglingTarget(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// navigate(Nonexistent, { order: data(list, .selected) }) — target doesn't exist
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "select_order", FromState: "start",
		Action: "navigate(Nonexistent, { order: data(list, .selected) })",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "dangling-navigate")
	if len(matched) == 0 {
		t.Error("expected dangling-navigate finding for nonexistent target with params")
	}
}

func TestUndefinedDataType_OptionalSuffix(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")

	// Optional primitives should not trigger
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "name", FieldType: "string?"})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "tags", FieldType: "string[]?"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "undefined-data-type")
	if len(matched) != 0 {
		t.Errorf("unexpected undefined-data-type finding for optional types: %v", matched)
	}
}

func TestUndefinedDataType_OptionalCustomType(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Define a custom type, then reference it with ? — should not trigger
	s.InsertDataType(&model.DataType{AppID: appID, Name: "email", Fields: `{"address": "string"}`})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "primary", FieldType: "email?"})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "all", FieldType: "email[]?"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "undefined-data-type")
	if len(matched) != 0 {
		t.Errorf("unexpected undefined-data-type finding for optional custom type: %v", matched)
	}
}

func TestOrphanFixture_ExtendedBase(t *testing.T) {
	s := setup(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// base_fixture is not directly referenced by a state, but child_fixture extends it
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "base_fixture", Data: `{"key": "base"}`})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "child_fixture", Extends: "base_fixture", Data: `{"extra": "val"}`})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "start", FixtureName: "child_fixture",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "orphan-fixture")
	if len(matched) != 0 {
		t.Errorf("unexpected orphan-fixture finding for base fixture used via extends: %v", matched)
	}
}

// setupWithEntry creates an app with an entry screen.
func setupWithEntry(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	app := &model.App{Name: "TestApp", Description: "test app"}
	if err := s.InsertApp(app); err != nil {
		t.Fatalf("InsertApp: %v", err)
	}

	sc := &model.Screen{AppID: app.ID, Name: "Main", Description: "main screen"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatalf("InsertScreen: %v", err)
	}

	if err := s.SetEntryScreen(app.ID, "Main"); err != nil {
		t.Fatalf("SetEntryScreen: %v", err)
	}

	return s
}

// --- Entry screen rules ---

func TestEntryScreenMissing(t *testing.T) {
	// setup() creates a screen without entry=true
	s := setup(t)

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "entry-screen-missing")
	if len(matched) == 0 {
		t.Error("expected entry-screen-missing finding when no screen has entry=true")
	}
}

func TestEntryScreenMissing_HasEntry(t *testing.T) {
	s := setupWithEntry(t)

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "entry-screen-missing")
	if len(matched) != 0 {
		t.Errorf("unexpected entry-screen-missing finding when entry screen exists: %v", matched)
	}
}

func TestEntryScreenMultiple(t *testing.T) {
	s := setupWithEntry(t)
	appID := int64(1)

	// Add a second screen
	s.InsertScreen(&model.Screen{AppID: appID, Name: "Other", Description: "other screen"})
	// Set entry=1 directly via SQL since SetEntryScreen clears existing entry flags
	s.DB.Exec("UPDATE screens SET entry = 1 WHERE name = 'Other'")

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "entry-screen-multiple")
	if len(matched) == 0 {
		t.Error("expected entry-screen-multiple finding when two screens have entry=true")
	}
}

func TestEntryScreenMultiple_SingleEntry(t *testing.T) {
	s := setupWithEntry(t)

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "entry-screen-multiple")
	if len(matched) != 0 {
		t.Errorf("unexpected entry-screen-multiple finding with single entry screen: %v", matched)
	}
}

// --- Leaf region no content ---

func TestLeafRegionNoContent(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Create a leaf region with no component
	s.InsertRegion(&model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "empty_leaf", Description: "leaf"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "leaf-region-no-content")
	if len(matched) == 0 {
		t.Error("expected leaf-region-no-content finding for region with no component")
	}
}

func TestLeafRegionNoContent_HasComponent(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	region := &model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "with_comp", Description: "has comp"}
	s.InsertRegion(region)

	// Insert component directly via SQL
	s.DB.Exec("INSERT INTO components (entity_type, entity_id, component, props) VALUES ('region', ?, 'card', '{}')", region.ID)

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "leaf-region-no-content")
	if len(matched) != 0 {
		t.Errorf("unexpected leaf-region-no-content for region with component: %v", matched)
	}
}

func TestLeafRegionNoContent_HasChildren(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	parent := &model.Region{AppID: appID, ParentType: "screen", ParentID: screenID, Name: "parent_reg", Description: "parent"}
	s.InsertRegion(parent)
	child := &model.Region{AppID: appID, ParentType: "region", ParentID: parent.ID, Name: "child_reg", Description: "child"}
	s.InsertRegion(child)

	// parent has children so is not a leaf — but child is a leaf with no component
	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "leaf-region-no-content")
	// parent_reg should not appear as the REGION being flagged (it may appear as a parent name).
	// Only child_reg should be flagged as a leaf.
	for _, f := range matched {
		if contains(f.Message, `leaf region "parent_reg"`) {
			t.Errorf("parent region with children should not be flagged as leaf: %v", f)
		}
	}
	// child_reg should be flagged
	foundChild := false
	for _, f := range matched {
		if contains(f.Message, `leaf region "child_reg"`) {
			foundChild = true
		}
	}
	if !foundChild {
		t.Error("expected child_reg to be flagged as leaf with no component")
	}
}

// --- Unreferenced data type ---

func TestUnreferencedDataType(t *testing.T) {
	s := setupWithEntry(t)
	appID := int64(1)

	// Create a data type not referenced anywhere
	s.InsertDataType(&model.DataType{AppID: appID, Name: "orphan_type", Fields: `{"x": "string"}`})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "unreferenced-data-type")
	if len(matched) == 0 {
		t.Error("expected unreferenced-data-type finding")
	}
}

func TestUnreferencedDataType_UsedInContext(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	s.InsertDataType(&model.DataType{AppID: appID, Name: "used_type", Fields: `{"x": "string"}`})
	s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screenID, FieldName: "items", FieldType: "used_type[]"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "unreferenced-data-type")
	if len(matched) != 0 {
		t.Errorf("unexpected unreferenced-data-type for type used in context: %v", matched)
	}
}

// --- State-region no fixture ---

func TestStateRegionNoFixture(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")

	// State region without a corresponding state fixture
	s.InsertStateRegion(&model.StateRegion{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "active", RegionName: "sidebar",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "state-region-no-fixture")
	if len(matched) == 0 {
		t.Error("expected state-region-no-fixture finding")
	}
}

func TestStateRegionNoFixture_HasFixture(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	s.InsertStateRegion(&model.StateRegion{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "active", RegionName: "sidebar",
	})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "active_data", Data: `{}`})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "active", FixtureName: "active_data",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "state-region-no-fixture")
	if len(matched) != 0 {
		t.Errorf("unexpected state-region-no-fixture when fixture exists: %v", matched)
	}
}

// --- State without fixture (screen-level) ---

func TestStateWithoutFixture(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")

	// Add screen-level transitions defining states, but no fixtures
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "init", FromState: "idle", ToState: "active",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "state-without-fixture")
	if len(matched) == 0 {
		t.Error("expected state-without-fixture finding for screen states without fixtures")
	}
}

func TestStateWithoutFixture_HasFixture(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "init", FromState: "idle", ToState: "active",
	})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "idle_data", Data: `{}`})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "active_data", Data: `{}`})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "idle", FixtureName: "idle_data",
	})
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "active", FixtureName: "active_data",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "state-without-fixture")
	if len(matched) != 0 {
		t.Errorf("unexpected state-without-fixture when all states have fixtures: %v", matched)
	}
}

// --- Fixture extends cycle ---

func TestFixtureExtendsCycle(t *testing.T) {
	s := setupWithEntry(t)
	appID := int64(1)

	// Create a circular extends chain: a -> b -> a
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "fixture_a", Extends: "fixture_b", Data: `{}`})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "fixture_b", Extends: "fixture_a", Data: `{}`})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "fixture-extends-cycle")
	if len(matched) == 0 {
		t.Error("expected fixture-extends-cycle finding for circular extends")
	}
}

func TestFixtureExtendsCycle_NoCycle(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Linear chain: child -> base (no cycle)
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "base", Data: `{"x": 1}`})
	s.InsertFixture(&model.Fixture{AppID: appID, Name: "child", Extends: "base", Data: `{"y": 2}`})
	// Reference child so it's not orphaned
	s.InsertStateFixture(&model.StateFixture{
		OwnerType: "screen", OwnerID: screenID,
		StateName: "start", FixtureName: "child",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "fixture-extends-cycle")
	if len(matched) != 0 {
		t.Errorf("unexpected fixture-extends-cycle for linear chain: %v", matched)
	}
}

// --- Screen unreachable ---

func TestScreenUnreachable(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	// Create a reachable screen via navigate
	s.InsertScreen(&model.Screen{AppID: appID, Name: "Detail", Description: "detail screen"})
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "click", Action: "navigate(Detail)",
	})

	// Create an unreachable screen (no navigate to it)
	s.InsertScreen(&model.Screen{AppID: appID, Name: "Hidden", Description: "hidden screen"})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "screen-unreachable")
	// Should flag Hidden but not Detail or Main
	foundHidden := false
	for _, f := range matched {
		if contains(f.Message, `screen "Hidden"`) {
			foundHidden = true
		}
		if contains(f.Message, `screen "Main" is not reachable`) {
			t.Errorf("entry screen should not be flagged as unreachable: %v", f)
		}
		if contains(f.Message, `screen "Detail" is not reachable`) {
			t.Errorf("reachable screen should not be flagged: %v", f)
		}
	}
	if !foundHidden {
		t.Error("expected screen-unreachable finding for Hidden screen")
	}
}

func TestScreenUnreachable_AllReachable(t *testing.T) {
	s := setupWithEntry(t)
	screenID, _ := s.ResolveScreen("Main")
	appID := int64(1)

	detailScreen := &model.Screen{AppID: appID, Name: "Detail", Description: "detail"}
	s.InsertScreen(detailScreen)
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: screenID,
		OnEvent: "click", Action: "navigate(Detail)",
	})
	// Detail navigates back to Main
	s.InsertTransition(&model.Transition{
		OwnerType: "screen", OwnerID: detailScreen.ID,
		OnEvent: "back", Action: "navigate(Main)",
	})

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "screen-unreachable")
	if len(matched) != 0 {
		t.Errorf("unexpected screen-unreachable when all screens are reachable: %v", matched)
	}
}

func TestScreenUnreachable_NoEntryScreen(t *testing.T) {
	// When no entry screen exists, screen-unreachable should not fire
	s := setup(t) // setup() creates screen without entry=true

	findings, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	matched := findRule(findings, "screen-unreachable")
	if len(matched) != 0 {
		t.Errorf("screen-unreachable should not fire when no entry screen: %v", matched)
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Verify that the sql.DB field is accessible for fn-based rules
func TestFnDispatch(t *testing.T) {
	s := setupWithEntry(t)

	// Validate should work with mixed SQL and fn rules without error
	_, err := Validate(s.DB)
	if err != nil {
		t.Fatalf("Validate with fn rules: %v", err)
	}
}
