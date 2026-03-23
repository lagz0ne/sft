package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/lagz0ne/sft/internal/diagram"
	"github.com/lagz0ne/sft/internal/diff"
	"github.com/lagz0ne/sft/internal/format"
	"github.com/lagz0ne/sft/internal/loader"
	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/query"
	"github.com/lagz0ne/sft/internal/render"
	"github.com/lagz0ne/sft/internal/show"
	"github.com/lagz0ne/sft/internal/store"
	"github.com/lagz0ne/sft/internal/validator"
	"github.com/lagz0ne/sft/internal/view"
	"github.com/lagz0ne/sft/web"
)

var version = "dev"

func main() {
	jsonMode := false
	args := make([]string, 0, len(os.Args))
	for _, a := range os.Args {
		if a == "--json" {
			jsonMode = true
		} else {
			args = append(args, a)
		}
	}
	os.Args = args
	format.Init(jsonMode)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	if os.Args[1] == "version" || os.Args[1] == "--version" {
		fmt.Printf("sft %s\n", version)
		return
	}

	s, err := store.Open(store.DefaultPath())
	if err != nil {
		die("open: %v", err)
	}
	defer s.Close()

	cmd := os.Args[1]
	rest := os.Args[2:]

	switch cmd {
	case "show":
		runShow(s)
	case "query", "q":
		runQuery(s, rest)
	case "validate", "check":
		runValidate(s)
	case "init":
		runInit(s, rest)
	case "import":
		runInit(s, rest) // legacy alias
	case "diff":
		runDiff(s, rest)
	case "add":
		runAdd(s, rest)
	case "set":
		runSet(s, rest)
	case "rename":
		runRename(s, rest)
	case "rm":
		runRm(s, rest)
	case "mv":
		runMv(s, rest)
	case "reorder":
		runReorder(s, rest)
	case "impact":
		runImpact(s, rest)
	case "component", "comp":
		runComponent(s, rest)
	case "render":
		runRender(s)
	case "attach":
		runAttach(s, rest)
	case "detach":
		runDetach(s, rest)
	case "list", "ls":
		runList(s, rest)
	case "cat":
		runCat(s, rest)
	case "view":
		runView(s, rest)
	case "diagram", "diag":
		runDiagram(s, rest)
	default:
		die("unknown command %q\n%s", cmd, usage)
	}
}

const usage = `sft — behavioral spec tool for UI screens, regions, events, flows, and components.

All output includes @refs (e.g. @s1, @r3) for stable entity addressing.
All output supports --json. The spec lives in .sft/db.

Workflow:
  sft init spec.yaml             # bootstrap from YAML (one-time, empty DB only)
  sft show                       # full spec tree with @refs
  sft query screens              # list screens, regions, events, flows
  sft validate                   # check for issues
  sft view                       # open in browser

Reading:
  show                             full spec tree with @refs (text or --json)
  query  <type>                    screens | regions | events | flows | tags | types | enums | fixtures | contexts | steps <flow>
  query  states <name>             transitions for a screen/region
  query  <SELECT ...>              raw SQL against the spec DB
  impact <screen|region> <name>    what depends on this entity
  render                           json-render element tree

Mutating (use @refs or names):
  add app <name> <desc>
  add screen <name> <desc>
  add region <name> <desc> --in <@ref|parent>
  add event <name(annotation)> --in <@ref|region>
  add transition --on <event> --in <@ref|owner> [--from <s>] [--to <s>] [--action <a>]
  add tag <tag> --on <@ref|entity>
  add flow <name> <sequence> [--description <d>] [--on <event>]
  set <screen|region> <name> --description <new> [--in <parent>]
  set type <name> --fields <json>
  set enum <name> --values <json>
  set context <field> --type <t> [--on <screen>]
  set field <name> --type <t> --in <region>
  set ambient <name> --ref <data_ref> --in <region>
  set fixture <name> --data <json> [--extends <f>]
  set state-fixture --in <owner> --state <s> --fixture <f>
  rename <screen|region|flow|type|enum|fixture> <old> <new> [--in <parent>]
  rm <type> <name> [--in/--on <parent>]
  mv region <name> --to <@ref|parent> [--in <current-parent>]
  reorder <parent> <child1> <child2> ...

Domain & Data:
  add type <name> <fields_json>
  add enum <name> <values_json>
  add context <field> <type> [--on <screen>]
  add field <name> <type> --in <region>
  add ambient <name> <data_ref> --in <region>

States & Fixtures:
  add fixture <name> <data_json> [--extends <f>]
  add state-fixture <fixture> --in <owner> --state <s>
  add state-region <region> --in <owner> --state <s>

Components:
  component <@ref|entity>                     show bound component
  component <@ref|entity> <Type> [--props {}] bind a component
  component <@ref|entity> --rm                unbind

Attachments:
  attach <@ref|entity> <file> [--as <name>]
  detach <@ref|entity> <name>
  list [entity]
  cat <@ref|entity> <name>

Diff:
  diff <file.yaml>                compare current spec vs a YAML file

Diagrams:
  diagram states <name>           state machine for a screen/region
  diagram nav                     navigation graph
  diagram flow <name>             flow sequence

View:
  view [--port N]                 open spec in browser

Version:
  version                         show sft version
  # If outdated, export your spec with 'sft show --json > backup.json'
  # then update sft and use an LLM to migrate if needed

Aliases: q=query, check=validate, ls=list, comp=component, diag=diagram`

// --- show ---

func runShow(s *store.Store) {
	spec, err := show.Load(s.DB, s)
	if err != nil {
		die("%v", err)
	}
	if format.JSONMode {
		format.JSON(spec)
		return
	}
	show.Render(os.Stdout, spec)
}

// --- query ---

func runQuery(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft query <screens|events|states|flows|tags|regions|types|enums|fixtures|contexts|SELECT ...>")
	}
	name := args[0]
	var results []map[string]any
	var err error
	queryKey := name
	if name == "states" {
		if len(args) < 2 {
			die("usage: sft query states <name>")
		}
		results, err = query.States(s.DB, args[1])
	} else if name == "steps" {
		if len(args) < 2 {
			die("usage: sft query steps <flow-name>")
		}
		results, err = query.Steps(s.DB, args[1])
	} else {
		results, err = query.Run(s.DB, name)
		if name != "screens" && name != "regions" && name != "events" &&
			name != "flows" && name != "tags" && name != "types" &&
			name != "enums" && name != "fixtures" && name != "contexts" {
			queryKey = ""
		}
	}
	if err != nil {
		die("%v", err)
	}
	format.Table(queryKey, results)
}

// --- validate ---

func runValidate(s *store.Store) {
	findings, err := validator.Validate(s.DB)
	if err != nil {
		die("validate: %v", err)
	}
	ff := make([]format.Finding, len(findings))
	for i, f := range findings {
		ff[i] = format.Finding{Rule: f.Rule, Severity: string(f.Severity), Message: f.Message}
	}
	format.Findings(ff)
	for _, f := range findings {
		if f.Severity == validator.Error {
			os.Exit(1)
		}
	}
}

// --- add ---

func runAdd(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft add <app|screen|region|event|transition|tag|flow> ...")
	}
	entity := args[0]
	args = args[1:]

	switch entity {
	case "app":
		need(args, 2, "sft add app <name> <description>")
		a := &model.App{Name: args[0], Description: args[1]}
		must(s.InsertApp(a))
		ok("app %s", a.Name)

	case "screen":
		need(args, 2, "sft add screen <name> <description>")
		appID := mustResolveApp(s)
		sc := &model.Screen{AppID: appID, Name: args[0], Description: args[1]}
		must(s.InsertScreen(sc))
		ok("screen %s", sc.Name)

	case "region":
		inIdx := flagIndex(args, "--in")
		if inIdx == -1 || inIdx+1 >= len(args) || inIdx < 2 {
			die("usage: sft add region <name> <description> --in <parent>")
		}
		parentName := args[inIdx+1]
		parentType, parentID, err := s.ResolveParent(parentName)
		if err != nil {
			die("%v", err)
		}
		appID := mustResolveApp(s)
		r := &model.Region{AppID: appID, ParentType: parentType, ParentID: parentID, Name: args[0], Description: args[1]}
		must(s.InsertRegion(r))
		ok("region %s → %s", r.Name, parentName)

	case "event":
		inIdx := flagIndex(args, "--in")
		if inIdx == -1 || inIdx+1 >= len(args) || inIdx < 1 {
			die("usage: sft add event <name> --in <region>")
		}
		regionName := args[inIdx+1]
		regionID, err := s.ResolveRegion(regionName)
		if err != nil {
			die("%v", err)
		}
		evName, annotation := loader.ParseEventName(args[0])
		e := &model.Event{RegionID: regionID, Name: evName, Annotation: annotation}
		must(s.InsertEvent(e))
		ok("event %s in %s", args[0], regionName)

	case "transition":
		on := flagVal(args, "--on")
		from := flagVal(args, "--from")
		to := flagVal(args, "--to")
		action := flagVal(args, "--action")
		in := flagVal(args, "--in")
		if on == "" || in == "" {
			die("usage: sft add transition --on <event> [--from <s>] [--to <s>] [--action <a>] --in <owner>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		t := &model.Transition{OwnerType: ownerType, OwnerID: ownerID, OnEvent: on, FromState: from, ToState: to, Action: action}
		must(s.InsertTransition(t))
		desc := on
		if from != "" {
			desc += " " + from + " → " + to
		}
		if action != "" {
			desc += " ⇒ " + action
		}
		ok("transition %s in %s", desc, in)

	case "tag":
		onIdx := flagIndex(args, "--on")
		if onIdx == -1 || onIdx+1 >= len(args) || onIdx < 1 {
			die("usage: sft add tag <tag> --on <screen-or-region>")
		}
		tagVal := args[0]
		targetName := args[onIdx+1]
		entityType, entityID, err := s.ResolveScreenOrRegion(targetName)
		if err != nil {
			die("%v", err)
		}
		t := &model.Tag{EntityType: entityType, EntityID: entityID, Tag: tagVal}
		must(s.InsertTag(t))
		ok("tag [%s] on %s", tagVal, targetName)

	case "flow":
		if len(args) < 2 {
			die("usage: sft add flow <name> <sequence> [--description <d>] [--on <event>]")
		}
		appID := mustResolveApp(s)
		f := &model.Flow{
			AppID:       appID,
			Name:        args[0],
			Sequence:    args[1],
			Description: flagVal(args, "--description"),
			OnEvent:     flagVal(args, "--on"),
		}
		must(s.InsertFlow(f))
		ok("flow %s", f.Name)

	case "type":
		need(args, 2, "sft add type <name> <fields_json>")
		appID := mustResolveApp(s)
		dt := &model.DataType{AppID: appID, Name: args[0], Fields: args[1]}
		must(s.InsertDataType(dt))
		ok("type %s", dt.Name)

	case "enum":
		need(args, 2, "sft add enum <name> <values_json>")
		appID := mustResolveApp(s)
		e := &model.Enum{AppID: appID, Name: args[0], Values: args[1]}
		must(s.InsertEnum(e))
		ok("enum %s", e.Name)

	case "context":
		need(args, 2, "sft add context <field> <type> [--on <screen>]")
		on := flagVal(args, "--on")
		ownerType, ownerID := resolveContextOwner(s, on)
		cf := &model.ContextField{OwnerType: ownerType, OwnerID: ownerID, FieldName: args[0], FieldType: args[1]}
		must(s.InsertContextField(cf))
		if on != "" {
			ok("context %s on %s", args[0], on)
		} else {
			ok("context %s", args[0])
		}

	case "field":
		in := flagVal(args, "--in")
		if in == "" || len(args) < 2 {
			die("usage: sft add field <name> <type> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		rd := &model.RegionData{RegionID: regionID, FieldName: args[0], FieldType: args[1]}
		must(s.InsertRegionData(rd))
		ok("field %s in %s", rd.FieldName, in)

	case "ambient":
		in := flagVal(args, "--in")
		if in == "" || len(args) < 2 {
			die("usage: sft add ambient <name> <data_ref> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		source, query, err := loader.ParseDataRef(args[1])
		if err != nil {
			die("%v", err)
		}
		ar := &model.AmbientRef{RegionID: regionID, LocalName: args[0], Source: source, Query: query}
		must(s.InsertAmbientRef(ar))
		ok("ambient %s in %s", ar.LocalName, in)

	case "fixture":
		need(args, 2, "sft add fixture <name> <data_json> [--extends <f>]")
		appID := mustResolveApp(s)
		f := &model.Fixture{AppID: appID, Name: args[0], Data: args[1], Extends: flagVal(args, "--extends")}
		must(s.InsertFixture(f))
		ok("fixture %s", f.Name)

	case "state-fixture":
		in := flagVal(args, "--in")
		state := flagVal(args, "--state")
		if in == "" || state == "" {
			die("usage: sft add state-fixture <fixture> --in <owner> --state <s>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		sf := &model.StateFixture{OwnerType: ownerType, OwnerID: ownerID, StateName: state, FixtureName: args[0]}
		must(s.InsertStateFixture(sf))
		ok("state-fixture %s → %s/%s", sf.FixtureName, in, state)

	case "state-region":
		in := flagVal(args, "--in")
		state := flagVal(args, "--state")
		if in == "" || state == "" {
			die("usage: sft add state-region <region> --in <owner> --state <s>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		sr := &model.StateRegion{OwnerType: ownerType, OwnerID: ownerID, StateName: state, RegionName: args[0]}
		must(s.InsertStateRegion(sr))
		ok("state-region %s → %s/%s", sr.RegionName, in, state)

	default:
		die("unknown entity %q (use: app, screen, region, event, transition, tag, flow, type, enum, context, field, ambient, fixture, state-fixture, state-region)", entity)
	}
}

// --- set [H6 fix] ---

func runSet(s *store.Store, args []string) {
	if len(args) > 0 && store.IsRef(args[0]) {
		entityType, _, entityName, err := s.ResolveRef(args[0])
		if err != nil {
			die("%v", err)
		}
		args = append([]string{entityType, entityName}, args[1:]...)
	}
	if len(args) < 2 {
		die("usage: sft set <entity> <name> --<field> <value>")
	}
	entity, name := args[0], args[1]

	switch entity {
	case "screen":
		desc := flagVal(args, "--description")
		if desc == "" {
			die("usage: sft set screen <name> --description <new>")
		}
		must(s.UpdateScreen(name, desc))

	case "region":
		desc := flagVal(args, "--description")
		if desc == "" {
			die("usage: sft set region <name> --description <new> [--in <parent>]")
		}
		in := flagVal(args, "--in")
		must(s.UpdateRegion(name, desc, in))

	case "type":
		fields := flagVal(args, "--fields")
		if fields == "" {
			die("usage: sft set type <name> --fields <json>")
		}
		must(s.UpdateDataType(name, fields))

	case "enum":
		values := flagVal(args, "--values")
		if values == "" {
			die("usage: sft set enum <name> --values <json>")
		}
		must(s.UpdateEnum(name, values))

	case "context":
		newType := flagVal(args, "--type")
		if newType == "" {
			die("usage: sft set context <field> --type <t> [--on <screen>]")
		}
		on := flagVal(args, "--on")
		ownerType, ownerID := resolveContextOwner(s, on)
		must(s.UpdateContextField(name, ownerType, ownerID, newType))

	case "field":
		newType := flagVal(args, "--type")
		in := flagVal(args, "--in")
		if newType == "" || in == "" {
			die("usage: sft set field <name> --type <t> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		must(s.UpdateRegionData(name, regionID, newType))

	case "ambient":
		ref := flagVal(args, "--ref")
		in := flagVal(args, "--in")
		if ref == "" || in == "" {
			die("usage: sft set ambient <name> --ref <data_ref> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		source, query, err := loader.ParseDataRef(ref)
		if err != nil {
			die("%v", err)
		}
		must(s.UpdateAmbientRef(name, regionID, source, query))

	case "fixture":
		data := flagVal(args, "--data")
		if data == "" {
			die("usage: sft set fixture <name> --data <json> [--extends <f>]")
		}
		extends := flagVal(args, "--extends")
		must(s.UpdateFixture(name, data, extends))

	case "state-fixture":
		in := flagVal(args, "--in")
		state := flagVal(args, "--state")
		fixture := flagVal(args, "--fixture")
		if in == "" || state == "" || fixture == "" {
			die("usage: sft set state-fixture --in <owner> --state <s> --fixture <f>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		must(s.UpdateStateFixture(ownerType, ownerID, state, fixture))
		// name is the first positional arg but state-fixture uses flags only
		ok("updated state-fixture %s/%s → %s", in, state, fixture)
		return

	default:
		die("set supports: screen, region, type, enum, context, field, ambient, fixture, state-fixture")
	}
	ok("updated %s %s", entity, name)
}

// --- import ---

func runInit(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft init <file.yaml>")
	}
	if err := loader.Load(s, args[0]); err != nil {
		die("import: %v", err)
	}
	ok("imported %s", args[0])
}


// --- diff ---

func runDiff(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft diff <file.yaml>")
	}
	// Load current spec from DB
	currentSpec, err := show.Load(s.DB, s)
	if err != nil {
		die("%v", err)
	}
	// Load target spec from YAML into in-memory DB
	memStore, err := store.OpenMemory()
	if err != nil {
		die("memory store: %v", err)
	}
	defer memStore.Close()
	if err := loader.Load(memStore, args[0]); err != nil {
		die("import target: %v", err)
	}
	targetSpec, err := show.Load(memStore.DB, nil)
	if err != nil {
		die("load target: %v", err)
	}
	// Compare
	changes := diff.Compare(currentSpec, targetSpec)
	if format.JSONMode {
		format.JSON(changes)
		return
	}
	fmt.Print(diff.Format(changes))
}

// --- rename ---

func runRename(s *store.Store, args []string) {
	if len(args) < 3 {
		die("usage: sft rename <screen|region|flow|type|enum|fixture> <old> <new> [--in <parent>]")
	}
	entity, old, newName := args[0], args[1], args[2]
	in := flagVal(args, "--in")
	switch entity {
	case "screen":
		must(s.RenameScreen(old, newName))
	case "region":
		must(s.RenameRegion(old, newName, in))
	case "flow":
		must(s.RenameFlow(old, newName))
	case "type":
		must(s.RenameDataType(old, newName))
	case "enum":
		must(s.RenameEnum(old, newName))
	case "fixture":
		must(s.RenameFixture(old, newName))
	default:
		die("rename supports: screen, region, flow, type, enum, fixture")
	}
	ok("renamed %s %s → %s", entity, old, newName)
}

// --- reorder ---

func runReorder(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft reorder <parent> <child1> <child2> ...")
	}
	parent := args[0]
	children := args[1:]
	must(s.ReorderRegions(parent, children))
	ok("reordered %d regions in %s", len(children), parent)
}

// --- rm [H7 fix: extended to all entity types] ---

func runRm(s *store.Store, args []string) {
	if len(args) > 0 && store.IsRef(args[0]) {
		entityType, _, entityName, err := s.ResolveRef(args[0])
		if err != nil {
			die("%v", err)
		}
		args = append([]string{entityType, entityName}, args[1:]...)
	}
	if len(args) < 2 {
		die("usage: sft rm <screen|region|event|transition|tag|flow> <name> [--in/--on <parent>]")
	}
	entity, name := args[0], args[1]

	switch entity {
	case "screen":
		impacts := getImpacts(s, entity, name, "")
		showImpacts(entity, name, impacts, false)
		must(s.DeleteScreen(name))
		ok("deleted screen %s", name)

	case "region":
		in := flagVal(args, "--in")
		impacts := getImpacts(s, entity, name, in)
		showImpacts(entity, name, impacts, false)
		must(s.DeleteRegion(name, in))
		ok("deleted region %s", name)

	case "event":
		in := flagVal(args, "--in")
		if in == "" {
			die("usage: sft rm event <name> --in <region>")
		}
		must(s.DeleteEvent(name, in))
		ok("deleted event %s from %s", name, in)

	case "transition":
		in := flagVal(args, "--in")
		if in == "" {
			die("usage: sft rm transition <on-event> --in <owner> [--from <state>]")
		}
		from := flagVal(args, "--from")
		must(s.DeleteTransition(name, in, from))
		if from != "" {
			ok("deleted transition on %s from %s in %s", name, from, in)
		} else {
			ok("deleted transition on %s from %s", name, in)
		}

	case "tag":
		on := flagVal(args, "--on")
		if on == "" {
			die("usage: sft rm tag <tag> --on <entity>")
		}
		must(s.DeleteTag(name, on))
		ok("deleted tag [%s] from %s", name, on)

	case "flow":
		must(s.DeleteFlow(name))
		ok("deleted flow %s", name)

	case "type":
		must(s.DeleteDataType(name))
		ok("deleted type %s", name)

	case "enum":
		must(s.DeleteEnum(name))
		ok("deleted enum %s", name)

	case "context":
		on := flagVal(args, "--on")
		ownerType, ownerID := resolveContextOwner(s, on)
		must(s.DeleteContextField(name, ownerType, ownerID))
		if on != "" {
			ok("deleted context %s from %s", name, on)
		} else {
			ok("deleted context %s", name)
		}

	case "field":
		in := flagVal(args, "--in")
		if in == "" {
			die("usage: sft rm field <name> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		must(s.DeleteRegionData(name, regionID))
		ok("deleted field %s from %s", name, in)

	case "ambient":
		in := flagVal(args, "--in")
		if in == "" {
			die("usage: sft rm ambient <name> --in <region>")
		}
		regionID, err := s.ResolveRegion(in)
		if err != nil {
			die("%v", err)
		}
		must(s.DeleteAmbientRef(name, regionID))
		ok("deleted ambient %s from %s", name, in)

	case "fixture":
		must(s.DeleteFixture(name))
		ok("deleted fixture %s", name)

	case "state-fixture":
		in := flagVal(args, "--in")
		state := flagVal(args, "--state")
		if in == "" || state == "" {
			die("usage: sft rm state-fixture --in <owner> --state <state>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		must(s.DeleteStateFixture(ownerType, ownerID, state))
		ok("deleted state-fixture %s/%s", in, state)

	case "state-region":
		in := flagVal(args, "--in")
		state := flagVal(args, "--state")
		if in == "" || state == "" {
			die("usage: sft rm state-region <region> --in <owner> --state <state>")
		}
		ownerType, ownerID, err := s.ResolveOwner(in)
		if err != nil {
			die("%v", err)
		}
		must(s.DeleteStateRegion(name, ownerType, ownerID, state))
		ok("deleted state-region %s from %s/%s", name, in, state)

	default:
		die("rm supports: screen, region, event, transition, tag, flow, type, enum, context, field, ambient, fixture, state-fixture, state-region")
	}
}

// --- mv ---

func runMv(s *store.Store, args []string) {
	if len(args) < 2 || args[0] != "region" {
		die("usage: sft mv region <name> --to <parent> [--in <current-parent>]")
	}
	name := args[1]
	to := flagVal(args, "--to")
	in := flagVal(args, "--in")
	if to == "" {
		die("usage: sft mv region <name> --to <parent> [--in <current-parent>]")
	}
	impacts := getImpacts(s, "region", name, in)
	showImpacts("region", name, impacts, false)
	if err := s.MoveRegion(name, to, in); err != nil {
		die("move: %v", err)
	}
	format.OK(fmt.Sprintf("moved region %s → %s", name, to))
}

// --- impact ---

func runImpact(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft impact <screen|region> <name> [--in <parent>]")
	}
	entity, name := args[0], args[1]
	in := flagVal(args, "--in")
	impacts := getImpacts(s, entity, name, in)
	showImpacts(entity, name, impacts, true)
}

func getImpacts(s *store.Store, entity, name, in string) []store.Impact {
	var impacts []store.Impact
	var err error
	switch entity {
	case "screen":
		impacts, err = s.ImpactScreen(name)
	case "region":
		impacts, err = s.ImpactRegion(name, in)
	default:
		die("impact supports: screen, region")
	}
	if err != nil {
		die("impact: %v", err)
	}
	return impacts
}

func showImpacts(entity, name string, impacts []store.Impact, toStdout bool) {
	fi := make([]format.Impact, len(impacts))
	for i, imp := range impacts {
		fi[i] = format.Impact{Entity: imp.Entity, Type: imp.Type, Name: imp.Name, Detail: imp.Detail}
	}
	if toStdout {
		format.Impacts(entity, name, fi)
	} else {
		format.ImpactInfo(entity, name, fi)
	}
}

// --- component [H9 fix: --props-file support] ---

func runComponent(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft component <entity> [Type] [--props/--props-file/--on/--visible]")
	}
	entity := args[0]

	if len(args) == 1 {
		comp := s.GetComponentByName(entity)
		if comp == nil {
			die("no component set on %s", entity)
		}
		format.JSON(comp)
		return
	}

	if args[1] == "--rm" {
		must(s.RemoveComponent(entity))
		ok("removed component from %s", entity)
		return
	}

	// [F4] If args[1] starts with --, it's a flag not a component type
	componentType := args[1]
	if strings.HasPrefix(componentType, "--") {
		die("usage: sft component <entity> <Type> [--props/--props-file/--on/--visible]\nmissing component type — got flag %q instead", componentType)
	}
	props := flagVal(args, "--props")
	propsFile := flagVal(args, "--props-file")
	if propsFile != "" {
		data, err := os.ReadFile(propsFile)
		if err != nil {
			die("read props file: %v", err)
		}
		props = string(data)
	}
	if props == "" {
		props = "{}"
	}
	onActions := flagVal(args, "--on")
	visible := flagVal(args, "--visible")

	must(s.SetComponent(entity, componentType, props, onActions, visible))
	ok("%s → %s", entity, componentType)
}

// --- render ---

func runRender(s *store.Store) {
	spec, err := show.Load(s.DB, s)
	if err != nil {
		die("%v", err)
	}
	jr := render.FromSFT(spec)
	render.Hydrate(jr, func(name string) *render.CompDef {
		comp := s.GetComponentByName(name)
		if comp == nil {
			return nil
		}
		return &render.CompDef{
			Component: comp.Component,
			Props:     comp.Props,
			OnActions: comp.OnActions,
			Visible:   comp.Visible,
		}
	})
	format.JSON(jr)
}

// --- attach ---

func runAttach(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft attach <entity> <file> [--as <name>]")
	}
	entity, file := args[0], args[1]
	asName := flagVal(args, "--as")
	name, err := s.Attach(entity, file, asName)
	if err != nil {
		die("attach: %v", err)
	}
	ok("%s on %s", name, entity)
}

func runDetach(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft detach <entity> <name>")
	}
	if err := s.Detach(args[0], args[1]); err != nil {
		die("detach: %v", err)
	}
	ok("removed %s from %s", args[1], args[0])
}

func runList(s *store.Store, args []string) {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}
	attachments, err := s.ListAttachments(filter)
	if err != nil {
		die("list: %v", err)
	}
	if format.JSONMode {
		format.JSON(attachments)
		return
	}
	if len(attachments) == 0 {
		fmt.Println(format.C(format.Dim, "(no attachments)"))
		return
	}
	groups := make(map[string][]store.Attachment)
	order := []string{}
	for _, a := range attachments {
		if _, exists := groups[a.Entity]; !exists {
			order = append(order, a.Entity)
		}
		groups[a.Entity] = append(groups[a.Entity], a)
	}
	for _, entity := range order {
		label := entity
		if entity == store.GlobalEntity {
			label = "(global)"
		} else {
			if _, err := s.ResolveScreen(entity); err == nil {
				label = entity + " " + format.C(format.Dim, "(screen)")
			} else if _, err := s.ResolveRegion(entity); err == nil {
				label = entity + " " + format.C(format.Dim, "(region)")
			}
		}
		fmt.Println(format.C(format.Bold, label))
		for _, a := range groups[entity] {
			fmt.Printf("  %s\n", a.Name)
		}
	}
}

func runCat(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft cat <entity> <name>")
	}
	data, err := s.ReadAttachment(args[0], args[1])
	if err != nil {
		die("cat: %v", err)
	}
	os.Stdout.Write(data)
}

// --- view ---

func runView(s *store.Store, args []string) {
	// Check for empty spec before starting server
	if _, err := s.ResolveApp(); err != nil {
		die("no spec found in %s — import one first:\n  sft import spec.yaml\n  sft add app MyApp \"description\"", store.DefaultPath())
	}

	var port int
	var webDir string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			}
		case "--web-dir":
			if i+1 < len(args) {
				webDir = args[i+1]
				i++
			}
		}
	}

	var clientFS fs.FS = web.ClientFS
	if webDir != "" {
		clientFS = os.DirFS(webDir)
	}

	srv := view.NewServer(s, view.Options{Port: port, ClientFS: clientFS})
	if err := srv.Start(); err != nil {
		die("view: %v", err)
	}
}

// --- diagram ---

func runDiagram(s *store.Store, args []string) {
	if len(args) == 0 {
		die("usage: sft diagram <states <name> | nav | flow <name>>")
	}
	var out string
	var err error
	switch args[0] {
	case "states", "state":
		need(args, 2, "sft diagram states <name>")
		out, err = diagram.States(s.DB, args[1])
	case "nav":
		out, err = diagram.Nav(s.DB)
	case "flow":
		need(args, 2, "sft diagram flow <name>")
		out, err = diagram.Flow(s.DB, args[1])
	default:
		die("unknown diagram type %q (available: states, nav, flow)", args[0])
	}
	if err != nil {
		die("%v", err)
	}
	fmt.Print(out)
}

// --- helpers ---

func ok(msg string, args ...any) {
	format.OK(fmt.Sprintf(msg, args...))
}

func die(msg string, args ...any) {
	format.Err(fmt.Sprintf(msg, args...))
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		die("%v", err)
	}
}

func mustResolveApp(s *store.Store) int64 {
	id, err := s.ResolveApp()
	if err != nil {
		die("%v", err)
	}
	return id
}

func need(args []string, n int, usage string) {
	if len(args) < n {
		die("usage: %s", usage)
	}
}

func flagIndex(args []string, flag string) int {
	for i, a := range args {
		if a == flag {
			return i
		}
	}
	return -1
}

func flagVal(args []string, flag string) string {
	i := flagIndex(args, flag)
	if i == -1 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func resolveContextOwner(s *store.Store, on string) (string, int64) {
	if on != "" {
		ownerType, ownerID, err := s.ResolveOwner(on)
		if err != nil {
			die("%v", err)
		}
		return ownerType, ownerID
	}
	return "app", mustResolveApp(s)
}
