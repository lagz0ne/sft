#!/usr/bin/env bash
set -euo pipefail

# SFT Autoresearch Benchmark
# Measures: vocab_count (lower=better), coverage (higher=better)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCHEMA_FILE="$SCRIPT_DIR/README.md"
EXAMPLES_DIR="$SCRIPT_DIR/examples"

# --- 1. Extract defined vocab from the YAML Schema section of README.md ---
# The schema section defines the structural keywords of SFT.
# We parse the schema block between "## YAML Schema" and the next "##" heading.

extract_schema_vocab() {
  local in_schema=0
  local in_code=0
  local vocab=()

  while IFS= read -r line; do
    if [[ "$line" == "## YAML Schema"* ]]; then
      in_schema=1
      continue
    fi
    if [[ $in_schema -eq 1 && "$line" == "## "* ]]; then
      break
    fi
    if [[ $in_schema -eq 1 ]]; then
      if [[ "$line" == '```'* ]]; then
        in_code=$((1 - in_code))
        continue
      fi
      if [[ $in_code -eq 1 ]]; then
        # Extract keywords: word followed by : or word inside [] or standalone keywords
        # Strip comments first
        local clean="${line%%#*}"
        # Match keywords (alphanumeric + underscore) followed by : , ? ) or space/}
        while [[ "$clean" =~ ([a-z_]+)[,:\?\)\ \}] ]]; do
          local kw="${BASH_REMATCH[1]}"
          vocab+=("$kw")
          clean="${clean#*"${BASH_REMATCH[0]}"}"
        done
      fi
    fi
  done < "$SCHEMA_FILE"

  # Also capture prose-documented variants (e.g., "apps:" mentioned outside code block)
  # Scan for backtick-quoted keywords with colon: `keyword:`
  local in_schema2=0
  while IFS= read -r line; do
    if [[ "$line" == "## YAML Schema"* ]]; then
      in_schema2=1
      continue
    fi
    if [[ $in_schema2 -eq 1 && "$line" == "## "* ]]; then
      break
    fi
    if [[ $in_schema2 -eq 1 ]]; then
      while [[ "$line" =~ \`([a-z_]+):\` ]]; do
        vocab+=("${BASH_REMATCH[1]}")
        line="${line#*"${BASH_REMATCH[0]}"}"
      done
    fi
  done < "$SCHEMA_FILE"

  # Deduplicate and sort
  printf '%s\n' "${vocab[@]}" | sort -u
}

# --- 2. Extract keywords used in example files ---

extract_example_keys() {
  local file="$1"
  # Extract all YAML keys (words before a colon at start of line or after spaces)
  grep -oE '^\s*[a-z_]+:' "$file" | sed 's/^\s*//; s/:$//' | sort -u
}

# --- 3. Validate examples against schema vocab ---

validate_example() {
  local file="$1"
  shift
  local schema_vocab=("$@")
  local example_keys
  example_keys=$(extract_example_keys "$file")

  local invalid=0
  while IFS= read -r key; do
    local found=0
    for sv in "${schema_vocab[@]}"; do
      if [[ "$key" == "$sv" ]]; then
        found=1
        break
      fi
    done
    if [[ $found -eq 0 ]]; then
      echo "  UNKNOWN KEY: '$key' in $(basename "$file")" >&2
      invalid=1
    fi
  done <<< "$example_keys"

  return $invalid
}

# --- Run ---

echo "=== SFT Benchmark ===" >&2

# Step 1: Schema vocab
mapfile -t SCHEMA_VOCAB < <(extract_schema_vocab)
VOCAB_COUNT=${#SCHEMA_VOCAB[@]}

echo "Schema vocab ($VOCAB_COUNT):" >&2
printf '  %s\n' "${SCHEMA_VOCAB[@]}" >&2
echo "" >&2

# Step 2: Validate each example
TOTAL=0
PASSED=0

for f in "$EXAMPLES_DIR"/*.sft.yaml; do
  TOTAL=$((TOTAL + 1))
  fname=$(basename "$f")
  if validate_example "$f" "${SCHEMA_VOCAB[@]}"; then
    echo "PASS: $fname" >&2
    PASSED=$((PASSED + 1))
  else
    echo "FAIL: $fname" >&2
  fi
done

echo "" >&2
echo "=== Results ===" >&2
echo "Vocab: $VOCAB_COUNT keywords" >&2
echo "Coverage: $PASSED/$TOTAL examples valid" >&2

# Step 3: Count distinct UI patterns demonstrated across all examples
PATTERN_COUNT=0
PATTERNS_FOUND=()
ALL_EXAMPLES=$(cat "$EXAMPLES_DIR"/*.sft.yaml)
ALL_README=$(cat "$SCHEMA_FILE")

check_pattern() {
  local name="$1"
  local test_result="$2"
  if [[ "$test_result" == "true" ]]; then
    PATTERN_COUNT=$((PATTERN_COUNT + 1))
    PATTERNS_FOUND+=("$name")
  fi
}

# 1. Browse → detail (navigate in states)
check_pattern "browse-detail" "$(echo "$ALL_EXAMPLES" | grep -q 'navigate(' && echo true || echo false)"
# 2. Multi-select / bulk actions (browsing→selecting)
check_pattern "multi-select" "$(echo "$ALL_EXAMPLES" | grep -q 'to: selecting' && echo true || echo false)"
# 3. Confirmation dialogs (overlay + confirm/cancel events)
check_pattern "confirmation-dialog" "$(echo "$ALL_EXAMPLES" | grep -q 'confirm-' && echo true || echo false)"
# 4. Sub-machines (states on a region, not screen)
check_pattern "sub-machine" "$(echo "$ALL_EXAMPLES" | grep -qP '^\s{8,}states:' && echo true || echo false)"
# 5. Event emit (emit in action)
check_pattern "event-emit" "$(echo "$ALL_EXAMPLES" | grep -q 'emit(' && echo true || echo false)"
# 6. Persistent overlay (app-level region with overlay tag)
check_pattern "persistent-overlay" "$(echo "$ALL_EXAMPLES" | grep -q '\[.*overlay.*\]' && echo true || echo false)"
# 7. Cross-app composition (contains tag)
check_pattern "cross-app" "$(echo "$ALL_EXAMPLES" | grep -q 'contains:' && echo true || echo false)"
# 8. Data-conditional regions (has-X/no-X/loading/error tags)
check_pattern "data-conditional" "$(echo "$ALL_EXAMPLES" | grep -qE '\[(has-|no-|loading|error)' && echo true || echo false)"
# 9. Destructive actions (destructive tag)
check_pattern "destructive-action" "$(echo "$ALL_EXAMPLES" | grep -q 'destructive' && echo true || echo false)"
# 10. History re-entry (H) in flows
check_pattern "history-reentry" "$(echo "$ALL_EXAMPLES" | grep -q '(H)' && echo true || echo false)"
# 11. Ambient events (Escape, keyboard shortcuts in states)
check_pattern "ambient-events" "$(echo "$ALL_EXAMPLES" | grep -qE 'on: (Escape|[A-Z][a-z]*)$' && echo true || echo false)"
# 12. Inline editing (viewing→editing sub-machine)
check_pattern "inline-editing" "$(echo "$ALL_EXAMPLES" | grep -q 'to: editing' && echo true || echo false)"
# 13. Search + results
check_pattern "search-results" "$(echo "$ALL_EXAMPLES" | grep -qi 'search' && echo true || echo false)"
# 14. Multi-app (app: as list)
check_pattern "multi-app" "$(echo "$ALL_EXAMPLES" | grep -qP '^app:' "$EXAMPLES_DIR/shopify.sft.yaml" && echo true || echo false)"
# 15. Action weight (primary/secondary tags)
check_pattern "action-weight" "$(echo "$ALL_EXAMPLES" | grep -qE '\[.*(primary|secondary).*\]' && echo true || echo false)"
# 16. Role-based visibility (admin/role tags)
check_pattern "role-based" "$(echo "$ALL_EXAMPLES" | grep -qE '\[.*(admin|role).*\]' && echo true || echo false)"
# 17. Overlay activation in flows
check_pattern "overlay-activation-flow" "$(echo "$ALL_EXAMPLES" | grep -q 'activates' && echo true || echo false)"
# 18. Unhappy path flows
check_pattern "unhappy-path" "$(echo "$ALL_EXAMPLES" | grep -qi 'failed\|error\|retry' && echo true || echo false)"
# 19. Wizard / multi-step form
check_pattern "wizard-multistep" "$(echo "$ALL_EXAMPLES" | grep -qiE 'step|wizard|stepper|multi.step' && echo true || echo false)"
# 20. Tabs within screen
check_pattern "tabs" "$(echo "$ALL_EXAMPLES" | grep -qi 'tab' && echo true || echo false)"
# 21. Drag-and-drop
check_pattern "drag-drop" "$(echo "$ALL_EXAMPLES" | grep -qi 'drag' && echo true || echo false)"
# 22. Toast / snackbar
check_pattern "toast-snackbar" "$(echo "$ALL_EXAMPLES" | grep -qiE 'toast|snackbar|notification.*dismiss' && echo true || echo false)"
# 23. Optimistic update + rollback
check_pattern "optimistic-update" "$(echo "$ALL_EXAMPLES" | grep -qiE 'optimistic|rollback|undo.*action' && echo true || echo false)"
# 24. Infinite scroll / pagination
check_pattern "pagination" "$(echo "$ALL_EXAMPLES" | grep -qiE 'pagination|paginate|load.more|infinite' && echo true || echo false)"
# 25. File upload with progress
check_pattern "file-upload" "$(echo "$ALL_EXAMPLES" | grep -qiE 'upload|progress' && echo true || echo false)"
# 26. Auth flow (login/MFA)
check_pattern "auth-flow" "$(echo "$ALL_EXAMPLES" | grep -qiE 'login|sign.in|mfa|auth|password' && echo true || echo false)"
# 27. Onboarding / tour
check_pattern "onboarding" "$(echo "$ALL_EXAMPLES" | grep -qiE 'onboard|tour|welcome|getting.started' && echo true || echo false)"
# 28. Undo/redo
check_pattern "undo-redo" "$(echo "$ALL_EXAMPLES" | grep -qiE '\bundo\b|\bredo\b' && echo true || echo false)"
# 29. Context menu
check_pattern "context-menu" "$(echo "$ALL_EXAMPLES" | grep -qiE 'context.menu|right.click' && echo true || echo false)"
# 30. Real-time / live data
check_pattern "realtime" "$(echo "$ALL_EXAMPLES" | grep -qiE 'live|real.time|websocket|streaming' && echo true || echo false)"
# 31. Split/master-detail pane
check_pattern "split-pane" "$(echo "$ALL_EXAMPLES" | grep -qiE 'split|pane|master.detail|preview.*pane' && echo true || echo false)"
# 32. Theme toggle
check_pattern "theme-toggle" "$(echo "$ALL_EXAMPLES" | grep -qiE 'theme|dark.mode|light.mode' && echo true || echo false)"
# 33. Responsive / mobile nav
check_pattern "responsive" "$(echo "$ALL_EXAMPLES" | grep -qiE 'responsive|mobile|hamburger|breakpoint' && echo true || echo false)"
# 34. Offline / sync
check_pattern "offline-sync" "$(echo "$ALL_EXAMPLES" | grep -qiE 'offline|sync|reconnect' && echo true || echo false)"
# 35. Autocomplete / search-as-you-type
check_pattern "autocomplete" "$(echo "$ALL_EXAMPLES" | grep -qiE 'autocomplete|typeahead|suggest' && echo true || echo false)"
# 36. Parallel sub-machines
check_pattern "parallel-submachines" "$(grep -c 'states:' "$EXAMPLES_DIR"/*.sft.yaml | awk -F: '{s+=$2} END {print (s>6) ? "true" : "false"}')"

echo "" >&2
echo "=== Patterns ===" >&2
echo "Found $PATTERN_COUNT/36 patterns:" >&2
printf '  %s\n' "${PATTERNS_FOUND[@]}" >&2

# Metrics output (parsed by autoresearch)
echo "METRIC vocab_count=$VOCAB_COUNT"
echo "METRIC coverage=$PASSED/$TOTAL"
echo "METRIC coverage_pct=$((PASSED * 100 / TOTAL))"
echo "METRIC pattern_count=$PATTERN_COUNT"
echo "METRIC pattern_total=36"
