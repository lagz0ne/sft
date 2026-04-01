package show

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ApplyExperiment returns a deep-copy of the spec with the named experiment's
// overlay applied to its target region. The original spec is NOT modified.
func ApplyExperiment(spec *Spec, name string) (*Spec, error) {
	// Find the experiment
	var exp *Experiment
	for i := range spec.Experiments {
		if spec.Experiments[i].Name == name {
			exp = &spec.Experiments[i]
			break
		}
	}
	if exp == nil {
		return nil, fmt.Errorf("experiment %q not found", name)
	}

	// Parse scope: "screen.region" or "screen.parent.child"
	parts := strings.SplitN(exp.Scope, ".", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("experiment %q: scope %q must be screen.region", name, exp.Scope)
	}
	screenName := parts[0]
	regionPath := parts[1] // may contain dots for nested regions

	// Deep-copy via JSON round-trip
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("deep copy marshal: %w", err)
	}
	var copy Spec
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("deep copy unmarshal: %w", err)
	}

	// Find the target screen
	var targetScreen *Screen
	for i := range copy.Screens {
		if copy.Screens[i].Name == screenName {
			targetScreen = &copy.Screens[i]
			break
		}
	}
	if targetScreen == nil {
		return nil, fmt.Errorf("experiment %q: screen %q not found", name, screenName)
	}

	// Walk to the target region (supports nested paths like "parent.child")
	regionParts := strings.Split(regionPath, ".")
	region := findRegion(targetScreen.Regions, regionParts)
	if region == nil {
		return nil, fmt.Errorf("experiment %q: region %q not found in screen %q", name, regionPath, screenName)
	}

	// Parse overlay
	overlay, ok := exp.Overlay.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("experiment %q: overlay is not a map", name)
	}

	// Apply overlay fields
	applyOverlay(region, overlay)

	return &copy, nil
}

// findRegion walks a region tree following a path of names.
func findRegion(regions []Region, path []string) *Region {
	if len(path) == 0 {
		return nil
	}
	for i := range regions {
		if regions[i].Name == path[0] {
			if len(path) == 1 {
				return &regions[i]
			}
			return findRegion(regions[i].Regions, path[1:])
		}
	}
	return nil
}

// applyOverlay shallow-merges overlay keys onto a region.
func applyOverlay(r *Region, overlay map[string]any) {
	if v, ok := overlay["delivery"]; ok {
		if dm, ok := v.(map[string]any); ok {
			if classes, ok := dm["classes"]; ok {
				r.DeliveryClasses = toStringSlice(classes)
			}
			if comp, ok := dm["component"]; ok {
				if s, ok := comp.(string); ok {
					r.DeliveryComponent = s
				}
			}
		}
	}
	if v, ok := overlay["discovery"]; ok {
		if dm, ok := v.(map[string]any); ok {
			if layout, ok := dm["layout"]; ok {
				r.DiscoveryLayout = toStringSlice(layout)
			}
		}
	}
	if v, ok := overlay["component"]; ok {
		if s, ok := v.(string); ok {
			r.Component = s
		}
	}
	if v, ok := overlay["props"]; ok {
		switch p := v.(type) {
		case string:
			r.ComponentProps = p
		default:
			// JSON-encode non-string props
			if b, err := json.Marshal(p); err == nil {
				r.ComponentProps = string(b)
			}
		}
	}
	if v, ok := overlay["description"]; ok {
		if s, ok := v.(string); ok {
			r.Description = s
		}
	}
	if v, ok := overlay["tags"]; ok {
		r.Tags = toStringSlice(v)
	}
}

// toStringSlice converts an any (expected []any of strings) to []string.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []any:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return s
	}
	return nil
}
