# Specification structure inventories

One directory per GEDCOM version, each holding a machine-readable statement of
what that specification defines. `decoder/spec7_coverage_test.go` and
`decoder/spec55_coverage_test.go` read them to derive
[`docs/reference/gedcom-7-coverage.md`](../../docs/reference/gedcom-7-coverage.md)
and
[`docs/reference/gedcom-5.5-coverage.md`](../../docs/reference/gedcom-5.5-coverage.md),
and `internal/cmd/gengrammar` generates the validator's grammar tables
(`validator/grammar_tables.go`) from them; see
[ADR 0011](../../docs/decisions/0011-grammar-conformance-check.md).

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

Beyond `substructures.tsv`, `cardinalities.tsv` and `payloads.tsv`, which every
version has:

| File | Contents |
|------|----------|
| `primitives.tsv` | Value type, field size, the literal values it enumerates, and the alternatives it prints as placeholders (`<user defined>`, `<LANGUAGE_ID>`) |
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
files are read over the network into memory, parsed, and compared, and none of
their content is written anywhere.

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

### Against oxidged's 5.5.1 errata

[oxidged](https://github.com/trois-six/oxidged) (Rust, MIT) builds its 5.5.1
tables from this transcription and corrects it with
`tools/spec-tables/551-errata.tsv`, read here at commit `97f0629`: 23 rows,
each checked against the PDF. Most found one real gap — the transcription read
a primitive's values only from a list on one line made of plain words, so a list
that wraps, a value with `-` or `/` in it, or a `<user defined>` alternative
left the set empty. `grammar.py` now reads those, and the rows it fixes are
corrected at the source rather than patched into the TSVs.

| Row | Verdict | Why |
|-----|---------|-----|
| `SEX_VALUE` M, F, U | Accepted | p.61 defines the three values in prose with no list above them. Read from the definitions and recorded as a defect (5.5 defines M and F the same way) |
| `SOURCE_MEDIA_TYPE` | Accepted | p.62's list wraps across two lines |
| `LDS_BAPTISM_DATE_STATUS`, `LDS_CHILD_SEALING_DATE_STATUS`, `LDS_SPOUSE_SEALING_DATE_STATUS` | Accepted | pp.51–52: lists that wrap, with `PRE-1970` and `DNS/CAN` as values |
| `LDS_ENDOWMENT_DATE_STATUS` | Accepted | p.52's list omits `INFANT`, which the definitions beneath it define. Read with it added, and recorded as a defect |
| `NAME_TYPE`, `PHONETIC_TYPE`, `ROMANIZED_TYPE`, `ROLE_IN_EVENT` (open) | Accepted | pp.56–61. The literals go to `values`; `<user defined>` and `(<ROLE_DESCRIPTOR>)` go to `placeholders`, so the set reads as open |
| `CHARACTER_SET` | Accepted | p.44: `UTF-8` was not read as a value |
| `GEDCOM_FORM` | Accepted | p.50: `LINEAGE-LINKED` was not read as a value |
| `LANGUAGE_ID` | Accepted | pp.50–51: the table opens with a sentence and is printed as two lists, either side of a page break. Both are read |
| `LANGUAGE_OF_TEXT`, `LANGUAGE_PREFERENCE` → `LANGUAGE_ID` | Accepted | p.51 defines each as `[ <LANGUAGE_ID> ]`, now in `placeholders` |
| `MULTIMEDIA_FILE_REFN` → `MULTIMEDIA_FILE_REFERENCE` | Accepted as a defect | pp.26 and 37: both `FILE` lines name a value type no definition has; p.54, where they point, defines `MULTIMEDIA_FILE_REFERENCE`. Recorded, and the payload left as printed: which primitive was meant is a judgement, not a transcription. The harness already gives both names the same value |
| Calendars `GREGORIAN`, `JULIAN`, `HEBREW`, `FRENCH R` | Accepted in part | The escapes (p.45's `DATE_CALENDAR_ESCAPE`) and the three month lists (`MONTH`, `MONTH_HEBR`, `MONTH_FREN`, pp.52–53) are now read. Which months go with which calendar, and the `B.C.` epoch, are date *syntax* (`DATE_GREG` and the rest), out of scope here: the transcription records value sets, and `gedcom.Date` parses the syntax |
| Calendars `ROMAN`, `UNKNOWN` | Accepted | Both are in `DATE_CALENDAR_ESCAPE`'s list, p.45. That ROMAN is "for future definition" is prose, out of scope |
| `INDIVIDUAL_EVENT_STRUCTURE.EVEN` payload | Rejected | p.35 prints `n EVEN {1:1}` with no payload, and the example beneath it writes `1 EVEN` bare, typed only by `2 TYPE`. The transcription is right about what the grammar says; reading a payload into it would correct the specification, not the transcription |

The Appendix B and Gedcom.pm results above do not change: the structure set is
untouched, and only `primitives.tsv` and `defects.tsv` move.

## Updating

For 7.0, follow [`gedcom-7.0/README.md`](gedcom-7.0/README.md). For 5.5.x,
re-run the transcription — see
[`scripts/spec55/README.md`](../../scripts/spec55/README.md). Then, for either:

```bash
make spec-coverage
go generate ./validator
```
