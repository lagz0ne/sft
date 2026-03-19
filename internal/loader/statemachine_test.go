package loader

import (
	"testing"

	"github.com/lagz0ne/sft/internal/model"
	"gopkg.in/yaml.v3"
)

func parseYAML(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// doc is a DocumentNode; the mapping is its first child
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		t.Fatal("expected document node with content")
	}
	return doc.Content[0]
}

func findTransition(ts []model.Transition, event, from string) *model.Transition {
	for i := range ts {
		if ts[i].OnEvent == event && ts[i].FromState == from {
			return &ts[i]
		}
	}
	return nil
}

func TestSimpleStringTarget(t *testing.T) {
	node := parseYAML(t, `
start:
  on:
    click: selecting
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	tr := ts[0]
	if tr.OnEvent != "click" {
		t.Errorf("OnEvent = %q, want click", tr.OnEvent)
	}
	if tr.FromState != "start" {
		t.Errorf("FromState = %q, want start", tr.FromState)
	}
	if tr.ToState != "selecting" {
		t.Errorf("ToState = %q, want selecting", tr.ToState)
	}
	if tr.Action != "" {
		t.Errorf("Action = %q, want empty", tr.Action)
	}
	if len(states) != 1 || states[0] != "start" {
		t.Errorf("states = %v, want [start]", states)
	}
}

func TestStayDot(t *testing.T) {
	node := parseYAML(t, `
idle:
  on:
    tick: .
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	tr := ts[0]
	if tr.FromState != "idle" {
		t.Errorf("FromState = %q, want idle", tr.FromState)
	}
	if tr.ToState != "idle" {
		t.Errorf("ToState = %q, want idle (expanded from .)", tr.ToState)
	}
}

func TestActionShorthandNavigate(t *testing.T) {
	node := parseYAML(t, `
viewing:
  on:
    select_email: navigate(thread_view)
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	tr := ts[0]
	if tr.Action != "navigate(thread_view)" {
		t.Errorf("Action = %q, want navigate(thread_view)", tr.Action)
	}
	if tr.ToState != "" {
		t.Errorf("ToState = %q, want empty (action shorthand)", tr.ToState)
	}
}

func TestActionShorthandEmit(t *testing.T) {
	node := parseYAML(t, `
active:
  on:
    done: emit(completed)
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	tr := ts[0]
	if tr.Action != "emit(completed)" {
		t.Errorf("Action = %q, want emit(completed)", tr.Action)
	}
	if tr.ToState != "" {
		t.Errorf("ToState = %q, want empty", tr.ToState)
	}
}

func TestObjectForm(t *testing.T) {
	node := parseYAML(t, `
open:
  on:
    send_reply: { to: collapsed, action: "emit(reply_sent)" }
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	tr := ts[0]
	if tr.FromState != "open" {
		t.Errorf("FromState = %q, want open", tr.FromState)
	}
	if tr.ToState != "collapsed" {
		t.Errorf("ToState = %q, want collapsed", tr.ToState)
	}
	if tr.Action != "emit(reply_sent)" {
		t.Errorf("Action = %q, want emit(reply_sent)", tr.Action)
	}
}

func TestObjectFormWithDotTo(t *testing.T) {
	node := parseYAML(t, `
editing:
  on:
    save: { to: ., action: "emit(saved)" }
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	tr := ts[0]
	if tr.ToState != "editing" {
		t.Errorf("ToState = %q, want editing (expanded from .)", tr.ToState)
	}
	if tr.Action != "emit(saved)" {
		t.Errorf("Action = %q, want emit(saved)", tr.Action)
	}
}

func TestGuardedArray(t *testing.T) {
	node := parseYAML(t, `
form:
  on:
    submit:
      - { guard: "valid", to: saving }
      - { guard: "invalid", to: . }
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(ts))
	}

	// First guard
	t0 := ts[0]
	if t0.FromState != "form" {
		t.Errorf("[0] FromState = %q, want form", t0.FromState)
	}
	if t0.ToState != "saving" {
		t.Errorf("[0] ToState = %q, want saving", t0.ToState)
	}
	if t0.Action != "guard(valid)" {
		t.Errorf("[0] Action = %q, want guard(valid)", t0.Action)
	}

	// Second guard with dot
	t1 := ts[1]
	if t1.ToState != "form" {
		t.Errorf("[1] ToState = %q, want form (expanded from .)", t1.ToState)
	}
	if t1.Action != "guard(invalid)" {
		t.Errorf("[1] Action = %q, want guard(invalid)", t1.Action)
	}
}

func TestGuardedWithAction(t *testing.T) {
	node := parseYAML(t, `
review:
  on:
    approve:
      - { guard: "authorized", to: approved, action: "emit(approved)" }
`)
	ts, _, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1, got %d", len(ts))
	}
	tr := ts[0]
	if tr.Action != "guard(authorized), emit(approved)" {
		t.Errorf("Action = %q, want guard(authorized), emit(approved)", tr.Action)
	}
}

func TestTerminalState(t *testing.T) {
	node := parseYAML(t, `
start:
  on:
    go: done
done: {}
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	// Only 1 transition (from start), done is terminal
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[0] != "start" || states[1] != "done" {
		t.Errorf("states = %v, want [start, done]", states)
	}
}

func TestInitialStateIsFirst(t *testing.T) {
	node := parseYAML(t, `
alpha:
  on:
    next: beta
beta:
  on:
    back: alpha
`)
	_, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if states[0] != "alpha" {
		t.Errorf("initial state = %q, want alpha", states[0])
	}
}

func TestMultipleStatesMultipleTransitions(t *testing.T) {
	node := parseYAML(t, `
start:
  on:
    check_email: selecting
    select_email: navigate(thread_view)
    check_email_again: .
selecting:
  on:
    escape: start
    pick: viewing
empty: {}
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}

	// 3 states
	if len(states) != 3 {
		t.Fatalf("expected 3 states, got %d: %v", len(states), states)
	}
	if states[0] != "start" || states[1] != "selecting" || states[2] != "empty" {
		t.Errorf("states = %v", states)
	}

	// 5 transitions total
	if len(ts) != 5 {
		t.Fatalf("expected 5 transitions, got %d", len(ts))
	}

	// check_email: start -> selecting
	tr := findTransition(ts, "check_email", "start")
	if tr == nil {
		t.Fatal("missing check_email from start")
	}
	if tr.ToState != "selecting" {
		t.Errorf("check_email ToState = %q", tr.ToState)
	}

	// select_email: action shorthand
	tr = findTransition(ts, "select_email", "start")
	if tr == nil {
		t.Fatal("missing select_email from start")
	}
	if tr.Action != "navigate(thread_view)" {
		t.Errorf("select_email Action = %q", tr.Action)
	}

	// check_email_again: stay
	tr = findTransition(ts, "check_email_again", "start")
	if tr == nil {
		t.Fatal("missing check_email_again from start")
	}
	if tr.ToState != "start" {
		t.Errorf("check_email_again ToState = %q, want start", tr.ToState)
	}

	// escape: selecting -> start
	tr = findTransition(ts, "escape", "selecting")
	if tr == nil {
		t.Fatal("missing escape from selecting")
	}
	if tr.ToState != "start" {
		t.Errorf("escape ToState = %q", tr.ToState)
	}

	// pick: selecting -> viewing
	tr = findTransition(ts, "pick", "selecting")
	if tr == nil {
		t.Fatal("missing pick from selecting")
	}
	if tr.ToState != "viewing" {
		t.Errorf("pick ToState = %q", tr.ToState)
	}

	// OwnerType/OwnerID should be zero values
	for _, tr := range ts {
		if tr.OwnerType != "" || tr.OwnerID != 0 {
			t.Errorf("OwnerType/OwnerID should be unset, got %q/%d", tr.OwnerType, tr.OwnerID)
		}
	}
}

func TestEmptyOnBlock(t *testing.T) {
	node := parseYAML(t, `
idle:
  on: {}
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 0 {
		t.Errorf("expected 0 transitions, got %d", len(ts))
	}
	if len(states) != 1 || states[0] != "idle" {
		t.Errorf("states = %v, want [idle]", states)
	}
}

func TestMissingOnKey(t *testing.T) {
	node := parseYAML(t, `
idle:
  description: some state without on
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 0 {
		t.Errorf("expected 0 transitions, got %d", len(ts))
	}
	if len(states) != 1 || states[0] != "idle" {
		t.Errorf("states = %v, want [idle]", states)
	}
}

func TestNullStateDef(t *testing.T) {
	node := parseYAML(t, `
start:
  on:
    go: next
next:
`)
	ts, states, _, err := ParseStateMachine(*node)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(ts))
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[1] != "next" {
		t.Errorf("states[1] = %q, want next", states[1])
	}
}

func TestNotMappingNode(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("- a\n- b"), &doc); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := ParseStateMachine(*doc.Content[0])
	if err == nil {
		t.Error("expected error for non-mapping node")
	}
}
