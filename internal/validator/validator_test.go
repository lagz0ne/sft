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
