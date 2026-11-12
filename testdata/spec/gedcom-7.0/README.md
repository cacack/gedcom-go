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

The superstructure column in `substructures.tsv` is what makes a coverage
inventory possible. GEDCOM tag meaning is context-dependent, so support has to
be reported per (superstructure, tag) pair; a flat per-tag table would overstate
it.

## What consumes them

`decoder/spec7_coverage_test.go` derives
[`docs/reference/gedcom-7-coverage.md`](../../../docs/reference/gedcom-7-coverage.md)
from these files, and `internal/cmd/gengrammar` generates the validator's grammar
tables (`validator/grammar_tables.go`) from them at `go generate` time. They are
test data only: no library code reads them at run time, and they add no runtime
dependency.

## Updating

Re-download the five TSVs plus `LICENSE` and `NOTICE` from a single upstream
commit, update `SOURCE`, then regenerate the coverage document and the
validator's grammar tables:

```bash
make spec-coverage
go generate ./validator
```
