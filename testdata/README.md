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
├── encoding/            # Character encoding test files
├── edge-cases/          # Edge case and special scenario tests
└── malformed/           # Invalid GEDCOM files for error testing
```

## Test File Summary

**Total Test Files**: 55 GEDCOM files (as of 2026-01-12)

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

- **calendar-dates.ged** (~5K) - Non-Gregorian calendar date tests
  - Hebrew calendar dates (`@#DHEBREW@`) with all 13 month codes
  - Julian calendar dates (`@#DJULIAN@`) including BC dates and dual dating
  - French Republican calendar dates (`@#DFRENCH R@`) with all 13 month codes
  - Date modifiers (ABT, BEF, AFT, EST, BET...AND) with non-Gregorian calendars
  - Partial dates (year-only, month-year) in each calendar system
  - Historical figures: Julius Caesar, Augustus, Washington, Rashi, Maimonides, Napoleon
  - Tests: 10 individuals with various calendar date formats

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

Real exports from current software versions. See [docs/COMPATIBILITY.md](../docs/COMPATIBILITY.md) for detailed quirks.

| File | Source | Version | Records |
|------|--------|---------|---------|
| ancestry-2025-export.ged | Ancestry.com | 2025.08 | 14 indi, 5 fam |
| familysearch-2025-export.ged | FamilySearch.org | 2025 | 14 indi, 5 fam |
| gramps-2025-export.ged | Gramps | 6.0.6 | 14 indi, 5 fam |
| myheritage-2025-export.ged | MyHeritage.com | 5.5.1 | 14 indi, 5 fam |
| rootsmagic-2026-export.ged | RootsMagic | 11 Essentials | 14 indi, 5 fam, 2 sour, 1 repo |

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

### Malformed Files

Files with intentional errors for testing error handling and validation:

- **invalid-level.ged** (110B) - Invalid level number (level 99)
- **invalid-xref.ged** (76B) - Malformed cross-reference format
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
- **Synthetic Test Files**: Created specifically for this project
  - `gedcom-5.5.1/comprehensive.ged` - GEDCOM 5.5.1 features
  - `encoding/utf8-unicode.ged` - International character testing
  - `edge-cases/cont-conc.ged` - Line continuation testing
  - `edge-cases/calendar-dates.ged` - Non-Gregorian calendar dates (Hebrew, Julian, French Republican)
  - `edge-cases/structural-edge-cases.ged` - Parser structural stress tests
  - `malformed/circular-reference.ged` - Circular relationship loops
  - `malformed/duplicate-xref.ged` - Duplicate identifier testing

## License Notes

- Most files are provided for non-commercial testing purposes
- GEDCOM specification: (c) The Church of Jesus Christ of Latter-Day Saints
- Individual files may have their own copyright notices (see file headers)
- TestGED suite: Free for non-commercial use per included README
- gedcom4j files: MIT License (Copyright 2009-2016 Matthew R. Harrah)
- Gramps files: GPL-2.0 (used as test data inputs; GPL copyleft does not apply to data file consumers)

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
