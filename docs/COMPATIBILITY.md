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
| Gramps | 5.1.6 | 🧪 | Synthetic test file; custom tags preserved |
| MyHeritage | - | 🧪 | Synthetic test file; custom tags preserved |
| Ancestry.com | - | 🧪 | Synthetic test file; `_APID` and `_TREE` extensions parsed |
| FamilySearch | GEDCOM 7.0 | ✅ | Spec examples only (not real exports) |

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

### Ancestry.com (`_APID` Tags)

Ancestry exports include `_APID` tags that reference their database records. We parse these and can reconstruct URLs, but:

- Testing is with a synthetic file, not a real Ancestry export
- Real exports may have additional patterns not covered

### MyHeritage

- Testing is with a synthetic file
- Real exports may have different custom tag patterns

### FamilySearch GEDCOM 7.0

- Testing uses specification examples, which are intentionally minimal
- Real exports from familysearch.org may have additional complexity

## Related Documentation

- [GEDCOM Version Differences](GEDCOM_VERSIONS.md) - Detailed spec differences
- [Test Data README](../testdata/README.md) - Complete test file documentation
- [API Stability](API_STABILITY.md) - What APIs are stable vs experimental
