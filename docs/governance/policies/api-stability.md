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

#### Conformance fixes in minor releases

One exception to the first requirement. A change that alters results on
unchanged input MAY ship in a MINOR release — never a PATCH — when it brings
behaviour into line with the GEDCOM specification or with the library's own
documented contract (its godoc or FEATURES.md). The old behaviour was then a
bug that a careful caller could not have relied on as intended; shipping the
fix promptly is what
[Honor the Standards](../../../CONSTITUTION.md#5-honor-the-standards) asks.
It requires:

1. A row in [Behaviour changes in minor releases](#behaviour-changes-in-minor-releases)
   giving the old and new behaviour and what a caller must do.
2. A commit body stating that the change is a conformance fix that changes
   results, and naming that table.
3. A `make api-check` that shows no incompatible signature changes.

A change that alters what a value *means* by choice rather than to fix a bug —
a renumbered constant, a new default for a zero value, a removed map key, a
reinterpreted field — still requires a major, with everything listed above.
When unsure, treat it as a major.

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
| GEDCOM 7.0 negative assertions (`NO <EVENT>`, `Event.IsNegative`) skipped by the library's event readers | v3.0.0 | A bug fix, listed here because accessor, validator and converter results change for existing callers with no signature change, so `apidiff` reports nothing. `Individual.BirthEvent`/`DeathEvent`/`BirthDate`/`DeathDate` no longer return a negated event or its date period (they return the first non-negated one, or nil). The date-logic rules no longer raise `DEATH_BEFORE_BIRTH`, `MARRIAGE_BEFORE_BIRTH` and the other date checks from a negated event, and the quality report no longer counts one toward its birth, death or place coverage. A 7.0 → 5.5/5.5.1 conversion now actually drops each `NO` (raw `Record.Tags` and the typed `Events` entry) instead of reporting it as lost while still writing it; a document built in code, with no raw tags, now has its negated events reported as data loss too, so `StrictDataLoss` fails on it where v2 succeeded. `Events` itself is unchanged and still holds negated events; the additive `OccurredEvents` and `NegativeAssertions` helpers separate them. See the [migration guide](../../guides/migration-v3.md#negative-assertions-are-no-longer-read-as-events). |
| Inline note text no longer reported as an orphaned `NOTE` reference ([#473](https://github.com/cacack/gedcom-go/issues/473)) | v3.0.0 | A bug fix, listed here because it changes issue counts a caller may have snapshotted. `StreamingValidator` collected note references from the deprecated `Notes` slice, which interleaved pointers with inline text and applied no pointer test — so a record carrying ordinary note prose was reported as an orphaned reference to a record whose XRef was the prose itself. Collection now reads `NoteXRefs`, which holds only pointer-shaped values. Callers who suppressed `ORPHANED_NOTE` wholesale, or who assert on issue counts, will see fewer issues. |

### Behaviour changes in minor releases

Conformance fixes shipped under
[Conformance fixes in minor releases](#conformance-fixes-in-minor-releases).
No signature changed and `apidiff` reports none of them; each row says what a
caller relying on the old behaviour now sees.

| Change | Release | Caller impact |
|--------|---------|---------------|
| `parser.Parse`, `ParseWithOptions`, the decoder and `LazyParser.FindRecord` accept lines up to `MaxLineBytes` (1 MiB) ([#578](https://github.com/cacack/gedcom-go/issues/578)) | v3.1.0 | Lines between 64 KiB and 1 MiB, such as a long GEDCOM 7 `SNOTE`, previously failed with a scanner error; they now parse, and `FindRecord` can read back every record `BuildIndex` indexed. The limit counts content, not the line terminator. Lines over 1 MiB still error. |
| `RecordIterator` and `LazyParser` iterators report exact byte offsets ([#502](https://github.com/cacack/gedcom-go/issues/502)) | v3.1.0 | `ByteOffset`/`ByteLength` previously drifted by the buffered level-0 line per record and by one byte per CRLF line. `LazyParser.Iterate`, `Records`, `IterateFrom` and `RecordsFrom` now report absolute file offsets, so a caller that added its own base after seeking must stop, or it double-counts. |
| `charset.NewReader` and `DetectEncodingFromHeader` probe only the header ([#577](https://github.com/cacack/gedcom-go/issues/577), [#592](https://github.com/cacack/gedcom-go/issues/592), [#593](https://github.com/cacack/gedcom-go/issues/593)) | v3.1.0 | Detection previously read the whole input and searched only its first 1000 bytes for `CHAR`; it now reads the header (up to 64 KiB), so a `CHAR` declared later in a long header is honoured and changes the decoding of that file. A read error during the probe or BOM check is now surfaced after the bytes already read are replayed, where it was swallowed and those bytes dropped. `DetectBOM` returns a non-nil replay reader alongside its error; callers that check `err` first are unaffected. |
| Spaced identifiers accepted in documents declaring GEDCOM 5.5 or 5.5.1 ([#579](https://github.com/cacack/gedcom-go/issues/579), [#591](https://github.com/cacack/gedcom-go/issues/591)) | v3.1.0 | Both grammars allow a space in an identifier. In a document whose header declares 5.5/5.5.1, `0 @I 1@ INDI` no longer fails `StrictMode` or emits `INVALID_XREF`, and a pointer such as `1 NOTE @N 1@` moves from `InlineNotes` to `NoteXRefs`. The new `Header.DeclaredVersion` records the declaration, and `Visit`, `Subset`, `merge` and the validator's broken-pointer check now follow such pointers. Undeclared and 7.0 documents keep the 7.0 rule. |
| `gedcom.Date.Validate` checks range/period end dates and BCE leap years ([#580](https://github.com/cacack/gedcom-go/issues/580)) | v3.1.0 | `BET 1900 AND 31 FEB 1901` now returns an error prefixed "invalid end date" where it returned nil. BCE years are counted astronomically (1 BCE = year 0), so `29 FEB 1 BCE` is now valid and `29 FEB 4 BCE` is not. |
| `gedcom.ParseDate` accepts GEDCOM 7 calendar keywords ([#581](https://github.com/cacack/gedcom-go/issues/581)) | v3.1.0 | `GREGORIAN`, `JULIAN`, `HEBREW` and `FRENCH_R` (case-sensitive) are accepted wherever an `@#D` escape is. Such dates previously failed, leaving `ParsedDate` nil and an `INVALID_VALUE` diagnostic; they now parse, and that diagnostic disappears. |
| `VERS 7.0.N` detected as GEDCOM 7.0 ([#587](https://github.com/cacack/gedcom-go/issues/587)) | v3.1.0 | A spec-conformant patch version such as `7.0.14` was unrecognized and detection fell back to tag heuristics, typically yielding 5.5 and 5.5 decoding rules. It is now 7.0, and `Header.Version` is normalized to `"7.0"`; the raw value stays in `Header.Tags`. |
| `gedcom.Date.ToTime` uses astronomical years and rejects nonexistent days ([#588](https://github.com/cacack/gedcom-go/issues/588), [#601](https://github.com/cacack/gedcom-go/issues/601)) | v3.1.0 | `1 JAN 44 BCE` now converts to year -43 where it gave AD 44, and `YearsBetween` counts across the BCE/CE boundary the same way. `31 FEB 1900` now returns an error where it silently rolled over to 3 March; `YearsBetween` then falls back to an inexact year count. |
| `converter` produces valid output for the target version ([#589](https://github.com/cacack/gedcom-go/issues/589), [#590](https://github.com/cacack/gedcom-go/issues/590), [#602](https://github.com/cacack/gedcom-go/issues/602), [#603](https://github.com/cacack/gedcom-go/issues/603), [#604](https://github.com/cacack/gedcom-go/issues/604), [#605](https://github.com/cacack/gedcom-go/issues/605)) | v3.1.0 | Calendar escapes and keywords are translated in record and header dates. XRefs invalid in 7.0 are rewritten to collision-free identifiers (not just uppercased), including on a 7.0 → 7.0 conversion that previously returned the document unchanged and for a 5.5.x record named `@VOID@`. `VERS` is added under an existing `GEDC` instead of writing a second `GEDC`. Output bytes, identifiers and `ConversionReport` entries change; a caller holding converted identifiers should map them through the report. |
| `validator` reports impossible dates and compares duplicate birth years astronomically ([#606](https://github.com/cacack/gedcom-go/issues/606), [#613](https://github.com/cacack/gedcom-go/issues/613)) | v3.1.0 | `ValidateAll` emits the new `INVALID_DATE` warning for a date such as `31 FEB 1900`, so issue counts can rise; skip it with `ValidateOptions.SkipRules`. Duplicate detection no longer scores 44 BCE and 44 CE as the same birth year, and a birth date with no real year earns no year credit, so some pairs' confidence drops. |
| `gedcom.ParseDate` resolves a short dual-year suffix to the following year, and `Date.Validate` reports a dual year that is not ([#621](https://github.com/cacack/gedcom-go/issues/621)) | v3.2.0 | `1699/00` now gives `DualYear` 1700 where it gave 1600, and `1624/5` gives 1625 where it gave 1605; on a B.C. date the suffix resolves downward, so `100/99 BC` gives 99 where it gave 199. A four-digit suffix is unchanged. `Date.Validate` now errors on a dual year other than the following year (`1750/53`, `1750/1749`, `1850/1860`), and `ValidateAll` reports it as `INVALID_DATE`, so issue counts can rise on files that use a slash for "one year or the other". `ParseDate` still accepts such values and keeps them as written. |
| `validator` date-ordering checks raise `ERROR` only when the dates prove the order ([#566](https://github.com/cacack/gedcom-go/issues/566)) | v3.2.0 | `DEATH_BEFORE_BIRTH`, `CHILD_BEFORE_PARENT` and `MARRIAGE_BEFORE_BIRTH` were an `ERROR` whenever `Compare` put the dates out of order, though `Compare` reads a partial date as its first day and ignores modifiers. They are now an `ERROR` only when the new `gedcom.Date.Bounds` proves the order; otherwise the same code is a `WARNING` whose message says "possibly before" (death `1503` against birth `2 FEB 1503`, `AFT 1078` against `ABT 1090`). `Bounds` reads a dual date in New Style, so death `10 FEB 1731/32` against birth `20 DEC 1731` is no longer reported, and birth `10 FEB 1731/32` against death `20 DEC 1731` is now an `ERROR`. A caller counting or gating on `SeverityError` sees fewer; filter on `Code` to keep every finding. Under `StrictnessRelaxed`, which keeps only errors, the unproven orderings are no longer reported at all; use `StrictnessNormal` or stricter to keep them. `Compare` and `IsBefore` are unchanged by this fix; see the [#563](https://github.com/cacack/gedcom-go/issues/563) row for their dual-date change. |
| `validator` date-ordering checks cap approximate dates at `WARNING` ([#632](https://github.com/cacack/gedcom-go/issues/632)) | v3.2.0 | `DEATH_BEFORE_BIRTH`, `CHILD_BEFORE_PARENT` and `MARRIAGE_BEFORE_BIRTH` are now at most a `WARNING` ("possibly before") when either date is `ABT`, `CAL` or `EST`, even when `Bounds` proves the order: death `ABT 1849` against birth `ABT 1850`, or `1849` against `ABT 1850`, was an `ERROR`. Exact dates and `INT` dates are unchanged, and `Bounds` itself is unchanged. A caller counting or gating on `SeverityError` sees fewer; under `StrictnessRelaxed` these findings are no longer reported at all. |
| `gedcom.PersonalName` types its source citations and notes ([#643](https://github.com/cacack/gedcom-go/issues/643)) | v3.3.0 | `SOUR`, `NOTE` and `SNOTE` under `INDI.NAME` were reachable only through `Record.Tags`; they now also fill the new `SourceCitations`, `NoteXRefs` and `InlineNotes` fields. The fields are additive, but traversal over unchanged input returns more: `Visit` reports a name's citation and note pointers, and `Clone`, `Subset`, `merge.RemapXRefs` and the 5.x → 7.0 calendar conversion now follow them. For a decoded document `Subset` already reached these records through the raw tags, and `RemapXRefs` already rewrote them there, so only the typed copies are new. A hand-built document with name citations and notes now writes them where the encoder previously dropped them; as on every other note carrier, a `NoteXRefs` pointer is written as `NOTE` even for 7.0, never `SNOTE` ([#471](https://github.com/cacack/gedcom-go/issues/471)). A conversion now reports their `DATA.DATE` under `NAME > SOUR`. |
| `gedcom.Date.Compare` and `YearsBetween` read a dual date in New Style ([#563](https://github.com/cacack/gedcom-go/issues/563)) | v3.2.0 | `Compare` ordered a dual date by `Year`, so `11 FEB 1731/32` sorted before `1 DEC 1731`; it now reads the date as `Bounds` does, as 11 FEB 1732, and compares equal to `11 FEB 1732`. A dual date from 1 January to 24 March orders by `DualYear`; one from 25 March, or one `Bounds` cannot place (`1731/32`, `MAR 1731/32`, a dual year that is not the following year, or one not in the Gregorian or Julian calendar), still orders by `Year`. `IsBefore`, `IsAfter`, `IsEqual` and `ValidateBirthBeforeDeath` follow `Compare`. `YearsBetween` counts from the New Style year, so `11 FEB 1731/32` to `11 FEB 1760` is 28 years where it was 29, which moves the validator's age checks by a year on such dates. A caller wanting the old order can compare `Year` directly. |
| `gedcom.YearsBetween` and duplicate detection compare dates in different calendars via JDN ([#618](https://github.com/cacack/gedcom-go/issues/618)) | v3.2.0 | `YearsBetween` subtracted the raw `Year` fields of dates in different calendars, so a Hebrew birth `1 TSH 5660` against a Gregorian death `1 JAN 1950` gave 3710 years and the validator raised `IMPOSSIBLE_AGE` on conformant data. It now converts both dates to Gregorian through their Julian Day Number first, giving 50; complete dates in different calendars now count exactly (`exact` is true) and can differ by a year from the old count (`@#DJULIAN@ 25 DEC 1700` to `25 DEC 1750` is 49 where it was 50). `IMPOSSIBLE_AGE`, `UNREASONABLE_PARENT_AGE`, `ValidateParentChildDates` and `ValidateMarriageDates` follow. A date whose calendar cannot be converted now returns an error naming the first or second date. Duplicate detection likewise compares birth dates in different calendars as the Gregorian years of their first days, reading a dual date in New Style, so Hebrew `1900` and Gregorian `1900` lose the same-birth-year credit and `POTENTIAL_DUPLICATE` confidence moves with it; a day or month the calendar does not have, such as `31 FEB`, is dropped, and such a pair earns no credit only when a year cannot be placed (see `Bounds`). Dates in the same calendar are counted as before. |
| ANSEL combining mark before a line terminator stays on its line ([#431](https://github.com/cacack/gedcom-go/issues/431)) | v3.2.0 | A mark with no base character before LF, CRLF or CR (malformed ANSEL) was emitted after the terminator, so a line after it failed to parse: `StrictMode` decoding returned an error naming the wrong line, and lenient decoding skipped the corrupted line, losing the mark and, after LF or CR, the whole next line. The mark is now emitted before the terminator, as at end of stream, so it combines with the line's last character (`AB` + acute decodes as `"AB\u0301"`) and the file decodes. The reader does not know GEDCOM structure, so a mark whose base starts a following `CONC` line also attaches to the previous line's last character (`abc` + acute, `CONC e` decodes as `"abc\u0301e"`); such input used to fail `StrictMode` decoding. Decoded text and diagnostics change for affected files. |
| `encoder` rejects a line break in a tag name or identifier ([#466](https://github.com/cacack/gedcom-go/issues/466)) | v3.2.0 | A CR or LF in `Tag.Tag`, `Tag.XRef`, `Record.Type`, `Record.XRef` or a hand-built `Header.Submitter` (or an entity field that becomes a tag name, such as `Event.Type`) was written through, producing a line that is not valid GEDCOM and could re-decode as a record the caller never wrote. A line break in a hand-built `Header.Version` (or `EncodeOptions.TargetVersion`), `SourceSystem` or `Language` was written through the same way and is now folded into `CONT` lines, as in any other value; re-decoding that output fills the typed field from the first line and keeps the rest in `Header.Tags`. `Encode`, `EncodeStreaming` and `StreamEncoder.WriteRecord`/`WriteHeader` now return an error wrapping `encoder.ErrLineBreak`; lines before the bad one are already written. Decoded documents never contain one; a caller building tags from external strings must strip line breaks first. |
| `gedcom.Date.ToGregorian` converts a dual date from its New Style year and clears `DualYear` ([#631](https://github.com/cacack/gedcom-go/issues/631)) | v3.2.0 | `ToGregorian` copied `DualYear` unchanged and converted from `Year`, so `@#DJULIAN@ 25 DEC 1750/51` became `5 JAN 1751` with `DualYear` 1751 and failed `Validate`, and `@#DJULIAN@ 11 FEB 1731/32` became `22 FEB 1731` with `DualYear` 1732. Converted dual dates now have `DualYear` 0 and are read as `Bounds` reads the source: those two give `5 JAN 1751` and `22 FEB 1732`. A dual date whose New Style year is unknown (`1731/32`, `MAR 1731/32`, an invalid suffix such as `1750/53`, or a non-Julian dual date such as `@#DHEBREW@ 1 TSH 5660/61`) converts from `Year` and loses its second year, so the result covers only that year and an invalid `DualYear` no longer fails `Validate` on it; validate the source to catch it. A date already Gregorian is returned unchanged. A caller wanting the Old Style year must read it from the source date. |
| `charset` decodes `CHAR ANSI`, `LATIN1` and `ISO-8859-1` as Windows-1252 ([#564](https://github.com/cacack/gedcom-go/issues/564)) | v3.3.0 | Bytes 0x80–0x9F were decoded as ISO-8859-1, which turned the curly quotes, dashes, `€` and other punctuation of a Windows-exported file into invisible C1 control characters: `O\x92Brien` decoded as `"O\u0092Brien"`. They now decode as Windows-1252, so that name is `O’Brien`, and `\x80`, `\x96` and `\x97` give `€`, `–` and `—`. Bytes 0x00–0x7F and 0xA0–0xFF decode as before, and the five bytes Windows-1252 leaves undefined (0x81, 0x8D, 0x8F, 0x90, 0x9D) keep their ISO-8859-1 code points (U+0081 and so on) rather than becoming U+FFFD, so no byte is lost; those five still decode to C1 controls. `DetectEncodingFromHeader` still returns `EncodingLATIN1` for all three labels. Decoded text changes only in files containing a byte in 0x80–0x9F; a caller that stored such text decoded by an earlier version holds C1 controls where this version gives punctuation, and the library offers no way to get the old decoding back. |
| `Source.Title`, `Author`, `Publication`, `Text` and `SourceCitation.Page` hold their `CONT`/`CONC` lines ([#442](https://github.com/cacack/gedcom-go/issues/442)) | v3.3.0 | The decoder kept only the tag's own value, so a multi-line source title or citation page read back as its first line. Each field now folds its continuation lines as `Note.Text` does: `CONT` joins with `"\n"`, `CONC` concatenates. A caller that stored, displayed or compared these fields sees longer values, some containing newlines; one that wants a single line must take the text before the first `"\n"` itself. Encoding is unchanged: a newline in a hand-built value was already written as a `CONT` line, so a document built in code with multi-line values now round-trips them where it previously lost all but the first line on re-decode. |
| A level-0 XRef line with no tag is kept instead of dropped ([#398](https://github.com/cacack/gedcom-go/issues/398)) | v3.4.0 | `0 @I1@` and `0 @I 1@` with nothing after the identifier were dropped in lenient mode, and their subordinate lines silently attached to the preceding record — adding a stray `NAME`, `SEX` or the like to an unrelated individual. `ParseLine` now returns a `Line` with the identifier as `Tag` alongside the same error, `ParseWithOptions` keeps it, and the decoder makes it a record with `Type` set to the identifier, `XRef == ""` and `Entity == nil` — the parse `0 @I1` already had. `Document.Records` gains that record, and the preceding record loses the lines that were never its own. The `INVALID_XREF` diagnostic and strict-mode failure are unchanged. |
| `Header.Encoding` is read from `HEAD.CHAR` at level 1 only ([#464](https://github.com/cacack/gedcom-go/issues/464)) | v3.4.0 | The decoder typed `Header.Encoding` from a `CHAR` at any depth in the header, while charset detection and the encoder's `CHAR` rewrite act only on level 1. A malformed header whose only `CHAR` is nested (`1 SOUR X` / `2 CHAR ANSEL`) now leaves `Header.Encoding` empty, where it held that value though the bytes were never decoded with it; a nested `CHAR` after `1 CHAR` no longer overwrites the level-1 value. The nested line is still kept in `Header.Tags` and written back verbatim, and `UNKNOWN_TAG` still reports it where its context is checked. A caller reading `Header.Encoding` from such a file must read the raw tag instead. |
| `version.DetectVersion` ignores a level-0 `HEAD` that carries an XRef ([#415](https://github.com/cacack/gedcom-go/issues/415)) | v3.4.0 | The GEDCOM grammar gives `HEAD` no identifier, and the decoder already treats `0 @X1@ HEAD` as an ordinary record ([#396](https://github.com/cacack/gedcom-go/issues/396)). Previously a `GEDC`/`VERS` beneath such a line was read as the header's declaration when the real header declared none, so `DetectVersion` and `Document.Header.Version` reported that version; they now fall back to tag heuristics as for any silent header (the record's other tags still count toward those heuristics, like any record's). Likewise, when such a line came before a real header that declares a version, that line's version won; the real header's now does. Documents without an identifier-bearing `HEAD` are unaffected. |
| The decoder reports unknown header tags ([#449](https://github.com/cacack/gedcom-go/issues/449)) | v3.4.0 | `buildHeader` was given no diagnostic collector, so no header line was ever reported, though FEATURES.md documents `UNKNOWN_TAG` for any unrecognized tag. `DecodeWithDiagnostics` now emits `UNKNOWN_TAG` for a non-underscore tag the GEDCOM 5.5, 5.5.1 or 7.0 header grammar does not define in its context — such as `1 ZZZZ` under `HEAD`, or a `1 TIME` beside `1 DATE` rather than under it — so diagnostic counts can rise on files with nonstandard header tags (one of the 102 corpus fixtures gains one). Underscore-prefixed tags stay unreported, every header line stays in `Header.Tags`, and the typed header fields are unchanged. A caller that wants the old silence filters out `UNKNOWN_TAG` diagnostics whose line falls in the header. |
| `validator` reports nil elements in a hand-built document ([#457](https://github.com/cacack/gedcom-go/issues/457)) | v3.4.0 | Same category as the `INVALID_DATE` row above: it changes the issues a document produces. A nil `*Record` in `Document.Records`, a nil element in any slice of a record, its entity or the header, and a `Record.Entity` holding a typed nil were skipped silently by every check, though ADR 0007 assigns reporting them to the validator. `ValidateAll` and `Validate` now emit one `NIL_ELEMENT` error per nil, with its path in `Details["field"]`, and the converter's `ConversionReport.ValidationIssues` lists them for a converted document that still holds nils. A decoded document never holds a nil, so only hand-built input is affected. Skip the code with `ValidateOptions.SkipRules` to keep the old results. |
| `validator.QualityReport` runs every check `ValidateAll` runs ([#544](https://github.com/cacack/gedcom-go/issues/544)) | v3.4.0 | Changes the issues and counts the report produces. `QualityAnalyzer.Analyze` was documented as running "all validators" but ran only date logic, references, duplicates, custom tags and completeness, so `ErrorCount`/`WarningCount` could read clean for a document `ValidateAll` flags; the quality report format is also listed as experimental below. The header, broken-pointer, record-structure, nil-element, date-validity, XRef-length, note-pointer and encoding checks now run too, landing in the new `ConformanceIssues` field (JSON `conformance_issues`) and in `Errors`/`Warnings`/`Info`. `TotalIssues`, `ErrorCount`, `WarningCount` and `InfoCount` rise for any document those checks flag — commonly `MISSING_SUBM` on a document with no header submitter. A caller wanting the old counts recomputes them by severity from the five category slices the report already had (`DateLogicIssues`, `ReferenceIssues`, `DuplicateIssues`, `CompletenessIssues`, `CustomTagIssues`), which are unchanged; filtering the severity slices by code is not equivalent, because a code can come from more than one check. |
| `validator` duplicate detection divides given-name edit distance by character count, not byte count ([#545](https://github.com/cacack/gedcom-go/issues/545)) | v3.4.0 | Levenshtein distance counts runes but was divided by the longer name's byte length, inflating every score for names outside ASCII: unrelated 太郎 and 次郎 scored 0.83 and were reported as duplicates at the default 0.8. Multibyte given names now score lower, so `FindDuplicates`, `FindPotentialDuplicates`, their `Report` forms and `QualityReport` can report fewer pairs and lower `DuplicatePair.Confidence`; ASCII-only names are unchanged. Callers that tuned `MinNameSimilarity` around the inflated scores should re-check it. |
| `merge.RemapXRefs` skips a record whose XRef is not pointer-shaped ([#397](https://github.com/cacack/gedcom-go/issues/397)) | v3.4.0 | A record whose `Record.XRef` fails `Document.IsPointerXRef` — a recovered `@I1` from `0 @I1 INDI`, or `@N 1@` in a document not declaring GEDCOM 5.5/5.5.1 — previously made `RemapXRefs` return a `*RemapError` for the whole document, and `Combine` with `PrefixDoc2` or `RenumberDoc2` fail with it. Such records are now skipped, matching `gedcom.Visit`: they keep their XRef, the transform is not called for them and the mapping has no entry for them, while every other record is remapped. A record named `@VOID@` is still remapped, but a transform may now return `@VOID@` unchanged for it, so `Combine` no longer fails on a 5.5/5.5.1 doc2 holding a non-colliding `@VOID@` record. Calls that failed now succeed; a caller that relied on the error to detect malformed identifiers should check `Document.IsPointerXRef` on each `Record.XRef` instead. |
| `Subset`, `VisitForVersion` and the `validator` broken-pointer check treat `@VOID@` as an ordinary identifier outside GEDCOM 7.0 ([#614](https://github.com/cacack/gedcom-go/issues/614)) | v3.4.0 | `@VOID@` is reserved as the null pointer only in GEDCOM 7.0, and `gedcom.Apply` already keyed on that. In a document whose `Header.Version` is not 7.0 (5.5, 5.5.1, or none, as for a document built in code), `Document.Subset` now follows a `@VOID@` pointer and pulls the `@VOID@` record into the closure (or errors with `ErrUnknownXRef` when no such record exists), accepts `@VOID@` as a seed, and the validator reports a dangling `@VOID@` as `BROKEN_XREF`. `VisitForVersion` reports `@VOID@` when given 5.5 or 5.5.1. 7.0 documents and `Visit` are unchanged. This includes the output of a `converter` downgrade from 7.0 to 5.5/5.5.1, which keeps 7.0 `@VOID@` pointers: `Subset` on it now returns `ErrUnknownXRef` when an included record holds one, where it previously skipped it. A caller that relied on `@VOID@` being skipped in a non-7.0 document should set `Header.Version` to 7.0 or check `Document.IsRecordPointer`. |
| Converted and retargeted headers follow the target grammar and carry `GEDC.FORM` on 5.x ([#465](https://github.com/cacack/gedcom-go/issues/465)) | v3.4.0 | Changes `Header.Tags` after a cross-version `converter.Convert`, the report it returns, and encoder output. Old: a conversion kept the source's level-1 `HEAD` order and appended any `GEDC`/`CHAR` it had to add, so a 5.x header converted to 7.0 kept `GEDC` late and a synthesized `GEDC` or `CHAR` landed last; no 5.5/5.5.1 output ever carried `GEDC.FORM`, which both versions require. New: a cross-version conversion reorders `Header.Tags` into the target grammar (each structure with its subtree; undefined structures last, in source order) and adds `2 FORM LINEAGE-LINKED` under `GEDC` for 5.x targets, reporting `GEDC_FORM_ADDED`; `EncodeOptions.TargetVersion` places a synthesized `GEDC` at its grammar position and writes `FORM` with it for 5.x; a hand-built 5.5/5.5.1 header (no `Tags`) gains `2 FORM LINEAGE-LINKED`. A same-version encode or conversion is unchanged, so byte-identical round trips still hold. A caller indexing `Header.Tags` by position after conversion, or comparing encoded output or report transformations byte-for-byte, must expect the new order and line. |
| The encoder writes a 7.0 pointer to an `SNOTE` record as `SNOTE` ([#471](https://github.com/cacack/gedcom-go/issues/471)) | v3.4.0 | Changes encoder output for records encoded from their entities (empty `Record.Tags`). When the output is GEDCOM 7.0 (`TargetVersion`, or `Header.Version` with no target), a `NoteXRefs`/`SharedNoteXRefs` entry naming an `SNOTE` record in the document was written as `NOTE @X@`, which 7.0 forbids (`NOTE` takes text); it is now `SNOTE @X@`, by `Encode` and `EncodeStreaming` alike. Pointers to a `NOTE` record or to nothing, all 5.5/5.5.1 output, a bare `StreamEncoder.WriteRecord`, and decoded records re-encoded from `Record.Tags` are unchanged. Re-decoding gives the same `NoteXRefs`; a `MediaObject` whose `NoteXRefs` named an `SNOTE` record now re-decodes with that pointer in `SharedNoteXRefs`, where the decoder puts every `SNOTE` pointer. |
| The encoder writes no `CONC` for GEDCOM 7.0 and splits 5.x `CONC` between non-spaces ([#565](https://github.com/cacack/gedcom-go/issues/565)) | v3.4.0 | Changes encoder output for values written from entities (empty `Record.Tags`) that exceed `MaxLineLength`; decoded values are unchanged. Old: a long value was split with `CONC` whatever the output version, so 7.0 output (`TargetVersion`, or `Header.Version` with no target) carried `CONC`, which 7.0 removed; and a 5.x split fell just after a space, so each piece ended in one and readers that trim lines lost a space per split. A split could also cut a multi-byte UTF-8 character in two. New: 7.0 output is never split, as if `DisableLineWrap` were set (the caller's options are not modified); a 5.x split falls between two non-space characters and on a character boundary, so it usually lands mid-word. Callers asserting exact 5.x line content, or relying on `CONC` in 7.0 output, see different lines. A 7.0 encode that is re-decoded and converted down to 5.x no longer carries `CONC` pieces into the result: the converter does not re-split by length and `Record.Tags` are written verbatim, so long values reach the 5.x file as one over-length line, as they already did for 7.0 input converted down. |
| Same-version `converter.Convert` adds a missing `GEDC.VERS` ([#615](https://github.com/cacack/gedcom-go/issues/615)) | v3.4.0 | Changes the result of a same-version conversion of a header whose `GEDC` has no `VERS`. Old: the input document was returned as-is, so its encoded output still declared no version, which 5.5, 5.5.1 and 7.0 all require. New: a deep copy is returned with `2 VERS <target>` inserted as the first child of that `GEDC`; no report entry is added. The input is not mutated. Headers that declare `VERS`, and headers with no `GEDC`, are unchanged. A caller relying on getting the same pointer back must not (the `Convert` contract already says so), and byte-for-byte output comparisons gain the `VERS` line. |

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

### `FamilyLink.Pedigree` stays a scalar

GEDCOM 5.5 defines `CHILD_TO_FAMILY_LINK.FAMC.PEDI` as `{0:M}`
(`testdata/spec/gedcom-5.5/cardinalities.tsv`); 5.5.1 and 7.0 both narrow it to
`{0:1}`. `FamilyLink.Pedigree` is a `string`, and on a 5.5 file that repeats
`PEDI` under one `FAMC` the decoder keeps the last value in the typed field.
Retyping it to `[]string` was proposed for v3 (audit item BW-21, and the scope
amendment on [#472](https://github.com/cacack/gedcom-go/issues/472)) and was
**declined** before v3.0.0 was tagged.

What the asymmetry costs:

- **Decode → encode** is unaffected: `Record.Tags` keeps every `PEDI` line and
  the encoder writes from those tags, so a round trip and `converter.Convert`
  preserve all of them.
- **The typed model and the entity-rebuild path** see one pedigree per link on
  such a file.
- No file in the test corpus repeats `PEDI` under a single `FAMC`; the evidence
  is the 1996 grammar alone.

It is declined because a slice would make every 5.5.1 and 7.0 caller — nearly
all real files — index `Pedigree[0]` behind a length check for a value that can
occur only once, to model a repeat that no observed file uses.

**Closing it later is additive.** A `Pedigrees []string` field (every `PEDI` in
order) can be added in a minor release alongside the scalar, with `Pedigree`
keeping the value it holds today; removing the scalar afterwards would follow
the normal deprecate-then-remove path. A diagnostic for a repeated `PEDI` —
which is invalid in 5.5.1 and 7.0 — is likewise additive. What should trigger
either is evidence per
[CONSTITUTION.md](../../../CONSTITUTION.md#what-counts-as-evidence): real
5.5 files that repeat `PEDI`, or a downstream request.

### GEDCOM 7.0 negative assertions stay in `Events`

A 7.0 negative assertion (`NO <EVEN>`, the `NON_EVENT_STRUCTURE`) is decoded
into `Individual.Events` or `Family.Events` as an `Event` with `IsNegative`
set, next to the events that did happen. The spec models it as a separate
structure: its only substructures are `DATE` — whose payload is a *period* in
which the event did **not** occur — `NOTE`, `SNOTE` and `SOUR`. Keeping it in
`Events` was the deliberate design when negative assertions were added
([#121](https://github.com/cacack/gedcom-go/issues/121)), and moving it into
its own type and slice was reconsidered and **not** taken for v3.0.0.

What the asymmetry costs: a caller that ranges over `Events` without checking
`IsNegative` treats "did not die between 1900 and 1910" as a death. The
library's own readers — `BirthEvent`, `DeathEvent`, the date-logic and quality
validators, the converter's 5.x output — skip or drop negated events, and
`OccurredEvents()` / `NegativeAssertions()` give callers the filtered views
directly. Nothing is lost: the `NO` line, its period and its citations
round-trip.

It is not split out in v3.0.0 because every harm the pre-release audit
reproduced was in library readers and is fixed without changing the shape,
and the only corpus file that uses `NO` is the spec exerciser `maximal70.ged`.
Moving `NO` out of `Events` would itself be a [semantic break](#semantic-breaks)
— the contents of an exported slice change — and removing `IsNegative` would be
a field removal, so it can only happen in a major release. What should trigger
it is evidence per
[CONSTITUTION.md](../../../CONSTITUTION.md#what-counts-as-evidence): vendor
files that use `NO`, or a downstream consumer misreading negated events despite
the helpers.

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
      a godoc note and a guide entry instead. `scripts/check-api-compat.sh`
      enforces this on every branch that declares a breaking change: each
      incompatible change apidiff reports must remove a symbol that carried a
      `Deprecated:` paragraph in the baseline release, or be listed in
      [`scripts/api-compat-allowlist.txt`](../../../scripts/api-compat-allowlist.txt)
      with a migration-guide heading that exists. Retypes, lost comparability
      and changed constant values can only pass through the allowlist. The
      exceptions below predate the gate and are kept as history; the baseline
      is now v3.0.0, so they are not in the allowlist
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
- [ ] The module path is bumped to the new major
      ([#516](https://github.com/cacack/gedcom-go/issues/516)). This step is
      **not optional** for any major release: tagging `vN.0.0` against a
      `/v(N-1)` path makes the release unreachable through `go get`, and
      consumers on `/v(N-1)@latest` never see it. Land it as the **last PR
      before the release-please PR merges** — it rewrites every import, so
      landing it earlier conflicts with every open branch.
      `scripts/check-module-path.sh` (CI and `make preflight`) warns while
      breaking commits are pending on the old path, accepts the bumped path
      once they are, and fails on any other mismatch.

## Stability Note

Version 1.0.0 marked the first stable release with full compatibility guarantees. All packages listed as "Stable" above follow strict semver.

## Reporting Compatibility Issues

If you encounter an unintentional breaking change:

1. Check the [CHANGELOG](../../../CHANGELOG.md) for documented changes
2. Open a [GitHub issue](https://github.com/cacack/gedcom-go/issues) with:
   - Version you upgraded from/to
   - Code that broke
   - Error message or behavior change
