package diff

import (
	"fmt"
	"strings"

	"github.com/lagz0ne/sft/internal/show"
)

type Change struct {
	Op     string `json:"op"`     // "+", "-", "~"
	Entity string `json:"entity"` // "screen", "region", "event", "transition", "tag"
	Name   string `json:"name"`
	In     string `json:"in,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Compare produces a list of changes needed to go from current to target.
func Compare(current, target *show.Spec) []Change {
	var changes []Change

	// App-level changes
	if current.App.Description != target.App.Description {
		changes = append(changes, Change{Op: "~", Entity: "app", Name: current.App.Name, Detail: "description changed"})
	}

	// App-level data types, enums, context
	changes = append(changes, diffDataTypes(current.App.DataTypes, target.App.DataTypes)...)
	changes = append(changes, diffEnums(current.App.Enums, target.App.Enums)...)
	changes = append(changes, diffContexts(current.App.Context, target.App.Context)...)

	// App-level regions
	changes = append(changes, diffRegions(current.App.Regions, target.App.Regions, current.App.Name)...)

	// Screens
	changes = append(changes, diffScreens(current.Screens, target.Screens)...)

	// Fixtures
	changes = append(changes, diffFixtures(current.Fixtures, target.Fixtures)...)

	// Layouts
	changes = append(changes, diffLayouts(current.Layouts, target.Layouts)...)

	return changes
}

func diffScreens(cur, tgt []show.Screen) []Change {
	var changes []Change
	curMap := make(map[string]show.Screen)
	for _, s := range cur {
		curMap[s.Name] = s
	}
	tgtMap := make(map[string]show.Screen)
	for _, s := range tgt {
		tgtMap[s.Name] = s
	}

	for _, s := range tgt {
		if _, ok := curMap[s.Name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "screen", Name: s.Name})
		}
	}
	for _, s := range cur {
		if _, ok := tgtMap[s.Name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "screen", Name: s.Name})
		}
	}
	for _, s := range tgt {
		cs, ok := curMap[s.Name]
		if !ok {
			continue
		}
		if cs.Description != s.Description {
			changes = append(changes, Change{Op: "~", Entity: "screen", Name: s.Name, Detail: "description changed"})
		}
		if cs.Component != s.Component || cs.ComponentProps != s.ComponentProps ||
			cs.ComponentOn != s.ComponentOn || cs.ComponentVis != s.ComponentVis {
			changes = append(changes, Change{Op: "~", Entity: "screen", Name: s.Name, Detail: "component changed"})
		}
		if !mapsEqual(cs.Context, s.Context) {
			changes = append(changes, Change{Op: "~", Entity: "screen", Name: s.Name, Detail: "context changed"})
		}
		if !mapsEqual(cs.StateFixtures, s.StateFixtures) {
			changes = append(changes, Change{Op: "~", Entity: "screen", Name: s.Name, Detail: "state_fixtures changed"})
		}
		if !mapSlicesEqual(cs.StateRegions, s.StateRegions) {
			changes = append(changes, Change{Op: "~", Entity: "screen", Name: s.Name, Detail: "state_regions changed"})
		}
		changes = append(changes, diffTags(cs.Tags, s.Tags, s.Name)...)
		changes = append(changes, diffRegions(cs.Regions, s.Regions, s.Name)...)
		changes = append(changes, diffTransitions(cs.Transitions, s.Transitions, s.Name)...)
	}
	return changes
}

func diffRegions(cur, tgt []show.Region, parent string) []Change {
	var changes []Change
	curMap := make(map[string]show.Region)
	for _, r := range cur {
		curMap[r.Name] = r
	}
	tgtMap := make(map[string]show.Region)
	for _, r := range tgt {
		tgtMap[r.Name] = r
	}

	for _, r := range tgt {
		if _, ok := curMap[r.Name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "region", Name: r.Name, In: parent})
		}
	}
	for _, r := range cur {
		if _, ok := tgtMap[r.Name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "region", Name: r.Name, In: parent})
		}
	}
	for _, r := range tgt {
		cr, ok := curMap[r.Name]
		if !ok {
			continue
		}
		if cr.Description != r.Description {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "description changed"})
		}
		if cr.Component != r.Component || cr.ComponentProps != r.ComponentProps ||
			cr.ComponentOn != r.ComponentOn || cr.ComponentVis != r.ComponentVis {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "component changed"})
		}
		if !sliceSetsEqual(cr.DeliveryClasses, r.DeliveryClasses) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "delivery_classes changed"})
		}
		if !sliceSetsEqual(cr.DiscoveryLayout, r.DiscoveryLayout) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "discovery_layout changed"})
		}
		if !mapsEqual(cr.Ambient, r.Ambient) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "ambient changed"})
		}
		if !mapsEqual(cr.RegionData, r.RegionData) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "region_data changed"})
		}
		if !mapsEqual(cr.StateFixtures, r.StateFixtures) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "state_fixtures changed"})
		}
		if !mapSlicesEqual(cr.StateRegions, r.StateRegions) {
			changes = append(changes, Change{Op: "~", Entity: "region", Name: r.Name, In: parent, Detail: "state_regions changed"})
		}
		changes = append(changes, diffTags(cr.Tags, r.Tags, r.Name)...)
		changes = append(changes, diffEvents(cr.Events, r.Events, r.Name)...)
		changes = append(changes, diffRegions(cr.Regions, r.Regions, r.Name)...)
		changes = append(changes, diffTransitions(cr.Transitions, r.Transitions, r.Name)...)
	}
	return changes
}

func diffTags(cur, tgt []string, parent string) []Change {
	var changes []Change
	curSet := setFrom(cur)
	tgtSet := setFrom(tgt)
	for _, t := range tgt {
		if !curSet[t] {
			changes = append(changes, Change{Op: "+", Entity: "tag", Name: t, In: parent})
		}
	}
	for _, t := range cur {
		if !tgtSet[t] {
			changes = append(changes, Change{Op: "-", Entity: "tag", Name: t, In: parent})
		}
	}
	return changes
}

func diffEvents(cur, tgt []string, parent string) []Change {
	var changes []Change
	curSet := setFrom(cur)
	tgtSet := setFrom(tgt)
	for _, e := range tgt {
		if !curSet[e] {
			changes = append(changes, Change{Op: "+", Entity: "event", Name: e, In: parent})
		}
	}
	for _, e := range cur {
		if !tgtSet[e] {
			changes = append(changes, Change{Op: "-", Entity: "event", Name: e, In: parent})
		}
	}
	return changes
}

func diffTransitions(cur, tgt []show.Transition, parent string) []Change {
	var changes []Change
	type tKey struct{ OnEvent, FromState string }
	curMap := make(map[tKey]show.Transition)
	for _, t := range cur {
		curMap[tKey{t.OnEvent, t.FromState}] = t
	}
	tgtMap := make(map[tKey]show.Transition)
	for _, t := range tgt {
		tgtMap[tKey{t.OnEvent, t.FromState}] = t
	}

	for k, t := range tgtMap {
		if _, ok := curMap[k]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "transition", Name: transitionName(t), In: parent})
		}
	}
	for k, t := range curMap {
		if _, ok := tgtMap[k]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "transition", Name: transitionName(t), In: parent})
		}
	}
	for k, t := range tgtMap {
		ct, ok := curMap[k]
		if !ok {
			continue
		}
		if ct.ToState != t.ToState || ct.Action != t.Action {
			changes = append(changes, Change{Op: "~", Entity: "transition", Name: transitionName(t), In: parent, Detail: "changed"})
		}
	}
	return changes
}

func diffDataTypes(cur, tgt map[string]map[string]string) []Change {
	var changes []Change
	for name := range tgt {
		if _, ok := cur[name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "data_type", Name: name})
		}
	}
	for name := range cur {
		if _, ok := tgt[name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "data_type", Name: name})
		}
	}
	for name, tgtFields := range tgt {
		curFields, ok := cur[name]
		if !ok {
			continue
		}
		if !mapsEqual(curFields, tgtFields) {
			var details []string
			for f := range tgtFields {
				if _, ok := curFields[f]; !ok {
					details = append(details, "+"+f)
				} else if curFields[f] != tgtFields[f] {
					details = append(details, "~"+f)
				}
			}
			for f := range curFields {
				if _, ok := tgtFields[f]; !ok {
					details = append(details, "-"+f)
				}
			}
			changes = append(changes, Change{Op: "~", Entity: "data_type", Name: name, Detail: "fields changed"})
		}
	}
	return changes
}

func diffEnums(cur, tgt map[string][]string) []Change {
	var changes []Change
	for name := range tgt {
		if _, ok := cur[name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "enum", Name: name})
		}
	}
	for name := range cur {
		if _, ok := tgt[name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "enum", Name: name})
		}
	}
	for name, tgtValues := range tgt {
		curValues, ok := cur[name]
		if !ok {
			continue
		}
		if !sliceSetsEqual(curValues, tgtValues) {
			changes = append(changes, Change{Op: "~", Entity: "enum", Name: name, Detail: "values changed"})
		}
	}
	return changes
}

func diffContexts(cur, tgt map[string]string) []Change {
	var changes []Change
	for name := range tgt {
		if _, ok := cur[name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "context", Name: name})
		}
	}
	for name := range cur {
		if _, ok := tgt[name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "context", Name: name})
		}
	}
	for name, tgtVal := range tgt {
		curVal, ok := cur[name]
		if !ok {
			continue
		}
		if curVal != tgtVal {
			changes = append(changes, Change{Op: "~", Entity: "context", Name: name, Detail: "type changed"})
		}
	}
	return changes
}

func diffFixtures(cur, tgt []show.Fixture) []Change {
	var changes []Change
	curMap := make(map[string]show.Fixture)
	for _, f := range cur {
		curMap[f.Name] = f
	}
	tgtMap := make(map[string]show.Fixture)
	for _, f := range tgt {
		tgtMap[f.Name] = f
	}

	for _, f := range tgt {
		if _, ok := curMap[f.Name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "fixture", Name: f.Name})
		}
	}
	for _, f := range cur {
		if _, ok := tgtMap[f.Name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "fixture", Name: f.Name})
		}
	}
	for _, f := range tgt {
		cf, ok := curMap[f.Name]
		if !ok {
			continue
		}
		var details []string
		if cf.Extends != f.Extends {
			details = append(details, "extends")
		}
		if fmt.Sprintf("%v", cf.Data) != fmt.Sprintf("%v", f.Data) {
			details = append(details, "data")
		}
		if len(details) > 0 {
			changes = append(changes, Change{Op: "~", Entity: "fixture", Name: f.Name, Detail: strings.Join(details, ", ") + " changed"})
		}
	}
	return changes
}

func diffLayouts(cur, tgt map[string][]string) []Change {
	var changes []Change
	for name := range tgt {
		if _, ok := cur[name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "layout", Name: name})
		}
	}
	for name := range cur {
		if _, ok := tgt[name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "layout", Name: name})
		}
	}
	for name, tgtClasses := range tgt {
		curClasses, ok := cur[name]
		if !ok {
			continue
		}
		if !sliceSetsEqual(curClasses, tgtClasses) {
			changes = append(changes, Change{Op: "~", Entity: "layout", Name: name, Detail: "classes changed"})
		}
	}
	return changes
}

// TODO: diffEntities and diffExperiments — wire in when show.Entity / show.Experiment types are available (Task 4)

func transitionName(t show.Transition) string {
	s := t.OnEvent
	if t.FromState != "" {
		s += " from " + t.FromState
	}
	return s
}

func setFrom(ss []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func sliceSetsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aSet := setFrom(a)
	for _, v := range b {
		if !aSet[v] {
			return false
		}
	}
	return true
}

func mapSlicesEqual(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !sliceSetsEqual(av, bv) {
			return false
		}
	}
	return true
}

// Format renders changes as human-readable text.
func Format(changes []Change) string {
	if len(changes) == 0 {
		return "no changes"
	}
	var b strings.Builder
	for _, c := range changes {
		fmt.Fprintf(&b, "%s %s %s", c.Op, c.Entity, c.Name)
		if c.In != "" {
			fmt.Fprintf(&b, " in %s", c.In)
		}
		if c.Detail != "" {
			fmt.Fprintf(&b, ": %s", c.Detail)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
