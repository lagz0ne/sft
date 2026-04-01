package show

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/lagz0ne/sft/internal/store"
)

// --- Spec tree types (nested, not flat tables) ---

type Spec struct {
	App         App                 `json:"app"`
	Screens     []Screen            `json:"screens"`
	Fixtures    []Fixture           `json:"fixtures,omitempty"`
	Layouts     map[string][]string `json:"layouts,omitempty"`
	Entities    []Entity            `json:"entities,omitempty"`
	Experiments []Experiment        `json:"experiments,omitempty"`
}

type Entity struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Experiment struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Overlay     any    `json:"overlay,omitempty"`
	Status      string `json:"status"`
}

type Fixture struct {
	Name    string      `json:"name"`
	Extends string      `json:"extends,omitempty"`
	Data    any `json:"data"`
}

type App struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	DataTypes   map[string]map[string]string `json:"data_types,omitempty"`
	Enums       map[string][]string          `json:"enums,omitempty"`
	Context     map[string]string            `json:"context,omitempty"`
	Regions     []Region                     `json:"regions,omitempty"`
	Transitions []Transition                 `json:"transitions,omitempty"`
}

type Screen struct {
	ID             int64               `json:"id"`
	Ref            string              `json:"ref"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Entry          bool                `json:"entry,omitempty"`
	Tags           []string            `json:"tags,omitempty"`
	Context        map[string]string   `json:"context,omitempty"`
	Component      string              `json:"component,omitempty"`
	ComponentProps string              `json:"component_props,omitempty"` // [F5]
	ComponentOn    string              `json:"component_on,omitempty"`    // [F5]
	ComponentVis   string              `json:"component_visible,omitempty"` // [F5]
	Regions        []Region            `json:"regions,omitempty"`
	Transitions    []Transition        `json:"transitions,omitempty"`
	States         []string            `json:"states,omitempty"`
	StateFixtures  map[string]string   `json:"state_fixtures,omitempty"`
	StateRegions   map[string][]string `json:"state_regions,omitempty"`
	Attachments    []string            `json:"attachments,omitempty"`
}

type Region struct {
	ID                int64               `json:"id"`
	Ref               string              `json:"ref"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Tags              []string            `json:"tags,omitempty"`
	Component         string              `json:"component,omitempty"`
	ComponentProps    string              `json:"component_props,omitempty"` // [F5]
	ComponentOn       string              `json:"component_on,omitempty"`    // [F5]
	ComponentVis      string              `json:"component_visible,omitempty"` // [F5]
	DiscoveryLayout   []string            `json:"discovery_layout,omitempty"`
	DeliveryClasses   []string            `json:"delivery_classes,omitempty"`
	DeliveryComponent string              `json:"delivery_component,omitempty"`
	Events            []string            `json:"events,omitempty"`
	Ambient           map[string]string   `json:"ambient,omitempty"`
	RegionData        map[string]string   `json:"region_data,omitempty"`
	Regions           []Region            `json:"regions,omitempty"`
	Transitions       []Transition        `json:"transitions,omitempty"`
	States            []string            `json:"states,omitempty"`
	StateFixtures     map[string]string   `json:"state_fixtures,omitempty"`
	StateRegions      map[string][]string `json:"state_regions,omitempty"`
	Attachments       []string            `json:"attachments,omitempty"`
}

type Transition struct {
	OnEvent   string `json:"on_event"`
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`
	Action    string `json:"action,omitempty"`
}

// Enricher provides attachment and component data during spec loading.
type Enricher interface {
	AttachmentsFor(entity string) []string
	ComponentFor(entityType string, entityID int64) string                    // type or ""
	ComponentInfoFor(entityType string, entityID int64) *store.ComponentInfo  // full details
}

// --- Load from DB ---

func Load(db *sql.DB, al Enricher) (*Spec, error) {
	spec := &Spec{}

	// App
	var appID int64
	row := db.QueryRow("SELECT id, name, description FROM apps LIMIT 1")
	if err := row.Scan(&appID, &spec.App.Name, &spec.App.Description); err != nil {
		return nil, fmt.Errorf("no app found: %w", err)
	}
	var err error
	if spec.App.DataTypes, err = loadDataTypes(db, appID); err != nil {
		return nil, err
	}
	if spec.App.Enums, err = loadEnums(db, appID); err != nil {
		return nil, err
	}
	spec.App.Context = loadContext(db, "app", appID)
	spec.App.Regions = loadRegions(db, "app", appID, al)
	spec.App.Transitions = loadTransitions(db, "app", appID)

	// Screens
	rows, err := db.Query("SELECT id, name, description, entry FROM screens ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var s Screen
		var entry int
		if err := rows.Scan(&id, &s.Name, &s.Description, &entry); err != nil {
			return nil, fmt.Errorf("scan screen: %w", err)
		}
		s.ID = id
		s.Entry = entry != 0
		s.Ref = fmt.Sprintf("@s%d", id)
		s.Tags = loadTags(db, "screen", id)
		s.Context = loadContext(db, "screen", id)
		s.Regions = loadRegions(db, "screen", id, al)
		s.Transitions = loadTransitions(db, "screen", id)
		s.States = deriveStates(s.Transitions)
		s.StateFixtures = loadStateFixtures(db, "screen", id)
		s.StateRegions = loadStateRegions(db, "screen", id)
		if al != nil {
			s.Attachments = al.AttachmentsFor(s.Name)
			s.Component = al.ComponentFor("screen", id)
			// [F5] Full component details
			if ci := al.ComponentInfoFor("screen", id); ci != nil {
				s.ComponentProps = ci.Props
				s.ComponentOn = ci.OnActions
				s.ComponentVis = ci.Visible
			}
		}
		spec.Screens = append(spec.Screens, s)
	}

	// Fixtures
	if spec.Fixtures, err = loadFixtures(db, appID); err != nil {
		return nil, err
	}

	// Layouts
	spec.Layouts = loadLayouts(db, appID)

	// Entities
	if spec.Entities, err = loadEntities(db, appID); err != nil {
		return nil, err
	}

	// Experiments
	if spec.Experiments, err = loadExperiments(db, appID); err != nil {
		return nil, err
	}

	return spec, nil
}

func loadTags(db *sql.DB, entityType string, entityID int64) []string {
	rows, _ := db.Query("SELECT tag FROM tags WHERE entity_type = ? AND entity_id = ?", entityType, entityID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tags = append(tags, t)
	}
	return tags
}

func loadRegions(db *sql.DB, parentType string, parentID int64, al Enricher) []Region {
	rows, _ := db.Query("SELECT id, name, description, discovery_layout, delivery_classes, delivery_component FROM regions WHERE parent_type = ? AND parent_id = ? ORDER BY position, id",
		parentType, parentID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var regions []Region
	for rows.Next() {
		var id int64
		var r Region
		var dlJSON, dcJSON, dcomp sql.NullString
		rows.Scan(&id, &r.Name, &r.Description, &dlJSON, &dcJSON, &dcomp)
		r.ID = id
		r.Ref = fmt.Sprintf("@r%d", id)
		if dlJSON.Valid {
			json.Unmarshal([]byte(dlJSON.String), &r.DiscoveryLayout)
		}
		if dcJSON.Valid {
			json.Unmarshal([]byte(dcJSON.String), &r.DeliveryClasses)
		}
		r.DeliveryComponent = dcomp.String
		r.Tags = loadTags(db, "region", id)
		r.Events = loadEvents(db, id)
		r.Ambient = loadAmbientRefs(db, id)
		r.RegionData = loadRegionData(db, id)
		r.Regions = loadRegions(db, "region", id, al) // recurse
		r.Transitions = loadTransitions(db, "region", id)
		r.States = deriveStates(r.Transitions)
		r.StateFixtures = loadStateFixtures(db, "region", id)
		r.StateRegions = loadStateRegions(db, "region", id)
		if al != nil {
			r.Attachments = al.AttachmentsFor(r.Name)
			r.Component = al.ComponentFor("region", id)
			// [F5] Full component details
			if ci := al.ComponentInfoFor("region", id); ci != nil {
				r.ComponentProps = ci.Props
				r.ComponentOn = ci.OnActions
				r.ComponentVis = ci.Visible
			}
		}
		regions = append(regions, r)
	}
	return regions
}

func loadEvents(db *sql.DB, regionID int64) []string {
	rows, _ := db.Query("SELECT name, annotation FROM events WHERE region_id = ?", regionID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var events []string
	for rows.Next() {
		var name string
		var annotation sql.NullString
		rows.Scan(&name, &annotation)
		if annotation.Valid && annotation.String != "" {
			events = append(events, name+"("+annotation.String+")")
		} else {
			events = append(events, name)
		}
	}
	return events
}

// deriveStates extracts an ordered, deduplicated list of states from transitions.
// The first from_state of the first transition (by rowid) is the initial state.
// Remaining states appear in encounter order (from_state then to_state per transition).
func deriveStates(transitions []Transition) []string {
	if len(transitions) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var states []string
	add := func(s string) {
		if s == "" || s == "." || seen[s] {
			return
		}
		seen[s] = true
		states = append(states, s)
	}
	for _, t := range transitions {
		add(t.FromState)
		add(t.ToState)
	}
	if len(states) == 0 {
		return nil
	}
	return states
}

func loadTransitions(db *sql.DB, ownerType string, ownerID int64) []Transition {
	rows, _ := db.Query("SELECT on_event, from_state, to_state, action FROM transitions WHERE owner_type = ? AND owner_id = ? ORDER BY id",
		ownerType, ownerID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var transitions []Transition
	for rows.Next() {
		var t Transition
		var from, to, action sql.NullString
		rows.Scan(&t.OnEvent, &from, &to, &action)
		t.FromState = from.String
		t.ToState = to.String
		t.Action = action.String
		transitions = append(transitions, t)
	}
	return transitions
}

func loadDataTypes(db *sql.DB, appID int64) (map[string]map[string]string, error) {
	rows, _ := db.Query("SELECT name, fields FROM data_types WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	result := map[string]map[string]string{}
	for rows.Next() {
		var name, fieldsJSON string
		rows.Scan(&name, &fieldsJSON)
		var fields map[string]string
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			return nil, fmt.Errorf("unmarshal fields for data type %s: %w", name, err)
		}
		result[name] = fields
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func loadEnums(db *sql.DB, appID int64) (map[string][]string, error) {
	rows, _ := db.Query(`SELECT name, "values" FROM enums WHERE app_id = ? ORDER BY name`, appID)
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	result := map[string][]string{}
	for rows.Next() {
		var name, valuesJSON string
		rows.Scan(&name, &valuesJSON)
		var values []string
		if err := json.Unmarshal([]byte(valuesJSON), &values); err != nil {
			return nil, fmt.Errorf("unmarshal values for enum %s: %w", name, err)
		}
		result[name] = values
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func loadContext(db *sql.DB, ownerType string, ownerID int64) map[string]string {
	rows, _ := db.Query("SELECT field_name, field_type FROM contexts WHERE owner_type = ? AND owner_id = ? ORDER BY field_name", ownerType, ownerID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var name, typ string
		rows.Scan(&name, &typ)
		result[name] = typ
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadAmbientRefs(db *sql.DB, regionID int64) map[string]string {
	rows, _ := db.Query("SELECT local_name, source, query FROM ambient_refs WHERE region_id = ? ORDER BY local_name", regionID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var name, source, query string
		rows.Scan(&name, &source, &query)
		result[name] = fmt.Sprintf("data(%s, %s)", source, query)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadRegionData(db *sql.DB, regionID int64) map[string]string {
	rows, _ := db.Query("SELECT field_name, field_type FROM region_data WHERE region_id = ? ORDER BY field_name", regionID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var name, typ string
		rows.Scan(&name, &typ)
		result[name] = typ
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadFixtures(db *sql.DB, appID int64) ([]Fixture, error) {
	rows, _ := db.Query("SELECT name, extends, data FROM fixtures WHERE app_id = ? ORDER BY id", appID)
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var fixtures []Fixture
	for rows.Next() {
		var f Fixture
		var extends sql.NullString
		var dataJSON string
		rows.Scan(&f.Name, &extends, &dataJSON)
		f.Extends = extends.String
		if err := json.Unmarshal([]byte(dataJSON), &f.Data); err != nil {
			return nil, fmt.Errorf("unmarshal data for fixture %s: %w", f.Name, err)
		}
		fixtures = append(fixtures, f)
	}
	return fixtures, nil
}

func loadLayouts(db *sql.DB, appID int64) map[string][]string {
	rows, _ := db.Query("SELECT name, classes FROM layouts WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string][]string{}
	for rows.Next() {
		var name, classesJSON string
		rows.Scan(&name, &classesJSON)
		var classes []string
		json.Unmarshal([]byte(classesJSON), &classes)
		result[name] = classes
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadStateRegions(db *sql.DB, ownerType string, ownerID int64) map[string][]string {
	rows, _ := db.Query("SELECT state_name, region_name FROM state_regions WHERE owner_type = ? AND owner_id = ? ORDER BY state_name, id",
		ownerType, ownerID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string][]string{}
	for rows.Next() {
		var stateName, regionName string
		rows.Scan(&stateName, &regionName)
		result[stateName] = append(result[stateName], regionName)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadStateFixtures(db *sql.DB, ownerType string, ownerID int64) map[string]string {
	rows, _ := db.Query("SELECT state_name, fixture_name FROM state_fixtures WHERE owner_type = ? AND owner_id = ? ORDER BY state_name",
		ownerType, ownerID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var stateName, fixtureName string
		rows.Scan(&stateName, &fixtureName)
		result[stateName] = fixtureName
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func loadEntities(db *sql.DB, appID int64) ([]Entity, error) {
	rows, _ := db.Query("SELECT name, type, data FROM entities WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var entities []Entity
	for rows.Next() {
		var e Entity
		var dataJSON string
		rows.Scan(&e.Name, &e.Type, &dataJSON)
		if err := json.Unmarshal([]byte(dataJSON), &e.Data); err != nil {
			return nil, fmt.Errorf("unmarshal data for entity %s: %w", e.Name, err)
		}
		entities = append(entities, e)
	}
	return entities, nil
}

func loadExperiments(db *sql.DB, appID int64) ([]Experiment, error) {
	rows, _ := db.Query("SELECT name, description, scope, overlay, status FROM experiments WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var experiments []Experiment
	for rows.Next() {
		var e Experiment
		var overlayJSON string
		rows.Scan(&e.Name, &e.Description, &e.Scope, &overlayJSON, &e.Status)
		if overlayJSON != "" {
			if err := json.Unmarshal([]byte(overlayJSON), &e.Overlay); err != nil {
				return nil, fmt.Errorf("unmarshal overlay for experiment %s: %w", e.Name, err)
			}
		}
		experiments = append(experiments, e)
	}
	return experiments, nil
}

// --- Text rendering ---

func Render(w io.Writer, spec *Spec) {
	fmt.Fprintf(w, "%s\n", spec.App.Name)
	fmt.Fprintf(w, "%s\n", spec.App.Description)

	if len(spec.Layouts) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "layouts:")
		for name, classes := range spec.Layouts {
			fmt.Fprintf(w, "  %s: [%s]\n", name, strings.Join(classes, ", "))
		}
	}

	// App-level regions [C2 fix]
	renderRegions(w, spec.App.Regions, "  ")

	if len(spec.App.Transitions) > 0 {
		fmt.Fprintln(w)
		for _, t := range spec.App.Transitions {
			fmt.Fprintf(w, "  on %s", t.OnEvent)
			writeTransitionDetail(w, t)
			fmt.Fprintln(w)
		}
	}

	for _, s := range spec.Screens {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s %s", s.Ref, s.Name)
		if s.Component != "" {
			fmt.Fprintf(w, " (%s)", s.Component)
		}
		if len(s.Tags) > 0 {
			fmt.Fprintf(w, " [%s]", strings.Join(s.Tags, ", "))
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", s.Description)

		if len(s.Attachments) > 0 {
			fmt.Fprintf(w, "  attached: %s\n", strings.Join(s.Attachments, ", "))
		}

		renderRegions(w, s.Regions, "  ")

		if len(s.Transitions) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "  states:\n")
			for _, t := range s.Transitions {
				fmt.Fprintf(w, "    %s", t.OnEvent)
				writeTransitionDetail(w, t)
				fmt.Fprintln(w)
			}
		}
	}

}

func renderRegions(w io.Writer, regions []Region, indent string) {
	if len(regions) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, r := range regions {
		fmt.Fprintf(w, "%s%s %s", indent, r.Ref, r.Name)
		if r.Component != "" {
			fmt.Fprintf(w, " (%s)", r.Component)
		}
		if len(r.Tags) > 0 {
			fmt.Fprintf(w, " [%s]", strings.Join(r.Tags, ", "))
		}
		fmt.Fprintln(w)
		if len(r.DiscoveryLayout) > 0 {
			fmt.Fprintf(w, "%s  discovery: [%s]\n", indent, strings.Join(r.DiscoveryLayout, ", "))
		}
		if len(r.DeliveryClasses) > 0 {
			fmt.Fprintf(w, "%s  delivery: [%s]", indent, strings.Join(r.DeliveryClasses, ", "))
			if r.DeliveryComponent != "" {
				fmt.Fprintf(w, " (%s)", r.DeliveryComponent)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s  %s\n", indent, r.Description)

		if len(r.Events) > 0 {
			fmt.Fprintf(w, "%s  emits: %s\n", indent, strings.Join(r.Events, ", "))
		}

		if len(r.Attachments) > 0 {
			fmt.Fprintf(w, "%s  attached: %s\n", indent, strings.Join(r.Attachments, ", "))
		}

		if len(r.Transitions) > 0 {
			fmt.Fprintf(w, "%s  states:\n", indent)
			for _, t := range r.Transitions {
				fmt.Fprintf(w, "%s    %s", indent, t.OnEvent)
				writeTransitionDetail(w, t)
				fmt.Fprintln(w)
			}
		}

		renderRegions(w, r.Regions, indent+"  ")
	}
}

func writeTransitionDetail(w io.Writer, t Transition) {
	hasFrom := t.FromState != ""
	hasTo := t.ToState != ""
	hasAction := t.Action != ""

	if hasFrom && hasTo {
		fmt.Fprintf(w, ": %s → %s", t.FromState, t.ToState)
	} else if hasFrom && !hasTo && !hasAction {
		fmt.Fprintf(w, " (from %s)", t.FromState)
	} else if hasFrom && hasAction {
		fmt.Fprintf(w, " (from %s)", t.FromState)
	}
	if hasAction {
		fmt.Fprintf(w, " → %s", t.Action)
	}
}
