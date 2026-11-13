# GEDCOM Test Data

This directory contains GEDCOM test files organized by version and type. These files are used for testing the parser, decoder, encoder, and validator implementations.

## Directory Structure

```
testdata/
├── gedcom-5.5/          # GEDCOM 5.5 test files
│   ├── torture-test/    # Comprehensive GEDCOM 5.5 validation suite
│   └── ...              # Various 5.5 samples
├── gedcom-5.5.1/        # GEDCOM 5.5.1 test files
├── gedcom-7.0/          # GEDCOM 7.0 test files
│   ├── familysearch-examples/  # FamilySearch GEDCOM 7.0 edge cases
│   └── ...              # Various 7.0 samples
├── gedcom7code/         # gedcom7code/test-files: matched 5.5.1 (5/) and 7.0 (7/) pairs
├── oxidged/             # oxidged's hand-written "every structure" files for 5.5.1 and 7.0
├── encoding/            # Character encoding test files
├── edge-cases/          # Edge case and special scenario tests
└── malformed/           # Invalid GEDCOM files for error testing
```

## Test File Summary

**Total Test Files**: 94 GEDCOM files (as of 2026-08-10)

### GEDCOM 5.5 Files

#### Standard Samples
- **minimal.ged** (170B) - Smallest valid GEDCOM 5.5 file
- **shakespeare.ged** (18K) - Shakespeare family genealogy
- **pres2020.ged** (1.1M) - US Presidents and families (2,322 individuals)
  - Source: https://github.com/arbre-app/public-gedcoms
  - Author: Paul E. Stobbe (2020)
  - Use case: Performance testing, large file handling
- **royal92.ged** (458K) - European royal families (3,010 individuals)
  - Source: https://github.com/arbre-app/public-gedcoms
  - Author: Denis R. Reid (1992)
  - Use case: Complex relationships, performance testing

#### Torture Test Suite (`torture-test/`)
Source: https://www.geditcom.com/gedcom.html (TestGED.zip, updated Feb 2003)

Comprehensive validation suite that exercises every tag in GEDCOM 5.5:

- **TGC551.ged** (66K) - Full torture test, CR line endings, single NAME structure
- **TGC551LF.ged** (68K) - Same as TGC551, CRLF line endings
- **TGC55C.ged** (67K) - Full torture test, CR line endings, multiple NAME structures
- **TGC55CLF.ged** (69K) - Same as TGC55C, CRLF line endings

**What the torture tests cover:**
- All GEDCOM 5.5 tags for every record type (INDI, FAM, SOUR, REPO, SUBM, OBJE, SUBN, NOTE)
- Special ANSEL character encoding
- Multimedia file links (images, audio, video, documents)
- Privacy/restriction tags (RESN locked/privacy)
- Multiple submitters
- Line breaking in notes and text continuation (CONC/CONT)
- LDS ordinances (BAPL, CONL, ENDL, SLGC)
- All event types (birth, death, marriage, adoption, etc.)
- Custom tags testing
- Edge cases: unknown sex, multiple names, complex relationships

**Note**: The torture-test directory also includes multimedia reference files (images, audio, video) used by the GEDCOM files.

### GEDCOM 5.5.1 Files

- **minimal.ged** (204B) - Smallest valid GEDCOM 5.5.1 file
- **comprehensive.ged** (~7K) - Comprehensive GEDCOM 5.5.1 test with all new tags
  - Tests EMAIL, FAX, and WWW tags (new in 5.5.1)
  - EMAIL tags in: HEAD.SOUR.CORP, SUBM, INDI.RESI, REPO
  - FAX tags in: HEAD.SOUR.CORP, SUBM, REPO
  - WWW tags in: HEAD.SOUR.CORP, SUBM, INDI, REPO
  - Multi-generational family structure
  - Complete source and repository citations
  - UTF-8 encoding

**GEDCOM 5.5.1 Key Features Tested:**
- Independent PHON, EMAIL, FAX, WWW subrecords (without requiring ADDR)
- Enhanced address structures
- Extended multimedia support

#### Scale Fixture
- **longsword.ged** (46.1 MiB) - William Longsword descendant tree (203,154 individuals / 90,085 families; Legacy 10.0 export, GEDCOM 5.5.1, UTF-8 BOM)
  - Source: [D-Jeffrey/gedcom-samples](https://github.com/D-Jeffrey/gedcom-samples) `longsword/WilliamLongsword.ged`, vendored byte-identical
  - License: MIT OR CC0-1.0 (Copyright (c) 2025 D Jeffrey); the corpus README applies MIT-with-attribution to files gathered from Internet sources that name an author in the file
  - Use case: scale/performance testing (`decoder/scale_test.go`); see [docs/guides/performance.md](../docs/guides/performance.md) for measured figures

### Character Encoding Tests (`encoding/`)

Testing various character encodings and special characters:

- **utf8-bom.ged** (1.9K) - GEDCOM 5.5.5 with UTF-8 BOM
  - Source: https://www.gedcom.org/samples/555SAMPLE.GED
  - Tests UTF-8 Byte Order Mark handling
- **utf16le.ged** (3.9K) - GEDCOM 5.5.5 with UTF-16 Little Endian
  - Source: https://www.gedcom.org/samples/555SAMPLE16LE.GED
  - Tests UTF-16 LE with BOM
- **utf16be.ged** (3.9K) - GEDCOM 5.5.5 with UTF-16 Big Endian
  - Source: https://www.gedcom.org/samples/555SAMPLE16BE.GED
  - Tests UTF-16 BE with BOM
- **utf8-unicode.ged** (~4K) - UTF-8 with extensive Unicode characters
  - Latin-1 Supplement: àáâãäåæçèéêëìíîï
  - Cyrillic: АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюя
  - Greek: ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩαβγδεζηθικλμνξοπρστυφχψω
  - Arabic (RTL): أبتثجحخدذرزسشصضطظعغفقكلمنهوي
  - CJK (Chinese/Japanese/Korean): 中文漢字日本語ひらがなカタカナ한글
  - Hebrew (RTL): אבגדהוזחטיכלמנסעפצקרשת
  - Thai: กขฃคฅฆงจฉชซฌญฎฏฐฑฒณดตถทธนบปผฝพฟภมยรลวศษสหฬอฮ
  - Symbols and emoji: ☺☻♠♣♥♦★☆♪♫✓✗✘
  - Tests multi-byte UTF-8, RTL text, complex scripts, combining marks
- **ansel-trailing-combining.ged** (<1K) - Hand-written malformed ANSEL
  - `1 NOTE AB` ends in a combining acute (0xE2) with no base character before the LF
  - The mark must stay on its own line rather than corrupt the next one (issue #431)
- **ansi-cp1252-c1-range.ged** (<1K) - Hand-written `CHAR ANSI` file using Windows-1252 bytes in 0x80–0x9F
  - Curly apostrophe and quotes, en and em dash, euro sign, plus the undefined byte 0x81
  - These must decode to those characters, not C1 controls, and 0x81 must survive as U+0081, not U+FFFD (issue #564)

#### Gramps Encoding Tests
Source: https://github.com/gramps-project/gramps/tree/master/data/tests
License: GPL-2.0 (data files used as test inputs)

- **ansel-lf.ged** (10K) - ANSEL character encoding with LF line endings
  - Comprehensive ANSEL character set test
  - All combining/diacritical characters in ANSEL
  - Tests special characters (british pound, copyright, etc.)
  - 35 individuals testing different character combinations
  - Unix-style LF line endings (fills gap in line ending coverage)

- **utf8-nobom-lf.ged** (8.5K) - UTF-8 without BOM, LF line endings
  - Same character tests as ANSEL version but in UTF-8
  - No Byte Order Mark (tests BOM-less detection)
  - Unix-style LF line endings
  - Tests UTF-8 encoding of ANSEL special characters

#### Vintage Vendor Encoding Tests (D-Jeffrey corpus)
Real vintage exports vendored byte-identical (aside from submitter-contact redaction); see the D-Jeffrey/gedcom-samples attribution block below for source and license.

- **ansi-cp1252-ftm17.ged** (213K) - FTM 17.0.0.440 export, `CHAR ANSI` with genuine Windows-1252 bytes (ñ, ó, £)
  - Tests ANSI→UTF-8 conversion of bytes 0xA0–0xFF; it has no byte in 0x80–0x9F (see `ansi-cp1252-c1-range.ged`)
- **ibmpc-cp437-broskeep.ged** (380K) - Brother's Keeper 5.2 export, `CHAR IBMPC` with a real CP437 byte (0x82 'é')
  - Decodes as CP437 in strict mode with "Frémont" intact (`TestCorpusVendorCP437`, #735)
- **ibm-windows-easytree.ged** (17K) - EasyTree V1.0 export with nonstandard `CHAR IBM WINDOWS`
  - Tests raw CHAR value preservation in `Header.Encoding`

### Edge Cases (`edge-cases/`)

Testing parser robustness with edge cases:

- **cont-conc.ged** (~3K) - CONT/CONC line continuation tests
  - CONT (continuation with newline) vs CONC (concatenation without newline)
  - Mixed CONT and CONC usage
  - Very long lines split across multiple CONC tags
  - Empty CONT/CONC values
  - Multiple consecutive CONT or CONC tags
  - Unicode characters with CONT/CONC
  - Special characters and quotes in continued text
  - Tests that parser correctly reconstructs original text

- **calendar-dates.ged** (~7K) - Non-Gregorian calendar date tests
  - Hebrew calendar dates (`@#DHEBREW@`) with all 13 month codes
  - Julian calendar dates (`@#DJULIAN@`) including BC dates and dual dating
  - French Republican calendar dates (`@#DFRENCH R@`) with all 13 month codes
  - Date modifiers (ABT, BEF, AFT, EST, CAL, BET...AND, FROM...TO, TO, INT) with
    non-Gregorian calendars, including escapes on only the second date of a
    range or period
  - Partial dates (year-only, month-year) in each calendar system
  - Historical figures: Julius Caesar, Augustus, Washington, Rashi, Maimonides, Napoleon
  - Tests: 11 individuals with various calendar date formats

- **structural-edge-cases.ged** (~3K) - Parser structural stress tests
  - Very long XRef identifiers (>22 characters)
  - Lowercase XRef identifiers
  - Purely numeric XRef identifiers
  - Deeply nested structures (6+ levels)
  - Tags with empty or whitespace-only values
  - Multiple consecutive spaces in values
  - Tab characters in values
  - Root-level NOTE records
  - Tests: 9 individuals, 3 families, 1 source, 1 note record
  - Note: Empty lines are invalid per GEDCOM spec (tested in malformed/)

#### Vendor-Specific Test Files
Source: https://github.com/frizbog/gedcom4j (sample/ directory)
License: MIT

These files test vendor-specific custom tags and extensions from various genealogy software:

- **vendor-legacy.ged** (7.6K) - Legacy Family Tree 8.0 export
  - Custom tags: `_TODO`, `_UID`, `_PRIV`, `_EVENT_DEFN`, `_PLAC_DEFN`, `_TAG`
  - Source with inline citation data
  - Privacy markers on events and addresses
  - Custom event definitions with sentence templates
  - Tests: 5 individuals, 3 families, notes, sources, repositories

- **vendor-ftm.ged** (3.4K) - Family Tree Maker 22.2.5 for Mac
  - Custom tags: `_MISN`, `_DNA`, `_FUN`, `_ORDI`, `_MILTID`, `_EMPLOY`, `_WEIG`, `_INIT`, `_DEST`, `_NAMS`, `_MDCL`, `_ELEC`, `_HEIG`, `_CIRC`, `_ORIG`, `_MILT`, `_EXCM`, `_DCAUSE`, `_PHOTO`, `_FREL`, `_MREL`, `_SEPR`
  - Non-standard relationship types (Step, Guardian)
  - Source citation with justification and link
  - Tests: 3 individuals, 2 families, 1 source, 1 media object

#### FTM Edge Case Tests (Gramps Project)
Source: https://github.com/gramps-project/gramps/tree/master/data/tests
License: GPL-2.0 (data files used as test inputs)

- **ftm-general.ged** (1.8K) - Family Tree Maker 22.2.5 general export
- **ftm-link-test.ged** (1.4K) - FTM link handling patterns
- **ftm-occu-bug.ged** (781B) - FTM occupation field bug test
- **ftm-photo-test.ged** (1.3K) - FTM photo/media handling
- **ftm-conc-test.ged** (1.1K) - FTM CONC line continuation

- **vendor-familyhistorian.ged** (5.8K) - Family Historian 6.2.2
  - Custom tags: `_ATTR`, `_USED`, `_SHAN`, `_SHAR`, `_FLGS`, `_PLAC`, `_EMAIL`, `_WEB`, `_ROOT`, `_UID`, `_PEDI`
  - Shared event participants (witnesses)
  - Custom place records at root level
  - Census flag markers
  - Named list structures
  - Tests: 3 individuals, 1 family, 11 place records, 1 source, 1 media object

- **vendor-customtags-torture.ged** (11K) - Custom tags stress test
  - Deeply nested vendor extensions at every GEDCOM structure level
  - Custom tags under: header, submitter, submission, individual, family, source, repository, media
  - Multi-level custom tag hierarchies (up to 6 levels deep)
  - Custom tags with CONC line continuation
  - Root-level custom records (`_ROOT`, `_ROOT2`)
  - MIT License embedded in COPR tag
  - Tests: 3 individuals, 2 families, 1 source, 1 repository, 1 media object, 1 submission

- **relationships-complex.ged** (9.5K) - Complex family relationships
  - Step-parent and guardian relationships
  - Single-parent families (father only, mother only)
  - Remarriage scenarios (same person multiple FAMS)
  - Children with multiple FAMC (step-family membership)
  - Unknown sex individuals (`SEX U`)
  - ALIA tags for relationship annotations
  - Tests: 45 individuals, 18 families spanning multiple generations

- **vendor-myheritage.ged** (1.8K) - MyHeritage Family Tree Builder export (synthetic)
  - Custom tags: `_MHID`, `_MHTAG`, `_MHPID`, `_MHSID`, `_UID`, `_MARNM`, `_PHOTO`, `_PHOTO_RIN`, `_MYHERITAGE_ID`
  - Hebrew names with transliteration (`_MARNM` for married/Hebrew name)
  - Israeli locations (Tel Aviv, Jerusalem, Haifa)
  - Event-level MyHeritage IDs for tracking
  - Media objects with MyHeritage-specific tags
  - Tests: 3 individuals, 1 family, 1 source, 1 media object

- **vendor-gramps.ged** (2.5K) - Gramps 5.1.6 export (synthetic)
  - Custom tags: `_GRAMPS_ID`, `_GRAMPS_PLACE_ID`, `_GRAMPS_ATTR`
  - Swedish genealogical data with proper character encoding (å, ä, ö)
  - Place references linked to Gramps place database
  - Custom attributes with nested DATA tags
  - Multiple name types (Birth Name, Also Known As)
  - Date modifiers: ABT, BEF, AFT, CAL (approximate dates)
  - Repository with WWW tag
  - Tests: 5 individuals, 2 families, 1 source, 1 repository, 1 note

#### Additional Vendor Files from Gramps Test Suite
Source: https://github.com/gramps-project/gramps/tree/master/data/tests
License: GPL-2.0 (data files used as test inputs)

- **vendor-rootsmagic.ged** (526B) - RootsMagic 7.0.2.2 export
  - Tests mixed inline and XREF notes in same record
  - Inline NOTE vs NOTE @N0@ reference patterns
  - Inline SOUR vs SOUR @S1@ reference patterns
  - Tests: 2 individuals, 2 sources, 1 note record

- **vendor-heredis.ged** (6K) - HEREDIS 14 PC export (French software)
  - French genealogy software with Paris arrondissement data
  - Non-standard PLAC FORM with 6 components
  - EVEN with multiple TYPE subtags
  - ORDN (ordination) events for church records
  - French place names and accented characters (Églises, Île-de-France)
  - Tests: 22 individuals, 1 family

#### Real Vendor Exports (2025+)

Real exports from current software versions. See [docs/governance/policies/compatibility.md](../docs/governance/policies/compatibility.md) for detailed quirks.

| File | Source | Version | Records |
|------|--------|---------|---------|
| ancestry-2025-export.ged | Ancestry.com | 2025.08 | 14 indi, 5 fam |
| familysearch-2025-export.ged | FamilySearch.org | 2025 | 14 indi, 5 fam |
| gramps-2025-export.ged | Gramps | 6.0.6 | 14 indi, 5 fam |
| myheritage-2025-export.ged | MyHeritage.com | 5.5.1 | 14 indi, 5 fam |
| rootsmagic-2026-export.ged | RootsMagic | 11 Essentials (2026) | 14 indi, 5 fam, 2 sour, 1 repo |

**RootsMagic 11 Essentials Export Details:**
- GEDCOM 5.5.1, UTF-8 with BOM
- Custom tags: `_UID` (unique identifiers), `_TMPLT` (source templates with nested FIELD/NAME/VALUE), `_SUBQ` (short footnote), `_BIBL` (bibliography)
- `_EVDEF` root-level event definitions with sentence templates and role definitions
- Complex ADDR structures with both CONT continuation and specific subfields (ADR1, ADR2, CITY, STAE, POST, CTRY)
- Multi-generational family with 4 generations (1920-2003)
- Various date formats: exact, ABT, BEF, AFT, FROM...TO, year-only
- Source citations with PAGE references and template fields
- Death with CAUS (cause of death)
- Stillbirth scenario

#### D-Jeffrey/gedcom-samples Corpus Fixtures

The following fixtures are verbatim copies from the
[D-Jeffrey/gedcom-samples](https://github.com/D-Jeffrey/gedcom-samples) corpus,
dual-licensed `MIT OR CC0-1.0` (MIT: Copyright (c) 2025 D Jeffrey). Per the
corpus README, MIT-with-attribution applies to files gathered from Internet
sources that name an author inside the GED file; CC0 applies where the author
is D Jeffrey or no author is referenced. Files are copied byte-identical —
encoding quirks, junk dates, and nonstandard structures are intentional —
with one exception: personal contact details in SUBM (submitter) records
(name, home address, phone, email) are redacted to placeholders before
vendoring. When vendoring any new corpus file, scan its SUBM records (and any
personal contact blocks) and redact them first; tests must never assert on
submitter contact values. Exercised by `decoder/corpus_vendor_test.go`.

| Fixture | Corpus source file | Origin/platform |
|---|---|---|
| edge-cases/legacy10-2025-export.ged | ivar/IvarKingOfDublin.ged | Legacy 10.0 export (FamilySearch-sourced data) |
| edge-cases/vendor-paf5.ged | bach.ged | PAF 5.2.18.0 export (Bach family) |
| edge-cases/vendor-familyorigins5.ged | washington/washington.ged | Family Origins 5.0 export (Washington family) |
| edge-cases/vendor-tmg12.ged | famous family trees/royalty/Hawaiian+Kings.ged | The Master Genealogist 1.2a export |
| edge-cases/vendor-ancestris11-export.ged | sample-bourbon/bourbon.ged | Ancestris 11 export (French Bourbon data) |
| edge-cases/mhftb8-export.ged | queen/Queen.ged | MyHeritage Family Tree Builder 8 export |
| edge-cases/vendor-myroots-palmos.ged | famous family trees/US presidents/Lincoln+Family.ged | My Roots 4.00 for Palm OS export |
| edge-cases/vendor-webtreeprint.ged | bronte.ged | webtreeprint.com 1.0 web generator |
| edge-cases/bare-header-geo-coords.ged | input.ged | Unknown tool; bare `0 HEAD`, PLAC MAP coordinates |
| encoding/ansi-cp1252-ftm17.ged | famous family trees/royalty/Irish+Kings.ged | FTM 17 export, CHAR ANSI with real Windows-1252 bytes |
| encoding/ibmpc-cp437-broskeep.ged | famous family trees/US presidents/US+Presidents+Trees+I.ged | Brother's Keeper 5.2, CHAR IBMPC with real CP437 bytes (decoded as CP437) |
| encoding/ibm-windows-easytree.ged | famous family trees/US presidents/Kennedy+Family.ged | EasyTree V1.0, nonstandard `CHAR IBM WINDOWS` |

The scale fixture `gedcom-5.5.1/longsword.ged` (see above) comes from the same corpus.

#### rodoslovje/ged-merge Corpus Fixtures (issue #719)

Privacy-scrubbed slices of real exports, vendored byte-identical (SHA-256
checked) from [rodoslovje/ged-merge](https://github.com/rodoslovje/ged-merge)
`src/__fixtures__/corpus/` at commit `78425cfb15c9ce859d59bff7619bfbe03add32e9`.
License: MIT (Copyright (c) 2026 Luka Renko). Upstream generated them from
real exports with `scripts/anonymize-corpus.ts`: personal names are replaced
by pseudonyms, NOTE/TEXT narrative by placeholder text, and email, phone,
submitter name/address and media paths are scrubbed; tags, nesting, CONT/CONC
layout, dates, places, line endings and character encoding are preserved
byte for byte. No further redaction was needed. Exporter, version, charset
and line endings below are from the upstream `manifest.json`. Exercised by
`decoder/corpus_vendor_test.go` in strict and lenient mode.

| Fixture | Exporter | GEDCOM | Charset | EOL | Quirk |
|---|---|---|---|---|---|
| edge-cases/synium-7.0-utf8.ged | MacFamilyTree 11.2.3 (Synium) | **7.0.3** | UTF-8 | LF | 5.5.1-style `GEDC.FORM`, mixed-case months, `_FID`/`_SMC`/`_SIF` without `SCHMA` |
| edge-cases/reunion-5.5.1-utf8.ged | Reunion 14 | 5.5.1 | UTF-8 | **CR** | classic-Mac CR-only endings |
| edge-cases/geneanet-5.5.1-utf8.ged | Geneanet 7.0.0 | 5.5.1 | UTF-8 | LF | |
| encoding/geneanet-5.5.1-w1250.ged | Geneanet 7.0.0 | 5.5.1 | Windows-1250 (`CHAR ANSI`) | LF | ANSI over 1250 (known gap) |
| edge-cases/webtrees-5.5.1-utf8.ged | webtrees 2.1.27 | 5.5.1 | UTF-8 | CRLF | |
| edge-cases/familyhistorian-5.5.1-utf8.ged | Family Historian 7.0.24 | 5.5.1 | UTF-8+BOM | CRLF | |
| encoding/familyhistorian-5.5.1-w1250.ged | Family Historian 7.0.24 | 5.5.1 | Windows-1250 (`CHAR ANSI`) | CRLF | ANSI over 1250 (known gap) |
| encoding/brotherskeeper-5.5.1-w1250.ged | Brother's Keeper 7.5.4 | 5.5.1 | Windows-1250 (`CHAR ANSI`) | CRLF | ANSI over 1250 (known gap) |
| edge-cases/ancestry-5.5.1-utf8.ged | Ancestry.com Family Trees 2025.08 | 5.5.1 | UTF-8 | CRLF | |

#### Structural Torture Fixtures (issue #301)

Deliberate structural edge cases mined from peer-library test suites (see
[gedcom7code/test-files](https://github.com/gedcom7code/test-files), Unlicense /
public domain, for the vendored ones). Exercised by `decoder/structural_torture_test.go`.

- **family-structure-variants.ged** - Children-only family, empty FAM record, self-marriage, same-sex marriage (two-HUSB and HUSB+WIFE encodings) in 5.5.1. Original work (issue #301).
- **sex-value-variants.ged** - Every SEX payload variant: M/F/U, 7.0-only X in a 5.5.1 file, empty payload, lowercase, and no SEX line. Original work (issue #301).
- **structural-torture.ged** - NOTE record with both an XRef ID and a pointer payload, a physical line over the 255-char 5.5.1 limit, and an unknown non-underscore tag. Original work (issue #301).
- **deep-nesting-levels.ged** - Generated chain nested one level at a time to level 99 (deepest two-digit level). Original work (issue #301).
- **indented-lines.ged** - Lines indented with leading spaces/tabs (Geni.com-style); tolerated by the parser in both modes. Original work (issue #301).
- **atsign-55.ged** - GEDCOM 5.5 at-sign torture: @@ doubling, @#...@ escapes in NOTE payloads, @@ at start of CONC/CONT. Vendored from gedcom7code/test-files (public domain).
- **xref-case.ged** - XRef case-mismatch (@test@ vs @TEST@) and an XRef containing a space. Vendored from gedcom7code/test-files (public domain).
- **age-keywords-551.ged** - 5.5.1 AGE keyword payloads (CHILD/INFANT/STILLBORN, case variants, </>). Vendored from gedcom7code/test-files (public domain).
- **date-dual-years.ged** - Dual-year dates (1699/00) in plain, ABT, FROM/TO, and BET/AND forms. Vendored from gedcom7code/test-files (public domain).

### GEDCOM 7.0 Files

#### Standard Samples
- **minimal.ged** (199B) - Minimal GEDCOM 7.0 file
- **minimal70.ged** (32B) - Smallest legal FamilySearch GEDCOM 7.0 file
- **maximal70.ged** (15K) - Exercises all standard tags and enumerations
- **remarriage1.ged** (388B) - Divorce/remarriage scenario

#### FamilySearch Examples (`familysearch-examples/`)
Source: https://gedcom.io/testfiles/gedcom70/

Comprehensive GEDCOM 7.0 edge case testing from FamilySearch:

**Feature-specific tests:**
- **age.ged** (2.3K) - AGE payload format variations
- **escapes.ged** (733B) - @ character escaping rules
- **lang.ged** (2.1K) - LANG payload examples
- **filename-1.ged** (1.8K) - FILE payload format variations
- **long-url.ged** (1.0K) - Very long line parsing tests
- **notes-1.ged** (390B) - NOTE and SNOTE usage patterns
- **obje-1.ged** (425B) - OBJE (multimedia object) record variations
- **voidptr.ged** (292B) - @VOID@ reference handling
- **xref.ged** (405B) - Cross-reference identifier formats

**Extension tests:**
- **extension-record.ged** (357B) - Custom _LOC record extensions
- **extensions.ged** (3.3K) - Various extension formats

**Relationship scenarios:**
- **remarriage2.ged** (447B) - Additional divorce/remarriage scenario
- **same-sex-marriage.ged** (173B) - Same-sex marriage example

**Comprehensive trees:**
- **maximal70-tree1.ged** (891B) - Individuals, families, sources, events
- **maximal70-tree2.ged** (2.3K) - Extends tree1 with attributes and events
- **maximal70-lds.ged** (1.2K) - LDS ordinance structures
- **maximal70-memories1.ged** (1.1K) - Multimedia object records
- **maximal70-memories2.ged** (1.2K) - Family and event multimedia objects

These files predate the licence check and carry no recorded licence; whether
they move to the fetched scheme below is a separate decision. The whole of
`gedcom70/` and `gedcom71/`, these 18 included, is also fetched at test time;
see the next section.

### gedcom7code Conversion Suite (`gedcom7code/`)
Source: https://github.com/gedcom7code/test-files (commit pinned in `gedcom7code/SOURCE`)
License: Unlicense (public domain; upstream `LICENSE` vendored alongside)

The whole suite, byte-identical: `5/` holds 26 GEDCOM 5.5.1 files and `7/` the
7.0 conversions of 25 of them (`date-dual-valid.ged` has no 7.0 partner). The
upstream `README.md` is vendored too; it defines when two files are equivalent
(trailing space on a valueless line, order of distinct-tag substructures,
`GREGORIAN`, SCHMA tag names, XRef spelling).

- **age-valid / age-invalid** - `AGE` payloads, including 5.5.1 keywords and malformed ages
- **atsign** - `@` escaping in each version
- **char_*** - ASCII, UTF-8 and UTF-16 (with and without BOM) encodings
- **date-all** (~10k lines) - the DateValue grammar; **date-dual-*** - dual-year dates
- **enum-ext**, **pedi-1**, **rela_1** - enumerations and their 7.0 `PHRASE` fallbacks
- **filename-1**, **lang-all**, **notes-1**, **obje-1**, **sour-1** - payloads and structures that change shape between versions
- **obsolete-1** - 5.5.1 structures removed in 7.0; **tiny-1** - smallest file; **xref-case** - XRef case and spaces

Exercised by `decoder/conformance_suites_test.go` (strict and lenient decode of
every file) and `converter/conformance_pairs_test.go` (each pair converted both
ways and compared under the README's rules). Both keep a self-cleaning list of
known failures. Four `5/` files (`atsign`, `xref-case`, `age-valid`,
`date-dual-valid`) were vendored earlier into `edge-cases/`; see the Structural
Torture Fixtures section.

### oxidged Every-Structure Files (`oxidged/`)
Source: https://github.com/trois-six/oxidged `tests/fixtures/conformance/` (commit pinned in `oxidged/SOURCE`)
License: MIT (Copyright (c) 2020 Robert Pirtle; Copyright (c) 2026 Pierre Erraud and the oxidged contributors; upstream `license.md` vendored as `LICENSE`)

Four fictitious files written by another project for its own 5.5.1 and 7.0
conformance reviews, byte-identical. They are independent of our own fixtures,
so a structure our fixtures' author overlooked is still exercised.

- **full551.ged** (6.1K) - Every GEDCOM 5.5.1 structure, hand-written
- **maximal551.ged** (10K) - `full551.ged` plus generated records completing every 5.5.1 context
- **full70.ged** (7.5K) - Every GEDCOM 7.0 structure, hand-written
- **extensions70.ged** (0.9K) - 7.0 extensions: SCHMA-documented tags, an extension record, extension enumeration values and calendars

Exercised by `decoder/conformance_suites_test.go` (strict and lenient decode),
`decoder/oxidged_fixtures_test.go` (no `UNKNOWN_TAG` on a standard tag, and every
line the coverage reports call typed reaches the typed model) and
`encoder/write_shapes_test.go` (decode, entity encode, decode). Each keeps a
self-cleaning list of known gaps citing the issue for each.

### Fetched at test time: gedcom.io test files

Not in this directory. The complete FamilySearch/GEDCOM.io `testfiles/gedcom70/`
and `testfiles/gedcom71/` (45 files: every `.ged`, the four `.gdz` archives, and
the `maximal70/`, `maximal71/`, `minimal70/` and `minimal71/` directories with
their media) are downloaded by `make fetch-corpora` into `.corpora/gedcomio/` at
the repository root.

- **Why not vendored:** the repository declares no licence and the site reads
  "All rights reserved", so the files may be used but not redistributed.
- **Pinned:** `internal/corpora/gedcomio.lock` names an immutable upstream
  commit and each file's size and SHA-256. A file that no longer matches fails
  the fetch; a network failure only warns, unless `-require` is given.
- **Outside `testdata/`** on purpose, so the corpus walkers here never see
  files that are only sometimes present.
- **Tests:** `decoder/conformance_gedcomio_test.go` decodes every `.ged` in
  strict and lenient mode and checks it decodes as 7.0;
  `validator/conformance_gedcomio_test.go` records which of the warnings
  `warnings70.ged`/`warnings71.ged` are built to raise we raise. Both keep
  self-cleaning known-failure lists; `.gdz` files skip until GEDZIP lands
  (#127). Without the files these tests skip naming `make fetch-corpora`;
  `GEDCOM_REQUIRE_FETCHED_CORPORA=1` makes that a failure.
- **CI:** the test job restores them from a cache keyed on the lock and
  tolerates a failed fetch; the nightly workflow fetches with `-require` and
  runs the tests with `GEDCOM_REQUIRE_FETCHED_CORPORA=1`.

### Malformed Files

Files with intentional errors for testing error handling and validation:

- **invalid-level.ged** (110B) - Invalid level number (level 99)
- **invalid-xref.ged** (76B) - Malformed cross-reference format
- **unterminated-xref.ged** (61B) - XRef with no closing `@` (`0 @I1 INDI`)
  - Strict mode rejects it; lenient mode recovers the record under the verbatim
    identifier `@I1` with an `INVALID_XREF` diagnostic
  - Original work (issue #385)
- **missing-header.ged** (37B) - Missing required HEAD record
- **missing-xref.ged** (76B) - Missing cross-reference target (broken XRef)
- **circular-reference.ged** (~500B) - Circular family relationships
  - Person is both parent and child in different families
  - Tests validator's ability to detect impossible genealogical loops
  - Should be accepted by parser/decoder, rejected by validator
- **duplicate-xref.ged** (~300B) - Duplicate cross-reference identifiers
  - Multiple records with same XRef (@I1@, @F1@)
  - Tests XRefMap handling of conflicts
  - Decoder behavior: last record wins, earlier records overwritten
- **blank-lines.ged** - Blank and whitespace-only lines mid-file
  - Strict mode errors, lenient mode skips with EMPTY_LINE diagnostics
  - Original work (issue #301)
- **level-over-99.ged** - Levels 100 and 101, both past the two-digit level ceiling; each is rejected with an INVALID_LEVEL diagnostic (issue #379)
  - Original work (issue #301)
- **coordinates.ged** - PLAC MAP with a non-finite `LATI` (`Nnan`) and a hex-float
  `LONG` (`E0x1p3`); each is reported with an INVALID_VALUE diagnostic while the raw
  text is preserved in the typed model. A second individual carries a well-formed MAP
  that must stay quiet.
  - Original work (issue #504)

## Usage Guidelines

### Test Coverage Strategy

1. **Unit Tests**: Use minimal files and specific feature tests
   - `minimal*.ged` - Basic parsing validation
   - `age.ged`, `escapes.ged`, etc. - Feature-specific validation
   - `cont-conc.ged` - Line continuation edge cases

2. **Integration Tests**: Use comprehensive files
   - `TGC55*.ged` - Full GEDCOM 5.5 specification coverage (ANSEL encoding)
   - `maximal70*.ged` - Full GEDCOM 7.0 specification coverage
   - `comprehensive.ged` (5.5.1) - EMAIL/FAX/WWW tag validation

3. **Performance Tests**: Use large real-world files
   - `pres2020.ged` (1.1M, 2,322 individuals)
   - `royal92.ged` (458K, 3,010 individuals)
   - `longsword.ged` (46.1M, 203,154 individuals) - scale class; see `decoder/scale_test.go`

4. **Encoding Tests**: Use encoding directory files
   - `utf8-bom.ged`, `utf16le.ged`, `utf16be.ged` - BOM handling
   - `utf8-unicode.ged` - Multi-byte UTF-8, international characters

5. **Error Handling Tests**: Use malformed files
   - `malformed/*.ged` - Various error conditions
   - Includes circular references, duplicate XRefs, broken references

### Version-Specific Testing

- **GEDCOM 5.5**: Use `torture-test/` files for comprehensive validation
- **GEDCOM 5.5.1**: Use `comprehensive.ged` for EMAIL/FAX/WWW tags
- **GEDCOM 7.0**: Use `familysearch-examples/` for edge cases
- **Character Encoding**: Use `encoding/` directory for UTF-8/UTF-16 tests

## Adding New Test Files

When adding new test files:

1. Place in appropriate version directory
2. Use descriptive filenames indicating what they test
3. Add entry to this README with:
   - Filename and size
   - Source/author if applicable
   - What the file tests or demonstrates
4. For large or complex files, include sample counts (individuals, families)

## Sources and Credits

- **FamilySearch GEDCOM 7.0 Examples**: https://gedcom.io/testfiles/gedcom70/
  - All files in `gedcom-7.0/familysearch-examples/`
  - The whole of `gedcom70/` and `gedcom71/` is fetched at test time, not held here (see above)
- **GEDCOM.org Official Samples**: https://www.gedcom.org/samples.html
  - UTF-8 with BOM, UTF-16 LE/BE samples in `encoding/`
- **TestGED Torture Test Suite**: https://www.geditcom.com/gedcom.html
  - Created by H. Eichmann
  - Modified by J. A. Nairn using GEDitCOM (1999-2001)
  - Files in `gedcom-5.5/torture-test/`
- **Public GEDCOM Collections**: https://github.com/arbre-app/public-gedcoms
  - royal92.ged by Denis R. Reid (1992)
  - pres2020.ged by Paul E. Stobbe (2020)
- **gedcom4j Project**: https://github.com/frizbog/gedcom4j
  - License: MIT
  - Author: Matthew R. Harrah
  - Files in `edge-cases/vendor-*.ged` and `edge-cases/relationships-complex.ged`
  - Tests vendor-specific extensions from Legacy, Family Tree Maker, Family Historian
- **Gramps Project**: https://github.com/gramps-project/gramps
  - License: GPL-2.0 (test data files used as inputs, not derivative works)
  - Files: `encoding/ansel-lf.ged`, `encoding/utf8-nobom-lf.ged`, `edge-cases/vendor-rootsmagic.ged`, `edge-cases/vendor-heredis.ged`
  - Comprehensive ANSEL character tests, LF line endings, RootsMagic/HEREDIS exports
- **D-Jeffrey/gedcom-samples Corpus**: https://github.com/D-Jeffrey/gedcom-samples
  - License: MIT OR CC0-1.0 (Copyright (c) 2025 D Jeffrey)
  - Files: 12 vintage vendor exports in `edge-cases/` and `encoding/` (see attribution table above) plus `gedcom-5.5.1/longsword.ged`
  - Real exports from Legacy 10, PAF 5.2, Family Origins 5, TMG 1.2a, Ancestris 11, MyHeritage FTB 8, and more
- **rodoslovje/ged-merge Corpus**: https://github.com/rodoslovje/ged-merge
  - License: MIT (Copyright (c) 2026 Luka Renko)
  - Files: 9 privacy-scrubbed real exports in `edge-cases/` and `encoding/` (see the ged-merge table above)
  - MacFamilyTree 11 GEDCOM 7.0, Reunion 14, Geneanet, webtrees, Family Historian 7, Brother's Keeper 7, Ancestry
- **gedcom7code/test-files**: https://github.com/gedcom7code/test-files
  - License: Unlicense (public domain)
  - Files: the whole suite in `gedcom7code/`, plus `edge-cases/atsign-55.ged`, `edge-cases/xref-case.ged`, `edge-cases/age-keywords-551.ged`, `edge-cases/date-dual-years.ged`
  - 5.5.1 ↔ 7.0 conversion pairs; at-sign escaping, XRef case quirks, 5.5.1 AGE keywords, dual-year dates
- **oxidged**: https://github.com/trois-six/oxidged
  - License: MIT (Copyright (c) 2020 Robert Pirtle; Copyright (c) 2026 Pierre Erraud and the oxidged contributors)
  - Files: the four files in `oxidged/`
  - Hand-written "every structure" files for 5.5.1 and 7.0, plus 7.0 extensions
- **Synthetic Test Files**: Created specifically for this project
  - `gedcom-5.5.1/comprehensive.ged` - GEDCOM 5.5.1 features
  - `encoding/utf8-unicode.ged` - International character testing
  - `edge-cases/cont-conc.ged` - Line continuation testing
  - `edge-cases/calendar-dates.ged` - Non-Gregorian calendar dates (Hebrew, Julian, French Republican)
  - `edge-cases/structural-edge-cases.ged` - Parser structural stress tests
  - `edge-cases/family-structure-variants.ged`, `sex-value-variants.ged`, `structural-torture.ged`, `deep-nesting-levels.ged`, `indented-lines.ged` - Structural torture cases (issue #301)
  - `malformed/circular-reference.ged` - Circular relationship loops
  - `malformed/duplicate-xref.ged` - Duplicate identifier testing
  - `malformed/blank-lines.ged`, `malformed/level-over-99.ged` - Lenient-mode error cases (issue #301)

## License Notes

- Most files are provided for non-commercial testing purposes
- GEDCOM specification: (c) The Church of Jesus Christ of Latter-Day Saints
- Individual files may have their own copyright notices (see file headers)
- TestGED suite: Free for non-commercial use per included README
- gedcom4j files: MIT License (Copyright 2009-2016 Matthew R. Harrah)
- Gramps files: GPL-2.0 (used as test data inputs; GPL copyleft does not apply to data file consumers)
- D-Jeffrey/gedcom-samples files: MIT OR CC0-1.0 (attribution given above per the corpus README)
- gedcom7code/test-files files: Unlicense (public domain)
- rodoslovje/ged-merge files: MIT (Copyright (c) 2026 Luka Renko)
- oxidged files: MIT (see `oxidged/LICENSE`)

## Testing Best Practices

1. **Always test with multiple versions**: 5.5, 5.5.1, and 7.0 have differences
2. **Test line ending variations**: Use both CR and CRLF variants (TGC551.ged vs TGC551LF.ged)
3. **Test character encodings**: Especially ANSEL (see TGC55*.ged files)
4. **Test edge cases**: Use FamilySearch examples for specific edge cases
5. **Test error handling**: Use malformed files to verify proper error messages
6. **Test performance**: Use large files (pres2020.ged, royal92.ged) for benchmarks
7. **Test streaming**: Large files should be parseable without loading entirely into memory

## Minimum Test Coverage

For compliance with project constitution (85% coverage requirement):

- Parse all files in gedcom-5.5/, gedcom-5.5.1/, gedcom-7.0/
- Properly reject all files in malformed/
- Successfully decode at least one torture test file
- Successfully handle large files (>1MB) in streaming fashion
