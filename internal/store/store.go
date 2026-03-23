package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/lagz0ne/sft/internal/flow"
	"github.com/lagz0ne/sft/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

const DefaultDir = ".sft"
const DefaultFile = "db"

type Store struct {
	DB   *sql.DB
	Path string
}

func DefaultPath() string {
	return filepath.Join(DefaultDir, DefaultFile)
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	db.Exec("PRAGMA journal_mode = WAL")
	db.Exec("PRAGMA busy_timeout = 5000")
	s := &Store{DB: db, Path: path}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	// Schema migration: add position column to regions if missing (existing DBs).
	db.Exec("ALTER TABLE regions ADD COLUMN position INTEGER NOT NULL DEFAULT 0")
	// Schema migration: add annotation column to events if missing (Phase 5).
	db.Exec("ALTER TABLE events ADD COLUMN annotation TEXT")
	// Schema migration: scoped region unique constraint (parent_type, parent_id, name) replacing global UNIQUE(name).
	s.migrateRegionScope()
	return s, nil
}

// memSeq ensures unique shared-cache names for concurrent in-memory stores.
var memSeq atomic.Int64

// OpenMemory opens an in-memory SQLite store (same schema, no persistence).
func OpenMemory() (*Store, error) {
	seq := memSeq.Add(1)
	dsn := fmt.Sprintf("file:sft_mem_%d?mode=memory&cache=shared", seq)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &Store{DB: db, Path: ":memory:"}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
}

// migrateRegionScope converts old UNIQUE(name) to UNIQUE(parent_type, parent_id, name).
func (s *Store) migrateRegionScope() {
	// Check if old global unique index exists (sqlite_autoindex_regions_1 with 1 column = old schema).
	var cnt int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM pragma_index_info('sqlite_autoindex_regions_1')`).Scan(&cnt)
	if err != nil || cnt != 1 {
		return // already migrated or no old index
	}
	tx, _ := s.DB.Begin()
	if tx == nil {
		return
	}
	defer tx.Rollback()
	tx.Exec(`CREATE TABLE regions_new (
		id INTEGER PRIMARY KEY, app_id INTEGER NOT NULL REFERENCES apps(id),
		parent_type TEXT NOT NULL CHECK(parent_type IN ('app','screen','region')),
		parent_id INTEGER NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 0, UNIQUE(parent_type, parent_id, name))`)
	tx.Exec(`INSERT INTO regions_new SELECT id, app_id, parent_type, parent_id, name, description, position FROM regions`)
	tx.Exec(`DROP TABLE regions`)
	tx.Exec(`ALTER TABLE regions_new RENAME TO regions`)
	tx.Commit()
}

func (s *Store) Close() error {
	return s.DB.Close()
}

// --- Resolve helpers ---

func (s *Store) ResolveApp() (int64, error) {
	var id int64
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM apps").Scan(&count); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("no apps exist — run: sft add app <name> <description>")
	}
	if count > 1 {
		return 0, fmt.Errorf("multiple apps exist — specify with --app")
	}
	err := s.DB.QueryRow("SELECT id FROM apps LIMIT 1").Scan(&id)
	return id, err
}

func (s *Store) ResolveScreen(name string) (int64, error) {
	var id int64
	err := s.DB.QueryRow("SELECT id FROM screens WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("screen %q not found", name)
	}
	return id, err
}

func (s *Store) ResolveRegion(name string) (int64, error) {
	rows, err := s.DB.Query("SELECT id FROM regions WHERE name = ?", name)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("region %q not found", name)
	}
	if len(ids) > 1 {
		// List parents for disambiguation
		var parents []string
		for _, id := range ids {
			var pt string
			var pid int64
			s.DB.QueryRow("SELECT parent_type, parent_id FROM regions WHERE id = ?", id).Scan(&pt, &pid)
			pName := s.parentName(pt, pid)
			parents = append(parents, pName)
		}
		return 0, fmt.Errorf("region %q is ambiguous — found in: %s (use --in to disambiguate)", name, strings.Join(parents, ", "))
	}
	return ids[0], nil
}

// ResolveRegionIn resolves a region name scoped to a parent name.
func (s *Store) ResolveRegionIn(name, parentName string) (int64, error) {
	parentType, parentID, err := s.ResolveParent(parentName)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.DB.QueryRow("SELECT id FROM regions WHERE name = ? AND parent_type = ? AND parent_id = ?",
		name, parentType, parentID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("region %q not found in %s", name, parentName)
	}
	return id, err
}

// resolveRegionWithScope resolves a region, using scoped resolution if inParent is provided.
func (s *Store) resolveRegionWithScope(name string, inParent ...string) (int64, error) {
	if len(inParent) > 0 && inParent[0] != "" {
		return s.ResolveRegionIn(name, inParent[0])
	}
	return s.ResolveRegion(name)
}

// parentName resolves a parent_type+parent_id to a name.
func (s *Store) parentName(pt string, pid int64) string {
	var name string
	switch pt {
	case "app":
		s.DB.QueryRow("SELECT name FROM apps WHERE id = ?", pid).Scan(&name)
	case "screen":
		s.DB.QueryRow("SELECT name FROM screens WHERE id = ?", pid).Scan(&name)
	case "region":
		s.DB.QueryRow("SELECT name FROM regions WHERE id = ?", pid).Scan(&name)
	}
	return name
}

// ResolveParent resolves a name to (type, id). Checks apps, screens, then regions. [C2 fix]
func (s *Store) ResolveParent(name string) (string, int64, error) {
	var id int64
	err := s.DB.QueryRow("SELECT id FROM apps WHERE name = ?", name).Scan(&id)
	if err == nil {
		return "app", id, nil
	}
	id, err = s.ResolveScreen(name)
	if err == nil {
		return "screen", id, nil
	}
	id, err = s.ResolveRegion(name)
	if err == nil {
		return "region", id, nil
	}
	return "", 0, fmt.Errorf("%q is not a known app, screen, or region", name)
}

// ResolveScreenOrRegion resolves to screen or region only (for tags, components).
func (s *Store) ResolveScreenOrRegion(name string) (string, int64, error) {
	id, err := s.ResolveScreen(name)
	if err == nil {
		return "screen", id, nil
	}
	id, err = s.ResolveRegion(name)
	if err == nil {
		return "region", id, nil
	}
	return "", 0, fmt.Errorf("%q is not a known screen or region", name)
}

// ResolveOwner resolves to (owner_type, owner_id). Checks apps, screens, regions.
func (s *Store) ResolveOwner(name string) (string, int64, error) {
	return s.ResolveParent(name)
}

// --- Insert helpers ---

func (s *Store) InsertApp(a *model.App) error {
	// [F1] Prevent duplicate apps — single-app model
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM apps").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("an app already exists — SFT supports a single app per project")
	}
	res, err := s.DB.Exec("INSERT INTO apps (name, description) VALUES (?, ?)", a.Name, a.Description)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

// [M1 fix] check cross-table name collision
func (s *Store) InsertScreen(sc *model.Screen) error {
	var regionCount int
	s.DB.QueryRow("SELECT COUNT(*) FROM regions WHERE name = ?", sc.Name).Scan(&regionCount)
	if regionCount > 0 {
		return fmt.Errorf("name %q already used by a region", sc.Name)
	}
	res, err := s.DB.Exec("INSERT INTO screens (app_id, name, description) VALUES (?, ?, ?)",
		sc.AppID, sc.Name, sc.Description)
	if err != nil {
		return err
	}
	sc.ID, _ = res.LastInsertId()
	return nil
}

// [M1 fix] check cross-table name collision (screen names must be globally unique vs regions)
func (s *Store) InsertRegion(r *model.Region) error {
	if _, err := s.ResolveScreen(r.Name); err == nil {
		return fmt.Errorf("name %q already used by a screen", r.Name)
	}
	res, err := s.DB.Exec("INSERT INTO regions (app_id, parent_type, parent_id, name, description) VALUES (?, ?, ?, ?, ?)",
		r.AppID, r.ParentType, r.ParentID, r.Name, r.Description)
	if err != nil {
		// Translate the scoped unique constraint error into a friendlier message
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("region %q already exists in this parent", r.Name)
		}
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertTag(t *model.Tag) error {
	res, err := s.DB.Exec("INSERT INTO tags (entity_type, entity_id, tag) VALUES (?, ?, ?)",
		t.EntityType, t.EntityID, t.Tag)
	if err != nil {
		return err
	}
	t.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertEvent(e *model.Event) error {
	res, err := s.DB.Exec("INSERT INTO events (region_id, name, annotation) VALUES (?, ?, ?)",
		e.RegionID, e.Name, e.Annotation)
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertTransition(t *model.Transition) error {
	res, err := s.DB.Exec("INSERT INTO transitions (owner_type, owner_id, on_event, from_state, to_state, action) VALUES (?, ?, ?, ?, ?, ?)",
		t.OwnerType, t.OwnerID, t.OnEvent, t.FromState, t.ToState, t.Action)
	if err != nil {
		return err
	}
	t.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertFlow(f *model.Flow) error {
	res, err := s.DB.Exec("INSERT INTO flows (app_id, name, description, on_event, sequence) VALUES (?, ?, ?, ?, ?)",
		f.AppID, f.Name, f.Description, f.OnEvent, f.Sequence)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()

	// Parse sequence into flow_steps
	steps := flow.ParseSequence(f.Sequence, f.ID, s)
	for i := range steps {
		if err := s.InsertFlowStep(&steps[i]); err != nil {
			return fmt.Errorf("flow step %d: %w", i+1, err)
		}
	}
	return nil
}

// IsEvent returns true if a name matches any known event.
func (s *Store) IsEvent(name string) bool {
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM events WHERE name = ?", name).Scan(&count)
	return count > 0
}

func (s *Store) InsertFlowStep(fs *model.FlowStep) error {
	res, err := s.DB.Exec("INSERT INTO flow_steps (flow_id, position, raw, type, name, history, data) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fs.FlowID, fs.Position, fs.Raw, fs.Type, fs.Name, fs.History, fs.Data)
	if err != nil {
		return err
	}
	fs.ID, _ = res.LastInsertId()
	return nil
}

// --- Phase 5: State-region visibility ---

func (s *Store) InsertStateRegion(sr *model.StateRegion) error {
	res, err := s.DB.Exec("INSERT INTO state_regions (owner_type, owner_id, state_name, region_name) VALUES (?, ?, ?, ?)",
		sr.OwnerType, sr.OwnerID, sr.StateName, sr.RegionName)
	if err != nil {
		return err
	}
	sr.ID, _ = res.LastInsertId()
	return nil
}

// --- Phase 5: Enum inserts ---

func (s *Store) InsertEnum(e *model.Enum) error {
	res, err := s.DB.Exec(`INSERT INTO enums (app_id, name, "values") VALUES (?, ?, ?)`,
		e.AppID, e.Name, e.Values)
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

// --- Phase 4: State template inserts ---

func (s *Store) InsertStateTemplate(st *model.StateTemplate) error {
	res, err := s.DB.Exec("INSERT INTO state_templates (app_id, name, definition) VALUES (?, ?, ?)",
		st.AppID, st.Name, st.Definition)
	if err != nil {
		return err
	}
	st.ID, _ = res.LastInsertId()
	return nil
}

// GetStateTemplate returns the definition JSON for a named template, or "" if not found.
func (s *Store) GetStateTemplate(appID int64, name string) (string, error) {
	var def string
	err := s.DB.QueryRow("SELECT definition FROM state_templates WHERE app_id = ? AND name = ?", appID, name).Scan(&def)
	if err != nil {
		return "", fmt.Errorf("state template %q not found", name)
	}
	return def, nil
}

// --- Phase 3: Fixture inserts ---

func (s *Store) InsertFixture(f *model.Fixture) error {
	res, err := s.DB.Exec("INSERT INTO fixtures (app_id, name, extends, data) VALUES (?, ?, ?, ?)",
		f.AppID, f.Name, f.Extends, f.Data)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertStateFixture(sf *model.StateFixture) error {
	res, err := s.DB.Exec("INSERT INTO state_fixtures (owner_type, owner_id, state_name, fixture_name) VALUES (?, ?, ?, ?)",
		sf.OwnerType, sf.OwnerID, sf.StateName, sf.FixtureName)
	if err != nil {
		return err
	}
	sf.ID, _ = res.LastInsertId()
	return nil
}

// --- Phase 2: Data model inserts ---

func (s *Store) InsertDataType(dt *model.DataType) error {
	res, err := s.DB.Exec("INSERT INTO data_types (app_id, name, fields) VALUES (?, ?, ?)",
		dt.AppID, dt.Name, dt.Fields)
	if err != nil {
		return err
	}
	dt.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertContextField(cf *model.ContextField) error {
	res, err := s.DB.Exec("INSERT INTO contexts (owner_type, owner_id, field_name, field_type) VALUES (?, ?, ?, ?)",
		cf.OwnerType, cf.OwnerID, cf.FieldName, cf.FieldType)
	if err != nil {
		return err
	}
	cf.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertAmbientRef(ar *model.AmbientRef) error {
	res, err := s.DB.Exec("INSERT INTO ambient_refs (region_id, local_name, source, query) VALUES (?, ?, ?, ?)",
		ar.RegionID, ar.LocalName, ar.Source, ar.Query)
	if err != nil {
		return err
	}
	ar.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) InsertRegionData(rd *model.RegionData) error {
	res, err := s.DB.Exec("INSERT INTO region_data (region_id, field_name, field_type) VALUES (?, ?, ?)",
		rd.RegionID, rd.FieldName, rd.FieldType)
	if err != nil {
		return err
	}
	rd.ID, _ = res.LastInsertId()
	return nil
}

// --- Update helpers [H6 fix] ---

func (s *Store) UpdateScreen(name, newDesc string) error {
	res, err := s.DB.Exec("UPDATE screens SET description = ? WHERE name = ?", newDesc, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("screen %q not found", name)
	}
	return nil
}

func (s *Store) UpdateRegion(name, newDesc string, inParent ...string) error {
	id, err := s.resolveRegionWithScope(name, inParent...)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec("UPDATE regions SET description = ? WHERE id = ?", newDesc, id)
	return err
}

func (s *Store) UpdateDataType(name, fields string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("UPDATE data_types SET fields = ? WHERE name = ? AND app_id = ?", fields, name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("data type %q not found", name)
	}
	return nil
}

func (s *Store) UpdateEnum(name, values string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec(`UPDATE enums SET "values" = ? WHERE name = ? AND app_id = ?`, values, name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("enum %q not found", name)
	}
	return nil
}

func (s *Store) UpdateContextField(field, ownerType string, ownerID int64, newType string) error {
	res, err := s.DB.Exec("UPDATE contexts SET field_type = ? WHERE field_name = ? AND owner_type = ? AND owner_id = ?",
		newType, field, ownerType, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("context field %q not found", field)
	}
	return nil
}

func (s *Store) UpdateRegionData(field string, regionID int64, newType string) error {
	res, err := s.DB.Exec("UPDATE region_data SET field_type = ? WHERE field_name = ? AND region_id = ?",
		newType, field, regionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("field %q not found", field)
	}
	return nil
}

func (s *Store) UpdateAmbientRef(name string, regionID int64, source, query string) error {
	res, err := s.DB.Exec("UPDATE ambient_refs SET source = ?, query = ? WHERE local_name = ? AND region_id = ?",
		source, query, name, regionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ambient ref %q not found", name)
	}
	return nil
}

func (s *Store) UpdateFixture(name, data, extends string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("UPDATE fixtures SET data = ?, extends = ? WHERE name = ? AND app_id = ?",
		data, extends, name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fixture %q not found", name)
	}
	return nil
}

func (s *Store) UpdateStateFixture(ownerType string, ownerID int64, state, fixture string) error {
	res, err := s.DB.Exec("UPDATE state_fixtures SET fixture_name = ? WHERE owner_type = ? AND owner_id = ? AND state_name = ?",
		fixture, ownerType, ownerID, state)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("state fixture for state %q not found", state)
	}
	return nil
}

// --- Impact analysis [H3 fix: add components + attachments] ---

type Impact struct {
	Entity string `json:"entity"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
}

func (s *Store) ImpactScreen(name string) ([]Impact, error) {
	id, err := s.ResolveScreen(name)
	if err != nil {
		return nil, err
	}
	var impacts []Impact

	rows, err := s.DB.Query("SELECT name FROM regions WHERE parent_type = 'screen' AND parent_id = ?", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		impacts = append(impacts, Impact{Entity: "region", Type: "child", Name: n})
	}
	rows.Close()

	rows, err = s.DB.Query(`SELECT e.name, r.name FROM events e
		JOIN regions r ON r.id = e.region_id
		WHERE r.parent_type = 'screen' AND r.parent_id = ?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var evName, regName string
		rows.Scan(&evName, &regName)
		impacts = append(impacts, Impact{Entity: "event", Type: "in-child-region", Name: evName, Detail: "in " + regName})
	}
	rows.Close()

	rows, err = s.DB.Query(`SELECT on_event, from_state, to_state FROM transitions
		WHERE owner_type = 'screen' AND owner_id = ?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var onEvent string
		var from, to sql.NullString
		rows.Scan(&onEvent, &from, &to)
		detail := onEvent
		if from.Valid {
			detail += " " + from.String + " → " + to.String
		}
		impacts = append(impacts, Impact{Entity: "transition", Type: "owned", Name: onEvent, Detail: detail})
	}
	rows.Close()

	rows, err = s.DB.Query(`SELECT f.name FROM flow_steps fs
		JOIN flows f ON f.id = fs.flow_id
		WHERE fs.type = 'screen' AND fs.name = ?`, name)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var flowName string
		rows.Scan(&flowName)
		impacts = append(impacts, Impact{Entity: "flow", Type: "references", Name: flowName})
	}
	rows.Close()

	rows, err = s.DB.Query("SELECT tag FROM tags WHERE entity_type = 'screen' AND entity_id = ?", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tag string
		rows.Scan(&tag)
		impacts = append(impacts, Impact{Entity: "tag", Type: "on-screen", Name: tag})
	}
	rows.Close()

	// [H3] Component
	if c := s.GetComponent("screen", id); c != nil {
		impacts = append(impacts, Impact{Entity: "component", Type: "bound", Name: c.Component})
	}

	// [H2] Attachments
	for _, a := range s.AttachmentsFor(name) {
		impacts = append(impacts, Impact{Entity: "attachment", Type: "on-screen", Name: a})
	}

	// [F6] Incoming navigate() references
	impacts = append(impacts, s.incomingNavigateRefs(name)...)

	return impacts, nil
}

func (s *Store) ImpactRegion(name string, inParent ...string) ([]Impact, error) {
	id, err := s.resolveRegionWithScope(name, inParent...)
	if err != nil {
		return nil, err
	}
	var impacts []Impact

	rows, err := s.DB.Query("SELECT name FROM regions WHERE parent_type = 'region' AND parent_id = ?", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		impacts = append(impacts, Impact{Entity: "region", Type: "child", Name: n})
	}
	rows.Close()

	rows, err = s.DB.Query("SELECT name FROM events WHERE region_id = ?", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		impacts = append(impacts, Impact{Entity: "event", Type: "emitted", Name: n})
	}
	rows.Close()

	rows, err = s.DB.Query(`SELECT on_event, from_state, to_state FROM transitions
		WHERE owner_type = 'region' AND owner_id = ?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var onEvent string
		var from, to sql.NullString
		rows.Scan(&onEvent, &from, &to)
		detail := onEvent
		if from.Valid {
			detail += " " + from.String + " → " + to.String
		}
		impacts = append(impacts, Impact{Entity: "transition", Type: "owned", Name: onEvent, Detail: detail})
	}
	rows.Close()

	rows, err = s.DB.Query(`SELECT f.name FROM flow_steps fs
		JOIN flows f ON f.id = fs.flow_id
		WHERE fs.type = 'region' AND fs.name = ?`, name)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var flowName string
		rows.Scan(&flowName)
		impacts = append(impacts, Impact{Entity: "flow", Type: "references", Name: flowName})
	}
	rows.Close()

	rows, err = s.DB.Query("SELECT tag FROM tags WHERE entity_type = 'region' AND entity_id = ?", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tag string
		rows.Scan(&tag)
		impacts = append(impacts, Impact{Entity: "tag", Type: "on-region", Name: tag})
	}
	rows.Close()

	// [H3] Component
	if c := s.GetComponent("region", id); c != nil {
		impacts = append(impacts, Impact{Entity: "component", Type: "bound", Name: c.Component})
	}

	// [H2] Attachments
	for _, a := range s.AttachmentsFor(name) {
		impacts = append(impacts, Impact{Entity: "attachment", Type: "on-region", Name: a})
	}

	// [F6] Incoming navigate() references
	impacts = append(impacts, s.incomingNavigateRefs(name)...)

	return impacts, nil
}

// [F6] incomingNavigateRefs finds transitions whose action is navigate(<name>) or navigate(<name>, ...).
func (s *Store) incomingNavigateRefs(name string) []Impact {
	target := "navigate(" + name + ")"
	targetWithParams := "navigate(" + name + ",%"
	rows, err := s.DB.Query(`SELECT `+ownerCase+` AS owner_name, t.on_event
		FROM transitions t
		WHERE t.action = ? OR t.action LIKE ?`, target, targetWithParams)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var impacts []Impact
	for rows.Next() {
		var ownerName, onEvent string
		rows.Scan(&ownerName, &onEvent)
		impacts = append(impacts, Impact{Entity: "transition", Type: "navigates-here", Name: onEvent, Detail: "from " + ownerName})
	}
	return impacts
}

// ownerCase for store package queries
const ownerCase = `CASE t.owner_type
  WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t.owner_id)
  WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = t.owner_id)
  WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = t.owner_id)
END`

// --- Mutations ---

// collectDescendantRegions returns all region IDs recursively under a parent. [H1 fix]
func (s *Store) collectDescendantRegions(tx *sql.Tx, parentType string, parentID int64) []int64 {
	rows, err := tx.Query("SELECT id, name FROM regions WHERE parent_type = ? AND parent_id = ?", parentType, parentID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	var names []string
	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)
		ids = append(ids, id)
		names = append(names, name)
	}
	for i, id := range ids {
		// Recurse into child regions
		childIDs := s.collectDescendantRegions(tx, "region", id)
		ids = append(ids, childIDs...)
		// Clean attachments by name [H2]
		tx.Exec("DELETE FROM attachments WHERE entity = ?", names[i])
	}
	return ids
}

func (s *Store) deleteRegionIDs(tx *sql.Tx, ids []int64) {
	for _, id := range ids {
		tx.Exec("DELETE FROM events WHERE region_id = ?", id)
		tx.Exec("DELETE FROM tags WHERE entity_type = 'region' AND entity_id = ?", id)
		tx.Exec("DELETE FROM transitions WHERE owner_type = 'region' AND owner_id = ?", id)
		tx.Exec("DELETE FROM state_regions WHERE owner_type = 'region' AND owner_id = ?", id)
		tx.Exec("DELETE FROM state_fixtures WHERE owner_type = 'region' AND owner_id = ?", id)
		tx.Exec("DELETE FROM components WHERE entity_type = 'region' AND entity_id = ?", id) // [H3]
		tx.Exec("DELETE FROM flow_steps WHERE type = 'region' AND name = (SELECT name FROM regions WHERE id = ?)", id)
		tx.Exec("DELETE FROM regions WHERE id = ?", id)
	}
}

func (s *Store) DeleteScreen(name string) error {
	id, err := s.ResolveScreen(name)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Collect ALL descendant regions recursively [H1]
	descendantIDs := s.collectDescendantRegions(tx, "screen", id)
	s.deleteRegionIDs(tx, descendantIDs)

	tx.Exec("DELETE FROM flow_steps WHERE type = 'screen' AND name = ?", name)
	tx.Exec("DELETE FROM tags WHERE entity_type = 'screen' AND entity_id = ?", id)
	tx.Exec("DELETE FROM transitions WHERE owner_type = 'screen' AND owner_id = ?", id)
	tx.Exec("DELETE FROM state_regions WHERE owner_type = 'screen' AND owner_id = ?", id)
	tx.Exec("DELETE FROM state_fixtures WHERE owner_type = 'screen' AND owner_id = ?", id)
	tx.Exec("DELETE FROM components WHERE entity_type = 'screen' AND entity_id = ?", id) // [H3]
	tx.Exec("DELETE FROM attachments WHERE entity = ?", name)                             // [H2]
	tx.Exec("DELETE FROM transitions WHERE action = ? OR action LIKE ?", "navigate("+name+")", "navigate("+name+",%") // [F2] cascade dangling navigate()
	tx.Exec("DELETE FROM screens WHERE id = ?", id)

	return tx.Commit()
}

func (s *Store) DeleteRegion(name string, inParent ...string) error {
	id, err := s.resolveRegionWithScope(name, inParent...)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Collect ALL descendant regions recursively [H1]
	descendantIDs := s.collectDescendantRegions(tx, "region", id)
	s.deleteRegionIDs(tx, descendantIDs)

	// Delete own data
	tx.Exec("DELETE FROM events WHERE region_id = ?", id)
	tx.Exec("DELETE FROM tags WHERE entity_type = 'region' AND entity_id = ?", id)
	tx.Exec("DELETE FROM transitions WHERE owner_type = 'region' AND owner_id = ?", id)
	tx.Exec("DELETE FROM components WHERE entity_type = 'region' AND entity_id = ?", id) // [H3]
	tx.Exec("DELETE FROM attachments WHERE entity = ?", name)                             // [H2]
	tx.Exec("DELETE FROM transitions WHERE action = ? OR action LIKE ?", "navigate("+name+")", "navigate("+name+",%") // [F2] cascade dangling navigate()
	tx.Exec("DELETE FROM flow_steps WHERE type = 'region' AND name = ?", name)
	tx.Exec("DELETE FROM regions WHERE id = ?", id)

	return tx.Commit()
}

// [H7 fix] Fine-grained delete methods
func (s *Store) DeleteEvent(name, regionName string) error {
	regionID, err := s.ResolveRegion(regionName)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM events WHERE name = ? AND region_id = ?", name, regionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event %q not found in %s", name, regionName)
	}
	return nil
}

func (s *Store) DeleteTransition(onEvent, ownerName, fromState string) error {
	ownerType, ownerID, err := s.ResolveOwner(ownerName)
	if err != nil {
		return err
	}
	q := "DELETE FROM transitions WHERE on_event = ? AND owner_type = ? AND owner_id = ?"
	args := []any{onEvent, ownerType, ownerID}
	if fromState != "" {
		q += " AND from_state = ?"
		args = append(args, fromState)
	}
	res, err := s.DB.Exec(q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if fromState != "" {
			return fmt.Errorf("transition on %q from %q not found in %s", onEvent, fromState, ownerName)
		}
		return fmt.Errorf("transition on %q not found in %s", onEvent, ownerName)
	}
	return nil
}

func (s *Store) DeleteTag(tag, entityName string) error {
	entityType, entityID, err := s.ResolveScreenOrRegion(entityName)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM tags WHERE tag = ? AND entity_type = ? AND entity_id = ?",
		tag, entityType, entityID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found on %s", tag, entityName)
	}
	return nil
}

func (s *Store) DeleteFlow(name string) error {
	var flowID int64
	err := s.DB.QueryRow("SELECT id FROM flows WHERE name = ?", name).Scan(&flowID)
	if err != nil {
		return fmt.Errorf("flow %q not found", name)
	}
	s.DB.Exec("DELETE FROM flow_steps WHERE flow_id = ?", flowID)
	s.DB.Exec("DELETE FROM flows WHERE id = ?", flowID)
	return nil
}

func (s *Store) DeleteDataType(name string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM data_types WHERE name = ? AND app_id = ?", name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("data type %q not found", name)
	}
	return nil
}

func (s *Store) DeleteEnum(name string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM enums WHERE name = ? AND app_id = ?", name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("enum %q not found", name)
	}
	return nil
}

func (s *Store) DeleteContextField(fieldName, ownerType string, ownerID int64) error {
	res, err := s.DB.Exec("DELETE FROM contexts WHERE field_name = ? AND owner_type = ? AND owner_id = ?",
		fieldName, ownerType, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("context field %q not found", fieldName)
	}
	return nil
}

func (s *Store) DeleteRegionData(fieldName string, regionID int64) error {
	res, err := s.DB.Exec("DELETE FROM region_data WHERE field_name = ? AND region_id = ?",
		fieldName, regionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("field %q not found", fieldName)
	}
	return nil
}

func (s *Store) DeleteAmbientRef(localName string, regionID int64) error {
	res, err := s.DB.Exec("DELETE FROM ambient_refs WHERE local_name = ? AND region_id = ?",
		localName, regionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ambient ref %q not found", localName)
	}
	return nil
}

func (s *Store) DeleteFixture(name string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM fixtures WHERE name = ? AND app_id = ?", name, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fixture %q not found", name)
	}
	return nil
}

func (s *Store) DeleteStateFixture(ownerType string, ownerID int64, stateName string) error {
	res, err := s.DB.Exec("DELETE FROM state_fixtures WHERE owner_type = ? AND owner_id = ? AND state_name = ?",
		ownerType, ownerID, stateName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("state fixture for state %q not found", stateName)
	}
	return nil
}

func (s *Store) DeleteStateRegion(regionName, ownerType string, ownerID int64, stateName string) error {
	res, err := s.DB.Exec("DELETE FROM state_regions WHERE region_name = ? AND owner_type = ? AND owner_id = ? AND state_name = ?",
		regionName, ownerType, ownerID, stateName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("state region %q for state %q not found", regionName, stateName)
	}
	return nil
}

// [H4 fix] MoveRegion with cycle detection
func (s *Store) MoveRegion(name, newParentName string, inParent ...string) error {
	id, err := s.resolveRegionWithScope(name, inParent...)
	if err != nil {
		return err
	}
	parentType, parentID, err := s.ResolveParent(newParentName)
	if err != nil {
		return err
	}

	// Cycle detection: cannot move under self or own descendant.
	if parentType == "region" && parentID == id {
		return fmt.Errorf("cannot move %q under itself", name)
	}
	if parentType == "region" {
		checkID := parentID
		for {
			var pt string
			var pid int64
			err := s.DB.QueryRow("SELECT parent_type, parent_id FROM regions WHERE id = ?", checkID).Scan(&pt, &pid)
			if err != nil {
				break
			}
			if pt == "region" && pid == id {
				return fmt.Errorf("cannot move %q into its own descendant %q — would create a cycle", name, newParentName)
			}
			if pt != "region" {
				break
			}
			checkID = pid
		}
	}

	_, err = s.DB.Exec("UPDATE regions SET parent_type = ?, parent_id = ? WHERE id = ?",
		parentType, parentID, id)
	return err
}

// --- Rename ---

func (s *Store) RenameScreen(old, newName string) error {
	if _, err := s.ResolveScreen(old); err != nil {
		return err
	}
	if _, err := s.ResolveScreen(newName); err == nil {
		return fmt.Errorf("screen %q already exists", newName)
	}
	if _, err := s.ResolveRegion(newName); err == nil {
		return fmt.Errorf("name %q already used by a region", newName)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE screens SET name = ? WHERE name = ?", newName, old)
	tx.Exec("UPDATE flow_steps SET name = ? WHERE type = 'screen' AND name = ?", newName, old)
	tx.Exec("UPDATE transitions SET action = ? WHERE action = ?", "navigate("+newName+")", "navigate("+old+")")
	tx.Exec("UPDATE transitions SET action = REPLACE(action, ?, ?) WHERE action LIKE ?",
		"navigate("+old+",", "navigate("+newName+",", "navigate("+old+",%")
	tx.Exec("UPDATE attachments SET entity = ? WHERE entity = ?", newName, old)
	return tx.Commit()
}

func (s *Store) RenameRegion(old, newName string, inParent ...string) error {
	id, err := s.resolveRegionWithScope(old, inParent...)
	if err != nil {
		return err
	}
	if _, err := s.ResolveScreen(newName); err == nil {
		return fmt.Errorf("name %q already used by a screen", newName)
	}
	// Scoped collision check: only block if newName exists under the same parent
	var parentType string
	var parentID int64
	s.DB.QueryRow("SELECT parent_type, parent_id FROM regions WHERE id = ?", id).Scan(&parentType, &parentID)
	var collision int
	s.DB.QueryRow("SELECT COUNT(*) FROM regions WHERE name = ? AND parent_type = ? AND parent_id = ? AND id != ?",
		newName, parentType, parentID, id).Scan(&collision)
	if collision > 0 {
		return fmt.Errorf("region %q already exists in %s", newName, s.parentName(parentType, parentID))
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE regions SET name = ? WHERE id = ?", newName, id)
	tx.Exec("UPDATE flow_steps SET name = ? WHERE type = 'region' AND name = ?", newName, old)
	tx.Exec("UPDATE transitions SET action = ? WHERE action = ?", "navigate("+newName+")", "navigate("+old+")")
	tx.Exec("UPDATE transitions SET action = REPLACE(action, ?, ?) WHERE action LIKE ?",
		"navigate("+old+",", "navigate("+newName+",", "navigate("+old+",%")
	tx.Exec("UPDATE attachments SET entity = ? WHERE entity = ?", newName, old)
	return tx.Commit()
}

func (s *Store) RenameFlow(old, newName string) error {
	res, err := s.DB.Exec("UPDATE flows SET name = ? WHERE name = ?", newName, old)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("flow %q not found", old)
	}
	return nil
}

// renameTypeInFields cascades a type/enum rename through contexts.field_type and region_data.field_type.
// Handles decorated names: "email", "email[]", "email?", "email[]?".
func (s *Store) renameTypeInFields(tx *sql.Tx, old, newName string) {
	// Cascade through contexts.field_type
	tx.Exec("UPDATE contexts SET field_type = ? WHERE field_type = ?", newName, old)
	tx.Exec("UPDATE contexts SET field_type = ? WHERE field_type = ?", newName+"[]", old+"[]")
	tx.Exec("UPDATE contexts SET field_type = ? WHERE field_type = ?", newName+"?", old+"?")
	tx.Exec("UPDATE contexts SET field_type = ? WHERE field_type = ?", newName+"[]?", old+"[]?")
	// Cascade through region_data.field_type
	tx.Exec("UPDATE region_data SET field_type = ? WHERE field_type = ?", newName, old)
	tx.Exec("UPDATE region_data SET field_type = ? WHERE field_type = ?", newName+"[]", old+"[]")
	tx.Exec("UPDATE region_data SET field_type = ? WHERE field_type = ?", newName+"?", old+"?")
	tx.Exec("UPDATE region_data SET field_type = ? WHERE field_type = ?", newName+"[]?", old+"[]?")
	// Cascade through events.annotation (same decoration pattern)
	tx.Exec("UPDATE events SET annotation = ? WHERE annotation = ?", newName, old)
	tx.Exec("UPDATE events SET annotation = ? WHERE annotation = ?", newName+"[]", old+"[]")
	tx.Exec("UPDATE events SET annotation = ? WHERE annotation = ?", newName+"?", old+"?")
	tx.Exec("UPDATE events SET annotation = ? WHERE annotation = ?", newName+"[]?", old+"[]?")
}

func (s *Store) RenameDataType(old, newName string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM data_types WHERE name = ? AND app_id = ?", old, appID).Scan(&count)
	if count == 0 {
		return fmt.Errorf("data type %q not found", old)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE data_types SET name = ? WHERE name = ? AND app_id = ?", newName, old, appID)
	s.renameTypeInFields(tx, old, newName)
	return tx.Commit()
}

func (s *Store) RenameEnum(old, newName string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM enums WHERE name = ? AND app_id = ?", old, appID).Scan(&count)
	if count == 0 {
		return fmt.Errorf("enum %q not found", old)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE enums SET name = ? WHERE name = ? AND app_id = ?", newName, old, appID)
	s.renameTypeInFields(tx, old, newName)
	return tx.Commit()
}

func (s *Store) RenameFixture(old, newName string) error {
	appID, err := s.ResolveApp()
	if err != nil {
		return err
	}
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM fixtures WHERE name = ? AND app_id = ?", old, appID).Scan(&count)
	if count == 0 {
		return fmt.Errorf("fixture %q not found", old)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE fixtures SET name = ? WHERE name = ? AND app_id = ?", newName, old, appID)
	tx.Exec("UPDATE state_fixtures SET fixture_name = ? WHERE fixture_name = ?", newName, old)
	return tx.Commit()
}

// --- Reorder ---

func (s *Store) ReorderRegions(parentName string, childNames []string) error {
	parentType, parentID, err := s.ResolveParent(parentName)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, name := range childNames {
		res, err := tx.Exec("UPDATE regions SET position = ? WHERE name = ? AND parent_type = ? AND parent_id = ?",
			i+1, name, parentType, parentID)
		if err != nil {
			return fmt.Errorf("reorder %s: %w", name, err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("region %q is not a child of %s", name, parentName)
		}
	}
	return tx.Commit()
}

// --- Components ---

type Component struct {
	EntityType string `json:"entity_type"`
	EntityID   int64  `json:"entity_id"`
	Component  string `json:"component"`
	Props      string `json:"props"`
	OnActions  string `json:"on_actions,omitempty"`
	Visible    string `json:"visible,omitempty"`
}

func (s *Store) SetComponent(entityName, component, props, onActions, visible string) error {
	// [F11] Accept app entities too (not just screen/region)
	entityType, entityID, err := s.ResolveParent(entityName)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		`INSERT INTO components (entity_type, entity_id, component, props, on_actions, visible)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(entity_type, entity_id) DO UPDATE SET
		   component = excluded.component, props = excluded.props,
		   on_actions = excluded.on_actions, visible = excluded.visible`,
		entityType, entityID, component, props, onActions, visible)
	return err
}

func (s *Store) GetComponent(entityType string, entityID int64) *Component {
	var c Component
	var onActions, visible sql.NullString
	err := s.DB.QueryRow(
		"SELECT entity_type, entity_id, component, props, on_actions, visible FROM components WHERE entity_type = ? AND entity_id = ?",
		entityType, entityID).Scan(&c.EntityType, &c.EntityID, &c.Component, &c.Props, &onActions, &visible)
	if err != nil {
		return nil
	}
	c.OnActions = onActions.String
	c.Visible = visible.String
	return &c
}

func (s *Store) GetComponentByName(entityName string) *Component {
	// [F11] Support app entities too
	entityType, entityID, err := s.ResolveParent(entityName)
	if err != nil {
		return nil
	}
	return s.GetComponent(entityType, entityID)
}

func (s *Store) ComponentFor(entityType string, entityID int64) string {
	c := s.GetComponent(entityType, entityID)
	if c == nil {
		return ""
	}
	return c.Component
}

// ComponentInfo holds full component details for enrichment.
type ComponentInfo struct {
	Component string
	Props     string
	OnActions string
	Visible   string
}

// ComponentInfoFor returns full component details.
func (s *Store) ComponentInfoFor(entityType string, entityID int64) *ComponentInfo {
	c := s.GetComponent(entityType, entityID)
	if c == nil {
		return nil
	}
	return &ComponentInfo{
		Component: c.Component,
		Props:     c.Props,
		OnActions: c.OnActions,
		Visible:   c.Visible,
	}
}

func (s *Store) RemoveComponent(entityName string) error {
	// [F11] Support app entities too
	entityType, entityID, err := s.ResolveParent(entityName)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec("DELETE FROM components WHERE entity_type = ? AND entity_id = ?", entityType, entityID)
	return err
}

// --- Attachments ---

const GlobalEntity = "_"

type Attachment struct {
	Entity string `json:"entity"`
	Name   string `json:"name"`
}

func (s *Store) Attach(entity, srcPath, asName string) (string, error) {
	// [F8] Validate entity exists (allow "_" global)
	if entity != GlobalEntity {
		if _, _, err := s.ResolveParent(entity); err != nil {
			return "", fmt.Errorf("entity %q not found — attach to an existing app, screen, or region (use %q for global)", entity, GlobalEntity)
		}
	}
	name := filepath.Base(srcPath)
	if asName != "" {
		name = asName
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", srcPath, err)
	}
	_, err = s.DB.Exec(
		`INSERT INTO attachments (entity, name, content) VALUES (?, ?, ?)
		 ON CONFLICT(entity, name) DO UPDATE SET content = excluded.content`,
		entity, name, data)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (s *Store) AttachContent(entity, name string, content []byte) error {
	_, err := s.DB.Exec(
		`INSERT INTO attachments (entity, name, content) VALUES (?, ?, ?)
		 ON CONFLICT(entity, name) DO UPDATE SET content = excluded.content`,
		entity, name, content)
	return err
}

func (s *Store) Detach(entity, name string) error {
	res, err := s.DB.Exec("DELETE FROM attachments WHERE entity = ? AND name = ?", entity, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("attachment %s/%s not found", entity, name)
	}
	return nil
}

func (s *Store) ListAttachments(filterEntity string) ([]Attachment, error) {
	q := "SELECT entity, name FROM attachments ORDER BY entity, name"
	args := []any{}
	if filterEntity != "" {
		q = "SELECT entity, name FROM attachments WHERE entity = ? ORDER BY name"
		args = append(args, filterEntity)
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []Attachment
	for rows.Next() {
		var a Attachment
		rows.Scan(&a.Entity, &a.Name)
		all = append(all, a)
	}
	return all, nil
}

func (s *Store) AttachmentsFor(entity string) []string {
	rows, _ := s.DB.Query("SELECT name FROM attachments WHERE entity = ? ORDER BY name", entity)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		names = append(names, n)
	}
	return names
}

func (s *Store) ReadAttachment(entity, name string) ([]byte, error) {
	var content []byte
	err := s.DB.QueryRow("SELECT content FROM attachments WHERE entity = ? AND name = ?", entity, name).Scan(&content)
	if err != nil {
		return nil, fmt.Errorf("attachment %s/%s not found", entity, name)
	}
	return content, nil
}
