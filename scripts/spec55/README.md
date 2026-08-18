# Transcribing the GEDCOM 5.5 and 5.5.1 grammars

GEDCOM 7.0 is published with `extracted-files/`, the standards body's own
machine-readable list of every structure the specification defines. 5.5 and
5.5.1 have no equivalent — their structure set exists only as the semi-formal
Lineage-Linked Grammar printed in Chapter 2 of each PDF:

```
HEADER: =
n HEAD                                {1:1}
  +1 SOUR <APPROVED_SYSTEM_ID>        {1:1} p.42
    +2 CORP <NAME_OF_BUSINESS>        {0:1} p.54
      +3 <<ADDRESS_STRUCTURE>>        {0:1} p.31
```

That encodes tag, nesting, cardinality, value type and cross-reference target —
everything a coverage inventory needs — as typeset text. These scripts read it
and write the same files 7.0 ships, so one harness in `decoder/` can measure all
three versions the same way.

They run **by hand**, not in CI, and nothing in the Go module imports them. The
output is checked in under `testdata/spec/`; this is how that output is
reproduced and audited.

## Running

The PDFs are not vendored, so download them first. Only the facts read out of
them are checked in, plus the individual grammar lines `defects.tsv` has to
quote; each document's own copyright terms are recorded verbatim in `SOURCE`.

```bash
curl -sSLO https://gedcom.io/specifications/ged55.pdf
curl -sSLO https://gedcom.io/specifications/ged551.pdf

SPEC55_RETRIEVED=$(date +%F) python3 scripts/spec55/transcribe.py ged55.pdf ged551.pdf
python3 scripts/spec55/crosscheck.py ged55.pdf ged551.pdf

make spec-coverage    # re-derive the published report from the new inventory
```

Python 3.8 or later, standard library only. `transcribe.py` refuses to run
without `SPEC55_RETRIEVED`, since the retrieval date is recorded in `SOURCE`,
and it checks that each PDF states the release it is expected to be, so passing
the two files in the wrong order fails rather than transcribing 5.5 as 5.5.1.

The files it writes record the SHA-256 of the PDF each was read from, so a
reviewer can confirm they are looking at the same bytes.

## The four modules

| File | What it does |
|------|--------------|
| `pdftext.py` | Reads positioned text out of a PDF, with no third-party dependency. Not a general extractor — just enough for these two documents |
| `grammar.py` | Parses the grammar notation into a structure graph: blocks, nesting, splices, alternations |
| `transcribe.py` | Walks that graph into the TSVs, and records provenance |
| `crosscheck.py` | Compares the result against the 5.5 specification's own Appendix B and against Gedcom.pm's independent transcriptions |

`pdftext.py` exists because the grammar chapters are laid out in columns — level
marker, tag, value type, cardinality, page reference, each a separately
positioned run — so an extractor that concatenated runs in drawing order would
interleave the cardinality column into the middle of the tags. It tracks the
text matrix and groups runs into lines by position instead.

## What needed judgement

Four things in the printed grammar are not mechanical, and each is handled in
one named place rather than by special cases scattered through the parser:

- **Splices.** `<<ADDRESS_STRUCTURE>>` contributes the named block's own
  top-level lines at that point. `Grammar._resolve` does that, and composes the
  two cardinalities: a block spliced `{0:M}` times, each allowing `{1:1}` of a
  tag, permits `{0:M}` of it. Across an alternation the branches are mutually
  exclusive, so the lower bound is not composed — `RECORD` is spliced `{1:M}`
  and each of its seven branches is `{1:1}`, which would otherwise read as
  though a conformant file must contain one of every record type.
- **Lines written beneath a splice.** 5.5's `FAM_RECORD` nests `HUSB` and `WIFE`
  under `<<FAMILY_EVENT_STRUCTURE>>`, meaning they are substructures of every
  family event tag rather than of the family. `Grammar.collect_extras` attaches
  them, and `Grammar.verify` fails if a block needing that treatment is ever
  spliced from two places, which is what would make the global attachment wrong.
- **Where a block ends.** The first prose line after a block's first grammar
  line ends it. That is what keeps 5.5.1's "some systems may have output the
  following 5.5 structure" illustration — written in grammar notation, inside
  `MULTIMEDIA_LINK`'s prose — out of the grammar.
- **Structure identity.** A structure is named for the production that defines
  it and the path of tags to it inside that production, so one block spliced ten
  places stays one structure. Where an alternation reuses a tag within a block,
  the repeat is suffixed `#2` rather than merged: both documents define `NOTE`
  twice in `NOTE_STRUCTURE`, once as a pointer and once as inline text, and they
  are genuinely different structures.

## Specification defects

Neither document is typographically clean. Cardinalities close with `)` or `]`,
footnote markers land inside the braces, value types lose an angle bracket,
cross-references are written with and without theirs, and two lines state no
cardinality at all.

The primitive definitions are no cleaner: one states `SIZE` where the rest state
`Size`, and four give a single width where the rest give a range.

**Nothing is corrected.** Every defect is recorded to `defects.tsv` with the
line as printed and the reading taken, and published in the coverage report. A
transcription that quietly repaired its source would describe a document that
does not exist.

`transcribe.py` fails rather than writing output if any grammar line is not
understood, so a defect it has no reading for stops the run instead of silently
dropping a structure.

## Checking it

`crosscheck.py` compares the transcription against two sources made without
reference to it — the 5.5 specification's own Appendix B, and
[pjcj/Gedcom.pm](https://github.com/pjcj/Gedcom.pm)'s `.grammar` files. Gedcom.pm
is Perl-licensed and nothing is copied from it: the files are fetched to a
temporary path, read, compared, and discarded.

The current differences, and why each is expected, are recorded in
[`testdata/spec/README.md`](../../testdata/spec/README.md). A new difference
after a change to the transcription is the signal to look.
