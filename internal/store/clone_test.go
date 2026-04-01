package store

import (
	"testing"

	"github.com/lagz0ne/sft/internal/model"
)

// seedScreenWithRegions creates a screen with nested regions, events, tags,
// components, ambient refs, region data, and context fields for clone testing.
func seedScreenWithRegions(t *testing.T, s *Store) (screenID int64) {
	t.Helper()
	a := seedApp(t, s)

	sc := &model.Screen{AppID: a.ID, Name: "inbox", Description: "email inbox"}
	if err := s.InsertScreen(sc); err != nil {
		t.Fatal(err)
	}

	// Screen-level context
	if err := s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: sc.ID, FieldName: "emails", FieldType: "email[]"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: sc.ID, FieldName: "selected", FieldType: "email[]"}); err != nil {
		t.Fatal(err)
	}

	// Screen-level tag
	if err := s.InsertTag(&model.Tag{EntityType: "screen", EntityID: sc.ID, Tag: "main_screen"}); err != nil {
		t.Fatal(err)
	}

	// Screen-level transitions (state machine)
	if err := s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: sc.ID, OnEvent: "select_email", FromState: "empty", ToState: "has_selection"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: sc.ID, OnEvent: "escape", FromState: "has_selection", ToState: "empty"}); err != nil {
		t.Fatal(err)
	}

	// Screen-level state fixtures
	if err := s.InsertStateFixture(&model.StateFixture{OwnerType: "screen", OwnerID: sc.ID, StateName: "empty", FixtureName: "inbox_empty"}); err != nil {
		t.Fatal(err)
	}

	// Screen-level state regions
	if err := s.InsertStateRegion(&model.StateRegion{OwnerType: "screen", OwnerID: sc.ID, StateName: "has_selection", RegionName: "email_detail"}); err != nil {
		t.Fatal(err)
	}

	// Region 1: email_list (direct child of screen)
	r1 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "email_list", Description: "email list"}
	if err := s.InsertRegion(r1); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(&model.Event{RegionID: r1.ID, Name: "select_email", Annotation: "email"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(&model.Event{RegionID: r1.ID, Name: "check_email", Annotation: "email"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTag(&model.Tag{EntityType: "region", EntityID: r1.ID, Tag: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertAmbientRef(&model.AmbientRef{RegionID: r1.ID, LocalName: "emails", Source: "inbox", Query: ".emails"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertRegionData(&model.RegionData{RegionID: r1.ID, FieldName: "selected_id", FieldType: "number"}); err != nil {
		t.Fatal(err)
	}

	// Nested region: email_row (child of email_list)
	r2 := &model.Region{AppID: a.ID, ParentType: "region", ParentID: r1.ID, Name: "email_row", Description: "single email row"}
	if err := s.InsertRegion(r2); err != nil {
		t.Fatal(err)
	}
	if err := s.SetComponent("email_row", "card", `{"variant":"compact"}`, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTag(&model.Tag{EntityType: "region", EntityID: r2.ID, Tag: "list_item"}); err != nil {
		t.Fatal(err)
	}

	// Deep nested: checkbox (child of email_row)
	r3 := &model.Region{AppID: a.ID, ParentType: "region", ParentID: r2.ID, Name: "checkbox", Description: "selection checkbox"}
	if err := s.InsertRegion(r3); err != nil {
		t.Fatal(err)
	}
	if err := s.SetComponent("checkbox", "input", `{"type":"checkbox"}`, "", ""); err != nil {
		t.Fatal(err)
	}

	// Region 2: toolbar (direct child of screen, sibling of email_list)
	r4 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "toolbar", Description: "action toolbar"}
	if err := s.InsertRegion(r4); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertTag(&model.Tag{EntityType: "region", EntityID: r4.ID, Tag: "toolbar"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(&model.Event{RegionID: r4.ID, Name: "archive"}); err != nil {
		t.Fatal(err)
	}

	// Region with delivery/discovery columns
	r5 := &model.Region{AppID: a.ID, ParentType: "screen", ParentID: sc.ID, Name: "sidebar", Description: "nav sidebar",
		DiscoveryLayout: "sidebar", DeliveryClasses: "w-64 shrink-0"}
	if err := s.InsertRegion(r5); err != nil {
		t.Fatal(err)
	}

	// Region-level transitions
	if err := s.InsertTransition(&model.Transition{OwnerType: "region", OwnerID: r1.ID, OnEvent: "select_email", ToState: "selected"}); err != nil {
		t.Fatal(err)
	}

	return sc.ID
}

func TestCloneScreen(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)

	err := s.CloneScreen("inbox", "inbox_v2")
	if err != nil {
		t.Fatalf("CloneScreen: %v", err)
	}

	origID, err := s.ResolveScreen("inbox")
	if err != nil {
		t.Fatalf("original missing: %v", err)
	}
	cloneID, err := s.ResolveScreen("inbox_v2")
	if err != nil {
		t.Fatalf("clone missing: %v", err)
	}
	if origID == cloneID {
		t.Error("clone should have different ID from original")
	}
}

func TestCloneScreen_RegionsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	origCount := countRegionsUnder(t, s, "screen", origID)
	if origCount == 0 {
		t.Fatal("original screen has no regions")
	}

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatalf("CloneScreen: %v", err)
	}

	cloneID, _ := s.ResolveScreen("inbox_v2")
	cloneCount := countRegionsUnder(t, s, "screen", cloneID)

	if origCount != cloneCount {
		t.Errorf("region count: orig=%d clone=%d", origCount, cloneCount)
	}
}

func TestCloneScreen_EventsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origEvents := countEventsUnderScreen(t, s, origID)
	cloneEvents := countEventsUnderScreen(t, s, cloneID)

	if origEvents == 0 {
		t.Fatal("original has no events")
	}
	if origEvents != cloneEvents {
		t.Errorf("event count: orig=%d clone=%d", origEvents, cloneEvents)
	}
}

func TestCloneScreen_TagsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origTags := countTagsUnderScreen(t, s, origID)
	cloneTags := countTagsUnderScreen(t, s, cloneID)

	if origTags == 0 {
		t.Fatal("original has no tags")
	}
	if origTags != cloneTags {
		t.Errorf("tag count: orig=%d clone=%d", origTags, cloneTags)
	}
}

func TestCloneScreen_ComponentsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origComps := countComponentsUnderScreen(t, s, origID)
	cloneComps := countComponentsUnderScreen(t, s, cloneID)

	if origComps == 0 {
		t.Fatal("original has no components")
	}
	if origComps != cloneComps {
		t.Errorf("component count: orig=%d clone=%d", origComps, cloneComps)
	}
}

func TestCloneScreen_AmbientRefsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origRefs := countAmbientRefsUnderScreen(t, s, origID)
	cloneRefs := countAmbientRefsUnderScreen(t, s, cloneID)

	if origRefs == 0 {
		t.Fatal("original has no ambient refs")
	}
	if origRefs != cloneRefs {
		t.Errorf("ambient ref count: orig=%d clone=%d", origRefs, cloneRefs)
	}
}

func TestCloneScreen_RegionDataCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origData := countRegionDataUnderScreen(t, s, origID)
	cloneData := countRegionDataUnderScreen(t, s, cloneID)

	if origData == 0 {
		t.Fatal("original has no region data")
	}
	if origData != cloneData {
		t.Errorf("region_data count: orig=%d clone=%d", origData, cloneData)
	}
}

func TestCloneScreen_TransitionsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	// Screen-level transitions
	origScreenT := countTransitions(t, s, "screen", origID)
	cloneScreenT := countTransitions(t, s, "screen", cloneID)
	if origScreenT == 0 {
		t.Fatal("original has no screen transitions")
	}
	if origScreenT != cloneScreenT {
		t.Errorf("screen transition count: orig=%d clone=%d", origScreenT, cloneScreenT)
	}

	// Region-level transitions (should be copied with new region IDs)
	origRegionT := countRegionTransitionsUnderScreen(t, s, origID)
	cloneRegionT := countRegionTransitionsUnderScreen(t, s, cloneID)
	if origRegionT == 0 {
		t.Fatal("original has no region transitions")
	}
	if origRegionT != cloneRegionT {
		t.Errorf("region transition count: orig=%d clone=%d", origRegionT, cloneRegionT)
	}
}

func TestCloneScreen_ContextCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origCtx := countContextFields(t, s, "screen", origID)
	cloneCtx := countContextFields(t, s, "screen", cloneID)

	if origCtx == 0 {
		t.Fatal("original has no context")
	}
	if origCtx != cloneCtx {
		t.Errorf("context count: orig=%d clone=%d", origCtx, cloneCtx)
	}
}

func TestCloneScreen_StateFixturesCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origSF := countStateFixtures(t, s, "screen", origID)
	cloneSF := countStateFixtures(t, s, "screen", cloneID)

	if origSF == 0 {
		t.Fatal("original has no state fixtures")
	}
	if origSF != cloneSF {
		t.Errorf("state fixture count: orig=%d clone=%d", origSF, cloneSF)
	}
}

func TestCloneScreen_StateRegionsCopied(t *testing.T) {
	s := mustOpen(t)
	origID := seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	origSR := countStateRegions(t, s, "screen", origID)
	cloneSR := countStateRegions(t, s, "screen", cloneID)

	if origSR == 0 {
		t.Fatal("original has no state regions")
	}
	if origSR != cloneSR {
		t.Errorf("state region count: orig=%d clone=%d", origSR, cloneSR)
	}
}

func TestCloneScreen_DeliveryDiscoveryCopied(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)

	if err := s.CloneScreen("inbox", "inbox_v2"); err != nil {
		t.Fatal(err)
	}
	cloneID, _ := s.ResolveScreen("inbox_v2")

	// The sidebar region has discovery_layout and delivery_classes set
	var layout, classes *string
	rows, _ := s.db().Query(`SELECT discovery_layout, delivery_classes FROM regions
		WHERE parent_type = 'screen' AND parent_id = ? AND name = 'sidebar'`, cloneID)
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&layout, &classes)
	}
	if layout == nil || *layout != "sidebar" {
		t.Errorf("discovery_layout not copied, got %v", layout)
	}
	if classes == nil || *classes != "w-64 shrink-0" {
		t.Errorf("delivery_classes not copied, got %v", classes)
	}
}

func TestCloneScreen_DuplicateName(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)
	err := s.CloneScreen("inbox", "inbox")
	if err == nil {
		t.Error("expected error when cloning to existing name")
	}
}

func TestCloneScreen_SourceNotFound(t *testing.T) {
	s := mustOpen(t)
	seedApp(t, s)
	err := s.CloneScreen("nonexistent", "new_name")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

// --- CloneRegion ---

func TestCloneRegion(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)

	err := s.CloneRegion("email_list", "email_list_v2", "inbox")
	if err != nil {
		t.Fatalf("CloneRegion: %v", err)
	}

	origID, _ := s.ResolveRegionIn("email_list", "inbox")
	cloneID, _ := s.ResolveRegionIn("email_list_v2", "inbox")

	if origID == cloneID {
		t.Error("clone should have different ID")
	}

	// Children were cloned
	origChildren := countRegionsUnder(t, s, "region", origID)
	cloneChildren := countRegionsUnder(t, s, "region", cloneID)

	if origChildren == 0 {
		t.Fatal("email_list has no children")
	}
	if origChildren != cloneChildren {
		t.Errorf("child count: orig=%d clone=%d", origChildren, cloneChildren)
	}
}

func TestCloneRegion_EventsAndTagsCopied(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)

	if err := s.CloneRegion("email_list", "email_list_v2", "inbox"); err != nil {
		t.Fatal(err)
	}

	origID, _ := s.ResolveRegionIn("email_list", "inbox")
	cloneID, _ := s.ResolveRegionIn("email_list_v2", "inbox")

	// Events on the root cloned region
	var origEvents, cloneEvents int
	s.db().QueryRow("SELECT COUNT(*) FROM events WHERE region_id = ?", origID).Scan(&origEvents)
	s.db().QueryRow("SELECT COUNT(*) FROM events WHERE region_id = ?", cloneID).Scan(&cloneEvents)
	if origEvents == 0 {
		t.Fatal("email_list has no events")
	}
	if origEvents != cloneEvents {
		t.Errorf("event count: orig=%d clone=%d", origEvents, cloneEvents)
	}

	// Tags on the root cloned region
	var origTags, cloneTags int
	s.db().QueryRow("SELECT COUNT(*) FROM tags WHERE entity_type='region' AND entity_id = ?", origID).Scan(&origTags)
	s.db().QueryRow("SELECT COUNT(*) FROM tags WHERE entity_type='region' AND entity_id = ?", cloneID).Scan(&cloneTags)
	if origTags == 0 {
		t.Fatal("email_list has no tags")
	}
	if origTags != cloneTags {
		t.Errorf("tag count: orig=%d clone=%d", origTags, cloneTags)
	}
}

func TestCloneRegion_SourceNotFound(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)
	err := s.CloneRegion("nonexistent", "new_name", "inbox")
	if err == nil {
		t.Error("expected error for nonexistent source region")
	}
}

func TestCloneRegion_TransitionsCopied(t *testing.T) {
	s := mustOpen(t)
	seedScreenWithRegions(t, s)

	if err := s.CloneRegion("email_list", "email_list_v2", "inbox"); err != nil {
		t.Fatal(err)
	}

	origID, _ := s.ResolveRegionIn("email_list", "inbox")
	cloneID, _ := s.ResolveRegionIn("email_list_v2", "inbox")

	origT := countTransitions(t, s, "region", origID)
	cloneT := countTransitions(t, s, "region", cloneID)
	if origT == 0 {
		t.Fatal("email_list has no transitions")
	}
	if origT != cloneT {
		t.Errorf("transition count: orig=%d clone=%d", origT, cloneT)
	}
}

// --- counting helpers ---

func countRegionsUnder(t *testing.T, s *Store, parentType string, parentID int64) int {
	t.Helper()
	var count int
	rows, err := s.db().Query("SELECT id FROM regions WHERE parent_type = ? AND parent_id = ?", parentType, parentID)
	if err != nil {
		t.Fatalf("query regions: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
		count++
	}
	for _, id := range ids {
		count += countRegionsUnder(t, s, "region", id)
	}
	return count
}

func countEventsUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow(`SELECT COUNT(*) FROM events WHERE region_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&count)
	return count
}

func countTagsUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM tags WHERE entity_type = 'screen' AND entity_id = ?", screenID).Scan(&count)
	var regionTags int
	s.db().QueryRow(`SELECT COUNT(*) FROM tags WHERE entity_type = 'region' AND entity_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&regionTags)
	return count + regionTags
}

func countComponentsUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM components WHERE entity_type = 'screen' AND entity_id = ?", screenID).Scan(&count)
	var regionComps int
	s.db().QueryRow(`SELECT COUNT(*) FROM components WHERE entity_type = 'region' AND entity_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&regionComps)
	return count + regionComps
}

func countAmbientRefsUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow(`SELECT COUNT(*) FROM ambient_refs WHERE region_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&count)
	return count
}

func countRegionDataUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow(`SELECT COUNT(*) FROM region_data WHERE region_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&count)
	return count
}

func countContextFields(t *testing.T, s *Store, ownerType string, ownerID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM contexts WHERE owner_type = ? AND owner_id = ?", ownerType, ownerID).Scan(&count)
	return count
}

func countTransitions(t *testing.T, s *Store, ownerType string, ownerID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM transitions WHERE owner_type = ? AND owner_id = ?", ownerType, ownerID).Scan(&count)
	return count
}

func countRegionTransitionsUnderScreen(t *testing.T, s *Store, screenID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow(`SELECT COUNT(*) FROM transitions WHERE owner_type = 'region' AND owner_id IN (
		WITH RECURSIVE tree(id) AS (
			SELECT id FROM regions WHERE parent_type = 'screen' AND parent_id = ?
			UNION ALL
			SELECT r.id FROM regions r JOIN tree ON r.parent_type = 'region' AND r.parent_id = tree.id
		) SELECT id FROM tree
	)`, screenID).Scan(&count)
	return count
}

func countStateFixtures(t *testing.T, s *Store, ownerType string, ownerID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM state_fixtures WHERE owner_type = ? AND owner_id = ?", ownerType, ownerID).Scan(&count)
	return count
}

func countStateRegions(t *testing.T, s *Store, ownerType string, ownerID int64) int {
	t.Helper()
	var count int
	s.db().QueryRow("SELECT COUNT(*) FROM state_regions WHERE owner_type = ? AND owner_id = ?", ownerType, ownerID).Scan(&count)
	return count
}
