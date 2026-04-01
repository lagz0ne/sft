package validator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	fn       func(db *sql.DB) ([]Finding, error)
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
		id:       "orphan-event",
		severity: Warning,
		query: `SELECT DISTINCT t.on_event, ` + ownerCase + ` AS owner_name
		        FROM transitions t
		        WHERE t.on_event NOT IN (SELECT e.name FROM events e)
		          AND t.on_event NOT IN (
		            SELECT CASE
		              WHEN INSTR(SUBSTR(t2.action, 6), ',') > 0
		              THEN SUBSTR(t2.action, 6, INSTR(SUBSTR(t2.action, 6), ',') - 1)
		              ELSE SUBSTR(t2.action, 6, INSTR(SUBSTR(t2.action, 6), ')') - 1)
		            END
		            FROM transitions t2 WHERE t2.action LIKE 'emit(%)'
		          )
		          AND t.on_event NOT IN (
		            SELECT CASE
		              WHEN INSTR(SUBSTR(t3.action, INSTR(t3.action, 'emit(') + 5), ',') > 0
		              THEN SUBSTR(t3.action, INSTR(t3.action, 'emit(') + 5, INSTR(SUBSTR(t3.action, INSTR(t3.action, 'emit(') + 5), ',') - 1)
		              ELSE SUBSTR(t3.action, INSTR(t3.action, 'emit(') + 5, INSTR(SUBSTR(t3.action, INSTR(t3.action, 'emit(') + 5), ')') - 1)
		            END
		            FROM transitions t3 WHERE t3.action LIKE '%emit(%'
		          )`,
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
		          )
		          AND NOT EXISTS (
		            SELECT 1 FROM state_fixtures sf
		            WHERE sf.owner_type = t1.owner_type AND sf.owner_id = t1.owner_id
		              AND sf.state_name = t1.to_state
		          )
		          AND NOT EXISTS (
		            SELECT 1 FROM state_regions sr
		            WHERE sr.owner_type = t1.owner_type AND sr.owner_id = t1.owner_id
		              AND sr.state_name = t1.to_state
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
	// Entry screen missing — no screen has entry=1
	{
		id:       "entry-screen-missing",
		severity: Warning,
		query:    `SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM screens WHERE entry = 1)`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var dummy int
				if err := rows.Scan(&dummy); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Message: "no entry screen defined (set entry: true on one screen)",
				})
			}
			return findings, nil
		},
	},
	// Entry screen multiple — more than one screen has entry=1
	{
		id:       "entry-screen-multiple",
		severity: Error,
		query:    `SELECT name FROM screens WHERE entry = 1`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var names []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				names = append(names, name)
			}
			if len(names) > 1 {
				return []Finding{{
					Message: fmt.Sprintf("multiple entry screens: %s", strings.Join(names, ", ")),
				}}, nil
			}
			return nil, nil
		},
	},
	// Leaf region with no component — leaf regions (no child regions) should have a component
	{
		id:       "leaf-region-no-content",
		severity: Warning,
		query: `SELECT r.name,
		          CASE r.parent_type
		            WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = r.parent_id)
		            WHEN 'region' THEN (SELECT r2.name FROM regions r2 WHERE r2.id = r.parent_id)
		            WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = r.parent_id)
		          END AS parent
		        FROM regions r
		        WHERE r.id NOT IN (SELECT r2.parent_id FROM regions r2 WHERE r2.parent_type = 'region')
		          AND r.id NOT IN (SELECT c.entity_id FROM components c WHERE c.entity_type = 'region')`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				var parent sql.NullString
				if err := rows.Scan(&name, &parent); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Message: fmt.Sprintf("leaf region %q in %s has no component", name, ns(parent)),
				})
			}
			return findings, nil
		},
	},
	// Unreferenced data type — data types not used in any context, region_data, or event annotation
	{
		id:       "unreferenced-data-type",
		severity: Warning,
		query: `SELECT dt.name FROM data_types dt
		        WHERE dt.name NOT IN (
		          SELECT REPLACE(REPLACE(c.field_type, '?', ''), '[]', '') FROM contexts c
		          UNION SELECT REPLACE(REPLACE(rd.field_type, '?', ''), '[]', '') FROM region_data rd
		          UNION SELECT REPLACE(REPLACE(e.annotation, '?', ''), '[]', '') FROM events e WHERE e.annotation IS NOT NULL
		        )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Message: fmt.Sprintf("data type %q is not referenced by any context, region_data, or event annotation", name),
				})
			}
			return findings, nil
		},
	},
	// State-region without fixture — states with state_regions but no state_fixtures
	{
		id:       "state-region-no-fixture",
		severity: Warning,
		query: `SELECT DISTINCT sr.state_name,
		          CASE sr.owner_type
		            WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = sr.owner_id)
		            WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = sr.owner_id)
		            WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = sr.owner_id)
		          END AS owner
		        FROM state_regions sr
		        WHERE NOT EXISTS (
		          SELECT 1 FROM state_fixtures sf
		          WHERE sf.owner_type = sr.owner_type AND sf.owner_id = sr.owner_id AND sf.state_name = sr.state_name
		        )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var stateName string
				var owner sql.NullString
				if err := rows.Scan(&stateName, &owner); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Message: fmt.Sprintf("state %q in %s has state_regions but no fixture", stateName, ns(owner)),
				})
			}
			return findings, nil
		},
	},
	// State without fixture (screen-level) — screen states with no fixture
	{
		id:       "state-without-fixture",
		severity: Warning,
		query: `SELECT DISTINCT sub.state_name, sub.screen_name FROM (
		          SELECT t.from_state AS state_name,
		            (SELECT s.name FROM screens s WHERE s.id = t.owner_id) AS screen_name,
		            t.owner_type, t.owner_id
		          FROM transitions t
		          WHERE t.owner_type = 'screen' AND t.from_state IS NOT NULL AND t.from_state != ''
		          UNION
		          SELECT t.to_state AS state_name,
		            (SELECT s.name FROM screens s WHERE s.id = t.owner_id) AS screen_name,
		            t.owner_type, t.owner_id
		          FROM transitions t
		          WHERE t.owner_type = 'screen' AND t.to_state IS NOT NULL AND t.to_state != ''
		        ) sub
		        WHERE NOT EXISTS (
		          SELECT 1 FROM state_fixtures sf
		          WHERE sf.owner_type = sub.owner_type AND sf.owner_id = sub.owner_id AND sf.state_name = sub.state_name
		        )`,
		format: func(rows *sql.Rows) ([]Finding, error) {
			var findings []Finding
			for rows.Next() {
				var stateName, screenName string
				if err := rows.Scan(&stateName, &screenName); err != nil {
					return nil, err
				}
				findings = append(findings, Finding{
					Message: fmt.Sprintf("screen %q state %q has no fixture", screenName, stateName),
				})
			}
			return findings, nil
		},
	},
	// Fixture extends cycle — circular extends chain in fixtures (Go-level DFS)
	{
		id:       "fixture-extends-cycle",
		severity: Error,
		fn: func(db *sql.DB) ([]Finding, error) {
			rows, err := db.Query("SELECT name, extends FROM fixtures WHERE extends IS NOT NULL AND extends != ''")
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			graph := map[string]string{}
			for rows.Next() {
				var name, ext string
				if err := rows.Scan(&name, &ext); err != nil {
					return nil, err
				}
				graph[name] = ext
			}
			var findings []Finding
			for start := range graph {
				visited := map[string]bool{}
				cur := start
				for cur != "" {
					if visited[cur] {
						findings = append(findings, Finding{
							Message: fmt.Sprintf("fixture %q has circular extends chain", start),
						})
						break
					}
					visited[cur] = true
					cur = graph[cur]
				}
			}
			return findings, nil
		},
	},
	// Screen unreachable — screens not reachable from entry screen via navigate() actions (Go-level BFS)
	{
		id:       "screen-unreachable",
		severity: Warning,
		fn: func(db *sql.DB) ([]Finding, error) {
			// 1. Get entry screen name
			var entryName string
			err := db.QueryRow("SELECT name FROM screens WHERE entry = 1").Scan(&entryName)
			if err != nil {
				// No entry screen — entry-screen-missing rule handles that
				return nil, nil
			}

			// 2. Get all screen names
			srows, err := db.Query("SELECT name FROM screens")
			if err != nil {
				return nil, err
			}
			allScreens := map[string]bool{}
			for srows.Next() {
				var name string
				if err := srows.Scan(&name); err != nil {
					srows.Close()
					return nil, err
				}
				allScreens[name] = true
			}
			srows.Close()

			// 3. Build adjacency: screen -> navigate() targets
			trows, err := db.Query("SELECT action, owner_type, owner_id FROM transitions WHERE action LIKE 'navigate(%'")
			if err != nil {
				return nil, err
			}
			// Map owner (screen) to navigate targets
			edges := map[string][]string{}
			for trows.Next() {
				var action, ownerType string
				var ownerID int64
				if err := trows.Scan(&action, &ownerType, &ownerID); err != nil {
					trows.Close()
					return nil, err
				}
				target := parseNavigateTarget(action)
				if target == "" {
					continue
				}
				// Resolve owner to screen name (could be nested region)
				var ownerScreen string
				if ownerType == "screen" {
					db.QueryRow("SELECT name FROM screens WHERE id = ?", ownerID).Scan(&ownerScreen)
				} else {
					// Walk up region tree to find owning screen
					ownerScreen = resolveOwnerScreen(db, ownerType, ownerID)
				}
				if ownerScreen != "" {
					edges[ownerScreen] = append(edges[ownerScreen], target)
				}
			}
			trows.Close()

			// 4. BFS from entry screen
			visited := map[string]bool{entryName: true}
			queue := []string{entryName}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				for _, next := range edges[cur] {
					if !visited[next] && allScreens[next] {
						visited[next] = true
						queue = append(queue, next)
					}
				}
			}

			// 5. Report unreachable
			var findings []Finding
			for name := range allScreens {
				if !visited[name] {
					findings = append(findings, Finding{
						Message: fmt.Sprintf("screen %q is not reachable from entry screen %q", name, entryName),
					})
				}
			}
			return findings, nil
		},
	},
	// Component prop unknown — component props contain keys not declared in schema
	{
		id:       "component-prop-unknown",
		severity: Warning,
		fn: func(db *sql.DB) ([]Finding, error) {
			// 1. Load schemas
			srows, err := db.Query("SELECT name, props FROM component_schemas")
			if err != nil {
				return nil, err
			}
			schemas := map[string]map[string]string{} // schema name -> prop name -> type
			for srows.Next() {
				var name, propsJSON string
				if err := srows.Scan(&name, &propsJSON); err != nil {
					srows.Close()
					return nil, err
				}
				var props map[string]string
				if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
					srows.Close()
					return nil, err
				}
				schemas[name] = props
			}
			srows.Close()
			if len(schemas) == 0 {
				return nil, nil
			}

			// 2. Check component props against schemas
			crows, err := db.Query("SELECT entity_type, entity_id, component, props FROM components")
			if err != nil {
				return nil, err
			}
			defer crows.Close()
			var findings []Finding
			for crows.Next() {
				var entType string
				var entID int64
				var comp, propsJSON string
				if err := crows.Scan(&entType, &entID, &comp, &propsJSON); err != nil {
					return nil, err
				}
				schema, ok := schemas[comp]
				if !ok {
					continue
				}
				var props map[string]any
				if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
					continue
				}
				for key := range props {
					if _, declared := schema[key]; !declared {
						findings = append(findings, Finding{
							Message: fmt.Sprintf("component %q on %s has unknown prop %q", comp, entType, key),
						})
					}
				}
			}
			return findings, nil
		},
	},
	// Fixture keys mismatch — fixture data keys don't match context fields
	{
		id:       "fixture-keys-mismatch",
		severity: Warning,
		fn: func(db *sql.DB) ([]Finding, error) {
			// 1. Get screen-level state_fixture bindings
			rows, err := db.Query(`SELECT sf.fixture_name, sf.state_name, s.name AS screen_name, sf.owner_id
				FROM state_fixtures sf
				JOIN screens s ON sf.owner_type = 'screen' AND sf.owner_id = s.id`)
			if err != nil {
				return nil, err
			}
			type binding struct {
				fixtureName string
				stateName   string
				screenName  string
				ownerID     int64
			}
			var bindings []binding
			for rows.Next() {
				var b binding
				if err := rows.Scan(&b.fixtureName, &b.stateName, &b.screenName, &b.ownerID); err != nil {
					rows.Close()
					return nil, err
				}
				bindings = append(bindings, b)
			}
			rows.Close()

			// 2. Load fixtures
			frows, err := db.Query("SELECT name, data FROM fixtures")
			if err != nil {
				return nil, err
			}
			fixtures := map[string]string{} // name -> data JSON
			for frows.Next() {
				var name, data string
				if err := frows.Scan(&name, &data); err != nil {
					frows.Close()
					return nil, err
				}
				fixtures[name] = data
			}
			frows.Close()

			var findings []Finding
			for _, b := range bindings {
				data, ok := fixtures[b.fixtureName]
				if !ok {
					continue // fixture-not-found handles this
				}
				// Parse fixture data
				var fixtureData map[string]any
				if err := json.Unmarshal([]byte(data), &fixtureData); err != nil {
					continue
				}
				// Get the screen section from fixture data
				screenData, ok := fixtureData[b.screenName]
				if !ok {
					continue // fixture may not have a section for this screen
				}
				screenMap, ok := screenData.(map[string]any)
				if !ok {
					continue
				}
				// Get context fields for this screen
				cfrows, err := db.Query("SELECT field_name FROM contexts WHERE owner_type='screen' AND owner_id=?", b.ownerID)
				if err != nil {
					return nil, err
				}
				contextFields := map[string]bool{}
				for cfrows.Next() {
					var fieldName string
					if err := cfrows.Scan(&fieldName); err != nil {
						cfrows.Close()
						return nil, err
					}
					contextFields[fieldName] = true
				}
				cfrows.Close()
				if len(contextFields) == 0 {
					continue
				}
				// Check each fixture key
				for key := range screenMap {
					if !contextFields[key] {
						findings = append(findings, Finding{
							Message: fmt.Sprintf("fixture %q for screen %q state %q has key %q not in context fields", b.fixtureName, b.screenName, b.stateName, key),
						})
					}
				}
			}
			return findings, nil
		},
	},
	// Entity type mismatch — entity type references a non-existent data type
	{
		id:       "entity-type-mismatch",
		severity: Warning,
		fn: func(db *sql.DB) ([]Finding, error) {
			// 1. Get all data type names
			dtrows, err := db.Query("SELECT name FROM data_types")
			if err != nil {
				return nil, err
			}
			dataTypes := map[string]bool{}
			for dtrows.Next() {
				var name string
				if err := dtrows.Scan(&name); err != nil {
					dtrows.Close()
					return nil, err
				}
				dataTypes[name] = true
			}
			dtrows.Close()

			// 2. Check entities
			erows, err := db.Query("SELECT name, type FROM entities")
			if err != nil {
				return nil, err
			}
			defer erows.Close()
			var findings []Finding
			for erows.Next() {
				var name, typ string
				if err := erows.Scan(&name, &typ); err != nil {
					return nil, err
				}
				if !dataTypes[typ] {
					findings = append(findings, Finding{
						Message: fmt.Sprintf("entity %q has type %q which is not a declared data type", name, typ),
					})
				}
			}
			return findings, nil
		},
	},
	// Experiment scope invalid — scope does not resolve to an existing screen/region
	{
		id:       "experiment-scope-invalid",
		severity: Warning,
		fn: func(db *sql.DB) ([]Finding, error) {
			rows, err := db.Query("SELECT name, scope FROM experiments WHERE status = 'active'")
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var findings []Finding
			for rows.Next() {
				var name, scope string
				if err := rows.Scan(&name, &scope); err != nil {
					return nil, err
				}
				parts := strings.SplitN(scope, ".", 2)
				screenName := parts[0]
				// Check screen exists
				var screenID int64
				err := db.QueryRow("SELECT id FROM screens WHERE name = ?", screenName).Scan(&screenID)
				if err != nil {
					findings = append(findings, Finding{
						Message: fmt.Sprintf("experiment %q scope %q references unknown screen %q", name, scope, screenName),
					})
					continue
				}
				// If region specified, check it exists under that screen
				if len(parts) == 2 {
					regionName := parts[1]
					var regionID int64
					err := db.QueryRow("SELECT id FROM regions WHERE name = ? AND parent_type = 'screen' AND parent_id = ?", regionName, screenID).Scan(&regionID)
					if err != nil {
						findings = append(findings, Finding{
							Message: fmt.Sprintf("experiment %q scope %q references unknown region %q in screen %q", name, scope, regionName, screenName),
						})
					}
				}
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

// parseNavigateTarget extracts the screen name from navigate(target) or navigate(target, ...).
func parseNavigateTarget(action string) string {
	if !strings.HasPrefix(action, "navigate(") {
		return ""
	}
	inner := action[9:] // after "navigate("
	// Find end: either comma or closing paren
	end := strings.IndexAny(inner, ",)")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(inner[:end])
}

// resolveOwnerScreen walks up the region tree to find the owning screen name.
func resolveOwnerScreen(db *sql.DB, ownerType string, ownerID int64) string {
	for ownerType == "region" {
		var pType string
		var pID int64
		err := db.QueryRow("SELECT parent_type, parent_id FROM regions WHERE id = ?", ownerID).Scan(&pType, &pID)
		if err != nil {
			return ""
		}
		ownerType = pType
		ownerID = pID
	}
	if ownerType == "screen" {
		var name string
		db.QueryRow("SELECT name FROM screens WHERE id = ?", ownerID).Scan(&name)
		return name
	}
	return ""
}

func Validate(db *sql.DB) ([]Finding, error) {
	var all []Finding
	for _, r := range rules {
		var findings []Finding
		var err error
		if r.fn != nil {
			findings, err = r.fn(db)
		} else {
			rows, qerr := db.Query(r.query)
			if qerr != nil {
				return nil, fmt.Errorf("rule %s: %w", r.id, qerr)
			}
			findings, err = r.format(rows)
			rows.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.id, err)
		}
		for i := range findings {
			findings[i].Rule = r.id
			findings[i].Severity = r.severity
		}
		all = append(all, findings...)
	}
	return all, nil
}
