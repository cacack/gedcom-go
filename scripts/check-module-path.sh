#!/usr/bin/env bash
# Verify go.mod's module-path major version matches the major the next release
# from this commit will carry.
#
# Background: Go's Semantic Import Versioning requires the module path to include
# a /vN suffix for v2 and later (e.g., github.com/foo/bar/v2). If we tag v2.0.0
# without updating go.mod, pkg.go.dev and `go get` will silently keep resolving
# the latest v1 — exactly what happened with our v2.0.0 release.
#
# The expected major is derived from the newest version tag in HEAD's history
# (not the newest tag anywhere, so a maintenance branch such as release-2.x keeps
# checking against its own line) plus the commits since it:
#
#   - no breaking commit since the tag  -> the next release keeps the tag's major
#   - a breaking commit since the tag    -> release-please will cut major+1
#
# A module path that matches the expected major passes. A path still on the
# tag's major while breaking commits are pending passes with a warning: the
# /vN bump rewrites every import and conflicts with every open branch, so it
# lands as the last PR before the release PR merges, not with the first
# breaking change. Anything else fails.
#
# Requires full history (actions/checkout with fetch-depth: 0).

set -euo pipefail

MODULE_PATH=$(awk '/^module / {print $2}' go.mod)
if [ -z "$MODULE_PATH" ]; then
  echo "✗ could not read module path from go.mod" >&2
  exit 1
fi

# Extract major from module path (e.g., .../v2 -> 2). Bare path implies v0/v1.
MOD_MAJOR=$(echo "$MODULE_PATH" | sed -nE 's|.*/v([0-9]+)$|\1|p')
MOD_MAJOR=${MOD_MAJOR:-1}

# Newest release tag reachable from HEAD (pre-release suffixes excluded).
LATEST_TAG=$(git describe --tags --abbrev=0 --match 'v[0-9]*' --exclude 'v*-*' HEAD 2>/dev/null || true)
if [ -z "$LATEST_TAG" ]; then
  if [ "$(git rev-parse --is-shallow-repository)" = "true" ]; then
    echo "✗ shallow clone with no version tag in the fetched history; the module-path" >&2
    echo "  check needs it (actions/checkout fetch-depth: 0, or git fetch --unshallow --tags)" >&2
    exit 1
  fi
  echo "✓ no version tags in history; skipping module-path check"
  exit 0
fi
TAG_MAJOR=$(echo "$LATEST_TAG" | sed -nE 's|^v([0-9]+).*|\1|p')

# Conventional-commit breaking markers: "type(scope)!:" subjects or a
# "BREAKING CHANGE:" / "BREAKING-CHANGE:" footer.
BREAKING=$(git log --format='%s%n%b' "$LATEST_TAG"..HEAD |
  grep -cE '^[a-z]+(\([^)]*\))?!:|^BREAKING[ -]CHANGE:' || true)

EXPECTED_MAJOR=$TAG_MAJOR
if [ "$BREAKING" -gt 0 ] && [ "$TAG_MAJOR" -ge 1 ]; then
  EXPECTED_MAJOR=$((TAG_MAJOR + 1))
fi

if [ "$MOD_MAJOR" = "$EXPECTED_MAJOR" ]; then
  if [ "$EXPECTED_MAJOR" != "$TAG_MAJOR" ]; then
    echo "✓ module path ($MODULE_PATH) matches the pending v$EXPECTED_MAJOR release ($BREAKING breaking change(s) since $LATEST_TAG)"
  else
    echo "✓ module path ($MODULE_PATH) matches latest tag major ($LATEST_TAG)"
  fi
  exit 0
fi

if [ "$MOD_MAJOR" = "$TAG_MAJOR" ] && [ "$EXPECTED_MAJOR" != "$TAG_MAJOR" ]; then
  MSG="$BREAKING breaking change(s) since $LATEST_TAG: the next release will be v$EXPECTED_MAJOR, but go.mod still declares $MODULE_PATH. Bump the module path to /v$EXPECTED_MAJOR in the last PR before the release PR merges (docs/governance/policies/api-stability.md, pre-major release checklist)."
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::warning title=Module path bump pending::$MSG"
  fi
  echo "⚠ $MSG"
  exit 0
fi

cat >&2 <<EOF
✗ Module path major version does not match the next release.

    go.mod module:   $MODULE_PATH    (major v$MOD_MAJOR)
    latest tag:      $LATEST_TAG    (major v$TAG_MAJOR)
    breaking since:  $BREAKING commit(s)
    next release:    major v$EXPECTED_MAJOR

For Go's Semantic Import Versioning, the module path must include /vN for
v2 and later. To fix:

  1. Update go.mod: 'module <path>/v$EXPECTED_MAJOR'
  2. Rewrite imports across the tree (find . -name '*.go' -exec sed ...)
  3. Update README/USAGE/docs to show the new import path

See https://go.dev/ref/mod#major-version-suffixes
EOF
exit 1
