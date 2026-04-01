package diff

import (
	"encoding/json"
	"testing"

	"github.com/lagz0ne/sft/internal/show"
)

// --- helpers ---

func findChange(changes []Change, op, entity, name string) *Change {
	for _, c := range changes {
		if c.Op == op && c.Entity == entity && c.Name == name {
			return &c
		}
	}
	return nil
}

func findChangeIn(changes []Change, op, entity, name, in string) *Change {
	for _, c := range changes {
		if c.Op == op && c.Entity == entity && c.Name == name && c.In == in {
			return &c
		}
	}
	return nil
}

func findChangeDetail(changes []Change, op, entity, name, detail string) *Change {
	for _, c := range changes {
		if c.Op == op && c.Entity == entity && c.Name == name && c.Detail == detail {
			return &c
		}
	}
	return nil
}

func assertChange(t *testing.T, changes []Change, op, entity, name string) {
	t.Helper()
	if findChange(changes, op, entity, name) == nil {
		t.Errorf("expected change {op:%q entity:%q name:%q}, got %s", op, entity, name, jsonChanges(changes))
	}
}

func assertChangeIn(t *testing.T, changes []Change, op, entity, name, in string) {
	t.Helper()
	if findChangeIn(changes, op, entity, name, in) == nil {
		t.Errorf("expected change {op:%q entity:%q name:%q in:%q}, got %s", op, entity, name, in, jsonChanges(changes))
	}
}

func assertChangeDetail(t *testing.T, changes []Change, op, entity, name, detail string) {
	t.Helper()
	if findChangeDetail(changes, op, entity, name, detail) == nil {
		t.Errorf("expected change {op:%q entity:%q name:%q detail:%q}, got %s", op, entity, name, detail, jsonChanges(changes))
	}
}

func assertNoChanges(t *testing.T, changes []Change) {
	t.Helper()
	if len(changes) > 0 {
		t.Errorf("expected no changes, got %s", jsonChanges(changes))
	}
}

func assertCount(t *testing.T, changes []Change, n int) {
	t.Helper()
	if len(changes) != n {
		t.Errorf("expected %d changes, got %d: %s", n, len(changes), jsonChanges(changes))
	}
}

func jsonChanges(changes []Change) string {
	b, _ := json.MarshalIndent(changes, "", "  ")
	return string(b)
}

// ===== Existing coverage: screens, regions, events, transitions, tags =====

func TestCompare_EmptySpecs(t *testing.T) {
	changes := Compare(&show.Spec{}, &show.Spec{})
	assertNoChanges(t, changes)
}

func TestCompare_AppDescriptionChanged(t *testing.T) {
	cur := &show.Spec{App: show.App{Name: "myapp", Description: "old"}}
	tgt := &show.Spec{App: show.App{Name: "myapp", Description: "new"}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "app", "myapp", "description changed")
}

func TestCompare_ScreenAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "screen", "inbox")
}

func TestCompare_ScreenRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "screen", "inbox")
}

func TestCompare_ScreenDescriptionChanged(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", Description: "old"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", Description: "new"}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "description changed")
}

func TestCompare_RegionAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", Regions: []show.Region{{Name: "list"}}}}}
	changes := Compare(cur, tgt)
	assertChangeIn(t, changes, "+", "region", "list", "inbox")
}

func TestCompare_RegionRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", Regions: []show.Region{{Name: "list"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	changes := Compare(cur, tgt)
	assertChangeIn(t, changes, "-", "region", "list", "inbox")
}

func TestCompare_EventAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", Events: []string{"click"}}}}}}
	changes := Compare(cur, tgt)
	assertChangeIn(t, changes, "+", "event", "click", "r")
}

func TestCompare_TagAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Tags: []string{"sidebar"}}}}
	changes := Compare(cur, tgt)
	assertChangeIn(t, changes, "+", "tag", "sidebar", "s")
}

func TestCompare_TransitionAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Transitions: []show.Transition{{OnEvent: "click", ToState: "active"}}}}}
	changes := Compare(cur, tgt)
	assertChangeIn(t, changes, "+", "transition", "click", "s")
}

// ===== New coverage: data_types =====

func TestCompare_DataTypeAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string"},
	}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "data_type", "email")
}

func TestCompare_DataTypeRemoved(t *testing.T) {
	cur := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string"},
	}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "data_type", "email")
}

func TestCompare_DataTypeFieldAdded(t *testing.T) {
	cur := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string"},
	}}}
	tgt := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string", "body": "string"},
	}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "data_type", "email")
	if c == nil {
		t.Fatalf("expected modification change for data_type email, got %s", jsonChanges(changes))
	}
}

func TestCompare_DataTypeFieldChanged(t *testing.T) {
	cur := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string"},
	}}}
	tgt := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "number"},
	}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "data_type", "email")
	if c == nil {
		t.Fatalf("expected modification change for data_type email, got %s", jsonChanges(changes))
	}
}

func TestCompare_DataTypeFieldRemoved(t *testing.T) {
	cur := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string", "body": "string"},
	}}}
	tgt := &show.Spec{App: show.App{DataTypes: map[string]map[string]string{
		"email": {"subject": "string"},
	}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "data_type", "email")
	if c == nil {
		t.Fatalf("expected modification change for data_type email, got %s", jsonChanges(changes))
	}
}

func TestCompare_DataTypeNoChange(t *testing.T) {
	dt := map[string]map[string]string{"email": {"subject": "string"}}
	cur := &show.Spec{App: show.App{DataTypes: dt}}
	tgt := &show.Spec{App: show.App{DataTypes: dt}}
	changes := Compare(cur, tgt)
	for _, c := range changes {
		if c.Entity == "data_type" {
			t.Errorf("unexpected data_type change: %+v", c)
		}
	}
}

// ===== New coverage: enums =====

func TestCompare_EnumAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "medium", "high"},
	}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "enum", "priority")
}

func TestCompare_EnumRemoved(t *testing.T) {
	cur := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "medium", "high"},
	}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "enum", "priority")
}

func TestCompare_EnumValueAdded(t *testing.T) {
	cur := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "high"},
	}}}
	tgt := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "medium", "high"},
	}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "enum", "priority")
	if c == nil {
		t.Fatalf("expected modification change for enum priority, got %s", jsonChanges(changes))
	}
}

func TestCompare_EnumValueRemoved(t *testing.T) {
	cur := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "medium", "high"},
	}}}
	tgt := &show.Spec{App: show.App{Enums: map[string][]string{
		"priority": {"low", "high"},
	}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "enum", "priority")
	if c == nil {
		t.Fatalf("expected modification change for enum priority, got %s", jsonChanges(changes))
	}
}

func TestCompare_EnumNoChange(t *testing.T) {
	enums := map[string][]string{"priority": {"low", "high"}}
	cur := &show.Spec{App: show.App{Enums: enums}}
	tgt := &show.Spec{App: show.App{Enums: enums}}
	changes := Compare(cur, tgt)
	for _, c := range changes {
		if c.Entity == "enum" {
			t.Errorf("unexpected enum change: %+v", c)
		}
	}
}

// ===== New coverage: context (app-level) =====

func TestCompare_AppContextAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{App: show.App{Context: map[string]string{"user": "User"}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "context", "user")
}

func TestCompare_AppContextRemoved(t *testing.T) {
	cur := &show.Spec{App: show.App{Context: map[string]string{"user": "User"}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "context", "user")
}

func TestCompare_AppContextChanged(t *testing.T) {
	cur := &show.Spec{App: show.App{Context: map[string]string{"user": "User"}}}
	tgt := &show.Spec{App: show.App{Context: map[string]string{"user": "Admin"}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "~", "context", "user")
}

func TestCompare_AppContextNoChange(t *testing.T) {
	ctx := map[string]string{"user": "User"}
	cur := &show.Spec{App: show.App{Context: ctx}}
	tgt := &show.Spec{App: show.App{Context: ctx}}
	changes := Compare(cur, tgt)
	for _, c := range changes {
		if c.Entity == "context" {
			t.Errorf("unexpected context change: %+v", c)
		}
	}
}

// ===== New coverage: fixtures =====

func TestCompare_FixtureAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{Fixtures: []show.Fixture{{Name: "default", Data: "{}"}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "fixture", "default")
}

func TestCompare_FixtureRemoved(t *testing.T) {
	cur := &show.Spec{Fixtures: []show.Fixture{{Name: "default", Data: "{}"}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "fixture", "default")
}

func TestCompare_FixtureExtendsChanged(t *testing.T) {
	cur := &show.Spec{Fixtures: []show.Fixture{{Name: "extra", Extends: "default", Data: "{}"}}}
	tgt := &show.Spec{Fixtures: []show.Fixture{{Name: "extra", Extends: "other", Data: "{}"}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "fixture", "extra")
	if c == nil {
		t.Fatalf("expected modification change for fixture extra, got %s", jsonChanges(changes))
	}
}

func TestCompare_FixtureDataChanged(t *testing.T) {
	cur := &show.Spec{Fixtures: []show.Fixture{{Name: "default", Data: "old"}}}
	tgt := &show.Spec{Fixtures: []show.Fixture{{Name: "default", Data: "new"}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "fixture", "default")
	if c == nil {
		t.Fatalf("expected modification change for fixture default, got %s", jsonChanges(changes))
	}
}

func TestCompare_FixtureNoChange(t *testing.T) {
	fixtures := []show.Fixture{{Name: "default", Data: "same"}}
	cur := &show.Spec{Fixtures: fixtures}
	tgt := &show.Spec{Fixtures: fixtures}
	changes := Compare(cur, tgt)
	for _, c := range changes {
		if c.Entity == "fixture" {
			t.Errorf("unexpected fixture change: %+v", c)
		}
	}
}

// ===== New coverage: layouts =====

func TestCompare_LayoutAdded(t *testing.T) {
	cur := &show.Spec{}
	tgt := &show.Spec{Layouts: map[string][]string{"wide": {"col-span-2"}}}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "+", "layout", "wide")
}

func TestCompare_LayoutRemoved(t *testing.T) {
	cur := &show.Spec{Layouts: map[string][]string{"wide": {"col-span-2"}}}
	tgt := &show.Spec{}
	changes := Compare(cur, tgt)
	assertChange(t, changes, "-", "layout", "wide")
}

func TestCompare_LayoutClassesChanged(t *testing.T) {
	cur := &show.Spec{Layouts: map[string][]string{"wide": {"col-span-2"}}}
	tgt := &show.Spec{Layouts: map[string][]string{"wide": {"col-span-3", "gap-4"}}}
	changes := Compare(cur, tgt)
	c := findChange(changes, "~", "layout", "wide")
	if c == nil {
		t.Fatalf("expected modification change for layout wide, got %s", jsonChanges(changes))
	}
}

func TestCompare_LayoutNoChange(t *testing.T) {
	layouts := map[string][]string{"wide": {"col-span-2"}}
	cur := &show.Spec{Layouts: layouts}
	tgt := &show.Spec{Layouts: layouts}
	changes := Compare(cur, tgt)
	for _, c := range changes {
		if c.Entity == "layout" {
			t.Errorf("unexpected layout change: %+v", c)
		}
	}
}

// ===== New coverage: region-level delivery_classes =====

func TestCompare_RegionDeliveryClassesAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", DeliveryClasses: []string{"bg-white"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected delivery_classes change for region r, got %s", jsonChanges(changes))
	}
}

func TestCompare_RegionDeliveryClassesRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", DeliveryClasses: []string{"bg-white"}}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected delivery_classes change for region r, got %s", jsonChanges(changes))
	}
}

// ===== New coverage: region-level discovery_layout =====

func TestCompare_RegionDiscoveryLayoutAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", DiscoveryLayout: []string{"grid-cols-2"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected discovery_layout change for region r, got %s", jsonChanges(changes))
	}
}

// ===== New coverage: region-level ambient =====

func TestCompare_RegionAmbientAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", Ambient: map[string]string{"email": "data(inbox, .selected)"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected ambient change for region r, got %s", jsonChanges(changes))
	}
}

func TestCompare_RegionAmbientChanged(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", Ambient: map[string]string{"email": "data(inbox, .selected)"}}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", Ambient: map[string]string{"email": "data(inbox, .all)"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected ambient change for region r, got %s", jsonChanges(changes))
	}
}

func TestCompare_RegionAmbientRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", Ambient: map[string]string{"email": "data(inbox, .selected)"}}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected ambient change for region r, got %s", jsonChanges(changes))
	}
}

// ===== New coverage: region-level region_data =====

func TestCompare_RegionDataAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", RegionData: map[string]string{"count": "number"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected region_data change for region r, got %s", jsonChanges(changes))
	}
}

// ===== New coverage: screen-level context =====

func TestCompare_ScreenContextAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", Context: map[string]string{"user": "User"}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "context changed")
}

func TestCompare_ScreenContextRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", Context: map[string]string{"user": "User"}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "context changed")
}

func TestCompare_ScreenContextChanged(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", Context: map[string]string{"user": "User"}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", Context: map[string]string{"user": "Admin"}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "context changed")
}

// ===== New coverage: screen-level state_fixtures =====

func TestCompare_ScreenStateFixturesAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateFixtures: map[string]string{"loading": "empty_fixture"}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "state_fixtures changed")
}

func TestCompare_ScreenStateFixturesRemoved(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateFixtures: map[string]string{"loading": "empty_fixture"}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "state_fixtures changed")
}

func TestCompare_ScreenStateFixturesChanged(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateFixtures: map[string]string{"loading": "empty_fixture"}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateFixtures: map[string]string{"loading": "full_fixture"}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "state_fixtures changed")
}

// ===== New coverage: screen-level state_regions =====

func TestCompare_ScreenStateRegionsAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox"}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateRegions: map[string][]string{"active": {"list", "detail"}}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "state_regions changed")
}

func TestCompare_ScreenStateRegionsChanged(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateRegions: map[string][]string{"active": {"list"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "inbox", StateRegions: map[string][]string{"active": {"list", "detail"}}}}}
	changes := Compare(cur, tgt)
	assertChangeDetail(t, changes, "~", "screen", "inbox", "state_regions changed")
}

// ===== New coverage: region-level state_fixtures =====

func TestCompare_RegionStateFixturesAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", StateFixtures: map[string]string{"loading": "empty"}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected state_fixtures change for region r, got %s", jsonChanges(changes))
	}
}

// ===== New coverage: region-level state_regions =====

func TestCompare_RegionStateRegionsAdded(t *testing.T) {
	cur := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r"}}}}}
	tgt := &show.Spec{Screens: []show.Screen{{Name: "s", Regions: []show.Region{{Name: "r", StateRegions: map[string][]string{"active": {"child"}}}}}}}
	changes := Compare(cur, tgt)
	c := findChangeIn(changes, "~", "region", "r", "s")
	if c == nil {
		t.Fatalf("expected state_regions change for region r, got %s", jsonChanges(changes))
	}
}

// ===== Comprehensive: multiple entities changing at once =====

func TestCompare_MultipleChanges(t *testing.T) {
	cur := &show.Spec{
		App: show.App{
			DataTypes: map[string]map[string]string{"email": {"subject": "string"}},
			Enums:     map[string][]string{"status": {"active"}},
			Context:   map[string]string{"user": "User"},
		},
		Screens: []show.Screen{{Name: "inbox"}},
		Fixtures: []show.Fixture{{Name: "default", Data: "old"}},
		Layouts: map[string][]string{"wide": {"col-span-2"}},
	}
	tgt := &show.Spec{
		App: show.App{
			DataTypes: map[string]map[string]string{
				"email":   {"subject": "string", "body": "string"}, // modified
				"contact": {"name": "string"},                      // added
			},
			Enums:   map[string][]string{"status": {"active", "archived"}}, // modified
			Context: map[string]string{"user": "Admin"},                    // modified
		},
		Screens: []show.Screen{
			{Name: "inbox"},
			{Name: "settings"}, // added
		},
		Fixtures: []show.Fixture{{Name: "default", Data: "new"}},                   // modified
		Layouts:  map[string][]string{"wide": {"col-span-3"}, "narrow": {"w-64"}}, // modified + added
	}
	changes := Compare(cur, tgt)

	assertChange(t, changes, "+", "data_type", "contact")
	assertChange(t, changes, "~", "data_type", "email")
	assertChange(t, changes, "~", "enum", "status")
	assertChange(t, changes, "~", "context", "user")
	assertChange(t, changes, "+", "screen", "settings")
	assertChange(t, changes, "~", "fixture", "default")
	assertChange(t, changes, "~", "layout", "wide")
	assertChange(t, changes, "+", "layout", "narrow")
}

// ===== Format =====

func TestFormat_NoChanges(t *testing.T) {
	out := Format(nil)
	if out != "no changes" {
		t.Errorf("expected 'no changes', got %q", out)
	}
}

func TestFormat_WithChanges(t *testing.T) {
	changes := []Change{
		{Op: "+", Entity: "screen", Name: "inbox"},
		{Op: "~", Entity: "region", Name: "list", In: "inbox", Detail: "description changed"},
	}
	out := Format(changes)
	if out == "" || out == "no changes" {
		t.Errorf("expected formatted output, got %q", out)
	}
}

