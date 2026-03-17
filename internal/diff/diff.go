package diff

import (
	"fmt"
	"strings"

	"github.com/lagz0ne/sft/internal/show"
)

type Change struct {
	Op     string `json:"op"`     // "+", "-", "~"
	Entity string `json:"entity"` // "screen", "region", "event", "transition", "flow", "tag"
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

	// App-level regions
	changes = append(changes, diffRegions(current.App.Regions, target.App.Regions, current.App.Name)...)

	// Screens
	changes = append(changes, diffScreens(current.Screens, target.Screens)...)

	// Flows
	changes = append(changes, diffFlows(current.Flows, target.Flows)...)

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

func diffFlows(cur, tgt []show.Flow) []Change {
	var changes []Change
	curMap := make(map[string]show.Flow)
	for _, f := range cur {
		curMap[f.Name] = f
	}
	tgtMap := make(map[string]show.Flow)
	for _, f := range tgt {
		tgtMap[f.Name] = f
	}

	for _, f := range tgt {
		if _, ok := curMap[f.Name]; !ok {
			changes = append(changes, Change{Op: "+", Entity: "flow", Name: f.Name})
		}
	}
	for _, f := range cur {
		if _, ok := tgtMap[f.Name]; !ok {
			changes = append(changes, Change{Op: "-", Entity: "flow", Name: f.Name})
		}
	}
	for _, f := range tgt {
		cf, ok := curMap[f.Name]
		if !ok {
			continue
		}
		var details []string
		if cf.Description != f.Description {
			details = append(details, "description")
		}
		if cf.Sequence != f.Sequence {
			details = append(details, "sequence")
		}
		if cf.OnEvent != f.OnEvent {
			details = append(details, "on_event")
		}
		if len(details) > 0 {
			changes = append(changes, Change{Op: "~", Entity: "flow", Name: f.Name, Detail: strings.Join(details, ", ") + " changed"})
		}
	}
	return changes
}

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
