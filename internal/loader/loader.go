package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/lagz0ne/sft/internal/model"
	"github.com/lagz0ne/sft/internal/show"
	"github.com/lagz0ne/sft/internal/store"
	"gopkg.in/yaml.v3"
)

// YAML types matching the SFT spec format.

type yamlFile struct {
	App yaml.Node `yaml:"app"`
}

type yamlApp struct {
	Name        string                       `yaml:"name"`
	Description string                       `yaml:"description"`
	Data        map[string]map[string]string `yaml:"data,omitempty"`
	Context     map[string]string            `yaml:"context,omitempty"`
	Regions     []yamlRegion                 `yaml:"regions,omitempty"`
	Screens     []yamlScreen                 `yaml:"screens,omitempty"`
	Flows       []yamlFlow                   `yaml:"flows,omitempty"`
}

type yamlScreen struct {
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Tags         []string         `yaml:"tags,omitempty"`
	Context      map[string]string `yaml:"context,omitempty"`
	Component    string           `yaml:"component,omitempty"`
	Props        string           `yaml:"props,omitempty"`
	OnActions    string           `yaml:"on_actions,omitempty"`
	Visible      string           `yaml:"visible,omitempty"`
	Regions      []yamlRegion     `yaml:"regions,omitempty"`
	States       []yamlTransition `yaml:"states,omitempty"`
	StateMachine yaml.Node        `yaml:"state_machine,omitempty"`
}

type yamlRegion struct {
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Tags         []string         `yaml:"tags,omitempty"`
	Component    string           `yaml:"component,omitempty"`
	Props        string           `yaml:"props,omitempty"`
	OnActions    string           `yaml:"on_actions,omitempty"`
	Visible      string           `yaml:"visible,omitempty"`
	Events       []string         `yaml:"events,omitempty"`
	Ambient      map[string]string `yaml:"ambient,omitempty"`
	Data         map[string]string `yaml:"data,omitempty"`
	Regions      []yamlRegion     `yaml:"regions,omitempty"`
	States       []yamlTransition `yaml:"states,omitempty"`
	StateMachine yaml.Node        `yaml:"state_machine,omitempty"`
}

type yamlTransition struct {
	On     string `yaml:"on"`
	From   string `yaml:"from,omitempty"`
	To     string `yaml:"to,omitempty"`
	Action string `yaml:"action,omitempty"`
}

type yamlFlow struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	On          string `yaml:"on,omitempty"`
	Sequence    string `yaml:"sequence"`
}

// Load parses an SFT YAML file and populates the store.
func Load(s *store.Store, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var f yamlFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// Handle app as mapping (single app) or sequence (list of apps).
	var app yamlApp
	switch f.App.Kind {
	case yaml.MappingNode:
		if err := f.App.Decode(&app); err != nil {
			return fmt.Errorf("decode app in %s: %w", path, err)
		}
	case yaml.SequenceNode:
		var apps []yamlApp
		if err := f.App.Decode(&apps); err != nil {
			return fmt.Errorf("decode app list in %s: %w", path, err)
		}
		if len(apps) == 0 {
			return fmt.Errorf("empty app list in %s", path)
		}
		app = apps[0]
		if len(apps) > 1 {
			fmt.Fprintf(os.Stderr, "warning: %s contains %d apps, importing first (%s) only\n", path, len(apps), app.Name)
		}
	default:
		return fmt.Errorf("app: expected mapping or sequence in %s", path)
	}

	if app.Name == "" {
		return fmt.Errorf("missing app.name in %s", path)
	}

	// App
	a := &model.App{Name: app.Name, Description: app.Description}
	if err := s.InsertApp(a); err != nil {
		return fmt.Errorf("app: %w", err)
	}

	// Data types
	for typeName, fields := range app.Data {
		fieldsJSON, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("data type %s: %w", typeName, err)
		}
		if err := s.InsertDataType(&model.DataType{AppID: a.ID, Name: typeName, Fields: string(fieldsJSON)}); err != nil {
			return fmt.Errorf("data type %s: %w", typeName, err)
		}
	}

	// App context
	for fieldName, fieldType := range app.Context {
		if err := s.InsertContextField(&model.ContextField{OwnerType: "app", OwnerID: a.ID, FieldName: fieldName, FieldType: fieldType}); err != nil {
			return fmt.Errorf("app context %s: %w", fieldName, err)
		}
	}

	// App-level regions
	for _, r := range app.Regions {
		if err := insertRegion(s, a.ID, "app", a.ID, r); err != nil {
			return err
		}
	}

	// Screens
	for _, sc := range app.Screens {
		screen := &model.Screen{AppID: a.ID, Name: sc.Name, Description: sc.Description}
		if err := s.InsertScreen(screen); err != nil {
			return fmt.Errorf("screen %s: %w", sc.Name, err)
		}
		for _, tag := range sc.Tags {
			if err := s.InsertTag(&model.Tag{EntityType: "screen", EntityID: screen.ID, Tag: tag}); err != nil {
				return fmt.Errorf("tag [%s] on screen %s: %w", tag, sc.Name, err)
			}
		}
		// Screen context
		for fieldName, fieldType := range sc.Context {
			if err := s.InsertContextField(&model.ContextField{OwnerType: "screen", OwnerID: screen.ID, FieldName: fieldName, FieldType: fieldType}); err != nil {
				return fmt.Errorf("screen context %s on %s: %w", fieldName, sc.Name, err)
			}
		}
		if sc.Component != "" {
			if err := s.SetComponent(sc.Name, sc.Component, sc.Props, sc.OnActions, sc.Visible); err != nil {
				return fmt.Errorf("component on screen %s: %w", sc.Name, err)
			}
		}
		for _, r := range sc.Regions {
			if err := insertRegion(s, a.ID, "screen", screen.ID, r); err != nil {
				return err
			}
		}
		if err := insertTransitions(s, "screen", screen.ID, sc.Name, sc.States, &sc.StateMachine); err != nil {
			return err
		}
	}

	// Flows
	for _, fl := range app.Flows {
		if err := s.InsertFlow(&model.Flow{
			AppID: a.ID, Name: fl.Name, Description: fl.Description,
			OnEvent: fl.On, Sequence: fl.Sequence,
		}); err != nil {
			return fmt.Errorf("flow %s: %w", fl.Name, err)
		}
	}

	return nil
}

func insertRegion(s *store.Store, appID int64, parentType string, parentID int64, r yamlRegion) error {
	region := &model.Region{
		AppID: appID, ParentType: parentType, ParentID: parentID,
		Name: r.Name, Description: r.Description,
	}
	if err := s.InsertRegion(region); err != nil {
		return fmt.Errorf("region %s: %w", r.Name, err)
	}
	if r.Component != "" {
		if err := s.SetComponent(r.Name, r.Component, r.Props, r.OnActions, r.Visible); err != nil {
			return fmt.Errorf("component on region %s: %w", r.Name, err)
		}
	}
	for _, ev := range r.Events {
		if err := s.InsertEvent(&model.Event{RegionID: region.ID, Name: ev}); err != nil {
			return fmt.Errorf("event %s in %s: %w", ev, r.Name, err)
		}
	}
	for _, tag := range r.Tags {
		if err := s.InsertTag(&model.Tag{EntityType: "region", EntityID: region.ID, Tag: tag}); err != nil {
			return fmt.Errorf("tag [%s] on %s: %w", tag, r.Name, err)
		}
	}
	// Ambient refs
	for localName, ref := range r.Ambient {
		source, query, err := parseDataRef(ref)
		if err != nil {
			return fmt.Errorf("ambient %s in %s: %w", localName, r.Name, err)
		}
		if err := s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: localName, Source: source, Query: query}); err != nil {
			return fmt.Errorf("ambient %s in %s: %w", localName, r.Name, err)
		}
	}
	// Region data
	for fieldName, fieldType := range r.Data {
		if err := s.InsertRegionData(&model.RegionData{RegionID: region.ID, FieldName: fieldName, FieldType: fieldType}); err != nil {
			return fmt.Errorf("region data %s in %s: %w", fieldName, r.Name, err)
		}
	}
	for _, child := range r.Regions {
		if err := insertRegion(s, appID, "region", region.ID, child); err != nil {
			return err
		}
	}
	if err := insertTransitions(s, "region", region.ID, r.Name, r.States, &r.StateMachine); err != nil {
		return err
	}
	return nil
}

// insertTransitions handles dual-format dispatch for state transitions.
// If both states and stateMachine are provided, it returns an error.
// If stateMachine is provided, it parses via ParseStateMachine and sets owner fields.
// If states is provided, it uses the legacy yamlTransition list.
// If neither is provided, no transitions are inserted (valid).
func insertTransitions(s *store.Store, ownerType string, ownerID int64, ownerName string, states []yamlTransition, stateMachine *yaml.Node) error {
	hasStateMachine := stateMachine != nil && stateMachine.Kind != 0
	if len(states) > 0 && hasStateMachine {
		return fmt.Errorf("%s %s: cannot specify both states and state_machine", ownerType, ownerName)
	}

	if hasStateMachine {
		transitions, _, err := ParseStateMachine(*stateMachine)
		if err != nil {
			return fmt.Errorf("state_machine in %s %s: %w", ownerType, ownerName, err)
		}
		for _, t := range transitions {
			t.OwnerType = ownerType
			t.OwnerID = ownerID
			if err := s.InsertTransition(&t); err != nil {
				return fmt.Errorf("transition on %s in %s %s: %w", t.OnEvent, ownerType, ownerName, err)
			}
		}
		return nil
	}

	for _, t := range states {
		if err := s.InsertTransition(&model.Transition{
			OwnerType: ownerType, OwnerID: ownerID,
			OnEvent: t.On, FromState: t.From, ToState: t.To, Action: t.Action,
		}); err != nil {
			return fmt.Errorf("transition on %s in %s %s: %w", t.On, ownerType, ownerName, err)
		}
	}
	return nil
}

// parseDataRef parses "data(source, query)" into source and query parts.
func parseDataRef(ref string) (source, query string, err error) {
	if !strings.HasPrefix(ref, "data(") || !strings.HasSuffix(ref, ")") {
		return "", "", fmt.Errorf("invalid data reference %q: must be data(source, query)", ref)
	}
	inner := ref[5 : len(ref)-1] // strip "data(" and ")"
	// Split on first ", "
	idx := strings.Index(inner, ", ")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid data reference %q: missing separator", ref)
	}
	return inner[:idx], inner[idx+2:], nil
}

// Export serializes a Spec tree to SFT YAML format.
func Export(spec *show.Spec, w io.Writer) error {
	app := yamlApp{
		Name:        spec.App.Name,
		Description: spec.App.Description,
		Data:        spec.App.DataTypes,
		Context:     spec.App.Context,
		Regions:     exportRegions(spec.App.Regions),
		Screens:     exportScreens(spec.Screens),
		Flows:       exportFlows(spec.Flows),
	}
	out := struct {
		App yamlApp `yaml:"app"`
	}{App: app}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return err
	}
	return enc.Close()
}

func exportScreens(screens []show.Screen) []yamlScreen {
	if len(screens) == 0 {
		return nil
	}
	var out []yamlScreen
	for _, s := range screens {
		ys := yamlScreen{
			Name:        s.Name,
			Description: s.Description,
			Tags:        s.Tags,
			Context:     s.Context,
			Component:   s.Component,
			Props:       s.ComponentProps,
			OnActions:   s.ComponentOn,
			Visible:     s.ComponentVis,
			Regions:     exportRegions(s.Regions),
		}
		if sm := exportStateMachine(s.Transitions); sm != nil {
			ys.StateMachine = *sm
		}
		out = append(out, ys)
	}
	return out
}

func exportRegions(regions []show.Region) []yamlRegion {
	if len(regions) == 0 {
		return nil
	}
	var out []yamlRegion
	for _, r := range regions {
		yr := yamlRegion{
			Name:        r.Name,
			Description: r.Description,
			Tags:        r.Tags,
			Component:   r.Component,
			Props:       r.ComponentProps,
			OnActions:   r.ComponentOn,
			Visible:     r.ComponentVis,
			Events:      r.Events,
			Ambient:     r.Ambient,
			Data:        r.RegionData,
			Regions:     exportRegions(r.Regions),
		}
		if sm := exportStateMachine(r.Transitions); sm != nil {
			yr.StateMachine = *sm
		}
		out = append(out, yr)
	}
	return out
}

// exportStateMachine converts a flat list of transitions into a state_machine yaml.Node.
// Groups transitions by FromState, producing ordered mappings. Terminal states (appear
// only as targets) are included as empty mappings.
func exportStateMachine(transitions []show.Transition) *yaml.Node {
	if len(transitions) == 0 {
		return nil
	}

	// Collect from-states in order, and group events per state.
	type eventEntry struct {
		event  string
		to     string
		action string
	}
	stateOrder := []string{}
	stateEvents := map[string][]eventEntry{}
	seenFrom := map[string]bool{}
	allTo := map[string]bool{}

	for _, t := range transitions {
		from := t.FromState
		if from == "" {
			continue
		}
		if !seenFrom[from] {
			seenFrom[from] = true
			stateOrder = append(stateOrder, from)
		}
		stateEvents[from] = append(stateEvents[from], eventEntry{
			event:  t.OnEvent,
			to:     t.ToState,
			action: t.Action,
		})
		if t.ToState != "" {
			allTo[t.ToState] = true
		}
	}

	// Find terminal states: appear as to but never as from.
	var terminalStates []string
	for s := range allTo {
		if !seenFrom[s] {
			terminalStates = append(terminalStates, s)
		}
	}
	// Sort terminal states for deterministic output.
	slices.Sort(terminalStates)

	// Build the top-level state_machine mapping node.
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	for _, stateName := range stateOrder {
		events := stateEvents[stateName]

		// State key.
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: stateName, Tag: "!!str"}

		// Build on: mapping for this state.
		onMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

		for _, e := range events {
			evKey := &yaml.Node{Kind: yaml.ScalarNode, Value: e.event, Tag: "!!str"}
			evVal := buildTransitionValueNode(stateName, e.to, e.action)
			onMapping.Content = append(onMapping.Content, evKey, evVal)
		}

		// State value: mapping with "on" key.
		stateVal := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		onKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "on", Tag: "!!str"}
		stateVal.Content = append(stateVal.Content, onKey, onMapping)

		root.Content = append(root.Content, keyNode, stateVal)
	}

	// Add terminal states as empty mappings.
	for _, s := range terminalStates {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"}
		valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, keyNode, valNode)
	}

	return root
}

// buildTransitionValueNode creates the yaml.Node for a single event's target value.
// Forms:
//   - to-state only → scalar: "selecting"
//   - stay (to == from) → scalar: "."
//   - action only (no to) → scalar if simple action, else flow object
//   - to + action → flow object: {to: x, action: y}
//   - guard in action → flow object: {guard: desc, to: x} or {guard: desc, to: x, action: y}
func buildTransitionValueNode(from, to, action string) *yaml.Node {
	guard, pureAction := parseGuardFromAction(action)

	hasTo := to != ""
	hasGuard := guard != ""
	hasAction := pureAction != ""

	// Determine the display to-value ("." for stay).
	displayTo := to
	if hasTo && to == from {
		displayTo = "."
	}

	// Simple case: to-state only, no action, no guard.
	if hasTo && !hasGuard && !hasAction {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: displayTo, Tag: "!!str"}
	}

	// Action only, no to-state, no guard → scalar shorthand.
	if !hasTo && !hasGuard && hasAction {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: pureAction, Tag: "!!str"}
	}

	// Everything else → flow-style mapping.
	obj := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}

	if hasGuard {
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "guard", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: guard, Tag: "!!str"},
		)
	}
	if hasTo {
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "to", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: displayTo, Tag: "!!str"},
		)
	}
	if hasAction {
		obj.Content = append(obj.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "action", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: pureAction, Tag: "!!str"},
		)
	}

	return obj
}

// parseGuardFromAction extracts guard and action from a combined action string.
// Format: "guard(description)" or "guard(description), action_name".
func parseGuardFromAction(action string) (guard, pureAction string) {
	if action == "" {
		return "", ""
	}
	if !strings.HasPrefix(action, "guard(") {
		return "", action
	}
	// Find closing paren for guard().
	idx := strings.Index(action, ")")
	if idx < 0 {
		return "", action
	}
	guard = action[len("guard("):idx]
	rest := strings.TrimSpace(action[idx+1:])
	rest = strings.TrimPrefix(rest, ",")
	rest = strings.TrimSpace(rest)
	return guard, rest
}

func exportFlows(flows []show.Flow) []yamlFlow {
	if len(flows) == 0 {
		return nil
	}
	var out []yamlFlow
	for _, f := range flows {
		out = append(out, yamlFlow{
			Name:        f.Name,
			Description: f.Description,
			On:          f.OnEvent,
			Sequence:    f.Sequence,
		})
	}
	return out
}
