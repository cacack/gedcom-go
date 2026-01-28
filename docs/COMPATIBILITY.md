# Compatibility Matrix

This document describes gedcom-go's compatibility with various GEDCOM sources. We aim to be transparent about what has been tested and at what confidence level.

## Vendor Compatibility

The following table shows which genealogy software exports have been tested with this library.

| Software | Version Tested | Import | Notes |
|----------|---------------|--------|-------|
| RootsMagic | 7.0.2.2 (2015) | ⚠️ | Older version; inline/xref note patterns work |
| Legacy Family Tree | 8.0 (2016) | ⚠️ | Older version; custom tags preserved (`_TODO`, `_UID`, `_PRIV`) |
| Family Tree Maker | 22.2.5 (2016) | ⚠️ | Older version; Ancestry format with custom tags |
| Family Historian | 6.2.2 | ⚠️ | Custom tags preserved (`_ATTR`, `_USED`, `_SHAN`, `_SHAR`) |
| HEREDIS | 14 PC | ⚠️ | French locales work; non-standard PLAC FORM handled |
| Gramps | 6.0.6 (2025) | ✅ | Real export tested; `CHAN` records, `TYPE birth`, note refs |
| MyHeritage | 5.5.1 (2025) | ✅ | Real export tested; `_UID`, `RIN`, HTML notes, `QUAY` tags |
| Ancestry.com | 2025.08 | ✅ | Real export tested; `_TREE` parsed, long XRefs, nickname handling |
| FamilySearch | 2025 | ✅ | Real export tested; `_HASH`/`_LHASH` tags, standardizer note |

### Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Verified with official specification examples |
| ⚠️ | Tested with older software version; current versions may differ |
| 🧪 | Synthetic test file only; not a real export |
| ❓ | Vendor detection exists but no test data available |

### Important Notes

- **Export column intentionally omitted**: This library produces standard GEDCOM output, not vendor-specific formats. All encoding is spec-compliant.
- **Older versions**: Test files for RootsMagic, Legacy, FTM, Family Historian, and HEREDIS are from 2015-2016 era software. Current versions may have different export patterns.
- **FamilySearch "verified"**: Testing uses [official spec examples](https://gedcom.io/tools/) from FamilySearch, not real-world exports from the FamilySearch website.

## GEDCOM Specification Support

| Spec Version | Parse | Encode | Validation | Test Coverage |
|--------------|:-----:|:------:|:----------:|---------------|
| GEDCOM 5.5 | ✅ | ✅ | ✅ | Torture test suite (TGC55*.ged) |
| GEDCOM 5.5.1 | ✅ | ✅ | ✅ | Comprehensive samples (EMAIL, FAX, WWW) |
| GEDCOM 7.0 | ✅ | ✅ | ✅ | FamilySearch spec examples |

### What Each Column Means

- **Parse**: Read GEDCOM files and convert to Document structure
- **Encode**: Write Document structure back to valid GEDCOM format
- **Validation**: Detect structural errors, broken references, invalid dates

## What "Import" Means

When we say a vendor's files are supported for import, we mean:

1. **Parsing**: The file parses without errors
2. **Structure preservation**: Record hierarchy is maintained correctly
3. **Custom tag handling**: Vendor-specific tags (e.g., `_APID`, `_MHID`) are preserved in `CustomTags` on each record
4. **Character encoding**: ANSEL, UTF-8, and UTF-16 files are decoded correctly
5. **Cross-references**: Links between records (XRefs) resolve correctly

We do **not** mean:

- Semantic interpretation of vendor-specific custom tags
- Format-specific optimizations for export
- Guaranteed compatibility with all versions of the software

## Vendor Detection

The library automatically detects the source software from the `HEAD.SOUR` tag. This enables:

- Logging which software created a file
- Future vendor-specific parsing behaviors if needed

Detection is implemented in [`gedcom/vendor.go`](../gedcom/vendor.go).

Currently detected vendors:
- Ancestry.com (including FamilyTreeMaker)
- FamilySearch
- RootsMagic
- Legacy Family Tree
- Gramps
- MyHeritage

## Test Data Sources

All test files are documented in [`testdata/README.md`](../testdata/README.md) with:

- Source attribution (where the file came from)
- License information
- What each file tests

Key sources include:

| Source | Files | License |
|--------|-------|---------|
| [FamilySearch GEDCOM 7.0](https://gedcom.io/tools/) | `gedcom-7.0/familysearch-examples/` | Public domain |
| [TestGED Torture Suite](https://www.geditcom.com/gedcom.html) | `gedcom-5.5/torture-test/` | Non-commercial |
| [gedcom4j Project](https://github.com/frizbog/gedcom4j) | `edge-cases/vendor-*.ged` | MIT |
| [Gramps Project](https://github.com/gramps-project/gramps) | `encoding/ansel-lf.ged`, `vendor-rootsmagic.ged`, `vendor-heredis.ged` | GPL-2.0 |
| Synthetic (this project) | Various test files | Apache-2.0 |

## Contributing Test Files

We actively seek real GEDCOM exports from:

- **Ancestry.com** - Real tree exports (not just linked records)
- **FamilySearch** - Actual exports from the FamilySearch website
- **FindMyPast** - No test coverage currently
- **Geni.com** - No test coverage currently
- **Current versions** of RootsMagic, Legacy, FTM, etc.

### How to Contribute

1. Export a GEDCOM from your genealogy software
2. Review it for sensitive personal information
3. Either:
   - Open an issue with the file attached (if small and non-sensitive)
   - Open an issue describing the software/version and any parsing issues encountered

### What We Need

- Software name and version
- Any custom tags your software uses
- Whether the file parses correctly with this library
- Any errors or unexpected behavior

Even failure reports are valuable - they help us understand what needs to be fixed.

## Known Limitations

### Ancestry.com

Tested with real export from Ancestry.com Member Trees (version 2025.08).

**Observed behaviors:**
- `_TREE` tag in header with tree name, RIN, and `_ENV prd` (production environment)
- Long numeric XRef IDs (e.g., `@I152733931767@`) instead of sequential `@I1@`
- Nicknames stored as separate NAME records: `1 NAME Jack //`
- HTML entities in notes: `&apos;` for apostrophe
- XML-like `<line>` tags within ADDR values
- Repository WWW URLs converted to NOTE records
- Source citations may be duplicated at individual level

**`_APID` tags**: Ancestry exports include `_APID` tags that reference their database records when you attach Ancestry hints/records. We parse these and can reconstruct URLs. Note: `_APID` tags only appear when records are attached to individuals, not in basic tree exports.

### MyHeritage

Tested with real export from MyHeritage.com (2025).

**Observed behaviors:**
- UTF-8 with BOM encoding
- `_RTLSAVE RTL` header tag (right-to-left language support indicator)
- `_PROJECT_GUID` and `_EXPORTED_FROM_SITE_ID` header tags
- `DEAT Y` explicit death marker with calculated `AGE` field
- `QUAY 0` quality assessment tags on source citations
- HTML `<p>` tags embedded in notes
- `RIN MH:I1` record identification numbers with MH prefix
- `_UID` tags on individuals for tracking
- Plain text addresses converted to `ADDR/ADR1` structure
- Removes REPO (repository) records from export

### Gramps

Tested with real export from Gramps 6.0.6 (2025).

**Observed behaviors:**
- `COPR` copyright tag in header
- Padded sequential XRef IDs: `@I0001@`, `@F0001@`, `@S0001@`
- `NAME` records with `TYPE birth` subrecord
- `CHAN` (change) records on each INDI/FAM with date and time
- Notes stored as references (`NOTE @N0000@`) to root-level NOTE records
- `SUBM` (submitter) record with empty NAME
- Standard-compliant output with minimal custom tags

### FamilySearch

Tested with real export from FamilySearch.org (2025).

**Observed behaviors:**
- Adds `_HASH` and `_LHASH` tags to each INDI and FAM record (MD5 checksums for change detection)
- Adds header note: `NOTE Unified System GEDCOM Standardizer 1.0`
- Reorders records: REPO, SOUR placed before INDI/FAM records
- Preserves original source header (doesn't claim authorship)
- Removes root-level NOTE records
- Removes SUBM (submitter) records
- Nicknames preserved as NICK subrecord (standard-compliant)

**Note**: FamilySearch exports GEDCOM 5.5.1, not GEDCOM 7.0, despite being the GEDCOM 7.0 spec maintainer. The GEDCOM 7.0 spec examples in our test suite are from gedcom.io documentation, not real FamilySearch exports.

## How We Test Compatibility

This section explains how compatibility claims in this document are verified, enabling you to audit our process or reproduce tests locally.

### Where Sample GEDCOMs Live

Test files are organized under `testdata/` by GEDCOM version and purpose:

```
testdata/
├── gedcom-5.5/          # GEDCOM 5.5 samples
│   └── torture-test/    # Comprehensive TGC55* validation suite
├── gedcom-5.5.1/        # GEDCOM 5.5.1 samples (EMAIL/FAX/WWW tags)
├── gedcom-7.0/          # GEDCOM 7.0 samples
│   └── familysearch-examples/  # Official FamilySearch edge cases
├── encoding/            # Character encoding tests (UTF-8, UTF-16, ANSEL)
├── edge-cases/          # Structural edge cases, vendor-specific exports
│   └── vendor-*.ged     # Vendor-specific custom tag tests
└── malformed/           # Invalid files for error handling tests
```

See [`testdata/README.md`](../testdata/README.md) for complete attribution, licensing, and descriptions of each file.

### What "Synthetic" Means

Files marked with 🧪 (synthetic) in the compatibility matrix are:

- **Created by this project** specifically to test parsing patterns
- **Not exports from actual software** - they simulate expected patterns based on documentation
- **Privacy-safe** - contain only fictional test data
- **License-clear** - created under this project's Apache-2.0 license

Why synthetic files? Real exports may contain sensitive personal information, and obtaining properly-licensed samples from every vendor version is impractical. Synthetic files let us test known patterns without these constraints.

**Important**: Synthetic tests verify that we *can* parse documented patterns, not that we *have* parsed real-world exports. When the compatibility matrix shows 🧪, treat it as "expected to work based on documentation" rather than "verified with production files."

### What "Round-Trip Fidelity" Means

Round-trip testing verifies: **decode -> encode -> decode produces semantically equivalent documents**.

**What IS preserved** (semantic equivalence):
- Record hierarchy and nesting structure
- Cross-references (XRefs) and their resolution
- Tag values and data content
- Custom/vendor-specific tags
- All typed entities (individuals, families, sources, etc.)

**Tolerated differences** (not considered failures):
- Line ending normalization (CR/LF/CRLF)
- Whitespace in certain contexts
- Encoding declaration changes (e.g., input ANSEL -> output UTF-8)
- Tag ordering within a record (when spec allows)

**How round-trip is tested**:

```go
// Simplified from actual test code
doc1, _ := decoder.Decode(input)       // Original decode
var buf bytes.Buffer
encoder.Encode(&buf, doc1)             // Re-encode
doc2, _ := decoder.Decode(&buf)        // Decode the output

// Compare semantic content
assertEqual(t, len(doc1.Individuals()), len(doc2.Individuals()))
assertEqual(t, len(doc1.Families()), len(doc2.Families()))
// ... and all other record types, values, references
```

Round-trip tests exist throughout the codebase:
- `encoder/encoder_test.go` - `TestEncodeRoundtrip`
- `gedcom_api_test.go` - `TestRoundTrip`
- `encoder/entity_writer_test.go` - Various `TestRoundTrip*` tests
- `encoder/streaming_test.go` - `TestStreamEncoder_RoundTrip`

### How to Reproduce Tests Locally

**Run the full test suite** (includes round-trip tests):

```bash
make test
```

**Run round-trip tests specifically**:

```bash
go test ./... -run TestRoundTrip -v
```

**Run tests against a specific GEDCOM version**:

```bash
# GEDCOM 5.5 torture test suite
go test ./decoder -run TestTortureTestSuite -v

# GEDCOM 7.0 tests
go test ./decoder -run "70|Gedcom7" -v
```

**Run with coverage** to see what code paths are exercised:

```bash
make test-coverage
```

### Adding New Vendor Test Files

To contribute a test file from your genealogy software:

1. **Export a GEDCOM** from your software
2. **Review for sensitive data** - remove or anonymize personal information
3. **Place in appropriate directory**:
   - `testdata/edge-cases/vendor-<software>.ged` for vendor-specific tests
   - `testdata/gedcom-<version>/` for version-specific tests
4. **Update `testdata/README.md`** with:
   - Filename and size
   - Source software name and version
   - What the file tests (custom tags, specific features)
   - License/attribution information
5. **Add tests** that exercise the new file:
   ```go
   func TestParseVendorNewSoftware(t *testing.T) {
       f, _ := os.Open("testdata/edge-cases/vendor-newsoftware.ged")
       doc, err := decoder.Decode(f)
       require.NoError(t, err)
       // Verify expected custom tags, structure, etc.
   }
   ```

See the "Contributing Test Files" section above for what vendor exports are most needed.

## Related Documentation

- [GEDCOM Version Differences](GEDCOM_VERSIONS.md) - Detailed spec differences
- [Test Data README](../testdata/README.md) - Complete test file documentation
- [API Stability](API_STABILITY.md) - What APIs are stable vs experimental
