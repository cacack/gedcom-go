# GEDCOM Library Comparison

> Comparing gedcom-go against other GEDCOM parsing libraries

## Overview

This document compares gedcom-go (github.com/cacack/gedcom-go) against other
GEDCOM libraries, in Go and in other languages, to help users choose a library
and to show where gedcom-go has gaps.

**Comparison scope:**
- Go libraries: elliotchance/gedcom, iand/gedcom, deaddr0p/go-gedcom (formerly
  funwithbots/go-gedcom), MJKWoolnough/gedcom
- Other languages: Rust (oxidged, gedlint), C# (GedcomSharp,
  ArmidaleSoftware/gedcom7), Python (python-gedcom7, python-gedcom), Java
  (gedcom5-java, gedcom4j), TypeScript/JavaScript (read-gedcom, js-gedcom)

**Research date:** November 2026. Every cell below was checked against the
library's repository (README, `go.mod`, source or release list) on that date. A
property that could not be established was left out rather than marked
"Unknown". gedcom-go's own claims link to [FEATURES.md](FEATURES.md), which is
authoritative for them.

---

## Go Libraries

| | gedcom-go | elliotchance | iand | deaddr0p (funwithbots) | MJKWoolnough |
|---|---|---|---|---|---|
| **License** | MIT | MIT | Unlicense | GPL-3.0 | ISC |
| **GEDCOM 5.5 / 5.5.1** | Read + write | Read + write | Read + write (core records) | No (roadmap) | Read (5.5) |
| **GEDCOM 7.0** | Read + write | No | Read + write (since 2026) | Read + write (7.0.11) | No |
| **Version conversion** | 5.5 ↔ 5.5.1 ↔ 7.0 | No | No | No (roadmap) | No |
| **Record-at-a-time reading** | Yes | No | No (`Decode` returns the whole file) | No | Yes (`Reader.Record`) |
| **ANSEL decoding** | Yes | No | No | No | No (ANSEL bytes are accepted, not converted) |
| **UTF-16** | Yes | No | No | No | No |
| **Direct dependencies** | 1 (`golang.org/x/text`) | 6 | 1 (`go-cmp`) | 4 | 1 |
| **Latest release** | v3.6.0 (Nov 2026) | v39.6.0 (Nov 2023) | v0.2.20 (May 2026) | v0.1.2 (Aug 2023) | v1.0.1 (tag) |
| **Stars** | 7 | 124 | 41 | 8 | 2 |

**Analysis:**
- iand/gedcom added GEDCOM 7.0 reading and writing in February 2026
  (iand/gedcom PR #50), including `TRAN`, so gedcom-go is no longer the only Go
  library with 5.5, 5.5.1 and 7.0, nor the only one handling transliterations.
- gedcom-go remains the only Go library here that converts between versions,
  decodes ANSEL and UTF-16, and reads record by record.
- elliotchance/gedcom is the broadest Go toolkit (query language, HTML output,
  comparison and merging) but has had no release since 2023 and does not read
  GEDCOM 7.0.

### API Shape

gedcom-go, elliotchance/gedcom, iand/gedcom and MJKWoolnough/gedcom read from an
`io.Reader` (elliotchance/gedcom also offers `NewDocumentFromGEDCOMFile`);
deaddr0p/go-gedcom reads a file name or a `bufio.Scanner`. MJKWoolnough/gedcom yields one record per
`Reader.Record` call; iand/gedcom and gedcom-go decode a whole document into
typed structs. gedcom-go additionally exposes `parser.Records` and
`decoder.DecodeRecord` for record-at-a-time typed access — see
[FEATURES.md](FEATURES.md) for the full API.

---

## Other Languages

| Library | Language | License | Versions | Notable | Latest release |
|---------|----------|---------|----------|---------|----------------|
| [oxidged](https://github.com/trois-six/oxidged) | Rust | MIT | 5.5.1, 7.0, 7.1 draft (read + write) | Conversion, streaming, GEDZIP, ANSEL/UTF-16, producer profiles, editing | v0.3.2 (Oct 2026) |
| [gedlint](https://github.com/pmontp19/gedlint) | Rust | MIT | 5.5.1, 7.0 | A linter (CLI, GitHub Action, web viewer), not a library | v0.12.0 (Oct 2026) |
| [GedcomSharp](https://github.com/gordon-matt/GedcomSharp) | C# | MIT | 5.5, 5.5.1, 5.5.5, 7.0 (read + write) | GEDZIP, GEDCOM X, ANSEL/UTF-16 | v1.0.1 (Nov 2026) |
| [ArmidaleSoftware/gedcom7](https://github.com/ArmidaleSoftware/gedcom7) | C# | MIT | 7.0 | Validator and FamilySearch compatibility tooling | v1.6.2 (Aug 2026) |
| [python-gedcom7](https://github.com/DavidMStraub/python-gedcom7) | Python | MIT | 7.0 (read + write since 1.x) | Strict: rejects non-conforming input | v1.2.0 (Oct 2026) |
| [python-gedcom](https://github.com/nickreynke/python-gedcom) | Python | GPL-2.0 | 5.5 | | v1.1.0 (Feb 2026) |
| [gedcom5-java](https://github.com/FamilySearch/gedcom5-java) | Java | Apache-2.0 | 5.x (read + write) | A "de facto" model: common non-standard tags in, rarely used standard tags out | 1.15.0 (tag) |
| [gedcom4j](https://github.com/frizbog/gedcom4j) | Java | MIT | 5.5, 5.5.1 (read + write) | Last commit Jun 2023 | 4.0.1 (tag) |
| [read-gedcom](https://github.com/arbre-app/read-gedcom) | TypeScript | MIT | 5.5.5-oriented (read) | Encoding detection incl. UTF-16, CP1252, CP850 | 0.3.2 (Jun 2022) |
| [js-gedcom](https://github.com/gedcom7code/js-gedcom) | JavaScript | MIT | 7.0 (read + write) | Type-checker and validator | No releases; last commit Feb 2024 |

rust-gedcom (last commit Jan 2023) and GeneGenie.Gedcom (last commit Mar 2025),
listed in earlier editions, are superseded in their languages by oxidged and by
GedcomSharp and ArmidaleSoftware/gedcom7 respectively.

**Analysis:**
- oxidged and GedcomSharp are the closest peers outside Go: both read and write
  several versions, preserve unknown data, and decode legacy encodings.
- Both read and write GEDZIP packages, as gedcom-go does
  ([FEATURES.md](FEATURES.md#gedzip-archives-gdz)). oxidged goes further than
  gedcom-go in one place: the GEDCOM 7.1 draft.

---

## gedcom-go Strengths

1. **Version conversion in Go.** The only Go library here that converts 5.5 ↔
   5.5.1 ↔ 7.0.
2. **Lossless representation.** Every readable line is kept, typed or not
   ([ADR 0003](docs/decisions/0003-lossless-dual-storage.md)); dates keep their
   original text beside the parsed form.
3. **Encodings.** UTF-8, UTF-16 LE/BE, ANSEL, ASCII, Windows-1252/1250 and DOS
   and Mac code pages, detected automatically
   ([FEATURES.md](FEATURES.md#character-encoding)). No other Go library here
   decodes ANSEL or UTF-16.
4. **Measured coverage.** Typed coverage of each version is measured, not
   claimed ([FEATURES.md](FEATURES.md#multi-version-support)).
5. **Streaming.** Record-at-a-time parse, typed decode, validation and encoding.
6. **Document operations.** Traversal, subset extraction, deep copy, merge and
   duplicate detection ([FEATURES.md](FEATURES.md#document-operations)).
7. **Validation.** Structural, version-specific and semantic checks (death
   before birth, child before parent, and others), plus a quality report.
8. **Published benchmarks.** Figures and the environment they were measured on
   live in [docs/guides/performance.md](docs/guides/performance.md).

---

## gedcom-go Gaps

Since the January 2026 edition, relationship traversal, semantic validation
warnings, document merging and duplicate detection, progress callbacks and
GEDZIP packages have all shipped and are no longer gaps.

| Gap | Available In | Notes |
|-----|--------------|-------|
| **GEDCOM 7.1 draft** | oxidged | gedcom-go reads `VERS 7.1` as 7.0 |
| **Query language** | elliotchance (gedcomq) | jq-like querying |
| **HTML generation** | elliotchance | Library-focused, but useful for tooling |
| **Alternative serializations** | oxidged (serde), GedcomSharp (GEDCOM X) | gedcom-go writes GEDCOM only |

### Not Gaps (by design)

| Feature | Notes |
|---------|-------|
| CLI tools | Library-first design; a CLI would be a separate package |
| GUI | Out of scope for a library |
| Database storage | Application-level concern |

---

## Methodology

### Data Sources
- GitHub repositories: README, license, `go.mod`, releases and tags, last
  commit, stars
- Source code, where a README did not settle a cell (encodings, record-at-a-time
  reading, writing)

### Limitations
- No performance comparison: few libraries publish benchmarks, and none on a
  shared environment
- Star counts reflect popularity, not quality
- The research is a snapshot at the date above

---

## References

### Go Libraries
- [cacack/gedcom-go](https://github.com/cacack/gedcom-go) - This library
- [elliotchance/gedcom](https://github.com/elliotchance/gedcom) - MIT, broadest Go toolkit
- [iand/gedcom](https://github.com/iand/gedcom) - Unlicense, 5.5/5.5.1/7.0
- [deaddr0p/go-gedcom](https://github.com/deaddr0p/go-gedcom) - GPL-3.0, GEDCOM 7.0 only (formerly funwithbots/go-gedcom)
- [MJKWoolnough/gedcom](https://github.com/MJKWoolnough/gedcom) - ISC, 5.5 reader

### GEDCOM Specifications
- [GEDCOM 5.5 Specification](https://www.gedcom.org/gedcom.html)
- [GEDCOM 5.5.1 Specification](https://edge.fscdn.org/assets/img/documents/ged551-5bac5e57fe88dd37df0e153d9c515335.pdf)
- [GEDCOM 7.0 Specification](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html)
