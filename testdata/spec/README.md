# Specification structure inventories

One directory per GEDCOM version, each holding a machine-readable statement of
what that specification defines. `decoder/spec7_coverage_test.go` and
`decoder/spec55_coverage_test.go` read them to derive
[`docs/reference/gedcom-7-coverage.md`](../../docs/reference/gedcom-7-coverage.md)
and
[`docs/reference/gedcom-5.5-coverage.md`](../../docs/reference/gedcom-5.5-coverage.md).

| Directory | Where it came from |
|-----------|--------------------|
| `gedcom-7.0` | Vendored unmodified from the standards body's own `extracted-files/`. Apache-2.0; see [its own README](gedcom-7.0/README.md) |
| `gedcom-5.5`, `gedcom-5.5.1` | Transcribed from the specification PDFs — the rest of this file |

Every directory's `SOURCE` records its provenance and is the single place that
is recorded.

## Why 5.5.x is transcribed and 7.0 is not

7.0 is published with `extracted-files/`, the standards body's own
machine-readable structure list. 5.5 and 5.5.1 have no equivalent: their
structure set exists only as the semi-formal Lineage-Linked Grammar printed in
Chapter 2 of each PDF. `scripts/spec55/transcribe.py` reads those chapters and
writes the same files 7.0 ships, so one harness can measure all three versions.
How it does that, and what needed judgement, is in
[`scripts/spec55/README.md`](../../scripts/spec55/README.md).

Neither PDF is vendored, and neither is its prose. What is checked in is the
facts read out of them — tag, superstructure, cardinality, value type — plus the
individual grammar lines `defects.tsv` has to quote to say what is wrong with
them. Each document's own terms are recorded verbatim in `SOURCE`: 5.5 permits
copying "for purposes of review or programming of genealogical software,
provided this notice is included", and 5.5.1 states the bare reservation. Rather
than reason about whether a vendored grammar chapter falls inside 5.5's
permission and outside 5.5.1's silence, only the derived data is kept.

## Files the 5.5.x directories add

Beyond `substructures.tsv` and `payloads.tsv`, which every version has:

| File | Contents |
|------|----------|
| `cardinalities.tsv` | (superstructure, structure, cardinality) triples |
| `primitives.tsv` | (value type, field size, literal values) triples |
| `defects.tsv` | Places the printed grammar does not say what it means, with the reading taken |

## How the transcription was checked

`scripts/spec55/crosscheck.py` compares it against two sources made without
reference to it. Re-run it after any change:

```bash
python3 scripts/spec55/crosscheck.py ged55.pdf ged551.pdf
```

### Against the 5.5 specification's own Appendix B

5.5 ships a Structure Cross-Reference listing every place each substructure is
spliced — the grammar chapter and the appendix were typeset from the same
source, so they can be checked against each other. **67 splice sites agree.
Three disagree, and in every case the appendix is wrong and the grammar chapter
is right**, corroborated by 5.5.1 and by Gedcom.pm:

| Difference | Reading taken |
|------------|---------------|
| Appendix B places `<<LDS_INDIVIDUAL_ORDINANCE>>` under `INDIVIDUAL_EVENT_STRUCTURE`; the grammar puts it under `INDIVIDUAL_RECORD` | The grammar chapter's. 5.5.1 also puts it under `INDIVIDUAL_RECORD` |
| Appendix B lists `<<SOURCE_CITATION>>` inside `SOURCE_CITATION` | Not in the grammar chapter, and a citation citing itself is not a structure either document defines |
| Appendix B lists `<<SOURCE_CITATION>>` inside `SOURCE_RECORD` | Not in the grammar chapter, and 5.5.1's `SOURCE_RECORD` has none either |

The page numbers in those appendix rows are wrong too — the self-citing entry
cites p. 24, which is `FAM_RECORD` — so the section reads like a table that was
edited by hand and not re-derived.

### Against Gedcom.pm

[pjcj/Gedcom.pm](https://github.com/pjcj/Gedcom.pm) ships `gedcom-5.5.grammar`
and `gedcom-5.5.1.grammar`, transcriptions of the same chapters by someone else,
in the specification's own notation. **Nothing is copied from them.** Gedcom.pm
is Perl-licensed (Artistic/GPL), which this MIT repository cannot vendor; the
files are fetched to a temporary path, parsed, compared, and discarded.

Its README says the author "had to modify slightly" the standard versions "to
correct a few errors", so the two are *expected* to differ. Each difference
below is a place Gedcom.pm departs from the printed grammar and this follows it.

**5.5: 818 pairs agree. Twelve are in Gedcom.pm and not here, none the reverse.**
All twelve are structures Gedcom.pm adds, not corrections of a printed error:

| Only in Gedcom.pm | What it is |
|-------------------|------------|
| `_EVENT_DEFN` at level 0, with `TYPE`, `TITL`, `ABBR`, `TITL.ABBR` under it | A vendor extension; no specification defines it |
| `ADDR` and `PHON` under `INDI` | Gedcom.pm splices `<<ADDRESS_STRUCTURE>>` into `INDIVIDUAL_RECORD`; neither 5.5 nor 5.5.1 does |
| `REFN` under a source citation's `SOUR` | Added |
| `CONC` and `CONT` under `SOUR.PAGE` | Added |
| `QUAY` under a source *record*'s `SOUR` | Added; 5.5 has `QUAY` only in the citation |
| `ABBR` under `SOUR.AUTH` | Added |

That nothing is in this transcription and absent from Gedcom.pm's is the useful
half of the result: it means no structure here was invented.

**5.5.1: 1,124 pairs agree, two are only here, three only in Gedcom.pm.** Two of
those three are the same disagreement seen from both sides:

| Difference | Why |
|------------|-----|
| Here: `TITL` under `MULTIMEDIA_LINK`'s second `OBJE`. Gedcom.pm: under its `FILE` | 5.5.1 prints `TITL` at `+1`, under `OBJE`. Its `MULTIMEDIA_RECORD` puts the equivalent under `FILE`, so the specification is internally inconsistent; Gedcom.pm harmonizes the two and this does not |
| Here: `MEDI` under that `FORM`. Gedcom.pm: `TYPE` | The same inconsistency: 5.5.1 writes `MEDI` in `MULTIMEDIA_LINK` and `TYPE` in `MULTIMEDIA_RECORD` for the same thing |
| Gedcom.pm: `DATE` under `LDS_INDIVIDUAL_ORDINANCE.CONL.STAT` | A naming artifact, not a disagreement about the grammar. `BAPL` and `CONL` are one alternation line here, so their shared substructures are named for the first; Gedcom.pm writes the branches separately and names them twice. Both allow the same lines |

## Updating

For 7.0, follow [`gedcom-7.0/README.md`](gedcom-7.0/README.md). For 5.5.x,
re-run the transcription — see
[`scripts/spec55/README.md`](../../scripts/spec55/README.md). Then, for either:

```bash
make spec-coverage
```
