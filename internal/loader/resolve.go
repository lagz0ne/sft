package loader

import "fmt"

// resolveValue recursively resolves $name references in arbitrary data.
// $name is recognized only as a standalone string value matching a pool key.
// Strings like "price is $50" where "$50" is not a pool key pass through unchanged.
func resolveValue(v any, pool map[string]any, seen map[string]bool) (any, error) {
	if seen == nil {
		seen = make(map[string]bool)
	}
	switch val := v.(type) {
	case string:
		if len(val) > 1 && val[0] == '$' {
			name := val[1:]
			if _, ok := pool[name]; !ok {
				return val, nil // not a ref — literal string
			}
			if seen[name] {
				return nil, fmt.Errorf("entity reference cycle detected: $%s", name)
			}
			seen[name] = true
			resolved, err := resolveValue(pool[name], pool, seen)
			delete(seen, name)
			return resolved, err
		}
		return val, nil
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			r, err := resolveValue(v2, pool, seen)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			r, err := resolveValue(v2, pool, seen)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	default:
		return val, nil
	}
}
