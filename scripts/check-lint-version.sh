#!/usr/bin/env bash
# Verify the golangci-lint version is pinned identically in its two sites.
#
# `make lint` and CI read the same .golangci.yml, so the rules cannot drift,
# but the linter *version* is pinned twice and Dependabot updates neither: the
# gomod ecosystem cannot see a Makefile variable, and the github-actions
# ecosystem bumps the action's `uses:` SHA, not its `version:` input. A
# one-sided bump would let `make lint` and CI run different analyzers, so this
# check fails the moment the two pins disagree (#469).

set -euo pipefail

MAKEFILE=Makefile
WORKFLOW=.github/workflows/ci.yml

MAKE_VERSION=$(sed -nE 's/^GOLANGCI_LINT_VERSION[[:space:]]*:?=[[:space:]]*([^[:space:]]+).*/\1/p' "$MAKEFILE")

# The `version:` input in the `with:` block that follows the
# golangci-lint-action `uses:` line.
CI_VERSION=$(awk '
  /uses:[[:space:]]*golangci\/golangci-lint-action/ { in_step = 1; next }
  in_step && /^[[:space:]]*-[[:space:]]/            { in_step = 0 }
  in_step && /^[[:space:]]*version:/                { sub(/^[[:space:]]*version:[[:space:]]*/, ""); gsub(/["'\'']/, ""); print; exit }
' "$WORKFLOW")

if [ -z "$MAKE_VERSION" ] || [ -z "$CI_VERSION" ]; then
  echo "✗ could not read the golangci-lint version from both sites" >&2
  echo "    $MAKEFILE GOLANGCI_LINT_VERSION:          '${MAKE_VERSION}'" >&2
  echo "    $WORKFLOW golangci-lint-action version: '${CI_VERSION}'" >&2
  exit 1
fi

if [ "$MAKE_VERSION" != "$CI_VERSION" ]; then
  cat >&2 <<MSG
✗ golangci-lint version pins disagree; bump both to the same version:

    $MAKEFILE GOLANGCI_LINT_VERSION:          $MAKE_VERSION
    $WORKFLOW golangci-lint-action version: $CI_VERSION
MSG
  exit 1
fi

echo "✓ golangci-lint pinned to $MAKE_VERSION in $MAKEFILE and $WORKFLOW"
