package loader

import (
	"strings"
	"testing"
)

func TestResolveEntityRefs(t *testing.T) {
	pool := map[string]any{
		"sarah": map[string]any{"name": "Sarah Chen", "email": "sarah@co.com"},
		"deal":  map[string]any{"name": "Acme", "rep": "$sarah"},
	}

	// Simple scalar ref
	result, err := resolveValue("$sarah", pool, nil)
	if err != nil {
		t.Fatalf("resolve $sarah: %v", err)
	}
	m := result.(map[string]any)
	if m["name"] != "Sarah Chen" {
		t.Errorf("$sarah name = %q, want Sarah Chen", m["name"])
	}

	// Nested ref (recursive)
	result2, err := resolveValue("$deal", pool, nil)
	if err != nil {
		t.Fatalf("resolve $deal: %v", err)
	}
	deal := result2.(map[string]any)
	rep := deal["rep"].(map[string]any)
	if rep["name"] != "Sarah Chen" {
		t.Errorf("$deal.rep.name = %q, want Sarah Chen", rep["name"])
	}

	// Array with refs
	result3, err := resolveValue([]any{"$sarah", "$deal"}, pool, nil)
	if err != nil {
		t.Fatalf("resolve array: %v", err)
	}
	arr := result3.([]any)
	if len(arr) != 2 {
		t.Fatalf("array len = %d, want 2", len(arr))
	}

	// Cycle detection
	cyclicPool := map[string]any{
		"a": map[string]any{"ref": "$b"},
		"b": map[string]any{"ref": "$a"},
	}
	_, err = resolveValue("$a", cyclicPool, nil)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q, want to contain 'cycle'", err.Error())
	}

	// Literal dollar string (not a ref)
	result4, err := resolveValue("price is $50", pool, nil)
	if err != nil {
		t.Fatalf("resolve literal: %v", err)
	}
	if result4 != "price is $50" {
		t.Errorf("literal = %q, want 'price is $50'", result4)
	}

	// Non-ref dollar (key not in pool)
	result5, err := resolveValue("$unknown", pool, nil)
	if err != nil {
		t.Fatalf("resolve $unknown: %v", err)
	}
	if result5 != "$unknown" {
		t.Errorf("unknown ref = %q, want '$unknown'", result5)
	}
}

func TestResolveEntityRefPaths(t *testing.T) {
	pool := map[string]any{
		"entity": map[string]any{
			"title": "Pumpkin Soup",
			"author": map[string]any{
				"name": "Ada Lovelace",
			},
			"ingredients": []any{
				map[string]any{"name": "Pumpkin"},
				map[string]any{"name": "Salt"},
			},
		},
	}

	t.Run("one level", func(t *testing.T) {
		result, err := resolveValue("$entity.title", pool, nil)
		if err != nil {
			t.Fatalf("resolve $entity.title: %v", err)
		}
		if result != "Pumpkin Soup" {
			t.Fatalf("$entity.title = %v, want %q", result, "Pumpkin Soup")
		}
	})

	t.Run("two levels", func(t *testing.T) {
		result, err := resolveValue("$entity.author.name", pool, nil)
		if err != nil {
			t.Fatalf("resolve $entity.author.name: %v", err)
		}
		if result != "Ada Lovelace" {
			t.Fatalf("$entity.author.name = %v, want %q", result, "Ada Lovelace")
		}
	})

	t.Run("array access", func(t *testing.T) {
		result, err := resolveValue("$entity.ingredients[0].name", pool, nil)
		if err != nil {
			t.Fatalf("resolve $entity.ingredients[0].name: %v", err)
		}
		if result != "Pumpkin" {
			t.Fatalf("$entity.ingredients[0].name = %v, want %q", result, "Pumpkin")
		}
	})

	t.Run("missing field", func(t *testing.T) {
		_, err := resolveValue("$entity.missing", pool, nil)
		if err == nil {
			t.Fatal("expected missing field error, got nil")
		}
		if err.Error() != "entity ref $entity.missing: field 'missing' not found" {
			t.Fatalf("error = %q", err.Error())
		}
	})

	t.Run("unknown root passes through", func(t *testing.T) {
		result, err := resolveValue("$unknown.field", pool, nil)
		if err != nil {
			t.Fatalf("resolve $unknown.field: %v", err)
		}
		if result != "$unknown.field" {
			t.Fatalf("$unknown.field = %v, want %q", result, "$unknown.field")
		}
	})
}
