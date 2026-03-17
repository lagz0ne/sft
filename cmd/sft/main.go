package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lagz0ne/sft/internal/format"
	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/query"
	"github.com/lagz0ne/sft/internal/render"
	"github.com/lagz0ne/sft/internal/show"
	"github.com/lagz0ne/sft/internal/store"
	"github.com/lagz0ne/sft/internal/validator"
)

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
		die(usage)
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
	case "add":
		runAdd(s, rest)
	case "set":
		runSet(s, rest)
	case "rm":
		runRm(s, rest)
	case "mv":
		runMv(s, rest)
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
	default:
		die("unknown command %q\n%s", cmd, usage)
	}
}

const usage = `usage: sft <command> [args] [--json]

commands:
  show                                render full spec (human/LLM readable)
  query    <name|SELECT>              screens, events, states, flows, tags, regions
  validate                            run validation rules
  add      <type> ...                 app, screen, region, event, transition, tag, flow
  set      <screen|region> <name> --description <d>   update entity
  rm       <type> <name> [--in <p>]   remove entity (screen, region, event, transition, tag, flow)
  mv       region <name> --to <dst>   move a region
  impact   <screen|region> <name>     show dependents
  component <entity> [Type] [--props/--props-file/--on/--visible]
  render                              generate json-render spec
  attach   <entity> <file> [--as n]   attach a file to an entity
  detach   <entity> <name>            remove an attachment
  list     [entity]                   show all attachments
  cat      <entity> <name>            read an attachment

aliases: q=query, check=validate, ls=list, comp=component`

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
		die("usage: sft query <screens|events|states|flows|tags|regions|SELECT ...>")
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
	} else {
		results, err = query.Run(s.DB, name)
		if name != "screens" && name != "regions" && name != "events" &&
			name != "flows" && name != "tags" {
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
		e := &model.Event{RegionID: regionID, Name: args[0]}
		must(s.InsertEvent(e))
		ok("event %s in %s", e.Name, regionName)

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

	default:
		die("unknown entity %q (use: app, screen, region, event, transition, tag, flow)", entity)
	}
}

// --- set [H6 fix] ---

func runSet(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft set <screen|region> <name> --description <new>")
	}
	entity, name := args[0], args[1]
	desc := flagVal(args, "--description")
	if desc == "" {
		die("usage: sft set <screen|region> <name> --description <new>")
	}
	switch entity {
	case "screen":
		must(s.UpdateScreen(name, desc))
	case "region":
		must(s.UpdateRegion(name, desc))
	default:
		die("set supports: screen, region")
	}
	ok("updated %s %s", entity, name)
}

// --- rm [H7 fix: extended to all entity types] ---

func runRm(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft rm <screen|region|event|transition|tag|flow> <name> [--in/--on <parent>]")
	}
	entity, name := args[0], args[1]

	switch entity {
	case "screen":
		impacts := getImpacts(s, entity, name)
		showImpacts(entity, name, impacts, false)
		must(s.DeleteScreen(name))
		ok("deleted screen %s", name)

	case "region":
		impacts := getImpacts(s, entity, name)
		showImpacts(entity, name, impacts, false)
		must(s.DeleteRegion(name))
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
			die("usage: sft rm transition <on-event> --in <owner>")
		}
		must(s.DeleteTransition(name, in))
		ok("deleted transition on %s from %s", name, in)

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

	default:
		die("rm supports: screen, region, event, transition, tag, flow")
	}
}

// --- mv ---

func runMv(s *store.Store, args []string) {
	if len(args) < 2 || args[0] != "region" {
		die("usage: sft mv region <name> --to <parent>")
	}
	name := args[1]
	to := flagVal(args, "--to")
	if to == "" {
		die("usage: sft mv region <name> --to <parent>")
	}
	impacts := getImpacts(s, "region", name)
	showImpacts("region", name, impacts, false)
	if err := s.MoveRegion(name, to); err != nil {
		die("move: %v", err)
	}
	format.OK(fmt.Sprintf("moved region %s → %s", name, to))
}

// --- impact ---

func runImpact(s *store.Store, args []string) {
	if len(args) < 2 {
		die("usage: sft impact <screen|region> <name>")
	}
	entity, name := args[0], args[1]
	impacts := getImpacts(s, entity, name)
	showImpacts(entity, name, impacts, true)
}

func getImpacts(s *store.Store, entity, name string) []store.Impact {
	var impacts []store.Impact
	var err error
	switch entity {
	case "screen":
		impacts, err = s.ImpactScreen(name)
	case "region":
		impacts, err = s.ImpactRegion(name)
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
