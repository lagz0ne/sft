package testdata

import (
	"testing"

	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/store"
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
