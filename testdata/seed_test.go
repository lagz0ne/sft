package testdata

import (
	"os"
	"testing"

	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/store"
	"github.com/lagz0ne/sft/internal/validator"
)

func TestSeedGmail(t *testing.T) {
	path := t.TempDir() + "/gmail.db"

	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "Gmail", Description: "Google email client"}
	must(t, s.InsertApp(app))

	inbox := &model.Screen{AppID: app.ID, Name: "Inbox", Description: "Main email list view"}
	compose := &model.Screen{AppID: app.ID, Name: "Compose", Description: "New email composition"}
	detail := &model.Screen{AppID: app.ID, Name: "EmailDetail", Description: "Single email view"}
	must(t, s.InsertScreen(inbox))
	must(t, s.InsertScreen(compose))
	must(t, s.InsertScreen(detail))

	emailList := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: inbox.ID, Name: "EmailList", Description: "Scrollable list of email threads"}
	must(t, s.InsertRegion(emailList))
	searchBar := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: inbox.ID, Name: "SearchBar", Description: "Search input and filters"}
	must(t, s.InsertRegion(searchBar))
	toolbar := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: inbox.ID, Name: "Toolbar", Description: "Bulk actions toolbar"}
	must(t, s.InsertRegion(toolbar))
	composeForm := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: compose.ID, Name: "ComposeForm", Description: "To/CC/BCC/Subject/Body fields"}
	must(t, s.InsertRegion(composeForm))
	attachBar := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: compose.ID, Name: "AttachmentBar", Description: "File attachments area"}
	must(t, s.InsertRegion(attachBar))
	emailRow := &model.Region{AppID: app.ID, ParentType: "region", ParentID: emailList.ID, Name: "EmailRow", Description: "Single email row in list"}
	must(t, s.InsertRegion(emailRow))

	must(t, s.InsertTag(&model.Tag{EntityType: "screen", EntityID: inbox.ID, Tag: "contains:EmailList"}))
	must(t, s.InsertTag(&model.Tag{EntityType: "screen", EntityID: compose.ID, Tag: "overlay"}))
	must(t, s.InsertTag(&model.Tag{EntityType: "region", EntityID: composeForm.ID, Tag: "form"}))

	must(t, s.InsertEvent(&model.Event{RegionID: emailList.ID, Name: "select-email"}))
	must(t, s.InsertEvent(&model.Event{RegionID: searchBar.ID, Name: "search-submit"}))
	must(t, s.InsertEvent(&model.Event{RegionID: composeForm.ID, Name: "send-email"}))
	must(t, s.InsertEvent(&model.Event{RegionID: emailRow.ID, Name: "select-email"}))

	must(t, s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: inbox.ID, OnEvent: "select-email", FromState: "list", ToState: "detail"}))
	must(t, s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: inbox.ID, OnEvent: "back", FromState: "detail", ToState: "list"}))
	must(t, s.InsertTransition(&model.Transition{OwnerType: "app", OwnerID: app.ID, OnEvent: "compose-new", Action: "open Compose"}))

	flow := &model.Flow{AppID: app.ID, Name: "SendEmail", Description: "Compose and send a new email", OnEvent: "compose-new", Sequence: "Compose > ComposeForm > send-email > Inbox"}
	must(t, s.InsertFlow(flow))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 1, Raw: "Compose", Type: "screen", Name: "Compose"}))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 2, Raw: "ComposeForm", Type: "region", Name: "ComposeForm"}))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 3, Raw: "send-email", Type: "event", Name: "send-email"}))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 4, Raw: "Inbox", Type: "screen", Name: "Inbox"}))

	// Verify round-trip
	appID, err := s.ResolveApp()
	if err != nil || appID != app.ID {
		t.Fatalf("ResolveApp: got %d, err %v", appID, err)
	}
	sid, err := s.ResolveScreen("Inbox")
	if err != nil || sid != inbox.ID {
		t.Fatalf("ResolveScreen: got %d, err %v", sid, err)
	}
	rid, err := s.ResolveRegion("EmailList")
	if err != nil || rid != emailList.ID {
		t.Fatalf("ResolveRegion: got %d, err %v", rid, err)
	}

	// Verify impact
	impacts, err := s.ImpactScreen("Inbox")
	if err != nil {
		t.Fatal(err)
	}
	if len(impacts) == 0 {
		t.Fatal("expected impacts for Inbox")
	}

	// Verify move
	must(t, s.MoveRegion("EmailRow", "Compose"))
	pt, pid, err := s.ResolveParent("Compose")
	if err != nil {
		t.Fatal(err)
	}
	var newParentType string
	var newParentID int64
	s.DB.QueryRow("SELECT parent_type, parent_id FROM regions WHERE name = 'EmailRow'").Scan(&newParentType, &newParentID)
	if newParentType != pt || newParentID != pid {
		t.Fatalf("move failed: got %s/%d, want %s/%d", newParentType, newParentID, pt, pid)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAutoInit(t *testing.T) {
	path := t.TempDir() + "/sub/dir/test.db"
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Should be able to insert immediately — no explicit Init needed
	a := &model.App{Name: "Test", Description: "test"}
	if err := s.InsertApp(a); err != nil {
		t.Fatal(err)
	}

	// Open again — IF NOT EXISTS should be idempotent
	s2, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	id, err := s2.ResolveApp()
	if err != nil || id != a.ID {
		t.Fatalf("reopen: got %d, err %v", id, err)
	}
}

func TestDeleteScreen(t *testing.T) {
	path := t.TempDir() + "/del.db"
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	sc := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen"}
	must(t, s.InsertScreen(sc))
	r := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: sc.ID, Name: "R1", Description: "region"}
	must(t, s.InsertRegion(r))
	must(t, s.InsertEvent(&model.Event{RegionID: r.ID, Name: "ev1"}))

	must(t, s.DeleteScreen("S1"))

	_, err = s.ResolveScreen("S1")
	if err == nil {
		t.Fatal("screen should be deleted")
	}
	_, err = s.ResolveRegion("R1")
	if err == nil {
		t.Fatal("child region should be deleted")
	}
}

// --- Round 2 Fixes ---

// F1: Duplicate app guard
func TestDuplicateAppBlocked(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f1.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	must(t, s.InsertApp(&model.App{Name: "App1", Description: "first"}))
	err = s.InsertApp(&model.App{Name: "App2", Description: "second"})
	if err == nil {
		t.Fatal("expected error inserting duplicate app")
	}
}

// F2: Deletion cascades navigate() references
func TestDeleteCascadesNavigate(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f2.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	s1 := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen 1"}
	s2 := &model.Screen{AppID: app.ID, Name: "S2", Description: "screen 2"}
	must(t, s.InsertScreen(s1))
	must(t, s.InsertScreen(s2))

	// S1 has a navigate(S2) transition
	must(t, s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: s1.ID, OnEvent: "go", Action: "navigate(S2)"}))

	// Delete S2 should cascade the navigate reference
	must(t, s.DeleteScreen("S2"))

	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM transitions WHERE action = 'navigate(S2)'").Scan(&count)
	if count != 0 {
		t.Fatalf("expected navigate(S2) transitions to be deleted, got %d", count)
	}
}

// F3: Dangling navigate validation
func TestDanglingNavigateValidation(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f3.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	sc := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen"}
	must(t, s.InsertScreen(sc))

	// Add a transition with navigate to non-existent screen
	must(t, s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: sc.ID, OnEvent: "go", Action: "navigate(NoSuchScreen)"}))

	findings, err := validator.Validate(s.DB)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if f.Rule == "dangling-navigate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected dangling-navigate finding")
	}
}

// F6: Impact shows incoming navigate refs
func TestImpactIncomingNavigate(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f6.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	s1 := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen 1"}
	s2 := &model.Screen{AppID: app.ID, Name: "S2", Description: "screen 2"}
	must(t, s.InsertScreen(s1))
	must(t, s.InsertScreen(s2))

	must(t, s.InsertTransition(&model.Transition{OwnerType: "screen", OwnerID: s1.ID, OnEvent: "go", Action: "navigate(S2)"}))

	impacts, err := s.ImpactScreen("S2")
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, imp := range impacts {
		if imp.Type == "navigates-here" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected navigates-here impact")
	}
}

// F7: Extended flow-ref validation for region and event references
func TestFlowRefValidation(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f7.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	sc := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen"}
	must(t, s.InsertScreen(sc))

	flow := &model.Flow{AppID: app.ID, Name: "F1", Sequence: "S1 > BadRegion > BadEvent > S1"}
	must(t, s.InsertFlow(flow))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 1, Raw: "S1", Type: "screen", Name: "S1"}))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 2, Raw: "BadRegion", Type: "region", Name: "BadRegion"}))
	must(t, s.InsertFlowStep(&model.FlowStep{FlowID: flow.ID, Position: 3, Raw: "BadEvent", Type: "event", Name: "BadEvent"}))

	findings, err := validator.Validate(s.DB)
	if err != nil {
		t.Fatal(err)
	}

	regionRef, eventRef := false, false
	for _, f := range findings {
		if f.Rule == "invalid-flow-ref" {
			if f.Message == `flow "F1" references unknown region "BadRegion"` {
				regionRef = true
			}
			if f.Message == `flow "F1" references unknown event "BadEvent"` {
				eventRef = true
			}
		}
	}
	if !regionRef {
		t.Fatal("expected invalid-flow-ref for bad region")
	}
	if !eventRef {
		t.Fatal("expected invalid-flow-ref for bad event")
	}
}

// F8: Attach validates entity existence
func TestAttachValidatesEntity(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f8.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	must(t, s.InsertApp(&model.App{Name: "App", Description: "test"}))

	// Create a temp file to attach
	tmp := t.TempDir() + "/test.txt"
	if err := writeFile(tmp, "test"); err != nil {
		t.Fatal(err)
	}

	// Attaching to non-existent entity should fail
	_, err = s.Attach("NonExistent", tmp, "", "")
	if err == nil {
		t.Fatal("expected error attaching to non-existent entity")
	}

	// Attaching to "_" (global) should work
	_, err = s.Attach("_", tmp, "", "")
	if err != nil {
		t.Fatalf("global attach failed: %v", err)
	}

	// Attaching to existing app should work
	_, err = s.Attach("App", tmp, "", "")
	if err != nil {
		t.Fatalf("app attach failed: %v", err)
	}
}

// F10: Unhandled event validation
func TestUnhandledEventValidation(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f10.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "App", Description: "test"}
	must(t, s.InsertApp(app))
	sc := &model.Screen{AppID: app.ID, Name: "S1", Description: "screen"}
	must(t, s.InsertScreen(sc))
	r := &model.Region{AppID: app.ID, ParentType: "screen", ParentID: sc.ID, Name: "R1", Description: "region"}
	must(t, s.InsertRegion(r))

	// Add an event with no handler
	must(t, s.InsertEvent(&model.Event{RegionID: r.ID, Name: "orphan-ev"}))

	findings, err := validator.Validate(s.DB)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range findings {
		if f.Rule == "unhandled-event" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected unhandled-event finding")
	}
}

// F11: App root component
func TestAppComponent(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/f11.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	app := &model.App{Name: "MyApp", Description: "test"}
	must(t, s.InsertApp(app))

	must(t, s.SetComponent("MyApp", "AppShell", `{"theme":"dark"}`, "", ""))

	comp := s.GetComponentByName("MyApp")
	if comp == nil {
		t.Fatal("expected component on app")
	}
	if comp.Component != "AppShell" {
		t.Fatalf("expected AppShell, got %s", comp.Component)
	}
	if comp.EntityType != "app" {
		t.Fatalf("expected entity_type app, got %s", comp.EntityType)
	}
}

func TestAttachmentContentIDAndHash(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/cid.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	must(t, s.InsertApp(&model.App{Name: "App", Description: "test"}))

	tmp := t.TempDir() + "/mock.png"
	if err := writeFile(tmp, "fake-image-data"); err != nil {
		t.Fatal(err)
	}

	_, err = s.Attach("App", tmp, "", "figma:node123")
	if err != nil {
		t.Fatalf("attach with content_id: %v", err)
	}

	attachments, err := s.ListAttachments("App")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	a := attachments[0]
	if a.ContentID == nil || *a.ContentID != "figma:node123" {
		t.Fatalf("expected content_id figma:node123, got %v", a.ContentID)
	}
	if a.ContentHash == "" {
		t.Fatal("expected non-empty content_hash")
	}
	if len(a.ContentHash) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d chars: %s", len(a.ContentHash), a.ContentHash)
	}
}

func TestAttachHashChangesOnReattach(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/rehash.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	must(t, s.InsertApp(&model.App{Name: "App", Description: "test"}))

	tmp := t.TempDir() + "/data.txt"
	if err := writeFile(tmp, "version-1"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Attach("App", tmp, "", "")
	if err != nil {
		t.Fatal(err)
	}

	list1, _ := s.ListAttachments("App")
	hash1 := list1[0].ContentHash

	// Re-attach different content
	if err := writeFile(tmp, "version-2"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Attach("App", tmp, "", "ext:updated")
	if err != nil {
		t.Fatal(err)
	}

	list2, _ := s.ListAttachments("App")
	hash2 := list2[0].ContentHash

	if hash1 == hash2 {
		t.Fatalf("expected hash to change on re-attach, both are %s", hash1)
	}
	if list2[0].ContentID == nil || *list2[0].ContentID != "ext:updated" {
		t.Fatal("expected content_id to be updated on re-attach")
	}
}

func TestSetContentID(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/setcid.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	must(t, s.InsertApp(&model.App{Name: "App", Description: "test"}))

	tmp := t.TempDir() + "/file.txt"
	if err := writeFile(tmp, "content"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Attach("App", tmp, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetContentID("App", "file.txt", "gdoc:abc"); err != nil {
		t.Fatalf("SetContentID: %v", err)
	}

	list, _ := s.ListAttachments("App")
	if list[0].ContentID == nil || *list[0].ContentID != "gdoc:abc" {
		t.Fatal("expected content_id gdoc:abc after SetContentID")
	}

	if list[0].ContentHash == "" {
		t.Fatal("hash should still be present after SetContentID")
	}

	// SetContentID on non-existent attachment should fail
	if err := s.SetContentID("App", "nope.txt", "x"); err == nil {
		t.Fatal("expected error for non-existent attachment")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
