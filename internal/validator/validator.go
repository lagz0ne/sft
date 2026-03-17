package validator

import (
	"database/sql"
	"fmt"
)

type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

type rule struct {
	id       string
	severity Severity
	query    string
	format   func(rows *sql.Rows) ([]Finding, error)
}

// ownerCase resolves owner_type+owner_id to a human-readable name via subquery.
const ownerCase = `CASE t.owner_type
  WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t.owner_id)
  WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = t.owner_id)
  WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = t.owner_id)
END`

var rules = []rule{
	{
		id:       "missing-description",
		severity: Error,
		query: `SELECT 'screen' AS type, name FROM screens WHERE description = ''
		        UNION ALL
		        SELECT 'region' AS type, name FROM regions WHERE description = ''`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var typ, name string
				if err := rows.Scan(&typ, &name); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "missing-description",
					Severity: Error,
					Message:  fmt.Sprintf("%s %q has no description", typ, name),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "orphan-emit",
		severity: Error,
		query: `SELECT t.action, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.action LIKE 'emit(%)'
		          AND SUBSTR(t.action, 6, LENGTH(t.action) - 6) NOT IN (
		            SELECT t2.on_event FROM transitions t2 WHERE t2.id != t.id
		          )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var action string
				var owner sql.NullString
				if err := rows.Scan(&action, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "orphan-emit",
					Severity: Error,
					Message:  fmt.Sprintf("%s in %s emits event with no handler", action, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "unreachable-state",
		severity: Error,
		query: `SELECT DISTINCT t1.from_state, ` + `CASE t1.owner_type
		  WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t1.owner_id)
		  WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = t1.owner_id)
		  WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = t1.owner_id)
		END` + ` AS owner_name
		        FROM transitions t1
		        WHERE t1.from_state IS NOT NULL
		          AND t1.from_state NOT IN (
		            SELECT t2.to_state FROM transitions t2
		            WHERE t2.owner_type = t1.owner_type AND t2.owner_id = t1.owner_id AND t2.to_state IS NOT NULL
		          )
		          AND t1.rowid != (
		            SELECT MIN(t3.rowid) FROM transitions t3
		            WHERE t3.owner_type = t1.owner_type AND t3.owner_id = t1.owner_id AND t3.from_state IS NOT NULL
		          )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var state string
				var owner sql.NullString
				if err := rows.Scan(&state, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "unreachable-state",
					Severity: Error,
					Message:  fmt.Sprintf("state %q in %s is unreachable", state, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "duplicate-transition",
		severity: Error,
		query: `SELECT ` + ownerCase + ` AS owner_name, t.on_event, t.from_state, COUNT(*) AS cnt
		        FROM transitions t
		        GROUP BY t.owner_type, t.owner_id, t.on_event, t.from_state
		        HAVING cnt > 1`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var owner sql.NullString
				var onEvent string
				var fromState sql.NullString
				var cnt int64
				if err := rows.Scan(&owner, &onEvent, &fromState, &cnt); err != nil {
					return nil, err
				}
				from := "*"
				if fromState.Valid {
					from = fromState.String
				}
				findings = append(findings, Finding{
					Rule:     "duplicate-transition",
					Severity: Error,
					Message:  fmt.Sprintf("%dx %q from %q in %s", cnt, onEvent, from, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "nesting-depth",
		severity: Warning,
		query: `SELECT r1.name
		        FROM regions r1
		        JOIN regions r2 ON r1.parent_type = 'region' AND r1.parent_id = r2.id
		        WHERE r2.parent_type = 'region'`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "nesting-depth",
					Severity: Warning,
					Message:  fmt.Sprintf("region %q is nested 3+ levels deep", name),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "invalid-flow-ref",
		severity: Error,
		// [F7] Check screen, region, AND event references in flow steps
		query: `SELECT fs.name, f.name AS flow_name, fs.type
		        FROM flow_steps fs
		        JOIN flows f ON f.id = fs.flow_id
		        WHERE (fs.type = 'screen' AND fs.name NOT IN (SELECT s.name FROM screens s))
		           OR (fs.type = 'region' AND fs.name NOT IN (SELECT r.name FROM regions r))
		           OR (fs.type = 'event'  AND fs.name NOT IN (SELECT e.name FROM events e))`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var stepName, flowName, stepType string
				if err := rows.Scan(&stepName, &flowName, &stepType); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "invalid-flow-ref",
					Severity: Error,
					Message:  fmt.Sprintf("flow %q references unknown %s %q", flowName, stepType, stepName),
				})
			}
			return findings, nil
		},
	},
	{
		id:       "orphan-event",
		severity: Warning,
		query: `SELECT DISTINCT t.on_event, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.on_event NOT IN (SELECT e.name FROM events e)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var onEvent string
				var owner sql.NullString
				if err := rows.Scan(&onEvent, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "orphan-event",
					Severity: Warning,
					Message:  fmt.Sprintf("%s handles %q but no region emits it", ns(owner), onEvent),
				})
			}
			return findings, nil
		},
	},
	// [F3] Dangling navigate() targets — action references a screen/region that doesn't exist
	{
		id:       "dangling-navigate",
		severity: Error,
		query: `SELECT t.action, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.action LIKE 'navigate(%)'
		          AND SUBSTR(t.action, 10, LENGTH(t.action) - 10) NOT IN (
		            SELECT s.name FROM screens s
		            UNION ALL
		            SELECT r.name FROM regions r
		          )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var action string
				var owner sql.NullString
				if err := rows.Scan(&action, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "dangling-navigate",
					Severity: Error,
					Message:  fmt.Sprintf("%s in %s targets unknown screen/region", action, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// Ambiguous region names — same name used by multiple regions (requires --in to disambiguate)
	{
		id:       "ambiguous-region-name",
		severity: Warning,
		query:    `SELECT name, COUNT(*) AS cnt FROM regions GROUP BY name HAVING cnt > 1`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				var cnt int
				if err := rows.Scan(&name, &cnt); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "ambiguous-region-name",
					Severity: Warning,
					Message:  fmt.Sprintf("region %q appears %dx — use --in to disambiguate", name, cnt),
				})
			}
			return findings, nil
		},
	},
	// [F10] Unhandled events — events emitted by regions but no transition handles them
	{
		id:       "unhandled-event",
		severity: Warning,
		query: `SELECT e.name, r.name AS region_name
		        FROM events e
		        JOIN regions r ON r.id = e.region_id
		        WHERE e.name NOT IN (SELECT t.on_event FROM transitions t)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var evName, regName string
				if err := rows.Scan(&evName, &regName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "unhandled-event",
					Severity: Warning,
					Message:  fmt.Sprintf("event %q emitted by %s has no handler", evName, regName),
				})
			}
			return findings, nil
		},
	},
}

func ns(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return "?"
}

func Validate(db *sql.DB) ([]Finding, error) {
	var all []Finding
	for _, r := range rules {
		rows, err := db.Query(r.query)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.id, err)
		}
		findings, err := r.format(rows)
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.id, err)
		}
		all = append(all, findings...)
	}
	return all, nil
}
