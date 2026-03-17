package loader

import (
	"fmt"
	"io"
	"os"

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
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Regions     []yamlRegion     `yaml:"regions,omitempty"`
	Screens     []yamlScreen     `yaml:"screens,omitempty"`
	Flows       []yamlFlow       `yaml:"flows,omitempty"`
}

type yamlScreen struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Tags        []string         `yaml:"tags,omitempty"`
	Component   string           `yaml:"component,omitempty"`
	Props       string           `yaml:"props,omitempty"`
	OnActions   string           `yaml:"on_actions,omitempty"`
	Visible     string           `yaml:"visible,omitempty"`
	Regions     []yamlRegion     `yaml:"regions,omitempty"`
	States      []yamlTransition `yaml:"states,omitempty"`
}

type yamlRegion struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Tags        []string         `yaml:"tags,omitempty"`
	Component   string           `yaml:"component,omitempty"`
	Props       string           `yaml:"props,omitempty"`
	OnActions   string           `yaml:"on_actions,omitempty"`
	Visible     string           `yaml:"visible,omitempty"`
	Events      []string         `yaml:"events,omitempty"`
	Regions     []yamlRegion     `yaml:"regions,omitempty"`
	States      []yamlTransition `yaml:"states,omitempty"`
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
		for _, t := range sc.States {
			if err := s.InsertTransition(&model.Transition{
				OwnerType: "screen", OwnerID: screen.ID,
				OnEvent: t.On, FromState: t.From, ToState: t.To, Action: t.Action,
			}); err != nil {
				return fmt.Errorf("transition on %s in screen %s: %w", t.On, sc.Name, err)
			}
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
	for _, child := range r.Regions {
		if err := insertRegion(s, appID, "region", region.ID, child); err != nil {
			return err
		}
	}
	for _, t := range r.States {
		if err := s.InsertTransition(&model.Transition{
			OwnerType: "region", OwnerID: region.ID,
			OnEvent: t.On, FromState: t.From, ToState: t.To, Action: t.Action,
		}); err != nil {
			return fmt.Errorf("transition on %s in %s: %w", t.On, r.Name, err)
		}
	}
	return nil
}

// Export serializes a Spec tree to SFT YAML format.
func Export(spec *show.Spec, w io.Writer) error {
	app := yamlApp{
		Name:        spec.App.Name,
		Description: spec.App.Description,
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
		out = append(out, yamlScreen{
			Name:        s.Name,
			Description: s.Description,
			Tags:        s.Tags,
			Component:   s.Component,
			Props:       s.ComponentProps,
			OnActions:   s.ComponentOn,
			Visible:     s.ComponentVis,
			Regions:     exportRegions(s.Regions),
			States:      exportTransitions(s.Transitions),
		})
	}
	return out
}

func exportRegions(regions []show.Region) []yamlRegion {
	if len(regions) == 0 {
		return nil
	}
	var out []yamlRegion
	for _, r := range regions {
		out = append(out, yamlRegion{
			Name:        r.Name,
			Description: r.Description,
			Tags:        r.Tags,
			Component:   r.Component,
			Props:       r.ComponentProps,
			OnActions:   r.ComponentOn,
			Visible:     r.ComponentVis,
			Events:      r.Events,
			Regions:     exportRegions(r.Regions),
			States:      exportTransitions(r.Transitions),
		})
	}
	return out
}

func exportTransitions(transitions []show.Transition) []yamlTransition {
	if len(transitions) == 0 {
		return nil
	}
	var out []yamlTransition
	for _, t := range transitions {
		out = append(out, yamlTransition{
			On:     t.OnEvent,
			From:   t.FromState,
			To:     t.ToState,
			Action: t.Action,
		})
	}
	return out
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
