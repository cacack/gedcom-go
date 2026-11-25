# ADR-013: The GEDCOM 7.0 ABNF as a Test Oracle for Payload Parsers

**Status**: Accepted
**Date**: 2026-11-25 ([#957](https://github.com/cacack/gedcom-go/issues/957))
**Context**: Extends [ADR-011](0011-grammar-conformance-check.md), which does for the value grammar what that record did for the structure grammar

## Decision

Hold the library's payload parsers and payload checks to the GEDCOM 7.0 value grammar
with the specification's own ABNF, `extracted-files/grammar.abnf`, vendored unmodified
as `testdata/spec/gedcom-7.0/grammar.abnf` at the commit the structure tables are pinned
to. The grammar is a **test oracle, not a runtime dependency**:

- `internal/abnf` reads ABNF (RFC 5234 and the case-sensitive strings of RFC 7405),
  matches a string against a rule, and generates strings from one by a seeded random walk.
  It imports nothing outside the standard library.
- `internal/payloadgrammar` loads the vendored grammar, adds the constraints the
  specification states beside it in prose, and supplies the inputs: the payload values of
  every 7.0 file under `testdata/gedcom-7.0`, `testdata/gedcom7code` and
  `testdata/oxidged` (and `.corpora/gedcomio` when fetched), typed through
  `substructures.tsv` and `payloads.tsv`; edge cases; 3,000 strings generated from each
  rule; and a near-miss mutation of each.
- A test per datatype feeds those inputs to the oracle and to the library, and fails on a
  disagreement unless a row of the
  [spec interpretation register](../governance/policies/spec-interpretation.md) allows
  it. The test names the row by ID, so a departure with no row fails, and so does a row
  that is deleted while the departure remains.

| Datatype | Library side | Test |
|----------|--------------|------|
| `DateValue` | `gedcom.ParseDate` | `gedcom.TestParseDateMatchesGrammar`, `gedcom.FuzzParseDateMatchesGrammar` |
| `Latitude`, `Longitude` | `gedcom.ParseLatitude`, `ParseLongitude` | `gedcom.TestParseLatitudeMatchesGrammar`, `gedcom.TestParseLongitudeMatchesGrammar` |
| `PersonalName` | `gedcom.SplitPersonalName` | `gedcom.TestSplitPersonalNameMatchesGrammar` |
| `Xref` | `gedcom.IsPointerXRef`, `parser.Parser.ParseLine`, the grammar check's identifier and pointer checks | `gedcom.TestIsPointerXRefMatchesGrammar`, `parser.TestParseLineXRefMatchesGrammar`, `validator.TestGrammarCheckXRefsMatchGrammar` |
| `DateExact`, `Time`, `Age` | the grammar check's payload checks | `validator.TestGrammarCheck*MatchesGrammar` |

A test fails when the grammar lacks a rule the oracle names
(`payloadgrammar.TestRequiredRules`), and when the vendored file differs by a byte from
the upstream blob `SOURCE` records (`payloadgrammar.TestVendoredGrammarIsPinned`).

## Context

ADR-011 made the structure grammar mechanical: the tables say a `DATE` payload is a
`DateValue`. What a valid `DateValue` is stayed in hand-written code, never compared with
the specification. Landing this found real disagreements: `ParseDate` read a B.C. epoch
on Hebrew and French Republican dates, the grammar check took `31 FEB 2000` and
`1 JAN 0` as exact dates, and a dozen leniencies were nowhere recorded.

## Considered Options

### Option A: More hand-written test tables

- **Pros**: No new code.
- **Cons**: A table records what its author thought of, and checks the parser against
  the same reading of the specification that wrote it.
- **Verdict**: Rejected.

### Option B: Generate the library's parsers from the ABNF

- **Pros**: The parsers would match the grammar by construction.
- **Cons**: The parsers are deliberately lenient readers of 5.5.x and real exports (the
  register's "Readers are lenient"), a generated recognizer has no place for that, and the
  issue scopes runtime use out.
- **Verdict**: Rejected.

### Option C: An ABNF matcher in test code, as an oracle (Selected)

- **Pros**: The oracle is the standards body's text, not this project's reading of it;
  every departure becomes a named, reasoned register row; generated and fuzzed inputs find
  cases no table lists. No runtime cost and no module dependency.
- **Cons**: A matcher and generator to maintain (a few hundred lines, with their own
  tests), and the prose constraints are still hand-written.
- **Verdict**: Accepted.

## Consequences

### Positive

- A payload parser that accepts what 7.0 forbids, or rejects what it allows, fails a test
  unless the register says why.
- Changing a leniency is visible: the row's disagreement count is logged by each test.

### Negative

- The oracle is only as good as its prose constraints, which the grammar does not carry:
  a calendar's months and epochs, a day's range (the most days its month has in any year,
  so `29 FEB 1900` passes and leap years stay with `Date.Validate`), year 0, the 90 and
  180 degree limits, and `@VOID@`. They live in `payloadgrammar.Oracle.Accepts`, citing
  the sections they come from.
- Some validity depends on the document: a `SCHMA` may document an extension calendar,
  month or epoch as a standard one (gedcom.io's `extensions.ged` writes
  `FRENCH_R 2 _JOUR 8`). The oracle has no document, so it accepts what is valid in some
  document; the library reads no extension calendar at all (SI-018).
- 5.5 and 5.5.1 publish no value grammar and are not checked; their payload rules stay
  in `primitives.tsv`.

## Implementation notes

- The matcher works on code points and returns every end position of every alternative,
  memoized per rule and position, so the order a rule lists its alternatives in does not
  matter (`upto90 = "90" / [upto8] digit` must still match `N9`).
- Rule names are case-insensitive, as RFC 5234 says, which `Latitude`'s `digit` relies
  on; a quoted string is case-insensitive unless `%s`, so `n42` is a valid latitude.
- The generator steers the rules the grammar leaves open (`month = stdTag / extTag`)
  toward the values the prose allows, with near misses among them, and never produces a
  character the grammar's `banned` rule names.
- A disagreement is attributed by undoing one leniency at a time until the oracle accepts
  (whitespace, then case, then the 5.5.x forms, then the day range), naming the row of
  each step it took.

## References

- `internal/abnf` — the ABNF reader, matcher and generator
- `internal/payloadgrammar` — the oracle, its inputs and `Compare`
- `internal/specinterp` — the register the tests look departures up in
- `testdata/spec/gedcom-7.0/README.md` — where the grammar came from
