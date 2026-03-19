package loader

import (
	"fmt"
	"strings"

	"github.com/lagz0ne/sft/internal/model"
	"gopkg.in/yaml.v3"
)

// ParseStateMachine parses a state_machine YAML mapping node into transitions.
// Returns all transitions, ordered state names (first = initial), and any error.
// OwnerType/OwnerID are left unset — the caller fills those in.
func ParseStateMachine(node yaml.Node) ([]model.Transition, []string, error) {
	if node.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("state_machine: expected mapping, got kind %d", node.Kind)
	}

	var transitions []model.Transition
	var states []string

	// Iterate state-name / state-def pairs.
	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		stateName := keyNode.Value
		states = append(states, stateName)

		// Terminal or null state: no transitions.
		if valNode.Kind == yaml.ScalarNode || (valNode.Kind == yaml.MappingNode && len(valNode.Content) == 0) {
			continue
		}
		if valNode.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("state %q: expected mapping, got kind %d", stateName, valNode.Kind)
		}

		// Find the "on" key inside the state definition.
		onNode := findKey(valNode, "on")
		if onNode == nil {
			continue
		}

		// on: {} — empty mapping
		if onNode.Kind == yaml.MappingNode && len(onNode.Content) == 0 {
			continue
		}
		if onNode.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("state %q: on: expected mapping, got kind %d", stateName, onNode.Kind)
		}

		// Iterate event / target pairs inside on:.
		for j := 0; j < len(onNode.Content)-1; j += 2 {
			eventNode := onNode.Content[j]
			targetNode := onNode.Content[j+1]
			event := eventNode.Value

			parsed, err := parseTarget(stateName, event, targetNode)
			if err != nil {
				return nil, nil, fmt.Errorf("state %q event %q: %w", stateName, event, err)
			}
			transitions = append(transitions, parsed...)
		}
	}

	return transitions, states, nil
}

// findKey returns the value node for a given key in a MappingNode, or nil.
func findKey(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// parseTarget interprets the value side of an event entry.
func parseTarget(fromState, event string, node *yaml.Node) ([]model.Transition, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return []model.Transition{scalarTransition(fromState, event, node.Value)}, nil

	case yaml.MappingNode:
		return []model.Transition{objectTransition(fromState, event, node)}, nil

	case yaml.SequenceNode:
		return guardedTransitions(fromState, event, node)

	default:
		return nil, fmt.Errorf("unexpected node kind %d", node.Kind)
	}
}

// scalarTransition handles: `event: target_state`, `event: .`, and action shorthand.
func scalarTransition(fromState, event, value string) model.Transition {
	t := model.Transition{OnEvent: event, FromState: fromState}

	if isActionShorthand(value) {
		t.Action = value
		return t
	}

	t.ToState = resolveDot(value, fromState)
	return t
}

// objectTransition handles: `event: { to: x, action: y }`.
func objectTransition(fromState, event string, node *yaml.Node) model.Transition {
	t := model.Transition{OnEvent: event, FromState: fromState}
	for i := 0; i < len(node.Content)-1; i += 2 {
		k := node.Content[i].Value
		v := node.Content[i+1].Value
		switch k {
		case "to":
			t.ToState = resolveDot(v, fromState)
		case "action":
			t.Action = v
		}
	}
	return t
}

// guardedTransitions handles: `event: [{ guard: "a", to: x }, ...]`.
func guardedTransitions(fromState, event string, node *yaml.Node) ([]model.Transition, error) {
	var out []model.Transition
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("guarded entry: expected mapping, got kind %d", item.Kind)
		}
		t := model.Transition{OnEvent: event, FromState: fromState}
		var guard, action string
		for i := 0; i < len(item.Content)-1; i += 2 {
			k := item.Content[i].Value
			v := item.Content[i+1].Value
			switch k {
			case "guard":
				guard = v
			case "to":
				t.ToState = resolveDot(v, fromState)
			case "action":
				action = v
			}
		}
		t.Action = formatGuardAction(guard, action)
		out = append(out, t)
	}
	return out, nil
}

// formatGuardAction builds the Action string for guarded transitions.
func formatGuardAction(guard, action string) string {
	if guard == "" {
		return action
	}
	g := "guard(" + guard + ")"
	if action == "" {
		return g
	}
	return g + ", " + action
}

// resolveDot returns fromState when value is ".", otherwise returns value unchanged.
func resolveDot(value, fromState string) string {
	if value == "." {
		return fromState
	}
	return value
}

// isActionShorthand returns true for `navigate(...)` or `emit(...)` patterns.
func isActionShorthand(s string) bool {
	return (strings.HasPrefix(s, "navigate(") && strings.HasSuffix(s, ")")) ||
		(strings.HasPrefix(s, "emit(") && strings.HasSuffix(s, ")"))
}
