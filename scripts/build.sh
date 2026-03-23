#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLI_DIR="$ROOT/cmd/sft"
BIN_DIR="$ROOT/skills/sft/bin"

VERSION=$(tr -d '[:space:]' < "$BIN_DIR/VERSION")

echo "Building sft v${VERSION}"

# Build web SPA (required for go:embed)
echo "Building web assets..."
(cd "$ROOT/web" && bun install --frozen-lockfile && bun run --filter 'web' build)

TARGETS=(
  "linux:amd64"
  "linux:arm64"
  "darwin:amd64"
  "darwin:arm64"
)

mkdir -p "$BIN_DIR"

for target in "${TARGETS[@]}"; do
  OS="${target%%:*}"
  ARCH="${target##*:}"
  OUT="sft-${VERSION}-${OS}-${ARCH}"

  echo "  ${OS}/${ARCH} → ${OUT}"
  CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build \
    -ldflags="-s -w" \
    -o "$BIN_DIR/$OUT" \
    "$CLI_DIR"
done

chmod +x "$BIN_DIR"/sft-*
echo "Done."
ls -lh "$BIN_DIR"/sft-*
