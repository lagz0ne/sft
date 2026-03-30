package store

import (
	"database/sql"
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

func TestInsertDataType(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	dt := &model.DataType{AppID: a.ID, Name: "email", Fields: `{"subject":"string","sender":"contact","read":"boolean"}`}
	if err := s.InsertDataType(dt); err != nil {
		t.Fatal(err)
	}
	if dt.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Duplicate name should fail (unique constraint)
	dt2 := &model.DataType{AppID: a.ID, Name: "email", Fields: `{}`}
	if err := s.InsertDataType(dt2); err == nil {
		t.Error("expected unique constraint error for duplicate data_type name")
	}
}

func TestInsertContextField(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Inbox", Description: "inbox"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	cf := &model.ContextField{OwnerType: "screen", OwnerID: sc.ID, FieldName: "emails", FieldType: "email[]"}
	if err := s.InsertContextField(cf); err != nil {
		t.Fatal(err)
	}
	if cf.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Duplicate field_name in same owner should fail
	cf2 := &model.ContextField{OwnerType: "screen", OwnerID: sc.ID, FieldName: "emails", FieldType: "string"}
	if err := s.InsertContextField(cf2); err == nil {
		t.Error("expected unique constraint error for duplicate context field")
	}
}

func TestInsertAmbientRef(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Inbox", Description: "inbox"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}
	r := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "EmailList", Description: "list"}
	if err := s.InsertRegion(r); err != nil {
		t.Fatal(err)
	}

	ar := &model.AmbientRef{RegionID: r.ID, LocalName: "emails", Source: "inbox", Query: ".emails"}
	if err := s.InsertAmbientRef(ar); err != nil {
		t.Fatal(err)
	}
	if ar.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Duplicate local_name in same region should fail
	ar2 := &model.AmbientRef{RegionID: r.ID, LocalName: "emails", Source: "other", Query: ".other"}
	if err := s.InsertAmbientRef(ar2); err == nil {
		t.Error("expected unique constraint error for duplicate ambient ref")
	}
}

func TestInsertRegionData(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Inbox", Description: "inbox"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}
	r := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "EmailList", Description: "list"}
	if err := s.InsertRegion(r); err != nil {
		t.Fatal(err)
	}

	rd := &model.RegionData{RegionID: r.ID, FieldName: "selected_id", FieldType: "number"}
	if err := s.InsertRegionData(rd); err != nil {
		t.Fatal(err)
	}
	if rd.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Duplicate field_name in same region should fail
	rd2 := &model.RegionData{RegionID: r.ID, FieldName: "selected_id", FieldType: "string"}
	if err := s.InsertRegionData(rd2); err == nil {
		t.Error("expected unique constraint error for duplicate region data field")
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

func TestLayoutCRUD(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	// Insert layouts
	l1 := &model.Layout{AppID: a.ID, Name: "sidebar", Classes: `["col-span-2","row-start-2"]`}
	if err := s.InsertLayout(l1); err != nil {
		t.Fatal(err)
	}
	if l1.ID == 0 {
		t.Fatal("expected non-zero layout ID")
	}

	l2 := &model.Layout{AppID: a.ID, Name: "top-bar", Classes: `["col-span-full","row-start-1"]`}
	if err := s.InsertLayout(l2); err != nil {
		t.Fatal(err)
	}

	// GetLayouts
	layouts, err := s.GetLayouts(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 2 {
		t.Fatalf("got %d layouts, want 2", len(layouts))
	}

	// GetLayout by name
	got, err := s.GetLayout(a.ID, "sidebar")
	if err != nil {
		t.Fatal(err)
	}
	if got.Classes != `["col-span-2","row-start-2"]` {
		t.Errorf("classes = %q, want [\"col-span-2\",\"row-start-2\"]", got.Classes)
	}

	// Duplicate name → error
	dup := &model.Layout{AppID: a.ID, Name: "sidebar", Classes: `[]`}
	if err := s.InsertLayout(dup); err == nil {
		t.Fatal("expected error on duplicate layout name")
	}
}

func TestRegionLayout(t *testing.T) {
	s := mustOpen(t)
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "Home", Description: "home"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	// Insert with layout fields
	r := &model.Region{
		AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "nav", Description: "nav",
		DiscoveryLayout: `["sidebar"]`, DeliveryClasses: `["w-56","shrink-0"]`, DeliveryComponent: "CustomNav",
	}
	if err := s.InsertRegion(r); err != nil {
		t.Fatal(err)
	}

	// Read back
	var dl, dc, dcomp sql.NullString
	err := s.DB.QueryRow("SELECT discovery_layout, delivery_classes, delivery_component FROM regions WHERE id = ?", r.ID).
		Scan(&dl, &dc, &dcomp)
	if err != nil {
		t.Fatal(err)
	}
	if dl.String != `["sidebar"]` {
		t.Errorf("discovery_layout = %q, want [\"sidebar\"]", dl.String)
	}
	if dc.String != `["w-56","shrink-0"]` {
		t.Errorf("delivery_classes = %q, want [\"w-56\",\"shrink-0\"]", dc.String)
	}
	if dcomp.String != "CustomNav" {
		t.Errorf("delivery_component = %q, want CustomNav", dcomp.String)
	}

	// Empty layout fields → NULL
	r2 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "empty", Description: "empty"}
	if err := s.InsertRegion(r2); err != nil {
		t.Fatal(err)
	}
	err = s.DB.QueryRow("SELECT discovery_layout FROM regions WHERE id = ?", r2.ID).Scan(&dl)
	if err != nil {
		t.Fatal(err)
	}
	if dl.Valid {
		t.Error("expected NULL discovery_layout for empty region")
	}
}
