package store

import (
	"testing"

	"github.com/lagz0ne/sft/internal/model"
)

func mustOpen(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seedApp(t *testing.T, s *Store) *model.App {
	t.Helper()
	a := &model.App{Name: "TestApp", Description: "test"}
	if err := s.InsertApp(a); err != nil {
		t.Fatalf("InsertApp: %v", err)
	}
	return a
}

func TestRenameRegionByID(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Screen1", Description: "s1"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	r1 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "RegA", Description: "a"}
	r2 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "RegB", Description: "b"}
	if err := s.InsertRegion(r1); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertRegion(r2); err != nil {
		t.Fatal(err)
	}

	// Rename RegA → RegX; RegB should be untouched
	if err := s.RenameRegion("RegA", "RegX", "Screen1"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.ResolveRegion("RegX"); err != nil {
		t.Error("RegX should exist after rename")
	}
	if _, err := s.ResolveRegion("RegB"); err != nil {
		t.Error("RegB should still exist")
	}
	if _, err := s.ResolveRegion("RegA"); err == nil {
		t.Error("RegA should no longer exist")
	}
}

func TestRenameRegionScopedCollision(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc1 := &model.Screen{AppID: a.ID, Name: "S1", Description: "s1"}
	sc2 := &model.Screen{AppID: a.ID, Name: "S2", Description: "s2"}
	if err := s.InsertScreen(sc1); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertScreen(sc2); err != nil {
		t.Fatal(err)
	}

	r1 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc1.ID, Name: "Reg", Description: "a"}
	r2 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc1.ID, Name: "Other", Description: "b"}
	r3 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc2.ID, Name: "Reg", Description: "c"}
	if err := s.InsertRegion(r1); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertRegion(r2); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertRegion(r3); err != nil {
		t.Fatal(err)
	}

	// Same parent collision: renaming Other→Reg in S1 should fail
	if err := s.RenameRegion("Other", "Reg", "S1"); err == nil {
		t.Error("expected collision error for same-parent rename")
	}

	// Different parent: renaming Reg→NewName in S2 should succeed
	if err := s.RenameRegion("Reg", "NewName", "S2"); err != nil {
		t.Errorf("different-parent rename should succeed: %v", err)
	}
}

func TestInsertScreenBlockedByRegionName(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	r := &model.Region{AppID: a.ID, ParentType: "app", ParentID: a.ID, Name: "Shared", Description: "r"}
	if err := s.InsertRegion(r); err != nil {
		t.Fatal(err)
	}

	sc := &model.Screen{AppID: a.ID, Name: "Shared", Description: "should fail"}
	if err := s.InsertScreen(sc); err == nil {
		t.Error("expected error: screen name collides with region")
	}
}

func TestInsertFlowPopulatesSteps(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Home", Description: "h"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	f := &model.Flow{AppID: a.ID, Name: "TestFlow", Sequence: "Home → action → Home(H)"}
	if err := s.InsertFlow(f); err != nil {
		t.Fatal(err)
	}

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM flow_steps WHERE flow_id = ?", f.ID).Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 flow steps, got %d", count)
	}

	// First step should be classified as "screen"
	var stepType string
	s.DB.QueryRow("SELECT type FROM flow_steps WHERE flow_id = ? AND position = 1", f.ID).Scan(&stepType)
	if stepType != "screen" {
		t.Errorf("step 1 type = %q, want screen", stepType)
	}
}

func TestIsEvent(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	r := &model.Region{AppID: a.ID, ParentType: "app", ParentID: a.ID, Name: "R1", Description: "r"}
	if err := s.InsertRegion(r); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(&model.Event{RegionID: r.ID, Name: "click"}); err != nil {
		t.Fatal(err)
	}

	if !s.IsEvent("click") {
		t.Error("IsEvent(click) should be true")
	}
	if s.IsEvent("nonexistent") {
		t.Error("IsEvent(nonexistent) should be false")
	}
}

func TestSetAndGetComponent(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Detail", Description: "d"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	if err := s.SetComponent("Detail", "DataTable", `{"cols":3}`, "onClick", "auth"); err != nil {
		t.Fatal(err)
	}

	c := s.GetComponentByName("Detail")
	if c == nil {
		t.Fatal("component should exist")
	}
	if c.Component != "DataTable" {
		t.Errorf("component = %q, want DataTable", c.Component)
	}
	if c.Props != `{"cols":3}` {
		t.Errorf("props = %q, want {\"cols\":3}", c.Props)
	}
	if c.OnActions != "onClick" {
		t.Errorf("on_actions = %q, want onClick", c.OnActions)
	}
	if c.Visible != "auth" {
		t.Errorf("visible = %q, want auth", c.Visible)
	}
}
