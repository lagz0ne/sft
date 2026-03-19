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
// Uses table alias "t" — for other aliases, use ownerCaseAlias.
const ownerCase = `CASE t.owner_type
  WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t.owner_id)
  WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = t.owner_id)
  WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = t.owner_id)
END`

// ownerCaseAlias is like ownerCase but parameterized on the table alias.
func ownerCaseAlias(alias string) string {
	return fmt.Sprintf(`CASE %s.owner_type
  WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = %s.owner_id)
  WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = %s.owner_id)
  WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = %s.owner_id)
END`, alias, alias, alias, alias)
}

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
		          AND SUBSTR(t.action, 6,
		            CASE WHEN INSTR(SUBSTR(t.action, 6), ',') > 0
		            THEN INSTR(SUBSTR(t.action, 6), ',') - 1
		            ELSE INSTR(SUBSTR(t.action, 6), ')') - 1 END) NOT IN (
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
		query: `SELECT DISTINCT t1.from_state, ` + ownerCaseAlias("t1") + ` AS owner_name
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
	// Dead-end states — transition targets with no outgoing transitions (terminal)
	{
		id:       "dead-end",
		severity: Warning,
		query: `SELECT DISTINCT t1.to_state, ` + ownerCaseAlias("t1") + ` AS owner_name
		        FROM transitions t1
		        WHERE t1.to_state IS NOT NULL AND t1.to_state != ''
		          AND t1.to_state NOT IN (
		            SELECT t2.from_state FROM transitions t2
		            WHERE t2.owner_type = t1.owner_type AND t2.owner_id = t1.owner_id
		              AND t2.from_state IS NOT NULL AND t2.from_state != ''
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
					Rule:     "dead-end",
					Severity: Warning,
					Message:  fmt.Sprintf("state %q in %s has no outgoing transitions (terminal)", state, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// Guard ambiguity — same event+from_state appears 2+ times without guard distinction
	{
		id:       "guard-ambiguity",
		severity: Warning,
		query: `SELECT ` + ownerCase + ` AS owner_name, t.on_event, t.from_state, COUNT(*) AS cnt
		        FROM transitions t
		        WHERE t.from_state IS NOT NULL AND t.from_state != ''
		          AND (t.action IS NULL OR t.action = '' OR t.action NOT LIKE 'guard(%)')
		        GROUP BY t.owner_type, t.owner_id, t.on_event, t.from_state
		        HAVING cnt > 1`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var owner sql.NullString
				var onEvent, fromState string
				var cnt int64
				if err := rows.Scan(&owner, &onEvent, &fromState, &cnt); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "guard-ambiguity",
					Severity: Warning,
					Message:  fmt.Sprintf("%dx %q from %q in %s without guard", cnt, onEvent, fromState, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// [F3] Dangling navigate() targets — action references a screen/region that doesn't exist
	// Handles both navigate(target) and navigate(target, {params})
	{
		id:       "dangling-navigate",
		severity: Error,
		query: `SELECT t.action, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.action LIKE 'navigate(%)'
		          AND TRIM(CASE
		            WHEN INSTR(SUBSTR(t.action, 10), ',') > 0
		            THEN SUBSTR(t.action, 10, INSTR(SUBSTR(t.action, 10), ',') - 1)
		            ELSE SUBSTR(t.action, 10, LENGTH(t.action) - 10)
		          END) NOT IN (
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
	// Undefined data type — context or region_data field_type references a type not in data_types
	{
		id:       "undefined-data-type",
		severity: Error,
		query: `SELECT 'context' AS source, c.field_name, c.field_type,
		          CASE c.owner_type
		            WHEN 'app' THEN (SELECT a.name FROM apps a WHERE a.id = c.owner_id)
		            WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = c.owner_id)
		          END AS owner_name
		        FROM contexts c
		        WHERE REPLACE(REPLACE(c.field_type, '?', ''), '[]', '') NOT IN ('string', 'number', 'boolean', 'datetime', 'date')
		          AND REPLACE(REPLACE(c.field_type, '?', ''), '[]', '') NOT IN (SELECT dt.name FROM data_types dt)
		          AND REPLACE(REPLACE(c.field_type, '?', ''), '[]', '') NOT IN (SELECT e.name FROM enums e)
		        UNION ALL
		        SELECT 'region_data' AS source, rd.field_name, rd.field_type,
		          (SELECT r.name FROM regions r WHERE r.id = rd.region_id) AS owner_name
		        FROM region_data rd
		        WHERE REPLACE(REPLACE(rd.field_type, '?', ''), '[]', '') NOT IN ('string', 'number', 'boolean', 'datetime', 'date')
		          AND REPLACE(REPLACE(rd.field_type, '?', ''), '[]', '') NOT IN (SELECT dt.name FROM data_types dt)
		          AND REPLACE(REPLACE(rd.field_type, '?', ''), '[]', '') NOT IN (SELECT e.name FROM enums e)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var source, fieldName, fieldType string
				var ownerName sql.NullString
				if err := rows.Scan(&source, &fieldName, &fieldType, &ownerName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "undefined-data-type",
					Severity: Error,
					Message:  fmt.Sprintf("%s field %q has undefined type %q in %s", source, fieldName, fieldType, ns(ownerName)),
				})
			}
			return findings, nil
		},
	},
	// Fixture not found — state_fixtures references a fixture name that doesn't exist
	{
		id:       "fixture-not-found",
		severity: Error,
		query: `SELECT sf.fixture_name, sf.state_name,
		          CASE sf.owner_type
		            WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = sf.owner_id)
		            WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = sf.owner_id)
		            WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = sf.owner_id)
		          END AS owner_name
		        FROM state_fixtures sf
		        WHERE sf.fixture_name NOT IN (SELECT f.name FROM fixtures f)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var fixtureName, stateName string
				var ownerName sql.NullString
				if err := rows.Scan(&fixtureName, &stateName, &ownerName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "fixture-not-found",
					Severity: Error,
					Message:  fmt.Sprintf("state %q in %s references undefined fixture %q", stateName, ns(ownerName), fixtureName),
				})
			}
			return findings, nil
		},
	},
	// Orphan fixture — fixture not referenced by any state or extends
	{
		id:       "orphan-fixture",
		severity: Warning,
		query: `SELECT f.name
		        FROM fixtures f
		        WHERE f.name NOT IN (SELECT sf.fixture_name FROM state_fixtures sf)
		          AND f.name NOT IN (SELECT f2.extends FROM fixtures f2 WHERE f2.extends IS NOT NULL)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "orphan-fixture",
					Severity: Warning,
					Message:  fmt.Sprintf("fixture %q is not referenced by any state", name),
				})
			}
			return findings, nil
		},
	},
	// Invalid ambient path — ambient ref source must be "app" or a valid screen name, query must start with "."
	{
		id:       "invalid-ambient-path",
		severity: Error,
		query: `SELECT ar.local_name, ar.source, ar.query,
		          (SELECT r.name FROM regions r WHERE r.id = ar.region_id) AS region_name,
		          'bad-source' AS reason
		        FROM ambient_refs ar
		        WHERE ar.source != 'app'
		          AND ar.source NOT IN (SELECT s.name FROM screens s)
		        UNION ALL
		        SELECT ar.local_name, ar.source, ar.query,
		          (SELECT r.name FROM regions r WHERE r.id = ar.region_id) AS region_name,
		          'bad-query' AS reason
		        FROM ambient_refs ar
		        WHERE ar.query NOT LIKE '.%'`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var localName, source, query string
				var regionName sql.NullString
				var reason string
				if err := rows.Scan(&localName, &source, &query, &regionName, &reason); err != nil {
					return nil, err
				}
				var msg string
				if reason == "bad-source" {
					msg = fmt.Sprintf("ambient ref %q in %s has invalid source %q (not 'app' or a screen name)", localName, ns(regionName), source)
				} else {
					msg = fmt.Sprintf("ambient ref %q in %s has query %q that doesn't start with '.'", localName, ns(regionName), query)
				}
				findings = append(findings, Finding{
					Rule:     "invalid-ambient-path",
					Severity: Error,
					Message:  msg,
				})
			}
			return findings, nil
		},
	},
	// Invalid event annotation — annotation type not a builtin or defined data type
	{
		id:       "invalid-event-annotation",
		severity: Warning,
		query: `SELECT e.name, e.annotation, (SELECT r.name FROM regions r WHERE r.id = e.region_id) AS region_name
		        FROM events e
		        WHERE e.annotation IS NOT NULL AND e.annotation != ''
		          AND REPLACE(REPLACE(e.annotation, '?', ''), '[]', '') NOT IN ('string', 'number', 'boolean', 'datetime', 'date')
		          AND REPLACE(REPLACE(e.annotation, '?', ''), '[]', '') NOT IN (SELECT dt.name FROM data_types dt)
		          AND REPLACE(REPLACE(e.annotation, '?', ''), '[]', '') NOT IN (SELECT en.name FROM enums en)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var evName, annotation string
				var regionName sql.NullString
				if err := rows.Scan(&evName, &annotation, &regionName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "invalid-event-annotation",
					Severity: Warning,
					Message:  fmt.Sprintf("event %q in %s has unknown annotation type %q", evName, ns(regionName), annotation),
				})
			}
			return findings, nil
		},
	},
	// emit() without target: — may be intentional but worth flagging
	{
		id:       "emit-missing-target",
		severity: Warning,
		query: `SELECT t.action, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.action LIKE 'emit(%)'
		          AND t.action NOT LIKE '%target:%'`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var action string
				var owner sql.NullString
				if err := rows.Scan(&action, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "emit-missing-target",
					Severity: Warning,
					Message:  fmt.Sprintf("%s in %s has no target: specifier", action, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// Invalid state-region — state references a region that doesn't exist in the owner's children
	{
		id:       "invalid-state-region",
		severity: Error,
		query: `SELECT sr.state_name, sr.region_name, ` + ownerCaseAlias("sr") + ` AS owner_name
		        FROM state_regions sr
		        WHERE sr.region_name NOT IN (
		          SELECT r.name FROM regions r WHERE r.parent_type = sr.owner_type AND r.parent_id = sr.owner_id
		        )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var stateName, regionName string
				var ownerName sql.NullString
				if err := rows.Scan(&stateName, &regionName, &ownerName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "invalid-state-region",
					Severity: Error,
					Message:  fmt.Sprintf("state %q in %s references non-child region %q", stateName, ns(ownerName), regionName),
				})
			}
			return findings, nil
		},
	},
	// Enum-data collision — enum name collides with a data type name
	{
		id:       "enum-data-collision",
		severity: Warning,
		query: `SELECT e.name
		        FROM enums e
		        WHERE e.name IN (SELECT dt.name FROM data_types dt WHERE dt.app_id = e.app_id)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Rule:     "enum-data-collision",
					Severity: Warning,
					Message:  fmt.Sprintf("enum %q has the same name as a data type", name),
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
