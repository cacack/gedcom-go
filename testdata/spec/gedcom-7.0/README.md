# FamilySearch GEDCOM 7.0 extracted specification files

Vendored, unmodified, from the `extracted-files/` directory of
[FamilySearch/GEDCOM](https://github.com/FamilySearch/GEDCOM). Provenance —
commit, what it describes, retrieval date — lives in `SOURCE`, which is the
single place it is recorded; the generated coverage document reads it from
there. Note that a commit is pinned, not a release tag: the two differ, and
some tagged revisions ship these files without a header row.

Upstream generates these files from the specification text itself, so they are
the standards body's own machine-readable statement of the 7.0 structure set,
not a transcription by this project.

## Licensing

Apache-2.0. `LICENSE` and `NOTICE` in this directory are copied from upstream
and satisfy its attribution requirement. Apache-2.0 is compatible with this
project's MIT license.

## Files

| File | Contents |
|------|----------|
| `substructures.tsv` | (superstructure type, substructure tag, substructure type) triples |
| `cardinalities.tsv` | (superstructure type, substructure type, cardinality) triples |
| `payloads.tsv` | (structure type, payload type) pairs |
| `enumerations.tsv` | (structure type, enumeration set) pairs |
| `enumerationsets.tsv` | (enumeration set, enumeration value) pairs |
| `grammar.abnf` | The ABNF of the line and value grammar: `Line`, `Xref`, `DateValue`, `Time`, `Age`, `PersonalName`, `Latitude` and the rest. `SOURCE` records its git blob hash and retrieval date, as it was added after the TSVs |

The superstructure column in `substructures.tsv` is what makes a coverage
inventory possible. GEDCOM tag meaning is context-dependent, so support has to
be reported per (superstructure, tag) pair; a flat per-tag table would overstate
it.

## What consumes them

`decoder/spec7_coverage_test.go` derives
[`docs/reference/gedcom-7-coverage.md`](../../../docs/reference/gedcom-7-coverage.md)
from these files, and `internal/cmd/gengrammar` generates the validator's grammar
tables (`validator/grammar_tables.go`) from them at `go generate` time, as do
`internal/cmd/genplacement` (the decoder's placement tables) and
`internal/cmd/genforeign` (the converter's). `internal/payloadgrammar` reads
`grammar.abnf` as a test oracle for the library's payload parsers; see
[ADR 0013](../../../docs/decisions/0013-payload-grammar-oracle.md). They are
test data only: no library code reads them at run time, and they add no runtime
dependency.

## Updating

Re-download the five TSVs, `grammar.abnf`, `LICENSE` and `NOTICE` from a
single upstream commit, update `SOURCE` (including the `grammar.abnf` blob
hash, `git hash-object grammar.abnf`), then regenerate the coverage document and the
generated tables:

```bash
make spec-coverage
go generate ./validator ./decoder ./converter
```
