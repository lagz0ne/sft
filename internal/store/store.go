package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

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
	s := &Store{DB: db, Path: path}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
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
	var id int64
	err := s.DB.QueryRow("SELECT id FROM regions WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("region %q not found", name)
	}
	return id, err
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
	res, err := s.DB.Exec("INSERT INTO apps (name, description) VALUES (?, ?)", a.Name, a.Description)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

// [M1 fix] check cross-table name collision
func (s *Store) InsertScreen(sc *model.Screen) error {
	if _, err := s.ResolveRegion(sc.Name); err == nil {
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

// [M1 fix] check cross-table name collision
func (s *Store) InsertRegion(r *model.Region) error {
	if _, err := s.ResolveScreen(r.Name); err == nil {
		return fmt.Errorf("name %q already used by a screen", r.Name)
	}
	res, err := s.DB.Exec("INSERT INTO regions (app_id, parent_type, parent_id, name, description) VALUES (?, ?, ?, ?, ?)",
		r.AppID, r.ParentType, r.ParentID, r.Name, r.Description)
	if err != nil {
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
	res, err := s.DB.Exec("INSERT INTO events (region_id, name) VALUES (?, ?)",
		e.RegionID, e.Name)
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
	return nil
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

func (s *Store) UpdateRegion(name, newDesc string) error {
	res, err := s.DB.Exec("UPDATE regions SET description = ? WHERE name = ?", newDesc, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("region %q not found", name)
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

	return impacts, nil
}

func (s *Store) ImpactRegion(name string) ([]Impact, error) {
	id, err := s.ResolveRegion(name)
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

	return impacts, nil
}

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
	tx.Exec("DELETE FROM components WHERE entity_type = 'screen' AND entity_id = ?", id) // [H3]
	tx.Exec("DELETE FROM attachments WHERE entity = ?", name)                             // [H2]
	tx.Exec("DELETE FROM screens WHERE id = ?", id)

	return tx.Commit()
}

func (s *Store) DeleteRegion(name string) error {
	id, err := s.ResolveRegion(name)
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

func (s *Store) DeleteTransition(onEvent, ownerName string) error {
	ownerType, ownerID, err := s.ResolveOwner(ownerName)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec("DELETE FROM transitions WHERE on_event = ? AND owner_type = ? AND owner_id = ?",
		onEvent, ownerType, ownerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
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

// [H4 fix] MoveRegion with cycle detection
func (s *Store) MoveRegion(name, newParentName string) error {
	id, err := s.ResolveRegion(name)
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
	entityType, entityID, err := s.ResolveScreenOrRegion(entityName)
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
	entityType, entityID, err := s.ResolveScreenOrRegion(entityName)
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

func (s *Store) RemoveComponent(entityName string) error {
	entityType, entityID, err := s.ResolveScreenOrRegion(entityName)
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
