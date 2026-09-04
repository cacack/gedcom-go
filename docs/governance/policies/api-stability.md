# API Stability Policy

This document defines what constitutes a breaking change and how API stability is managed in gedcom-go.

## Versioning

gedcom-go follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.x.x → 2.0.0): Breaking changes
- **MINOR** (1.0.x → 1.1.0): New features, backward compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, backward compatible

### Automated Enforcement

CI automatically detects breaking API changes using [apidiff](https://pkg.go.dev/golang.org/x/exp/cmd/apidiff). PRs with breaking changes must declare them via conventional commits (`feat!:`, `fix!:`, or `BREAKING CHANGE:` footer). This ensures release-please correctly bumps the major version.

## What Constitutes a Breaking Change

### Breaking (Requires Major Version Bump)

| Category | Examples |
|----------|----------|
| **Remove exported symbol** | Removing `Document.GetIndividual()` |
| **Change function signature** | Adding required parameter, changing return type |
| **Change struct field type** | `Name.Given string` → `Name.Given []string` |
| **Remove struct field** | Removing `Individual.Sex` |
| **Change interface** | Adding method to existing interface |
| **Change behavior semantically** | `ParseDate()` returning different values for same input |
| **Change error types** | Removing error fields that consumers may check |

### Non-Breaking (Minor or Patch)

| Category | Examples |
|----------|----------|
| **Add exported function** | Adding `Document.GetSubmitter()` |
| **Add struct field** | Adding `Individual.FamilySearchID` |
| **Add method to concrete type** | Adding `Date.ToGregorian()` |
| **Fix bug** | Correcting incorrect date parsing |
| **Improve performance** | Faster encoding without API change |
| **Add new type** | Adding `MediaObject` struct |
| **Extend enum/const** | Adding `VendorRootsMagic` constant |

### Semantic Breaks

A semantic break changes what a value *means* without changing any signature.
The caller still compiles; the failure appears at runtime, in their code, on
data they had already stored.

`apidiff` catches some of these and not others. It does report a change to an
exported constant's value, so the `validator.Strictness` renumbering below
shows up under "Incompatible changes" — verify with `make api-check` rather
than assuming either way. What it can never see is a change to which code
paths consult an unchanged field, or to what a function does with unchanged
inputs.

Because the compiler gives the caller no signal at all, a semantic break
requires:

1. A major version, exactly like a signature break.
2. An explicit `BREAKING CHANGE:` footer. State in the commit body whether
   `make api-check` flags it, so a reader is not left guessing.
3. A migration note giving the old-to-new mapping in full — a caller cannot
   diff their way to it.

Known members of this category:

| Change | Release | Caller impact |
|--------|---------|---------------|
| `validator.Strictness` renumbered so `StrictnessNormal` is the zero value ([#489](https://github.com/cacack/gedcom-go/issues/489)) | v3.0.0 | A `Strictness` integer persisted to a config file, database column or API payload changes meaning on upgrade. Old: Relaxed=0, Normal=1, Strict=2. New: Normal=0, Relaxed=1, Strict=2. Reported by `apidiff` as two constant value changes. |
| `encoder.EncodeOptions.LineEnding` defaults to `"\n"` when empty ([#486](https://github.com/cacack/gedcom-go/issues/486)) | v3.0.0 | The field's type and name are unchanged; what the encoder does with an unchanged input changed. An empty `LineEnding` previously wrote every line with no separator, producing one unparseable line; it now writes `"\n"`. `apidiff` reports nothing at all. |
| `validator.Issue.Details["line_number"]` removed ([#498](https://github.com/cacack/gedcom-go/issues/498)) | v3.0.0 | A map key appears in no signature, so `apidiff` cannot see it disappear. A caller reading `Details["line_number"]` gets an empty string instead of a value, silently. Read `Issue.LineNumber` instead. Note that `Details["position"]` survives — it is a byte offset within a field's value, not a source line. |

## Stability Guarantees

### Stable (Full Compatibility Promise)

These packages/APIs are stable and follow semver strictly:

| Package | Status | Notes |
|---------|--------|-------|
| `gedcom` | Stable | Core types: Document, Individual, Family, etc. |
| `decoder` | Stable | `Decode()`, `DecodeWithOptions()` |
| `encoder` | Stable | `Encode()`, `EncodeWithOptions()`, `NewStreamEncoder()`, `NewStreamEncoderWithOptions()`, `EncodeStreaming()`, `EncodeStreamingWithOptions()` |
| `converter` | Stable | `Convert()`, `ConvertWithOptions()` |
| `parser` | Stable | `Parse()`, `ParseLine()`, `NewRecordIterator()`, `NewRecordIteratorWithOffset()`, `Records()`, `RecordsWithOffset()`, `NewLazyParser()` |
| `validator` | Stable | `Validate()`, `ValidateAll()`, `NewStreamingValidator()` |
| `charset` | Stable | `NewReader()` |
| `version` | Stable | `DetectVersion()`, version constants |

### Experimental (May Change)

Features marked experimental may change in minor versions:

- Duplicate detection algorithms
- Quality report format

Experimental features are documented as such in godoc.

## Downstream Consumer Considerations

This library is consumed by [my-family](https://github.com/cacack/my-family). When making changes:

1. **Prefer additive changes** over modifications
2. **Deprecate before removing** - mark deprecated in one minor version, remove in next major
3. **Test downstream** - verify my-family still builds after changes
4. **Document migration** - provide upgrade guidance for breaking changes

## Deprecation Process

1. Add `// Deprecated:` godoc comment explaining replacement
2. Keep deprecated API functional for at least one minor version
3. Remove in next major version
4. Document removal in CHANGELOG

Example:
```go
// Deprecated: Use GetIndividual instead. Will be removed in v2.0.0.
func (d *Document) FindIndividual(xref string) *Individual {
    return d.GetIndividual(xref)
}
```

### Released, Not Merged

**A staged breaking change may not land until its replacement is in a published
tag.** Merged to the default branch is not sufficient. The consumer resolves this
module through the Go module proxy, so an API that exists only on a branch cannot
be called: `go get` will never see it. The same applies to the `// Deprecated:`
marker itself — a marker that has not shipped in a release warns nobody.

The failure this prevents is silent. Each step of a staged removal passes review
on its own, the migration guide names a replacement that really does exist in the
code, and the break still arrives as one combined compile-and-semantics change
for the consumer, because the intermediate step was never obtainable. [#521](https://github.com/cacack/gedcom-go/issues/521)
is the worked example: `Event.PlaceName()` and the `PLAC` encoder fix sat on
`main` for a whole major cycle, unreleasable, while `v2.4.0` remained the newest
tag.

### Cutting the last minor of a major line

The default branch accumulates the next major's breaking commits during a major
cycle, so it cannot produce a release on the outgoing line — usually from its very
first commit after the last tag. That release is cut from a **maintenance branch**
instead:

1. Branch `release-N.x` from the previous tag (e.g. `release-2.x` from `v2.4.0`).
2. Backport only what a migration path names: the additive half of each staged
   change, plus the `// Deprecated:` markers. Cherry-pick where the commit
   applies; port it by hand where the default branch has since restructured the
   code around it.
3. Push. `.github/workflows/ci.yml` and `release-please.yml` both name the
   maintenance branch, and release-please runs with
   `target-branch: ${{ github.ref_name }}`, so the branch gets CI and its own
   release PR driven by its own `.release-please-manifest.json`.
4. Merge that PR to tag. Then verify the consumer can migrate against the tag.

The maintenance branch is terminal — **never merge it back into the default
branch**. Its `CHANGELOG.md` and manifest are branch-local, and the default
branch's manifest must keep naming the pre-major version so release-please still
computes the major there.

### Pre-major release checklist

Before release-please is allowed to cut the major:

- [ ] Every removal in the milestone has a replacement, or is documented as
      having none in [the migration guide](../../guides/migration-v3.md)
- [ ] Every such replacement is in a **published tag**, not merely merged
- [ ] Every removed exported symbol shipped a `// Deprecated:` marker in a
      released minor — a map key or a retyped field cannot carry one, so it gets
      a godoc note and a guide entry instead
- [ ] The consumer has migrated against that tag with no other change
- [ ] The module path is bumped to the new major ([#516](https://github.com/cacack/gedcom-go/issues/516))

## Stability Note

Version 1.0.0 marked the first stable release with full compatibility guarantees. All packages listed as "Stable" above follow strict semver.

## Reporting Compatibility Issues

If you encounter an unintentional breaking change:

1. Check the [CHANGELOG](../../../CHANGELOG.md) for documented changes
2. Open a [GitHub issue](https://github.com/cacack/gedcom-go/issues) with:
   - Version you upgraded from/to
   - Code that broke
   - Error message or behavior change
