#!/usr/bin/env bash
# Check the exported API against the latest release on this major line, and
# require any incompatible change to be declared as a breaking change.
#
# This is the single home for the rule: `make api-check`, `make preflight` and
# the CI api-compatibility job all run this script.
#
# The baseline is the highest release tag (pre-release suffixes excluded) whose
# major is at most the major in go.mod's module path. It is deliberately not
# `git describe`: that only sees tags in HEAD's history, so it skips releases
# cut on a maintenance branch (v2.5.0 lives on release-2.x, not main). The
# major cap keeps release-N.x comparing against its own line once vN+1 exists,
# while a vN+1 cycle that has not yet bumped the module path still compares
# against the newest vN release wherever it was tagged.
#
#   - apidiff errors                       -> fail, showing its output
#   - baseline module path differs         -> skip (distinct modules under SIV)
#   - compatible changes only              -> pass, listing them
#   - incompatible, declared since baseline -> pass, listing them
#   - incompatible, not declared           -> fail
#
# "Declared" means a commit in "$TAG"..HEAD (i.e. HEAD --not $TAG, so it also
# works when the tag is not an ancestor) carries a conventional-commit breaking
# marker, the same regex scripts/check-module-path.sh uses.
#
# Requires full history (actions/checkout with fetch-depth: 0) and apidiff
# (make install-tools).

set -euo pipefail

# Conventional-commit breaking markers: "type(scope)!:" subjects or a
# "BREAKING CHANGE:" / "BREAKING-CHANGE:" footer.
BREAKING_RE='^[a-z]+(\([^)]*\))?!:|^BREAKING[ -]CHANGE:'

module_path() {
  awk '/^module / {print $2}' "$1"
}

# Major version implied by a module path (e.g., .../v3 -> 3). Bare path implies v1.
module_major() {
  local major
  major=$(echo "$1" | sed -nE 's|.*/v([0-9]+)$|\1|p')
  echo "${major:-1}"
}

# Highest non-prerelease version tag whose major is <= $1. Empty if none.
baseline_tag() {
  # awk reads all input (no early exit) so pipefail never sees a SIGPIPE.
  git tag --list 'v[0-9]*' --sort=-v:refname |
    awk -v max="$1" '/-/ || found { next }
      { major = $0; sub(/^v/, "", major); sub(/\..*/, "", major) }
      major + 0 <= max + 0 { found = $0 }
      END { print found }'
}

# Locate apidiff the same way the Makefile locates its other tools.
resolve_apidiff() {
  local bin
  bin=$(command -v apidiff || echo "$HOME/go/bin/apidiff")
  if [ ! -x "$bin" ]; then bin="$(go env GOPATH 2>/dev/null)/bin/apidiff"; fi
  if [ ! -x "$bin" ]; then
    echo "✗ apidiff not found. Run 'make install-tools'" >&2
    exit 1
  fi
  echo "$bin"
}

error() {
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::error::$1"
  fi
  echo "✗ $1" >&2
}

main() {
  local apidiff new_module tag old_dir api_file old_module out status commits
  apidiff=$(resolve_apidiff)

  new_module=$(module_path go.mod)
  if [ -z "$new_module" ]; then
    echo "✗ could not read module path from go.mod" >&2
    exit 1
  fi

  tag=$(baseline_tag "$(module_major "$new_module")")
  if [ -z "$tag" ]; then
    if [ "$(git rev-parse --is-shallow-repository)" = "true" ]; then
      echo "✗ shallow clone with no version tags; the API compatibility check" >&2
      echo "  needs them (actions/checkout fetch-depth: 0, or git fetch --unshallow --tags)" >&2
      exit 1
    fi
    echo "✓ no release tags found; skipping API compatibility check"
    exit 0
  fi
  echo "Comparing API against $tag"

  old_dir=$(mktemp -d)
  api_file=$(mktemp)
  # shellcheck disable=SC2064 # expand now: locals are gone when EXIT fires
  trap "rm -rf '$old_dir' '$api_file'" EXIT
  git archive "$tag" | tar -x -C "$old_dir"

  # Different module paths are different modules under Go's Semantic Import
  # Versioning, so apidiff would report every type as changed.
  old_module=$(module_path "$old_dir/go.mod")
  if [ "$old_module" != "$new_module" ]; then
    echo "✓ module path changed ($old_module -> $new_module); skipping apidiff."
    echo "  The $tag module and the current module are distinct under"
    echo "  Go's Semantic Import Versioning rules and can coexist."
    exit 0
  fi

  (cd "$old_dir" && go mod download && "$apidiff" -m -w "$api_file" .)
  go mod download

  set +e
  out=$("$apidiff" -m "$api_file" . 2>&1)
  status=$?
  set -e

  # Here-strings, not pipes: grep -q exiting early would SIGPIPE the writer
  # and pipefail would turn a match into a failure.
  if ! grep -q "Incompatible changes:" <<<"$out"; then
    if [ "$status" -ne 0 ]; then
      error "apidiff failed to run:"
      echo "$out"
      exit 1
    fi
    echo "✓ no breaking API changes against $tag"
    if [ -n "$out" ]; then
      echo ""
      echo "Compatible changes:"
      echo "$out"
    fi
    exit 0
  fi

  echo "⚠ Breaking API changes detected against $tag:"
  echo "$out"
  echo ""

  commits=$(git log --format='%s%n%b' "$tag"..HEAD)
  if grep -qE "$BREAKING_RE" <<<"$commits"; then
    echo "✓ Breaking change properly declared in commit message"
    echo "  Release-please will bump major version"
    # Deprecation gate hook: every declared incompatible change must also have
    # shipped a Deprecated: marker or be allowlisted. Invoked here (#557).
    exit 0
  fi

  echo ""
  error "Breaking API changes detected but not declared in commits!"
  cat >&2 <<EOF

To fix, use one of these conventional commit formats:
  feat!: description of breaking change
  fix!: description of breaking change
  feat: description

  BREAKING CHANGE: explanation of what breaks

This ensures release-please creates a major version bump.
EOF
  exit 1
}

# Run only when executed, so the functions can be sourced and exercised.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
