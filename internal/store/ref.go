package store

import "fmt"

// ResolveRef takes a ref like "@s1", "@r7", "@f2" and returns (entityType, entityID, entityName, error).
// Prefix mapping: s=screen, r=region, f=flow, e=event, t=transition
func (s *Store) ResolveRef(ref string) (entityType string, entityID int64, entityName string, err error) {
	if len(ref) < 3 || ref[0] != '@' {
		return "", 0, "", fmt.Errorf("invalid ref: %s", ref)
	}

	prefix := ref[1]
	if _, err := fmt.Sscanf(ref[2:], "%d", &entityID); err != nil {
		return "", 0, "", fmt.Errorf("invalid ref: %s", ref)
	}

	var table, nameCol string
	switch prefix {
	case 's':
		table, nameCol, entityType = "screens", "name", "screen"
	case 'r':
		table, nameCol, entityType = "regions", "name", "region"
	case 'f':
		table, nameCol, entityType = "flows", "name", "flow"
	case 'e':
		table, nameCol, entityType = "events", "name", "event"
	case 't':
		table, nameCol, entityType = "transitions", "on_event", "transition"
	default:
		return "", 0, "", fmt.Errorf("unknown ref prefix: %c", prefix)
	}

	if err := s.DB.QueryRow(fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", nameCol, table), entityID).Scan(&entityName); err != nil {
		return "", 0, "", fmt.Errorf("ref %s not found", ref)
	}
	return
}

// IsRef returns true if the string looks like a ref (@s1, @r2, etc.)
func IsRef(s string) bool {
	return len(s) >= 3 && s[0] == '@'
}
