package store

import (
	"database/sql"
	"fmt"
)

// CloneScreen deep-copies a screen and all its children to a new name.
func (s *Store) cloneScreen(name, newName string) error {
	srcID, err := s.ResolveScreen(name)
	if err != nil {
		return err
	}

	// Check destination doesn't exist
	if _, err := s.ResolveScreen(newName); err == nil {
		return fmt.Errorf("screen %q already exists", newName)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Copy screen row with new name
	res, err := tx.Exec(`INSERT INTO screens (app_id, name, description, entry)
		SELECT app_id, ?, description, entry FROM screens WHERE id = ?`, newName, srcID)
	if err != nil {
		return fmt.Errorf("copy screen: %w", err)
	}
	dstID, _ := res.LastInsertId()

	// 2. Copy screen-level context fields
	if _, err := tx.Exec(`INSERT INTO contexts (owner_type, owner_id, field_name, field_type)
		SELECT owner_type, ?, field_name, field_type FROM contexts
		WHERE owner_type = 'screen' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy context: %w", err)
	}

	// 3. Copy screen-level tags
	if _, err := tx.Exec(`INSERT INTO tags (entity_type, entity_id, tag)
		SELECT entity_type, ?, tag FROM tags
		WHERE entity_type = 'screen' AND entity_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy screen tags: %w", err)
	}

	// 4. Copy screen-level transitions
	if _, err := tx.Exec(`INSERT INTO transitions (owner_type, owner_id, on_event, from_state, to_state, action)
		SELECT owner_type, ?, on_event, from_state, to_state, action FROM transitions
		WHERE owner_type = 'screen' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy screen transitions: %w", err)
	}

	// 5. Copy screen-level state fixtures
	if _, err := tx.Exec(`INSERT INTO state_fixtures (owner_type, owner_id, state_name, fixture_name)
		SELECT owner_type, ?, state_name, fixture_name FROM state_fixtures
		WHERE owner_type = 'screen' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy state fixtures: %w", err)
	}

	// 6. Copy screen-level state regions
	if _, err := tx.Exec(`INSERT INTO state_regions (owner_type, owner_id, state_name, region_name)
		SELECT owner_type, ?, state_name, region_name FROM state_regions
		WHERE owner_type = 'screen' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy state regions: %w", err)
	}

	// 7. Copy screen-level components
	if _, err := tx.Exec(`INSERT INTO components (entity_type, entity_id, component, props, on_actions, visible)
		SELECT entity_type, ?, component, props, on_actions, visible FROM components
		WHERE entity_type = 'screen' AND entity_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy screen components: %w", err)
	}

	// 8. Recursively copy all child regions with ID remapping
	if err := cloneRegionTree(tx, "screen", srcID, "screen", dstID); err != nil {
		return err
	}

	return tx.Commit()
}

// CloneRegion deep-copies a region and all its children to a new name under the given parent.
func (s *Store) cloneRegion(name, newName, parentName string) error {
	// Resolve the source region scoped to the parent
	srcID, err := s.ResolveRegionIn(name, parentName)
	if err != nil {
		return err
	}

	// Resolve the destination parent
	dstParentType, dstParentID, err := s.ResolveParent(parentName)
	if err != nil {
		return err
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Copy the root region
	newRootID, err := cloneSingleRegion(tx, srcID, newName, dstParentType, dstParentID)
	if err != nil {
		return err
	}

	// Copy region attachments: events, tags, components, ambient_refs, region_data, transitions
	if err := cloneRegionAttachments(tx, srcID, newRootID); err != nil {
		return err
	}

	// Recursively copy children
	if err := cloneRegionTree(tx, "region", srcID, "region", newRootID); err != nil {
		return err
	}

	return tx.Commit()
}

// cloneRegionTree recursively copies all child regions from srcParent to dstParent,
// remapping parent IDs as it goes.
func cloneRegionTree(tx *sql.Tx, srcParentType string, srcParentID int64, dstParentType string, dstParentID int64) error {
	rows, err := tx.Query(`SELECT id, name FROM regions WHERE parent_type = ? AND parent_id = ? ORDER BY position, id`,
		srcParentType, srcParentID)
	if err != nil {
		return fmt.Errorf("query children: %w", err)
	}
	defer rows.Close()

	type child struct {
		id   int64
		name string
	}
	var children []child
	for rows.Next() {
		var c child
		rows.Scan(&c.id, &c.name)
		children = append(children, c)
	}
	rows.Close()

	for _, c := range children {
		// Copy region row (keeping same name since parent is different)
		newID, err := cloneSingleRegion(tx, c.id, c.name, dstParentType, dstParentID)
		if err != nil {
			return fmt.Errorf("clone region %q: %w", c.name, err)
		}

		// Copy all attachments
		if err := cloneRegionAttachments(tx, c.id, newID); err != nil {
			return fmt.Errorf("clone attachments for %q: %w", c.name, err)
		}

		// Recurse into children
		if err := cloneRegionTree(tx, "region", c.id, "region", newID); err != nil {
			return err
		}
	}
	return nil
}

// cloneSingleRegion copies one region row and returns the new ID.
func cloneSingleRegion(tx *sql.Tx, srcID int64, newName, newParentType string, newParentID int64) (int64, error) {
	res, err := tx.Exec(`INSERT INTO regions (app_id, parent_type, parent_id, name, description, position, discovery_layout, delivery_classes, delivery_component)
		SELECT app_id, ?, ?, ?, description, position, discovery_layout, delivery_classes, delivery_component
		FROM regions WHERE id = ?`,
		newParentType, newParentID, newName, srcID)
	if err != nil {
		return 0, fmt.Errorf("copy region: %w", err)
	}
	return res.LastInsertId()
}

// cloneRegionAttachments copies events, tags, components, ambient_refs, region_data, and transitions for a region.
func cloneRegionAttachments(tx *sql.Tx, srcID, dstID int64) error {
	// Events
	if _, err := tx.Exec(`INSERT INTO events (region_id, name, annotation)
		SELECT ?, name, annotation FROM events WHERE region_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy events: %w", err)
	}

	// Tags
	if _, err := tx.Exec(`INSERT INTO tags (entity_type, entity_id, tag)
		SELECT 'region', ?, tag FROM tags WHERE entity_type = 'region' AND entity_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}

	// Components
	if _, err := tx.Exec(`INSERT INTO components (entity_type, entity_id, component, props, on_actions, visible)
		SELECT 'region', ?, component, props, on_actions, visible FROM components
		WHERE entity_type = 'region' AND entity_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy components: %w", err)
	}

	// Ambient refs
	if _, err := tx.Exec(`INSERT INTO ambient_refs (region_id, local_name, source, query)
		SELECT ?, local_name, source, query FROM ambient_refs WHERE region_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy ambient refs: %w", err)
	}

	// Region data
	if _, err := tx.Exec(`INSERT INTO region_data (region_id, field_name, field_type)
		SELECT ?, field_name, field_type FROM region_data WHERE region_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy region data: %w", err)
	}

	// Transitions
	if _, err := tx.Exec(`INSERT INTO transitions (owner_type, owner_id, on_event, from_state, to_state, action)
		SELECT 'region', ?, on_event, from_state, to_state, action FROM transitions
		WHERE owner_type = 'region' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy transitions: %w", err)
	}

	// State fixtures on region
	if _, err := tx.Exec(`INSERT INTO state_fixtures (owner_type, owner_id, state_name, fixture_name)
		SELECT 'region', ?, state_name, fixture_name FROM state_fixtures
		WHERE owner_type = 'region' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy state fixtures: %w", err)
	}

	// State regions on region
	if _, err := tx.Exec(`INSERT INTO state_regions (owner_type, owner_id, state_name, region_name)
		SELECT 'region', ?, state_name, region_name FROM state_regions
		WHERE owner_type = 'region' AND owner_id = ?`, dstID, srcID); err != nil {
		return fmt.Errorf("copy state regions: %w", err)
	}

	return nil
}
