package show

import "testing"

func TestApplyExperiment(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{
			Name: "dash",
			Regions: []Region{{
				Name: "kpi", DeliveryClasses: []string{"grid", "grid-cols-4"},
			}},
		}},
		Experiments: []Experiment{{
			Name: "compact", Scope: "dash.kpi", Status: "active",
			Overlay: map[string]any{"delivery": map[string]any{"classes": []any{"flex", "gap-6"}}},
		}},
	}

	applied, err := ApplyExperiment(spec, "compact")
	if err != nil {
		t.Fatal(err)
	}

	// Applied spec has new classes
	if len(applied.Screens[0].Regions[0].DeliveryClasses) != 2 {
		t.Fatalf("expected 2 delivery classes, got %d", len(applied.Screens[0].Regions[0].DeliveryClasses))
	}
	if applied.Screens[0].Regions[0].DeliveryClasses[0] != "flex" {
		t.Errorf("expected flex, got %s", applied.Screens[0].Regions[0].DeliveryClasses[0])
	}
	if applied.Screens[0].Regions[0].DeliveryClasses[1] != "gap-6" {
		t.Errorf("expected gap-6, got %s", applied.Screens[0].Regions[0].DeliveryClasses[1])
	}

	// Original unchanged
	if spec.Screens[0].Regions[0].DeliveryClasses[0] != "grid" {
		t.Error("original spec was mutated")
	}
}

func TestApplyExperiment_NotFound(t *testing.T) {
	spec := &Spec{}
	_, err := ApplyExperiment(spec, "nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestApplyExperiment_NestedRegion(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{
			Name: "dashboard",
			Regions: []Region{{
				Name: "sidebar",
				Regions: []Region{{
					Name:        "nav",
					Description: "original",
					Component:   "old-nav",
				}},
			}},
		}},
		Experiments: []Experiment{{
			Name: "new_nav", Scope: "dashboard.sidebar.nav", Status: "active",
			Overlay: map[string]any{
				"component":   "fancy-nav",
				"description": "updated nav",
			},
		}},
	}

	applied, err := ApplyExperiment(spec, "new_nav")
	if err != nil {
		t.Fatal(err)
	}

	nav := applied.Screens[0].Regions[0].Regions[0]
	if nav.Component != "fancy-nav" {
		t.Errorf("expected component fancy-nav, got %s", nav.Component)
	}
	if nav.Description != "updated nav" {
		t.Errorf("expected description 'updated nav', got %s", nav.Description)
	}

	// Original unchanged
	if spec.Screens[0].Regions[0].Regions[0].Component != "old-nav" {
		t.Error("original spec was mutated")
	}
}

func TestApplyExperiment_AllOverlayFields(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{
			Name: "home",
			Regions: []Region{{
				Name:              "hero",
				Description:       "old desc",
				Component:         "old-comp",
				ComponentProps:    `{"size":"large"}`,
				DeliveryClasses:   []string{"old-class"},
				DeliveryComponent: "old-delivery",
				DiscoveryLayout:   []string{"old-layout"},
				Tags:              []string{"old-tag"},
			}},
		}},
		Experiments: []Experiment{{
			Name: "full_overlay", Scope: "home.hero", Status: "active",
			Overlay: map[string]any{
				"delivery": map[string]any{
					"classes":   []any{"new-class-1", "new-class-2"},
					"component": "new-delivery",
				},
				"discovery": map[string]any{
					"layout": []any{"new-layout"},
				},
				"component":   "new-comp",
				"props":       `{"size":"small"}`,
				"description": "new desc",
				"tags":        []any{"new-tag-1", "new-tag-2"},
			},
		}},
	}

	applied, err := ApplyExperiment(spec, "full_overlay")
	if err != nil {
		t.Fatal(err)
	}

	r := applied.Screens[0].Regions[0]
	if r.Description != "new desc" {
		t.Errorf("description: got %q", r.Description)
	}
	if r.Component != "new-comp" {
		t.Errorf("component: got %q", r.Component)
	}
	if r.ComponentProps != `{"size":"small"}` {
		t.Errorf("props: got %q", r.ComponentProps)
	}
	if r.DeliveryComponent != "new-delivery" {
		t.Errorf("delivery_component: got %q", r.DeliveryComponent)
	}
	if len(r.DeliveryClasses) != 2 || r.DeliveryClasses[0] != "new-class-1" {
		t.Errorf("delivery_classes: got %v", r.DeliveryClasses)
	}
	if len(r.DiscoveryLayout) != 1 || r.DiscoveryLayout[0] != "new-layout" {
		t.Errorf("discovery_layout: got %v", r.DiscoveryLayout)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "new-tag-1" {
		t.Errorf("tags: got %v", r.Tags)
	}

	// Original unchanged
	if spec.Screens[0].Regions[0].Component != "old-comp" {
		t.Error("original spec was mutated")
	}
}

func TestApplyExperiment_BadScope(t *testing.T) {
	spec := &Spec{
		Experiments: []Experiment{{
			Name: "bad", Scope: "noregion", Status: "active",
			Overlay: map[string]any{},
		}},
	}
	_, err := ApplyExperiment(spec, "bad")
	if err == nil {
		t.Error("expected error for scope without region")
	}
}

func TestApplyExperiment_ScreenNotFound(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{Name: "home"}},
		Experiments: []Experiment{{
			Name: "x", Scope: "missing.region", Status: "active",
			Overlay: map[string]any{},
		}},
	}
	_, err := ApplyExperiment(spec, "x")
	if err == nil {
		t.Error("expected error for missing screen")
	}
}

func TestApplyExperiment_RegionNotFound(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{Name: "home", Regions: []Region{{Name: "header"}}}},
		Experiments: []Experiment{{
			Name: "x", Scope: "home.missing", Status: "active",
			Overlay: map[string]any{},
		}},
	}
	_, err := ApplyExperiment(spec, "x")
	if err == nil {
		t.Error("expected error for missing region")
	}
}

func TestApplyExperiment_PropsAsMap(t *testing.T) {
	spec := &Spec{
		Screens: []Screen{{
			Name:    "dash",
			Regions: []Region{{Name: "widget"}},
		}},
		Experiments: []Experiment{{
			Name: "map_props", Scope: "dash.widget", Status: "active",
			Overlay: map[string]any{
				"props": map[string]any{"key": "value", "num": float64(42)},
			},
		}},
	}

	applied, err := ApplyExperiment(spec, "map_props")
	if err != nil {
		t.Fatal(err)
	}

	// Props should be JSON-encoded
	r := applied.Screens[0].Regions[0]
	if r.ComponentProps == "" {
		t.Error("expected non-empty component_props")
	}
}
