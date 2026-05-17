#!/usr/bin/env bash
# Verify go.mod's module-path major version matches the latest git tag's major.
#
# Background: Go's Semantic Import Versioning requires the module path to include
# a /vN suffix for v2 and later (e.g., github.com/foo/bar/v2). If we tag v2.0.0
# without updating go.mod, pkg.go.dev and `go get` will silently keep resolving
# the latest v1 — exactly what happened with our v2.0.0 release.
#
# This guard fails when the majors don't match, before a broken release ships.

set -euo pipefail

MODULE_PATH=$(awk '/^module / {print $2}' go.mod)
if [ -z "$MODULE_PATH" ]; then
  echo "✗ could not read module path from go.mod" >&2
  exit 1
fi

# Extract major from module path (e.g., .../v2 -> 2). Bare path implies v0/v1.
MOD_MAJOR=$(echo "$MODULE_PATH" | sed -nE 's|.*/v([0-9]+)$|\1|p')
MOD_MAJOR=${MOD_MAJOR:-1}

# Latest tag major (excludes pre-release suffixes like v2.0.0-rc1).
LATEST_TAG=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -n1)
if [ -z "$LATEST_TAG" ]; then
  echo "✓ no version tags yet; skipping module-path check"
  exit 0
fi
TAG_MAJOR=$(echo "$LATEST_TAG" | sed -nE 's|^v([0-9]+).*|\1|p')

if [ "$MOD_MAJOR" != "$TAG_MAJOR" ]; then
  cat >&2 <<EOF
✗ Module path major version does not match latest tag.

    go.mod module:  $MODULE_PATH    (major v$MOD_MAJOR)
    latest tag:     $LATEST_TAG    (major v$TAG_MAJOR)

For Go's Semantic Import Versioning, the module path must include /vN for
v2 and later. To fix before releasing v$TAG_MAJOR:

  1. Update go.mod: 'module <path>/v$TAG_MAJOR'
  2. Rewrite imports across the tree (find . -name '*.go' -exec sed ...)
  3. Update README/USAGE/docs to show the new import path

See https://go.dev/ref/mod#major-version-suffixes
EOF
  exit 1
fi

echo "✓ module path ($MODULE_PATH) matches latest tag major ($LATEST_TAG)"
