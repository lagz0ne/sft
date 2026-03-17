package flow

import (
	"strings"
	"unicode"

	"github.com/lagz0ne/sft/internal/model"
)

// Resolver maps names to IDs for classification. Nil is safe (all steps become "action").
type Resolver interface {
	ResolveScreen(name string) (int64, error)
	ResolveRegion(name string) (int64, error)
	IsEvent(name string) bool
}

// ParseSequence splits a flow sequence string and classifies each token.
func ParseSequence(sequence string, flowID int64, r Resolver) []model.FlowStep {
	sep := "→"
	if !strings.Contains(sequence, "→") {
		sep = ">"
	}
	tokens := strings.Split(sequence, sep)

	var steps []model.FlowStep
	for i, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		step := model.FlowStep{
			FlowID:   flowID,
			Position: i + 1,
			Raw:      tok,
		}
		classify(tok, r, &step)
		steps = append(steps, step)
	}
	return steps
}

func classify(tok string, r Resolver, step *model.FlowStep) {
	// [Back]
	if tok == "[Back]" {
		step.Type = "back"
		step.Name = "Back"
		return
	}

	// "Name activates"
	if strings.HasSuffix(tok, " activates") {
		step.Type = "activate"
		step.Name = strings.TrimSuffix(tok, " activates")
		return
	}

	// Extract data suffix: Name{data}
	name, data := extractBraces(tok)

	// Extract history suffix: Name(H)
	name, history := extractHistory(name)

	step.Name = name
	step.Data = data
	if history {
		step.History = 1
	}

	// Resolve type
	if r == nil {
		step.Type = "action"
		return
	}

	if _, err := r.ResolveScreen(name); err == nil {
		step.Type = "screen"
		return
	}
	if _, err := r.ResolveRegion(name); err == nil {
		step.Type = "region"
		return
	}
	if r.IsEvent(name) {
		step.Type = "event"
		return
	}

	// Fallback: PascalCase heuristic — if first char is upper, guess screen
	if len(name) > 0 && unicode.IsUpper(rune(name[0])) {
		step.Type = "screen"
		return
	}

	step.Type = "action"
}

// extractBraces parses "Name{data}" → ("Name", "data") or ("Name", "").
func extractBraces(s string) (string, string) {
	if idx := strings.Index(s, "{"); idx > 0 && strings.HasSuffix(s, "}") {
		return s[:idx], s[idx+1 : len(s)-1]
	}
	return s, ""
}

// extractHistory parses "Name(H)" → ("Name", true) or ("Name", false).
func extractHistory(s string) (string, bool) {
	if strings.HasSuffix(s, "(H)") {
		return s[:len(s)-3], true
	}
	return s, false
}
