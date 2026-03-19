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
