package flow

import (
	"fmt"
	"testing"
)

// stubResolver implements Resolver for testing.
type stubResolver struct {
	screens map[string]int64
	regions map[string]int64
	events  map[string]bool
}

func (r *stubResolver) ResolveScreen(name string) (int64, error) {
	if id, ok := r.screens[name]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("not found")
}

func (r *stubResolver) ResolveRegion(name string) (int64, error) {
	if id, ok := r.regions[name]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("not found")
}

func (r *stubResolver) IsEvent(name string) bool {
	return r.events[name]
}

func TestBasicSequence(t *testing.T) {
	steps := ParseSequence("A → B → C", 1, nil)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	for i, want := range []string{"A", "B", "C"} {
		if steps[i].Name != want {
			t.Errorf("step %d: name = %q, want %q", i, steps[i].Name, want)
		}
		if steps[i].Position != i+1 {
			t.Errorf("step %d: position = %d, want %d", i, steps[i].Position, i+1)
		}
	}
}

func TestBackAndHistory(t *testing.T) {
	steps := ParseSequence("A → [Back] → A(H)", 1, nil)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[1].Type != "back" {
		t.Errorf("step 1: type = %q, want back", steps[1].Type)
	}
	if steps[1].Name != "Back" {
		t.Errorf("step 1: name = %q, want Back", steps[1].Name)
	}
	if steps[2].Name != "A" {
		t.Errorf("step 2: name = %q, want A", steps[2].Name)
	}
	if steps[2].History != 1 {
		t.Errorf("step 2: history = %d, want 1", steps[2].History)
	}
}

func TestActivate(t *testing.T) {
	steps := ParseSequence("Drawer activates → fill", 1, nil)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Type != "activate" {
		t.Errorf("step 0: type = %q, want activate", steps[0].Type)
	}
	if steps[0].Name != "Drawer" {
		t.Errorf("step 0: name = %q, want Drawer", steps[0].Name)
	}
}

func TestDataExtraction(t *testing.T) {
	steps := ParseSequence("A{x} → B{y+z}", 1, nil)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Data != "x" {
		t.Errorf("step 0: data = %q, want x", steps[0].Data)
	}
	if steps[1].Data != "y+z" {
		t.Errorf("step 1: data = %q, want y+z", steps[1].Data)
	}
}

func TestEventResolution(t *testing.T) {
	r := &stubResolver{
		screens: map[string]int64{},
		regions: map[string]int64{},
		events:  map[string]bool{"click": true},
	}
	steps := ParseSequence("click → Submit", 1, r)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Type != "event" {
		t.Errorf("step 0: type = %q, want event", steps[0].Type)
	}
	// Submit is PascalCase, no resolver match → heuristic → screen
	if steps[1].Type != "screen" {
		t.Errorf("step 1: type = %q, want screen", steps[1].Type)
	}
}

func TestNilResolverAllAction(t *testing.T) {
	steps := ParseSequence("foo → bar", 1, nil)
	for i, s := range steps {
		if s.Type != "action" {
			t.Errorf("step %d: type = %q, want action (nil resolver)", i, s.Type)
		}
	}
}

func TestEmptyTokensSkipped(t *testing.T) {
	steps := ParseSequence("A →  → B", 1, nil)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps (empty skipped), got %d", len(steps))
	}
}

func TestFallbackSeparator(t *testing.T) {
	steps := ParseSequence("A > B > C", 1, nil)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps with > separator, got %d", len(steps))
	}
}

func TestScreenAndRegionResolution(t *testing.T) {
	r := &stubResolver{
		screens: map[string]int64{"Home": 1},
		regions: map[string]int64{"Sidebar": 2},
		events:  map[string]bool{},
	}
	steps := ParseSequence("Home → Sidebar → other", 1, r)
	if steps[0].Type != "screen" {
		t.Errorf("step 0: type = %q, want screen", steps[0].Type)
	}
	if steps[1].Type != "region" {
		t.Errorf("step 1: type = %q, want region", steps[1].Type)
	}
	if steps[2].Type != "action" {
		t.Errorf("step 2: type = %q, want action", steps[2].Type)
	}
}
