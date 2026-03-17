package query

import (
	"database/sql"
	"fmt"
	"strings"
)

var namedQueries = map[string]string{
	"screens": "SELECT id, app_id, name, description FROM screens",
	"events":  "SELECT event, emitted_by, parent_type, handled_at, from_state, to_state, action FROM event_index",
	"flows":   "SELECT id, name, description, on_event, sequence FROM flows",
	"tags":    "SELECT tag, entity_type, entity_name FROM tag_index",
	"regions": "SELECT id, name, description, parent_type, parent_name, event_count, has_states FROM region_tree",
}

// Run executes a named query or raw SQL and returns rows as []map[string]any.
func Run(db *sql.DB, input string, args ...string) ([]map[string]any, error) {
	q, ok := namedQueries[input]
	if !ok {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(input)), "SELECT") {
			q = input
		} else {
			return nil, fmt.Errorf("unknown query %q (available: screens, events, flows, tags, regions, or raw SELECT)", input)
		}
	}

	// Handle "states <name>" query
	if input == "states" && len(args) > 0 {
		q = `SELECT owner_type, owner_name, on_event, from_state, to_state, action
		     FROM state_machines WHERE owner_name = ?`
		return execQuery(db, q, args[0])
	}

	return execQuery(db, q)
}

// States runs the state_machines view filtered by owner name.
func States(db *sql.DB, name string) ([]map[string]any, error) {
	q := `SELECT owner_type, owner_name, on_event, from_state, to_state, action
	      FROM state_machines WHERE owner_name = ?`
	return execQuery(db, q, name)
}

func execQuery(db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[col] = string(v)
			default:
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
