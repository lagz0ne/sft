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
	Name           string                       `yaml:"name"`
	Description    string                       `yaml:"description"`
	Data           map[string]map[string]string `yaml:"data,omitempty"`
	Enums          map[string][]string          `yaml:"enums,omitempty"`
	Context        map[string]string            `yaml:"context,omitempty"`
	StateTemplates yaml.Node                    `yaml:"state_templates,omitempty"`
	Regions        []yamlRegion                 `yaml:"regions,omitempty"`
	Screens        []yamlScreen                 `yaml:"screens,omitempty"`
	Flows          []yamlFlow                   `yaml:"flows,omitempty"`
	Fixtures       yaml.Node                    `yaml:"fixtures,omitempty"`
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
	Events       yaml.Node        `yaml:"events,omitempty"`
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

	// Enums
	for enumName, values := range app.Enums {
		valuesJSON, err := json.Marshal(values)
		if err != nil {
			return fmt.Errorf("enum %s: %w", enumName, err)
		}
		if err := s.InsertEnum(&model.Enum{AppID: a.ID, Name: enumName, Values: string(valuesJSON)}); err != nil {
			return fmt.Errorf("enum %s: %w", enumName, err)
		}
	}

	// App context
	for fieldName, fieldType := range app.Context {
		if err := validateTypeSuffix(fieldType); err != nil {
			return fmt.Errorf("app context %s: %w", fieldName, err)
		}
		if err := s.InsertContextField(&model.ContextField{OwnerType: "app", OwnerID: a.ID, FieldName: fieldName, FieldType: fieldType}); err != nil {
			return fmt.Errorf("app context %s: %w", fieldName, err)
		}
	}

	// State templates
	if app.StateTemplates.Kind == yaml.MappingNode {
		if err := loadStateTemplates(s, a.ID, &app.StateTemplates); err != nil {
			return err
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
			if err := validateTypeSuffix(fieldType); err != nil {
				return fmt.Errorf("screen context %s on %s: %w", fieldName, sc.Name, err)
			}
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
		if err := insertTransitions(s, a.ID, "screen", screen.ID, sc.Name, sc.States, &sc.StateMachine); err != nil {
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

	// Fixtures
	if app.Fixtures.Kind == yaml.MappingNode {
		if err := loadFixtures(s, a.ID, &app.Fixtures); err != nil {
			return err
		}
	}

	return nil
}

func loadFixtures(s *store.Store, appID int64, node *yaml.Node) error {
	for i := 0; i < len(node.Content)-1; i += 2 {
		name := node.Content[i].Value
		def := node.Content[i+1]

		extends := ""
		dataNode := &yaml.Node{Kind: yaml.MappingNode}

		if def.Kind == yaml.MappingNode {
			for j := 0; j < len(def.Content)-1; j += 2 {
				key := def.Content[j].Value
				if key == "extends" {
					extends = def.Content[j+1].Value
				} else {
					dataNode.Content = append(dataNode.Content, def.Content[j], def.Content[j+1])
				}
			}
		}

		var dataMap any
		if err := dataNode.Decode(&dataMap); err != nil {
			return fmt.Errorf("fixture %s: decode data: %w", name, err)
		}
		dataJSON, err := json.Marshal(dataMap)
		if err != nil {
			return fmt.Errorf("fixture %s: marshal data: %w", name, err)
		}

		if err := s.InsertFixture(&model.Fixture{
			AppID: appID, Name: name, Extends: extends, Data: string(dataJSON),
		}); err != nil {
			return fmt.Errorf("fixture %s: %w", name, err)
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
	events, err := parseEventsNode(&r.Events)
	if err != nil {
		return fmt.Errorf("events in %s: %w", r.Name, err)
	}
	for _, ev := range events {
		ev.RegionID = region.ID
		if err := s.InsertEvent(&ev); err != nil {
			return fmt.Errorf("event %s in %s: %w", ev.Name, r.Name, err)
		}
	}
	for _, tag := range r.Tags {
		if err := s.InsertTag(&model.Tag{EntityType: "region", EntityID: region.ID, Tag: tag}); err != nil {
			return fmt.Errorf("tag [%s] on %s: %w", tag, r.Name, err)
		}
	}
	// Ambient refs
	for localName, ref := range r.Ambient {
		source, query, err := ParseDataRef(ref)
		if err != nil {
			return fmt.Errorf("ambient %s in %s: %w", localName, r.Name, err)
		}
		if err := s.InsertAmbientRef(&model.AmbientRef{RegionID: region.ID, LocalName: localName, Source: source, Query: query}); err != nil {
			return fmt.Errorf("ambient %s in %s: %w", localName, r.Name, err)
		}
	}
	// Region data
	for fieldName, fieldType := range r.Data {
		if err := validateTypeSuffix(fieldType); err != nil {
			return fmt.Errorf("region data %s in %s: %w", fieldName, r.Name, err)
		}
		if err := s.InsertRegionData(&model.RegionData{RegionID: region.ID, FieldName: fieldName, FieldType: fieldType}); err != nil {
			return fmt.Errorf("region data %s in %s: %w", fieldName, r.Name, err)
		}
	}
	for _, child := range r.Regions {
		if err := insertRegion(s, appID, "region", region.ID, child); err != nil {
			return err
		}
	}
	if err := insertTransitions(s, appID, "region", region.ID, r.Name, r.States, &r.StateMachine); err != nil {
		return err
	}
	return nil
}

// insertTransitions handles dual-format dispatch for state transitions.
// If both states and stateMachine are provided, it returns an error.
// If stateMachine is provided, it parses via ParseStateMachine and sets owner fields.
// If states is provided, it uses the legacy yamlTransition list.
// If neither is provided, no transitions are inserted (valid).
func insertTransitions(s *store.Store, appID int64, ownerType string, ownerID int64, ownerName string, states []yamlTransition, stateMachine *yaml.Node) error {
	hasStateMachine := stateMachine != nil && stateMachine.Kind != 0
	if len(states) > 0 && hasStateMachine {
		return fmt.Errorf("%s %s: cannot specify both states and state_machine", ownerType, ownerName)
	}

	if hasStateMachine {
		// Check for extends: key before parsing
		smNode := stateMachine
		if extendsName := findKeyValue(stateMachine, "extends"); extendsName != "" {
			merged, err := mergeWithTemplate(s, appID, extendsName, stateMachine)
			if err != nil {
				return fmt.Errorf("state_machine extends in %s %s: %w", ownerType, ownerName, err)
			}
			smNode = merged
		}

		transitions, _, stateFixtures, stateRegions, err := ParseStateMachine(*smNode)
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
		// Insert state → fixture bindings
		for stateName, fixtureName := range stateFixtures {
			if err := s.InsertStateFixture(&model.StateFixture{
				OwnerType: ownerType, OwnerID: ownerID,
				StateName: stateName, FixtureName: fixtureName,
			}); err != nil {
				return fmt.Errorf("state fixture %s→%s in %s %s: %w", stateName, fixtureName, ownerType, ownerName, err)
			}
		}
		// Insert state → region visibility bindings
		for stateName, regionNames := range stateRegions {
			for _, regionName := range regionNames {
				if err := s.InsertStateRegion(&model.StateRegion{
					OwnerType: ownerType, OwnerID: ownerID,
					StateName: stateName, RegionName: regionName,
				}); err != nil {
					return fmt.Errorf("state region %s→%s in %s %s: %w", stateName, regionName, ownerType, ownerName, err)
				}
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

// loadStateTemplates parses state_templates from YAML and stores them as JSON.
func loadStateTemplates(s *store.Store, appID int64, node *yaml.Node) error {
	for i := 0; i < len(node.Content)-1; i += 2 {
		name := node.Content[i].Value
		def := node.Content[i+1]

		// Serialize the template definition as JSON via yaml → any → json
		var defMap any
		if err := def.Decode(&defMap); err != nil {
			return fmt.Errorf("state template %s: %w", name, err)
		}
		defJSON, err := json.Marshal(defMap)
		if err != nil {
			return fmt.Errorf("state template %s: %w", name, err)
		}

		if err := s.InsertStateTemplate(&model.StateTemplate{
			AppID: appID, Name: name, Definition: string(defJSON),
		}); err != nil {
			return fmt.Errorf("state template %s: %w", name, err)
		}
	}
	return nil
}

// findKeyValue returns the scalar value for a key in a MappingNode, or "".
func findKeyValue(node *yaml.Node, key string) string {
	if node == nil {
		return ""
	}
	if v := findKey(node, key); v != nil {
		return v.Value
	}
	return ""
}

// mergeWithTemplate loads a template from the DB, parses it into a yaml.Node,
// then merges the screen/region overrides on top of it.
// Returns a new yaml.Node with the merged state machine (no extends: key).
func mergeWithTemplate(s *store.Store, appID int64, templateName string, overrides *yaml.Node) (*yaml.Node, error) {
	defJSON, err := s.GetStateTemplate(appID, templateName)
	if err != nil {
		return nil, err
	}

	// Convert JSON → any → YAML bytes → yaml.Node
	var defMap any
	if err := json.Unmarshal([]byte(defJSON), &defMap); err != nil {
		return nil, fmt.Errorf("unmarshal template %q: %w", templateName, err)
	}
	yamlBytes, err := yaml.Marshal(defMap)
	if err != nil {
		return nil, fmt.Errorf("marshal template %q to yaml: %w", templateName, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse template %q: %w", templateName, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("template %q produced empty document", templateName)
	}
	base := doc.Content[0]
	if base.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("template %q is not a mapping", templateName)
	}

	// Build a map of state name → index in base for merging
	baseStates := map[string]int{}
	for i := 0; i < len(base.Content)-1; i += 2 {
		baseStates[base.Content[i].Value] = i
	}

	// Apply overrides: iterate through the override node, skip "extends:"
	for i := 0; i < len(overrides.Content)-1; i += 2 {
		key := overrides.Content[i].Value
		if key == "extends" {
			continue
		}
		if idx, ok := baseStates[key]; ok {
			// Override: replace the state definition in base.
			// Merge the on: blocks — override events replace base events, base events not overridden are kept.
			baseVal := base.Content[idx+1]
			overrideVal := overrides.Content[i+1]
			base.Content[idx+1] = mergeStateNodes(baseVal, overrideVal)
		} else {
			// New state: append to base
			base.Content = append(base.Content, overrides.Content[i], overrides.Content[i+1])
		}
	}

	return base, nil
}

// mergeStateNodes merges an override state definition into a base state definition.
// The override's on: events replace matching base events; base-only events are kept.
func mergeStateNodes(base, override *yaml.Node) *yaml.Node {
	// If override is scalar or empty, it fully replaces
	if override.Kind == yaml.ScalarNode || (override.Kind == yaml.MappingNode && len(override.Content) == 0) {
		return override
	}
	// If base is scalar/null, override wins entirely
	if base.Kind == yaml.ScalarNode || (base.Kind == yaml.MappingNode && len(base.Content) == 0) {
		return override
	}

	// Both are mappings — merge on: blocks
	baseOn := findKey(base, "on")
	overrideOn := findKey(override, "on")

	if baseOn == nil || overrideOn == nil {
		// No on: in one of them — just use override
		return override
	}

	// Merge on: mappings — override events replace base events
	mergedOn := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	baseEvents := map[string]int{}
	for i := 0; i < len(baseOn.Content)-1; i += 2 {
		baseEvents[baseOn.Content[i].Value] = i
		mergedOn.Content = append(mergedOn.Content, baseOn.Content[i], baseOn.Content[i+1])
	}
	for i := 0; i < len(overrideOn.Content)-1; i += 2 {
		eventName := overrideOn.Content[i].Value
		if _, ok := baseEvents[eventName]; ok {
			// Replace in mergedOn — find the index
			for j := 0; j < len(mergedOn.Content)-1; j += 2 {
				if mergedOn.Content[j].Value == eventName {
					mergedOn.Content[j+1] = overrideOn.Content[i+1]
					break
				}
			}
		} else {
			// New event
			mergedOn.Content = append(mergedOn.Content, overrideOn.Content[i], overrideOn.Content[i+1])
		}
	}

	// Build result: copy all non-"on" keys from override, then add merged on:
	result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	// Copy non-on keys from base first (like fixture:)
	for i := 0; i < len(base.Content)-1; i += 2 {
		if base.Content[i].Value != "on" {
			// Only add if override doesn't have this key
			if findKey(override, base.Content[i].Value) == nil {
				result.Content = append(result.Content, base.Content[i], base.Content[i+1])
			}
		}
	}
	// Copy non-on keys from override
	for i := 0; i < len(override.Content)-1; i += 2 {
		if override.Content[i].Value != "on" {
			result.Content = append(result.Content, override.Content[i], override.Content[i+1])
		}
	}
	// Add merged on:
	result.Content = append(result.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "on", Tag: "!!str"},
		mergedOn,
	)

	return result
}

// ParseDataRef parses "data(source, query)" into source and query parts.
func ParseDataRef(ref string) (source, query string, err error) {
	inner, ok := strings.CutPrefix(ref, "data(")
	if !ok || !strings.HasSuffix(inner, ")") {
		return "", "", fmt.Errorf("invalid data reference %q: must be data(source, query)", ref)
	}
	inner = strings.TrimSuffix(inner, ")")
	source, query, found := strings.Cut(inner, ", ")
	if !found {
		return "", "", fmt.Errorf("invalid data reference %q: missing separator", ref)
	}
	return source, query, nil
}

// parseEventsNode parses a yaml.Node for events into model.Event slices.
// Supports SequenceNode (list of strings) and MappingNode (keys are event names).
// Event strings may contain annotations: "name(type)" -> name="name", annotation="type".
func parseEventsNode(node *yaml.Node) ([]model.Event, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}
	switch node.Kind {
	case yaml.SequenceNode:
		var events []model.Event
		for _, item := range node.Content {
			name, annotation := ParseEventName(item.Value)
			events = append(events, model.Event{Name: name, Annotation: annotation})
		}
		return events, nil
	case yaml.MappingNode:
		var events []model.Event
		for i := 0; i < len(node.Content)-1; i += 2 {
			name, annotation := ParseEventName(node.Content[i].Value)
			events = append(events, model.Event{Name: name, Annotation: annotation})
		}
		return events, nil
	default:
		return nil, fmt.Errorf("events: expected sequence or mapping, got kind %d", node.Kind)
	}
}

// ParseEventName splits "name(annotation)" into bare name and annotation.
func ParseEventName(raw string) (name, annotation string) {
	idx := strings.Index(raw, "(")
	if idx < 0 {
		return raw, ""
	}
	name = raw[:idx]
	annotation = strings.TrimSuffix(raw[idx+1:], ")")
	return name, annotation
}

// validateTypeSuffix rejects invalid suffix ordering on field types.
// Valid: "type", "type?", "type[]", "type[]?" -- Invalid: "type?[]"
func validateTypeSuffix(fieldType string) error {
	if strings.Contains(fieldType, "?[]") {
		return fmt.Errorf("invalid type %q: use []? instead of ?[]", fieldType)
	}
	return nil
}

// Export serializes a Spec tree to SFT YAML format.
func Export(spec *show.Spec, w io.Writer) error {
	app := yamlApp{
		Name:        spec.App.Name,
		Description: spec.App.Description,
		Data:        spec.App.DataTypes,
		Enums:       spec.App.Enums,
		Context:     spec.App.Context,
		Regions:     exportRegions(spec.App.Regions),
		Screens:     exportScreens(spec.Screens),
		Flows:       exportFlows(spec.Flows),
		Fixtures:    exportFixtures(spec.Fixtures),
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
		if sm := exportStateMachine(s.Transitions, s.StateFixtures, s.StateRegions); sm != nil {
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
			Events:      exportEvents(r.Events),
			Ambient:     r.Ambient,
			Data:        r.RegionData,
			Regions:     exportRegions(r.Regions),
		}
		if sm := exportStateMachine(r.Transitions, r.StateFixtures, r.StateRegions); sm != nil {
			yr.StateMachine = *sm
		}
		out = append(out, yr)
	}
	return out
}

// exportEvents converts event strings (possibly with annotations) to a yaml.Node.
// If any event has an annotation, uses mapping format. Otherwise, uses sequence format.
func exportEvents(events []string) yaml.Node {
	if len(events) == 0 {
		return yaml.Node{}
	}
	// Check if any event has an annotation
	hasAnnotation := false
	for _, e := range events {
		if strings.Contains(e, "(") {
			hasAnnotation = true
			break
		}
	}
	if !hasAnnotation {
		// Simple sequence format
		node := yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, e := range events {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: e, Tag: "!!str"})
		}
		return node
	}
	// Mapping format: keys are "name(annotation)" or bare "name", values are null
	node := yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, e := range events {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: e, Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"},
		)
	}
	return node
}

// exportStateMachine converts a flat list of transitions into a state_machine yaml.Node.
// Groups transitions by FromState, producing ordered mappings. Terminal states (appear
// only as targets) are included as empty mappings. State fixtures are included as fixture: keys.
// State regions are included as regions: keys.
func exportStateMachine(transitions []show.Transition, stateFixtures map[string]string, stateRegions map[string][]string) *yaml.Node {
	if len(transitions) == 0 && len(stateFixtures) == 0 && len(stateRegions) == 0 {
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
	seenTerminal := map[string]bool{}
	for s := range allTo {
		if !seenFrom[s] {
			terminalStates = append(terminalStates, s)
			seenTerminal[s] = true
		}
	}

	// Include fixture-only or region-only states that aren't in stateOrder or terminalStates.
	extraStates := map[string]bool{}
	for s := range stateFixtures {
		if !seenFrom[s] && !seenTerminal[s] {
			extraStates[s] = true
		}
	}
	for s := range stateRegions {
		if !seenFrom[s] && !seenTerminal[s] {
			extraStates[s] = true
		}
	}
	for s := range extraStates {
		terminalStates = append(terminalStates, s)
	}
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

		// State value: mapping with optional "fixture", "regions", and "on" keys.
		stateVal := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if fixtureName, ok := stateFixtures[stateName]; ok {
			stateVal.Content = append(stateVal.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "fixture", Tag: "!!str"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: fixtureName, Tag: "!!str"},
			)
		}
		appendRegionsNode(stateVal, stateRegions[stateName])
		onKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "on", Tag: "!!str"}
		stateVal.Content = append(stateVal.Content, onKey, onMapping)

		root.Content = append(root.Content, keyNode, stateVal)
	}

	// Add terminal states as empty mappings (with optional fixture and regions).
	for _, s := range terminalStates {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"}
		valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if fixtureName, ok := stateFixtures[s]; ok {
			valNode.Content = append(valNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "fixture", Tag: "!!str"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: fixtureName, Tag: "!!str"},
			)
		}
		appendRegionsNode(valNode, stateRegions[s])
		root.Content = append(root.Content, keyNode, valNode)
	}

	return root
}

// appendRegionsNode adds a regions: flow-style sequence to a state mapping node.
func appendRegionsNode(parent *yaml.Node, regionNames []string) {
	if len(regionNames) == 0 {
		return
	}
	regSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, rn := range regionNames {
		regSeq.Content = append(regSeq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: rn, Tag: "!!str"})
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "regions", Tag: "!!str"},
		regSeq,
	)
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
	after, ok := strings.CutPrefix(action, "guard(")
	if !ok {
		return "", action
	}
	guard, rest, found := strings.Cut(after, ")")
	if !found {
		return "", action
	}
	rest = strings.TrimSpace(rest)
	rest = strings.TrimPrefix(rest, ",")
	rest = strings.TrimSpace(rest)
	return guard, rest
}

func exportFixtures(fixtures []show.Fixture) yaml.Node {
	if len(fixtures) == 0 {
		return yaml.Node{}
	}
	root := yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, f := range fixtures {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: f.Name, Tag: "!!str"}

		// Build fixture value mapping
		valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

		if f.Extends != "" {
			valNode.Content = append(valNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "extends", Tag: "!!str"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: f.Extends, Tag: "!!str"},
			)
		}

		// Marshal data back to yaml.Node
		if f.Data != nil {
			var dataNode yaml.Node
			dataBytes, _ := json.Marshal(f.Data)
			var dataMap any
			json.Unmarshal(dataBytes, &dataMap)
			if err := dataNode.Encode(dataMap); err == nil && dataNode.Kind == yaml.DocumentNode && len(dataNode.Content) > 0 {
				inner := dataNode.Content[0]
				if inner.Kind == yaml.MappingNode {
					valNode.Content = append(valNode.Content, inner.Content...)
				}
			}
		}

		root.Content = append(root.Content, keyNode, valNode)
	}
	return root
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
