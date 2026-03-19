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
	App     App      `json:"app"`
	Screens []Screen `json:"screens"`
	Flows   []Flow   `json:"flows,omitempty"`
}

type App struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	DataTypes   map[string]map[string]string `json:"data_types,omitempty"`
	Context     map[string]string            `json:"context,omitempty"`
	Regions     []Region                     `json:"regions,omitempty"`
	Transitions []Transition                 `json:"transitions,omitempty"`
}

type Screen struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Tags           []string          `json:"tags,omitempty"`
	Context        map[string]string `json:"context,omitempty"`
	Component      string            `json:"component,omitempty"`
	ComponentProps string            `json:"component_props,omitempty"` // [F5]
	ComponentOn    string            `json:"component_on,omitempty"`    // [F5]
	ComponentVis   string            `json:"component_visible,omitempty"` // [F5]
	Regions        []Region          `json:"regions,omitempty"`
	Transitions    []Transition      `json:"transitions,omitempty"`
	Attachments    []string          `json:"attachments,omitempty"`
}

type Region struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Tags           []string          `json:"tags,omitempty"`
	Component      string            `json:"component,omitempty"`
	ComponentProps string            `json:"component_props,omitempty"` // [F5]
	ComponentOn    string            `json:"component_on,omitempty"`    // [F5]
	ComponentVis   string            `json:"component_visible,omitempty"` // [F5]
	Events         []string          `json:"events,omitempty"`
	Ambient        map[string]string `json:"ambient,omitempty"`
	RegionData     map[string]string `json:"region_data,omitempty"`
	Regions        []Region          `json:"regions,omitempty"`
	Transitions    []Transition      `json:"transitions,omitempty"`
	Attachments    []string          `json:"attachments,omitempty"`
}

type Transition struct {
	OnEvent   string `json:"on_event"`
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`
	Action    string `json:"action,omitempty"`
}

type Flow struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OnEvent     string `json:"on_event,omitempty"`
	Sequence    string `json:"sequence"`
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
	row := db.QueryRow("SELECT name, description FROM apps LIMIT 1")
	if err := row.Scan(&spec.App.Name, &spec.App.Description); err != nil {
		return nil, fmt.Errorf("no app found: %w", err)
	}

	// App-level data [C2 fix: load app-level regions]
	appID := int64(0)
	db.QueryRow("SELECT id FROM apps LIMIT 1").Scan(&appID)
	spec.App.DataTypes = loadDataTypes(db, appID)
	spec.App.Context = loadContext(db, "app", appID)
	spec.App.Regions = loadRegions(db, "app", appID, al)
	spec.App.Transitions = loadTransitions(db, "app", appID)

	// Screens
	rows, err := db.Query("SELECT id, name, description FROM screens ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var s Screen
		rows.Scan(&id, &s.Name, &s.Description)
		s.Tags = loadTags(db, "screen", id)
		s.Context = loadContext(db, "screen", id)
		s.Regions = loadRegions(db, "screen", id, al)
		s.Transitions = loadTransitions(db, "screen", id)
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

	// Flows
	frows, err := db.Query("SELECT name, description, on_event, sequence FROM flows ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	for frows.Next() {
		var f Flow
		var desc, onEvent sql.NullString
		frows.Scan(&f.Name, &desc, &onEvent, &f.Sequence)
		f.Description = desc.String
		f.OnEvent = onEvent.String
		spec.Flows = append(spec.Flows, f)
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
	rows, _ := db.Query("SELECT id, name, description FROM regions WHERE parent_type = ? AND parent_id = ? ORDER BY position, id",
		parentType, parentID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var regions []Region
	for rows.Next() {
		var id int64
		var r Region
		rows.Scan(&id, &r.Name, &r.Description)
		r.Tags = loadTags(db, "region", id)
		r.Events = loadEvents(db, id)
		r.Ambient = loadAmbientRefs(db, id)
		r.RegionData = loadRegionData(db, id)
		r.Regions = loadRegions(db, "region", id, al) // recurse
		r.Transitions = loadTransitions(db, "region", id)
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
	rows, _ := db.Query("SELECT name FROM events WHERE region_id = ?", regionID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var events []string
	for rows.Next() {
		var e string
		rows.Scan(&e)
		events = append(events, e)
	}
	return events
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

func loadDataTypes(db *sql.DB, appID int64) map[string]map[string]string {
	rows, _ := db.Query("SELECT name, fields FROM data_types WHERE app_id = ? ORDER BY name", appID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	result := map[string]map[string]string{}
	for rows.Next() {
		var name, fieldsJSON string
		rows.Scan(&name, &fieldsJSON)
		var fields map[string]string
		json.Unmarshal([]byte(fieldsJSON), &fields)
		result[name] = fields
	}
	if len(result) == 0 {
		return nil
	}
	return result
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

// --- Text rendering ---

func Render(w io.Writer, spec *Spec) {
	fmt.Fprintf(w, "%s\n", spec.App.Name)
	fmt.Fprintf(w, "%s\n", spec.App.Description)

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
		fmt.Fprintf(w, "%s", s.Name)
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

	if len(spec.Flows) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "flows:\n")
		for _, f := range spec.Flows {
			fmt.Fprintf(w, "  %s", f.Name)
			if f.OnEvent != "" {
				fmt.Fprintf(w, " (on %s)", f.OnEvent)
			}
			fmt.Fprintln(w)
			if f.Description != "" {
				fmt.Fprintf(w, "    %s\n", f.Description)
			}
			fmt.Fprintf(w, "    %s\n", f.Sequence)
		}
	}
}

func renderRegions(w io.Writer, regions []Region, indent string) {
	if len(regions) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, r := range regions {
		fmt.Fprintf(w, "%s%s", indent, r.Name)
		if r.Component != "" {
			fmt.Fprintf(w, " (%s)", r.Component)
		}
		if len(r.Tags) > 0 {
			fmt.Fprintf(w, " [%s]", strings.Join(r.Tags, ", "))
		}
		fmt.Fprintln(w)
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
