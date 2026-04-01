package loader

import (
	"fmt"
	"strconv"
	"strings"
)

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
			return resolveEntityRef(val, pool, seen)
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

func resolveEntityRef(ref string, pool map[string]any, seen map[string]bool) (any, error) {
	parts := strings.Split(ref[1:], ".")
	name := parts[0]
	if _, ok := pool[name]; !ok {
		return ref, nil // not a ref — literal string
	}
	if seen[name] {
		return nil, fmt.Errorf("entity reference cycle detected: $%s", name)
	}

	seen[name] = true
	resolved, err := resolveValue(pool[name], pool, seen)
	delete(seen, name)
	if err != nil {
		return nil, err
	}
	if len(parts) == 1 {
		return resolved, nil
	}
	return walkEntityPath(ref, resolved, parts[1:])
}

func walkEntityPath(ref string, current any, parts []string) (any, error) {
	for _, part := range parts {
		next, err := walkEntityPathPart(ref, current, part)
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func walkEntityPathPart(ref string, current any, part string) (any, error) {
	field, indexes, err := parseEntityPathPart(part)
	if err != nil {
		return nil, fmt.Errorf("entity ref %s: %w", ref, err)
	}

	if field != "" {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("entity ref %s: cannot access field '%s' on %T", ref, field, current)
		}
		value, ok := obj[field]
		if !ok {
			return nil, fmt.Errorf("entity ref %s: field '%s' not found", ref, field)
		}
		current = value
	}

	for _, index := range indexes {
		items, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("entity ref %s: cannot access index [%d] on %T", ref, index, current)
		}
		if index < 0 || index >= len(items) {
			return nil, fmt.Errorf("entity ref %s: index [%d] out of range", ref, index)
		}
		current = items[index]
	}

	return current, nil
}

func parseEntityPathPart(part string) (string, []int, error) {
	if part == "" {
		return "", nil, fmt.Errorf("invalid empty path segment")
	}

	fieldEnd := strings.IndexByte(part, '[')
	if fieldEnd == -1 {
		return part, nil, nil
	}

	field := part[:fieldEnd]
	rest := part[fieldEnd:]
	indexes := make([]int, 0, strings.Count(rest, "["))
	for rest != "" {
		if rest[0] != '[' {
			return "", nil, fmt.Errorf("invalid path segment %q", part)
		}

		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx == -1 {
			return "", nil, fmt.Errorf("invalid path segment %q", part)
		}

		index, err := strconv.Atoi(rest[1:closeIdx])
		if err != nil {
			return "", nil, fmt.Errorf("invalid path segment %q", part)
		}
		indexes = append(indexes, index)
		rest = rest[closeIdx+1:]
	}

	return field, indexes, nil
}
