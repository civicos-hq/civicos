#!/usr/bin/env bash
# openapi-sync.sh — keep the embedded gateway copies of the canonical
# OpenAPI specs in lock-step with docs/api/openapi-*.yaml.
#
# The api-gateway serves Swagger UI at /docs by `go:embed`-ing files from
# services/api-gateway/internal/docs/openapi/. Because go:embed can't
# reach outside the module root, we can't symlink — we mirror. This
# script is the one place that mirror is enforced.
#
# Usage:
#   scripts/openapi-sync.sh             # sync docs/api → embedded
#   scripts/openapi-sync.sh --check     # exit 1 if they disagree
#
# CI runs the --check mode; developers run the default write mode after
# editing any docs/api/openapi-*.yaml.

set -euo pipefail

# Resolve repo root — scripts/ is a direct child, so parent is root.
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

SRC_DIR="$REPO_ROOT/docs/api"
DST_DIR="$REPO_ROOT/services/api-gateway/internal/docs/openapi"

# Enumerate the specs to keep in sync. New files added under docs/api/
# with the openapi- prefix are picked up automatically. Written to
# survive macOS bash 3.2 (no `mapfile`) as well as CI's bash 5.
SPECS=()
while IFS= read -r spec; do
  SPECS+=("$spec")
done < <(cd "$SRC_DIR" && ls openapi-*.yaml 2>/dev/null | sort)

if [ ${#SPECS[@]} -eq 0 ]; then
  echo "no openapi-*.yaml files found under $SRC_DIR" >&2
  exit 1
fi

MODE="write"
if [ "${1:-}" = "--check" ]; then
  MODE="check"
elif [ -n "${1:-}" ]; then
  echo "unknown argument: $1" >&2
  echo "usage: $0 [--check]" >&2
  exit 2
fi

drift=0
for spec in "${SPECS[@]}"; do
  src="$SRC_DIR/$spec"
  dst="$DST_DIR/$spec"
  if [ ! -f "$dst" ]; then
    if [ "$MODE" = "check" ]; then
      echo "MISSING: $dst (source exists at $src)" >&2
      drift=1
      continue
    fi
    cp "$src" "$dst"
    echo "created  $dst"
    continue
  fi
  if ! diff -q "$src" "$dst" >/dev/null; then
    if [ "$MODE" = "check" ]; then
      echo "DRIFT: $spec" >&2
      diff -u "$dst" "$src" >&2 || true
      drift=1
      continue
    fi
    cp "$src" "$dst"
    echo "synced   $dst"
  fi
done

if [ "$MODE" = "check" ] && [ "$drift" -ne 0 ]; then
  cat >&2 <<EOF

The embedded OpenAPI copies at services/api-gateway/internal/docs/openapi/
do not match docs/api/. Run:

  scripts/openapi-sync.sh

and commit the updated files.
EOF
  exit 1
fi

if [ "$MODE" = "check" ]; then
  echo "OpenAPI mirror is in sync."
fi
