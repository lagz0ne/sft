package render

import (
	"encoding/json"
	"strings"

	"github.com/lagz0ne/sft/internal/show"
)

type Spec struct {
	Root     string              `json:"root"`
	Elements map[string]*Element `json:"elements"`
}

type Element struct {
	Type      string         `json:"type"`
	Props     map[string]any `json:"props"`
	Children  []string       `json:"children"`
	ClassName string         `json:"className,omitempty"`
	Visible   any            `json:"visible,omitempty"`
	On        map[string]any `json:"on,omitempty"`
}

// CompDef holds a component definition from the DB.
type CompDef struct {
	Component string
	Props     string
	OnActions string
	Visible   string
}

// FromSFT generates a json-render spec from an SFT spec. [removed dead getComp param]
func FromSFT(spec *show.Spec) *Spec {
	jr := &Spec{
		Elements: make(map[string]*Element),
	}

	if len(spec.Screens) == 0 && len(spec.App.Regions) == 0 {
		return jr
	}

	// Determine root
	if len(spec.Screens) == 1 && len(spec.App.Regions) == 0 {
		jr.Root = spec.Screens[0].Name
	} else {
		jr.Root = spec.App.Name
		root := &Element{
			Type:     "Stack",
			Props:    map[string]any{"direction": "vertical"},
			Children: []string{},
		}
		// App-level regions first [C2]
		for _, r := range spec.App.Regions {
			root.Children = append(root.Children, r.Name)
		}
		for _, s := range spec.Screens {
			root.Children = append(root.Children, s.Name)
		}
		jr.Elements[spec.App.Name] = root
	}

	// App-level regions [C2]
	addRegions(jr, spec.App.Regions, spec.Layouts)

	for _, s := range spec.Screens {
		el := screenToElement(s)
		jr.Elements[s.Name] = el
		addRegions(jr, s.Regions, spec.Layouts)
	}

	return jr
}

func screenToElement(s show.Screen) *Element {
	el := &Element{
		Type:     "Card",
		Props:    map[string]any{"title": s.Name},
		Children: []string{},
	}
	if s.Component != "" {
		el.Type = s.Component
	}
	for _, r := range s.Regions {
		el.Children = append(el.Children, r.Name)
	}
	return el
}

func addRegions(jr *Spec, regions []show.Region, presets map[string][]string) {
	for _, r := range regions {
		el := &Element{
			Type:     "Stack",
			Props:    map[string]any{},
			Children: []string{},
		}
		if r.Component != "" {
			el.Type = r.Component
		}
		if len(r.DeliveryClasses) > 0 {
			el.ClassName = strings.Join(r.DeliveryClasses, " ")
		} else if len(r.DiscoveryLayout) > 0 {
			el.ClassName = strings.Join(expandPresets(r.DiscoveryLayout, presets), " ")
		}
		if r.DeliveryComponent != "" {
			el.Type = r.DeliveryComponent
		}
		for _, child := range r.Regions {
			el.Children = append(el.Children, child.Name)
		}
		jr.Elements[r.Name] = el
		addRegions(jr, r.Regions, presets)
	}
}

func expandPresets(layout []string, presets map[string][]string) []string {
	var classes []string
	for _, item := range layout {
		if preset, ok := presets[item]; ok {
			classes = append(classes, preset...)
		} else {
			classes = append(classes, item)
		}
	}
	return classes
}

// Hydrate enriches the generated spec with stored component props from the DB.
// [H5 fix: always apply stored props when component exists, even if "{}"]
func Hydrate(jr *Spec, getComp func(name string) *CompDef) {
	for name, el := range jr.Elements {
		comp := getComp(name)
		if comp == nil {
			continue
		}
		el.Type = comp.Component
		// Always apply stored props — even "{}" resets to clean state [H5]
		var props map[string]any
		if err := json.Unmarshal([]byte(comp.Props), &props); err == nil {
			el.Props = props
		}
		if comp.OnActions != "" {
			var on map[string]any
			if err := json.Unmarshal([]byte(comp.OnActions), &on); err == nil {
				el.On = on
			}
		}
		if comp.Visible != "" {
			var vis any
			if err := json.Unmarshal([]byte(comp.Visible), &vis); err == nil {
				el.Visible = vis
			}
		}
	}
}
