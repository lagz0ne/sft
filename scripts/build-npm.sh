#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLI_DIR="$ROOT/cmd/sft"
NPM_DIR="$ROOT/npm"

SKIP_WEB=false
PUBLISH=false
for arg in "$@"; do
  case "$arg" in
    --skip-web) SKIP_WEB=true ;;
    --publish)  PUBLISH=true ;;
  esac
done

VERSION=$(node -p "require('$NPM_DIR/sft-cli/package.json').version")
echo "Building sft-cli v${VERSION}"

# Build web SPA (required for go:embed)
if [[ "$SKIP_WEB" == false ]]; then
  echo "Building web assets..."
  (cd "$ROOT/web" && bun install --frozen-lockfile && bun run --filter 'web' build)
fi

# npm-dir:GOOS:GOARCH:binary-name
TARGETS=(
  "linux-x64:linux:amd64:sft"
  "linux-arm64:linux:arm64:sft"
  "darwin-x64:darwin:amd64:sft"
  "darwin-arm64:darwin:arm64:sft"
  "win32-x64:windows:amd64:sft.exe"
)

for target in "${TARGETS[@]}"; do
  IFS=: read -r DIR GOOS GOARCH BIN <<< "$target"
  OUT="$NPM_DIR/$DIR/bin/$BIN"
  mkdir -p "$(dirname "$OUT")"

  echo "  ${GOOS}/${GOARCH} → npm/${DIR}/bin/${BIN}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "$OUT" \
    "$CLI_DIR"
done

echo "Done. Binaries in npm/*/bin/"

if [[ "$PUBLISH" == true ]]; then
  for dir in linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64; do
    echo "Publishing sft-cli-${dir}..."
    (cd "$NPM_DIR/$dir" && npm publish --access public)
  done
  echo "Publishing sft-cli..."
  (cd "$NPM_DIR/sft-cli" && npm publish --access public)
fi
