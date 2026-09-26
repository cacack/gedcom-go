# API Stability Policy

This document defines what constitutes a breaking change and how API stability is managed in gedcom-go.

## Versioning

gedcom-go follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.x.x → 2.0.0): Breaking changes
- **MINOR** (1.0.x → 1.1.0): New features, backward compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, backward compatible

### Automated Enforcement

CI automatically detects breaking API changes using [apidiff](https://pkg.go.dev/golang.org/x/exp/cmd/apidiff). PRs with breaking changes must declare them via conventional commits (`feat!:`, `fix!:`, or `BREAKING CHANGE:` footer). This ensures release-please correctly bumps the major version.

The gate is a floor, not a verdict: it compares signatures, so a change that
alters behaviour without altering a signature passes it clean. See
[Semantic Breaks](#semantic-breaks).

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

**A clean `make api-check` is not evidence that a change is compatible.** It
is evidence only that no exported signature, type property (such as
comparability) or constant value changed. A reviewer must not treat a green
gate as sufficient: the question to ask of every change is whether a caller's
existing code, on the data it already holds, now gets a different answer.
The shapes to look for, each with an in-repo instance from the table below:

- **A constant renumbered** — `validator.Strictness`. (`apidiff` does flag
  this one; it is listed because the incompatibility is in stored values, which
  no tool sees.)
- **A function starting or ceasing to return an error on the same input** —
  `Decode` honouring `StrictMode`; `ParseCoordinate` rejecting non-decimal
  spellings.
- **The contents of an exported map key, slice or field changing** —
  `Issue.Details["line_number"]` removed; `MediaObject.NoteXRefs` and
  `SharedNoteXRefs` partitioning; `Document.Subset` returning an extra record.
- **A zero value changing meaning** — an empty `EncodeOptions.LineEnding` now
  meaning `"\n"`.

Because the compiler gives the caller no signal at all, a semantic break
requires:

1. A major version, exactly like a signature break.
2. An explicit `BREAKING CHANGE:` footer. State in the commit body why the
   change is breaking and whether `make api-check` flags it, so a reader is not
   left guessing.
3. A migration note giving the old-to-new mapping in full — a caller cannot
   diff their way to it.

Known members of this category:

| Change | Release | Caller impact |
|--------|---------|---------------|
| `validator.Strictness` renumbered so `StrictnessNormal` is the zero value ([#489](https://github.com/cacack/gedcom-go/issues/489)) | v3.0.0 | A `Strictness` integer persisted to a config file, database column or API payload changes meaning on upgrade. Old: Relaxed=0, Normal=1, Strict=2. New: Normal=0, Relaxed=1, Strict=2. Reported by `apidiff` as two constant value changes. |
| `encoder.EncodeOptions.LineEnding` defaults to `"\n"` when empty ([#486](https://github.com/cacack/gedcom-go/issues/486)) | v3.0.0 | The field's type and name are unchanged; what the encoder does with an unchanged input changed. An empty `LineEnding` previously wrote every line with no separator, producing one unparseable line; it now writes `"\n"`. `apidiff` reports nothing at all. |
| `decoder.DecodeOptions.StrictMode` honoured by `Decode` and `DecodeWithOptions` ([#490](https://github.com/cacack/gedcom-go/issues/490)) | v3.0.0 | The field was previously read only by `DecodeWithDiagnostics`. `Decode` and `DecodeWithOptions` now return a document with a nil error on input where they returned `nil, err`, because the default `StrictMode: false` recovers from malformed lines. Input that already decoded without error can also change: a level jump is now clamped (`1 BIRT` / `3 DATE` becomes level 2) where v2 kept the original level. A document can arrive together with an error when no line parses or a read fails partway. `apidiff` reports nothing: which code paths consult an unchanged field is invisible to it. A caller relying on v2's behaviour must set `StrictMode: true`. |
| `validator.Issue.Details["line_number"]` removed ([#498](https://github.com/cacack/gedcom-go/issues/498)) | v3.0.0 | A map key appears in no signature, so `apidiff` cannot see it disappear. A caller reading `Details["line_number"]` gets an empty string instead of a value, silently. Read `Issue.LineNumber` instead. Note that `Details["position"]` survives — it is a byte offset within a field's value, not a source line. |
| `gedcom.MediaObject.NoteXRefs` and `.SharedNoteXRefs` partition instead of overlapping ([#499](https://github.com/cacack/gedcom-go/issues/499)) | v3.0.0 | Both fields keep their name and `[]string` type, so `apidiff` reports nothing and `make api-check` is clean; what changed is which pointers each slice carries. Previously every SNOTE pointer was appended to *both*; now `NoteXRefs` holds NOTE pointers only and `SharedNoteXRefs` holds SNOTE pointers only. A caller reading `NoteXRefs` alone no longer sees SNOTE pointers there and must read both. A caller who concatenated the two and deduped must stop deduping — the dedup now discards a genuine repeat rather than an artefact. `MediaObject.AllNotes` already follows both and is unaffected. |
| `validator.Issue.Details["field"]` reports `NoteXRefs[N]` where it reported `Notes[N]` ([#473](https://github.com/cacack/gedcom-go/issues/473)) | v3.0.0 | Same category as the `line_number` row above: a map *value* appears in no signature, so `apidiff` cannot see it change. The streaming validator's note-reference issues now name the field they actually read. A caller pattern-matching `Details["field"]` for a `"Notes["` prefix silently stops matching. Match `"NoteXRefs["` instead. |
| `gedcom.Document.Subset` always includes the header's submitter record ([#503](https://github.com/cacack/gedcom-go/issues/503)) | v3.0.0 | The signature is unchanged and `apidiff` reports nothing; what changed is which records come back. Previously the submitter was included only when a seeded record happened to reference it, and `Header.Submitter` was cleared otherwise. It is now always pulled into the closure, along with anything it references, so **every** subset gains a record — including `Subset(nil)`, which the doc comment previously promised would return an empty document. A caller asserting record counts, or relying on empty-seeds-means-empty, sees a different result with no compile error. Note also that the submitter record carries whatever `Address`, `Phone` and `Email` the source recorded, so those contact details now leave the process in every extract; clear or replace the record if that is unwanted. |
| `encoder` writes hand-built header fields in GEDCOM grammar order ([#503](https://github.com/cacack/gedcom-go/issues/503)) | v3.0.0 | Affects only documents with no raw `Header.Tags` — a decoded document is written from its tags and is unchanged. For a hand-built header the field order changed from `GEDC, CHAR, SOUR, LANG` to `SOUR, SUBM, GEDC, CHAR, LANG` (5.5/5.5.1) or `GEDC, SOUR, SUBM, CHAR, LANG` (7.0). The old order violated the 5.5 grammar. This shifts the bytes for **every** hand-built header, including one that sets no `Submitter` at all, because SOUR, CHAR and LANG move relative to each other regardless. `apidiff` reports nothing; a caller with golden-file or byte-equality tests over encoder output breaks on upgrade. Separately, a hand-built header that set `Submitter` previously emitted no `1 SUBM` line at all — that silent drop is fixed. |
| `ParseCoordinate` and `Coordinates.AsDecimal` reject non-decimal coordinate values ([#504](https://github.com/cacack/gedcom-go/issues/504)) | v3.0.0 | A bug fix, listed here because input that previously returned a value now returns an error, on data a caller may have stored. The numeric part is now an unsigned plain decimal; the wider `strconv.ParseFloat` syntax is refused. `AsDecimal` on `{Latitude: "Nnan", Longitude: "Enan"}` was `(NaN, NaN, nil)` — a success return outside the documented range, since `NaN` fails both range comparisons — and is now an error. `"N1e2"` was `100`, `"N0x1p3"` was `8`, `"N1_0.5"` was `10.5`; all three now error. `apidiff` reports nothing: no signature changed. A caller who checked the error and trusted the value was already correct and needs no change; one who stored a non-decimal spelling must fix the data. |
| Decoding reports bad `MAP` coordinates ([#504](https://github.com/cacack/gedcom-go/issues/504)) | v3.0.0 | Same category as the `ORPHANED_NOTE` row below: it changes the diagnostics a document produces, which a caller may have snapshotted or suppressed. `LATI` and `LONG` previously received no decode-time validation at all. Each is now checked for value syntax, axis direction (`LATI` must use N/S, `LONG` E/W) and range, and a failure emits an `INVALID_VALUE` warning naming the line. The axis and range checks are the ones likely to fire on real data — ordinary data-entry mistakes rather than the pathological spellings in the row above. These are warnings, so `HasErrors` is unaffected and nothing that decoded before fails to decode now; the raw text is still preserved on the record (ADR 0003). |
| Inline note text no longer reported as an orphaned `NOTE` reference ([#473](https://github.com/cacack/gedcom-go/issues/473)) | v3.0.0 | A bug fix, listed here because it changes issue counts a caller may have snapshotted. `StreamingValidator` collected note references from the deprecated `Notes` slice, which interleaved pointers with inline text and applied no pointer test — so a record carrying ordinary note prose was reported as an orphaned reference to a record whose XRef was the prose itself. Collection now reads `NoteXRefs`, which holds only pointer-shaped values. Callers who suppressed `ORPHANED_NOTE` wholesale, or who assert on issue counts, will see fewer issues. |

## Accepted Future Breaks

A break recorded here has already been judged worth its cost. When the change
that causes it lands, it still needs everything any break needs — a major
release, a `BREAKING CHANGE:` footer, a migration-guide entry — but it does
**not** need to re-argue whether it should happen. Being breaking is not, on
its own, a reason to abandon or dilute the work named in an entry.

The register exists because the cheapest time to take a break is inside a
major release that is already breaking, and the obvious shortcut — adding
exported fields early just to spend the window — is ruled out: an exported
field the decoder does not populate is dead surface, the speculative API that
[CONSTITUTION.md](../../../CONSTITUTION.md#anti-patterns-to-avoid) warns
against. When the justifying work cannot land in the current major, the break
is recorded here instead.

| Break | Arrives with | Recorded | Why it is accepted |
|-------|--------------|----------|--------------------|
| `gedcom.Transliteration` stops being comparable with `==` | [#453](https://github.com/cacack/gedcom-go/issues/453) — typed decoding of 5.5.1 `FONE` / `ROMN` onto `PersonalName.Transliterations` | v3.0.0 cycle ([#501](https://github.com/cacack/gedcom-go/issues/501)) | See below. |

### `Transliteration` loses `==` when #453 lands

`Transliteration` is comparable today, and nothing about the type itself needs
that to change: every `NAME-TRAN` substructure is typed, and the 5.5.1 name
pieces it mirrors are `{0:1}`. The pressure comes from #453. 5.5.1's `FONE` and
`ROMN` are accepted and dropped today, and #453 maps both onto
`PersonalName.Transliterations` — onto this type. `FONE` and `ROMN` admit
`NOTE {0:M}` and `SOUR {0:M}` plus a required `TYPE` naming the scheme, so a
lossless mapping gives `Transliteration` at least `SourceCitations` and the
`NoteXRefs` / `InlineNotes` pair. A struct holding a slice cannot be compared,
so the type loses `==` — and use as a map key — the day #453 lands.

The maintainer decision ([#501](https://github.com/cacack/gedcom-go/issues/501))
was not to schedule #453 into v3.0.0, so this break falls to a later major. It
is accepted there because:

- #453 is a fidelity fix for 5.5.1, the most common GEDCOM format in the wild.
  Nothing about the feature is breaking; only the comparability side effect is.
  Gating it on "too breaking" would defer lossless decoding of real files for a
  purely mechanical reason.
- No known downstream call site depends on it: the my-family index surveyed
  for #501 has no use of `Transliteration`, and the type predates the version
  my-family pins, so the absence is genuine non-use rather than a version
  artefact.
- The alternatives were judged worse. Adding the note fields during v3 and
  leaving `SourceCitations` and the scheme field for later would take the same
  break twice. Adding empty fields with no decoder support would be dead
  surface.

Consequently `Transliteration` gains **no** exported field in v3.0.0 that the
decoder does not populate. #453 adds its fields together with the decoding and
encoding that fill them.

## Accepted Asymmetries

An asymmetry recorded here is a deliberate difference between API shapes that
look like they should match. Recording it stops the difference being "fixed"
as a drive-by, and states in advance what closing it later would cost.

### `LDSOrdinance.Place` stays a scalar with no `PlaceDetail` twin

v3.0.0 removed `Event.Place` and `Attribute.Place`, leaving `PlaceDetail` as the
sole place carrier on events and attributes. `LDSOrdinance.Place` is the one
place carrier left as a bare `string`, and it stays that way in v3.0.0
([#523](https://github.com/cacack/gedcom-go/issues/523)).

| Carrier | Typed place | Encoder writes `PLAC` when |
|---------|-------------|----------------------------|
| `Event`, `Attribute` | `PlaceDetail *PlaceDetail` | `PlaceDetail != nil` — a valueless `PLAC` survives |
| `LDSOrdinance` | `Place string` | `Place != ""` — a valueless `PLAC` does not |

What the asymmetry costs, stated so a reader can judge it:

- **Grammar.** In 5.5 and 5.5.1 an ordinance `PLAC` is
  `PLACE_LIVING_ORDINANCE`, a bare value with no subordinates, so the scalar is
  a complete model. GEDCOM 7.0 reuses the general `PLAC` structure there, which
  admits `FORM`, `LANG`, `TRAN`, `MAP`, `EXID` and `NOTE`. The 7.0 coverage
  report lists each structure's substructures once, at its first occurrence
  under an event, so its "typed" status for `PLAC`'s children describes events
  and attributes, not ordinances.
- **Decode → encode** is unaffected. `Record.Tags` preserves an ordinance's raw
  `PLAC` line and everything under it, and the encoder writes from those tags.
- **The entity-rebuild path** — a hand-built document, or one reconstructed
  from typed fields — loses what the scalar cannot hold: subordinates of an
  ordinance `PLAC`, and a `PLAC` line with no value. The decoder reads only an
  ordinance's direct children, so these subordinates produce no
  `UNKNOWN_TAG` diagnostic either; the raw tags are the only record of them.

It is accepted because nothing is lost on the ordinary decode → encode path,
the scalar is already a complete model for 5.5 and 5.5.1, and no corpus
evidence or downstream request yet shows structure under an ordinance `PLAC`.
The cost is confined to typed-model access and entity rebuild.

**Closing it later can be additive — but only if `Place` stays
authoritative.** Adding `PlaceDetail *PlaceDetail` to `LDSOrdinance` adds a
field, and `LDSOrdinance` already holds slices and is not comparable, so a new
pointer field costs nothing further there. The field is not what makes it safe
for a minor release; the encoder's behaviour is. The decoder would fill both
carriers from the same line, so a caller who decodes, edits `ord.Place` and
re-encodes through the entity path must still see the edit written — and a
caller who *clears* `ord.Place` must still see the line disappear, as it does
today. That holds only if `Place` is authoritative for presence as well as
value: the encoder writes `PLAC` only when `Place != ""`, takes the line value
from `Place`, and takes only the subordinates (`FORM`, `MAP`, and so on) from
`PlaceDetail`. Any rule that emits `PLAC` while `Place` is empty — for example
`Place != "" || PlaceDetail != nil` — changes the result of clearing `Place` on
a decoded ordinance and is itself a semantic break. Keeping the two in sync
through accessors does not help either: `LDSOrdinance` has no accessors today,
and callers assign the field directly. This is close to, but not the same as,
how v2's `Event` behaved while it carried both: the v2.5.0 encoder preferred
the `Place` scalar when both were set, but fell back to `PlaceDetail.Name` when
the scalar was empty.

Making the encoder prefer `PlaceDetail` over an edited `Place` would be a
[semantic break](#semantic-breaks): a change to which code path consults an
unchanged field, invisible to `make api-check`, and it would need a major
release. So would *removing* the `Place` scalar afterwards, staged through
`// Deprecated:` like `Event.Place` was. What should trigger any of this is
evidence per [CONSTITUTION.md](../../../CONSTITUTION.md#what-counts-as-evidence):
real files carrying subordinates under an ordinance `PLAC`, or a downstream
request.

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
      - **Named exception (v3.0.0):** `gedcom.Address.Phone`, `.Email` and
        `.Website` ([#494](https://github.com/cacack/gedcom-go/issues/494))
        shipped no marker in `v2.5.0`, and their replacement
        (`Repository.Phone`, `.Email`, `.Fax`, `.Website`,
        [#508](https://github.com/cacack/gedcom-go/issues/508)) exists only in
        v3. The maintainer accepted removing them anyway; the
        [migration guide](../../guides/migration-v3.md#addressphone-email-and-website-in-detail)
        records the exception and the upgrade path.
      - **Named exception (v3.0.0):** `validator.ValidationError`
        ([#500](https://github.com/cacack/gedcom-go/issues/500)) shipped no
        marker in `v2.5.0`; its replacement, `*validator.Issue` from
        `ValidateAll`, has existed throughout v2, so callers can stage the move
        before upgrading. The
        [migration guide](../../guides/migration-v3.md#validationerror-in-detail)
        records the exception and the upgrade path.
      - **Named exception (v3.0.0):** the changes the pre-release API audit
        brought into v3 —
        [#553](https://github.com/cacack/gedcom-go/issues/553)
        (`Source.RepositoryLink`, `SourceRepositoryLink.MediaType`,
        `.CallNumberMedia`), [#554](https://github.com/cacack/gedcom-go/issues/554)
        (scalar `RefNumber`/`UID` on `Individual`, `Family`, `Source`, `Event`)
        and [#555](https://github.com/cacack/gedcom-go/issues/555)
        (`DuplicateConfig.RequireExactSurname`, `.NormalizeNames`) — shipped no
        marker in `v2.5.0`, which predates the audit, and their replacements
        exist only in v3. Deferring them would have meant publishing v3 with
        known data loss and a no-op option until v4; the maintainer accepted
        removing them unmarked. Each has its own migration-guide entry.
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
