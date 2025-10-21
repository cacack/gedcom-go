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

**Total Test Files**: 42 GEDCOM files (as of 2025-10-21)

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
- **Synthetic Test Files**: Created specifically for this project
  - `gedcom-5.5.1/comprehensive.ged` - GEDCOM 5.5.1 features
  - `encoding/utf8-unicode.ged` - International character testing
  - `edge-cases/cont-conc.ged` - Line continuation testing
  - `malformed/circular-reference.ged` - Circular relationship loops
  - `malformed/duplicate-xref.ged` - Duplicate identifier testing

## License Notes

- Most files are provided for non-commercial testing purposes
- GEDCOM specification: © The Church of Jesus Christ of Latter-Day Saints
- Individual files may have their own copyright notices (see file headers)
- TestGED suite: Free for non-commercial use per included README

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
