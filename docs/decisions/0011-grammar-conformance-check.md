# ADR-011: Grammar Conformance Check Generated from the Vendored Spec Tables

**Status**: Accepted
**Date**: 2026-11-12 ([#742](https://github.com/cacack/gedcom-go/issues/742))
**Context**: Extends the validator architecture of [ADR-008](0008-validator-architecture.md)

## Decision

Add a **grammar conformance check** to the validator that holds every structure in a
document against the structure tables of the version its header declares: permitted
substructures, cardinality, required substructures, enumeration values, payload
datatypes, 7.0 record identifier form, version-foreign tags and 7.0 deprecations.

- The tables are **generated Go code** (`validator/grammar_tables.go`), produced by
  `go generate ./validator` from the specification inventories already vendored under
  `testdata/spec` for the coverage reports. The inventories stay test data: no library
  code opens them, and the module gains no runtime file dependency. A test fails while the
  generated file differs from what the inventories produce.
- The check is **opt-in**: `ValidateOptions.CheckGrammar` for `ValidateAll` and
  `QualityReport` (which files the issues under `GrammarIssues`), and `WithGrammarCheck()`
  for a `QualityAnalyzer`. It follows ADR-008's pattern for an expensive or opinionated
  check, as `CheckEntityTags` does.
- It reads the raw `Tags` (ADR-0003), not the typed entities, because the grammar is about
  lines, and the typed model deliberately keeps what the grammar forbids.
- It applies the **declared** version (`Header.DeclaredVersion`), never the decoder's
  guess, so a 7.1 file is not held to 7.0's tables. A header with no `GEDC.VERS` at all
  falls back to `Header.Version`.

## Context

The validator was a strong data-quality tool and a weak conformance tool: apart from
`HEAD.SUBM` it enforced no required substructure, no cardinality and almost no
enumeration or payload datatype. A file with `SEX Z`, two `SEX` lines, `TIME 25:99` or a
5.5.1 `AGE 0 y` validated clean. The rules it was missing are all in the
`substructures`, `cardinalities`, `payloads` and `enumerations` tables the project already
vendored, which is also where other implementations (js-gedcom, oxidged, python-gedcom7)
derive theirs.

## Considered Options

### Option A: Hand-written rules per structure

- **Pros**: No generator.
- **Cons**: Some 3,700 (superstructure, structure) pairs across three versions; every
  rule a hand transcription that drifts from the inventories the coverage reports read.
- **Verdict**: Rejected.

### Option B: Read the TSVs at run time (embed)

- **Pros**: No generated code.
- **Cons**: Parses some 9,000 rows on first use, ships test data in the library binary,
  and moves table errors from `go generate` time to a caller's run time.
- **Verdict**: Rejected.

### Option C: Generate Go tables from the TSVs (Selected)

- **Pros**: One source of truth for the grammar; table errors fail at generate time; the
  validator only does map lookups.
- **Cons**: A generated file to keep in step, which the up-to-date test enforces.
- **Verdict**: Accepted.

## Consequences

### Positive

- One authoritative home for the grammar: updating an inventory and regenerating updates
  the coverage reports and the validator together.
- gedcom.io's 7.0 test files report nothing except `warnings70.ged`'s designed
  deprecations; gedcom7code's `5/age-invalid.ged` reports one issue per malformed age.

### Negative

- Opt-in, so a caller must ask for it. Default `ValidateAll` output is unchanged.
- Strict by construction: real 5.5.x exports routinely depart from the printed grammar
  (`DEAT Age: 52`, `RIN` under events, `MULTIMEDIA_FORMAT` values beyond the list), and
  each departure is reported as an Error. That is what the check is for; callers wanting
  data quality rather than conformance leave it off.

## Implementation notes

- Payload datatypes not in the tables' own terms are mapped in the generator: 7.0 reads
  the `type-*` and XML Schema URIs; 5.5.x maps the primitives `CHANGE_DATE` and
  `TRANSMISSION_DATE` (DATE_EXACT), `TIME_VALUE`, `AGE_AT_EVENT` and the counts, and treats
  a primitive with a value list as an enumeration, open when it lists `<user defined>`.
- Where 5.5.x prints alternatives for one tag (`NOTE` as a pointer or as text), the line
  is matched to the alternative its payload's shape selects.
- Deprecations are not in the inventories; the 7.0 ones are a short hand-kept list in
  `validator/grammar.go`, citing the specification's deprecation notes.
- Not covered: pointer-target types, pointer cycles (7.0.17), 5.5's 255-byte line limit,
  and a 7.0 record with no identifier, which gedcom.io's `xref.ged` shows is legal.

## References

- `validator/grammar.go` — the check
- `internal/specgrammar` — the generator; `internal/cmd/gengrammar` — its command
- `testdata/spec/README.md` — where each inventory came from
