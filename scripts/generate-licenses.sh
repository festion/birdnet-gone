#!/usr/bin/env bash
# Generate frontend/static/LICENSES.md from the Go module graph.
#
# Runs at build time (Taskfile `licenses`, called by `frontend-build`), so
# every build that bundles the frontend serves a licence page that matches
# the dependencies it was built from. Vite copies frontend/static/ into
# dist/, and the About page links to /ui/assets/LICENSES.md.
#
# This used to be the "Update Licenses" workflow, which committed the file
# straight to main. The `main` ruleset requires the gitleaks check, which a
# direct push never has, so every run from 2026-09-01 on was refused and the
# page went stale (ops #4230). The file is no longer tracked in git.
set -euo pipefail

GO_LICENSES_VERSION="${GO_LICENSES_VERSION:-v1.6.0}"
MODULE="github.com/tphakala/birdnet-go"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${REPO_ROOT}/frontend/static/LICENSES.md"

cd "$REPO_ROOT"

GOBIN_DIR="$(go env GOBIN)"
GO_LICENSES="${GOBIN_DIR:-$(go env GOPATH)/bin}/go-licenses"
if [[ ! -x "$GO_LICENSES" ]]; then
  echo "Installing go-licenses ${GO_LICENSES_VERSION}"
  go install "github.com/google/go-licenses@${GO_LICENSES_VERSION}"
fi

tmp="$(mktemp)"
csv="$(mktemp)"
trap 'rm -f "$tmp" "$csv"' EXIT

# go-licenses exits non-zero for packages it cannot classify (stdlib, cgo
# packages without a module) while still printing every row it could. The
# workflow this replaces ignored that exit status through a pipe; keep that
# behaviour, and fail below only if no dependency was written at all.
"$GO_LICENSES" csv "${MODULE}/..." >"$csv" 2>/dev/null || true

{
  echo "# Licenses"
  echo ""
  echo "## BirdNET-Go"
  echo ""
  echo "Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International"
  echo "https://creativecommons.org/licenses/by-nc-sa/4.0/"
  echo ""
  echo "## Dependencies"
  echo ""
} >"$tmp"

count=0
while IFS=, read -r package license source; do
  # Skip internal and cmd packages unless they carry their own LICENSE file.
  if [[ $package == *"/internal/"* || $package == *"/cmd/"* ]] && ! [[ -f "${source}/LICENSE" ]]; then
    continue
  fi
  # Skip the main module, and Go files that npm packages ship inside
  # frontend/node_modules (present whenever this runs after `npm ci`).
  if [[ $package == "$MODULE" || $package == "${MODULE}/frontend/node_modules/"* ]]; then
    continue
  fi
  {
    echo "### $package"
    echo ""
    echo "License: $license"
    echo "Source: $source"
    echo ""
  } >>"$tmp"
  count=$((count + 1))
done <"$csv"

if [[ $count -eq 0 ]]; then
  echo "generate-licenses: go-licenses produced no dependency rows; refusing to write an empty licence page" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"
mv "$tmp" "$OUT"
chmod 0644 "$OUT"
trap 'rm -f "$csv"' EXIT
echo "generate-licenses: wrote ${count} dependencies to frontend/static/LICENSES.md"
