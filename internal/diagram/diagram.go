package diagram

import (
	"database/sql"
	"fmt"
	"strings"
)

// States generates a Mermaid stateDiagram-v2 for the given owner name.
func States(db *sql.DB, name string) (string, error) {
	rows, err := db.Query(`SELECT on_event, from_state, to_state, action
		FROM state_machines WHERE owner_name = ?`, name)
	if err != nil {
		return "", fmt.Errorf("diagram states: %w", err)
	}
	defer rows.Close()

	type transition struct {
		event, from, to, action string
	}
	var transitions []transition
	stateSet := map[string]bool{}

	for rows.Next() {
		var t transition
		var from, to, action sql.NullString
		if err := rows.Scan(&t.event, &from, &to, &action); err != nil {
			return "", err
		}
		t.from = from.String
		t.to = to.String
		t.action = action.String
		transitions = append(transitions, t)
		if t.from != "" {
			stateSet[t.from] = true
		}
		if t.to != "" {
			stateSet[t.to] = true
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(transitions) == 0 {
		return "", fmt.Errorf("no state machine found for %q", name)
	}

	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")

	// Find initial state: appears as to_state but never as from_state,
	// or first from_state if all states appear as from.
	fromStates := map[string]bool{}
	toStates := map[string]bool{}
	for _, t := range transitions {
		if t.from != "" {
			fromStates[t.from] = true
		}
		if t.to != "" {
			toStates[t.to] = true
		}
	}

	// Initial state = first from_state that is never a to_state, or first from_state
	initial := ""
	for _, t := range transitions {
		if t.from != "" && !toStates[t.from] {
			initial = t.from
			break
		}
	}
	if initial == "" {
		for _, t := range transitions {
			if t.from != "" {
				initial = t.from
				break
			}
		}
	}
	if initial != "" {
		fmt.Fprintf(&b, "  [*] --> %s\n", mermaidID(initial))
	}

	// Terminal states: appear as to_state but never as from_state
	for s := range stateSet {
		if !fromStates[s] && toStates[s] {
			fmt.Fprintf(&b, "  %s --> [*]\n", mermaidID(s))
		}
	}

	for _, t := range transitions {
		from := t.from
		to := t.to
		if from == "" {
			from = "[*]"
		} else {
			from = mermaidID(from)
		}
		if to == "" || to == "." {
			to = from // self-transition
		} else {
			to = mermaidID(to)
		}

		label := t.event
		if t.action != "" {
			label += " / " + t.action
		}
		fmt.Fprintf(&b, "  %s --> %s : %s\n", from, to, label)
	}

	return b.String(), nil
}

// Nav generates a Mermaid flowchart of screen-to-screen navigation.
func Nav(db *sql.DB) (string, error) {
	rows, err := db.Query(`SELECT
		CASE t.owner_type
			WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t.owner_id)
			WHEN 'region' THEN (SELECT r2.name FROM regions r2 WHERE r2.id = t.owner_id)
			WHEN 'app' THEN (SELECT a.name FROM apps a WHERE a.id = t.owner_id)
		END AS source,
		t.action
		FROM transitions t
		WHERE t.action LIKE 'navigate(%'`)
	if err != nil {
		return "", fmt.Errorf("diagram nav: %w", err)
	}
	defer rows.Close()

	type edge struct{ from, to string }
	seen := map[edge]bool{}
	var edges []edge

	for rows.Next() {
		var source, action string
		if err := rows.Scan(&source, &action); err != nil {
			return "", err
		}
		target := extractNavigateTarget(action)
		if target == "" {
			continue
		}
		e := edge{source, target}
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(edges) == 0 {
		return "", fmt.Errorf("no navigate() actions found")
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")
	for _, e := range edges {
		fmt.Fprintf(&b, "  %s --> %s\n", mermaidID(e.from), mermaidID(e.to))
	}
	return b.String(), nil
}

// Flow generates a Mermaid flowchart for a named flow.
func Flow(db *sql.DB, name string) (string, error) {
	rows, err := db.Query(`SELECT fs.position, fs.type, fs.name, fs.data, fs.history
		FROM flow_steps fs JOIN flows f ON f.id = fs.flow_id
		WHERE f.name = ? ORDER BY fs.position`, name)
	if err != nil {
		return "", fmt.Errorf("diagram flow: %w", err)
	}
	defer rows.Close()

	type step struct {
		pos     int
		typ     string
		name    string
		data    string
		history bool
	}
	var steps []step
	for rows.Next() {
		var s step
		var data sql.NullString
		var hist int
		if err := rows.Scan(&s.pos, &s.typ, &s.name, &data, &hist); err != nil {
			return "", err
		}
		s.data = data.String
		s.history = hist > 0
		steps = append(steps, s)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(steps) == 0 {
		return "", fmt.Errorf("no flow found for %q", name)
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")

	for i, s := range steps {
		id := fmt.Sprintf("s%d", i)
		label := s.name
		if s.data != "" {
			label += "\\n{" + s.data + "}"
		}

		switch s.typ {
		case "screen":
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		case "region":
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		case "event":
			fmt.Fprintf(&b, "  %s{{%s}}\n", id, label)
		case "back":
			fmt.Fprintf(&b, "  %s>Back]\n", id)
		case "action", "activate":
			fmt.Fprintf(&b, "  %s(%s)\n", id, label)
		default:
			fmt.Fprintf(&b, "  %s[%s]\n", id, label)
		}

		if i > 0 {
			prev := fmt.Sprintf("s%d", i-1)
			fmt.Fprintf(&b, "  %s --> %s\n", prev, id)
		}
	}

	return b.String(), nil
}

func mermaidID(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "_"), "-", "_")
}

func extractNavigateTarget(action string) string {
	// navigate(target) or navigate(target, {params})
	if !strings.HasPrefix(action, "navigate(") {
		return ""
	}
	inner := action[len("navigate("):]
	// Strip trailing )
	if idx := strings.IndexAny(inner, ",)"); idx >= 0 {
		inner = inner[:idx]
	}
	return strings.TrimSpace(inner)
}
