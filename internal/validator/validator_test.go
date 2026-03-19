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
