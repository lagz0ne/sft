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
        # Match keywords (alphanumeric + underscore) followed by : or ,
        while [[ "$clean" =~ ([a-z_]+)[,:\?\)] ]]; do
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

# Metrics output (parsed by autoresearch)
echo "METRIC vocab_count=$VOCAB_COUNT"
echo "METRIC coverage=$PASSED/$TOTAL"
echo "METRIC coverage_pct=$((PASSED * 100 / TOTAL))"
