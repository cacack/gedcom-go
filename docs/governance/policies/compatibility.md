# Compatibility Matrix

This document describes gedcom-go's compatibility with various GEDCOM sources. We aim to be transparent about what has been tested and at what confidence level.

## Vendor Compatibility

The following table shows which genealogy software exports have been tested with this library.

| Software | Version Tested | Import | Notes |
|----------|---------------|--------|-------|
| RootsMagic | 11 Essentials (2026) | ✅ | Real export tested; `_UID`, `_TMPLT` source templates, `_EVDEF` event defs |
| RootsMagic | 7.0.2.2 (2015) | ⚠️ | Older version; inline/xref note patterns work |
| Legacy Family Tree | 10.0 (2025) | ✅ | Real export tested (vendored corpus); UTF-8+BOM, real-world junk DATE values tolerated with diagnostics |
| Legacy Family Tree | 8.0 (2016) | ⚠️ | Older version; custom tags preserved (`_TODO`, `_UID`, `_PRIV`) |
| Family Tree Maker | 22.2.5 (2016) | ⚠️ | Older version; Ancestry format with custom tags |
| Family Tree Maker | 17.0 (2005) | ✅ | Real export tested (vendored corpus); `CHAR ANSI` with real Windows-1252 bytes converted correctly |
| Family Historian | 6.2.2 | ⚠️ | Custom tags preserved (`_ATTR`, `_USED`, `_SHAN`, `_SHAR`) |
| HEREDIS | 14 PC | ⚠️ | French locales work; non-standard PLAC FORM handled |
| Gramps | 6.0.6 (2025) | ✅ | Real export tested; `CHAN` records, `TYPE birth`, note refs |
| MyHeritage | 5.5.1 (2025) | ✅ | Real export tested; `_UID`, `RIN`, HTML notes, `QUAY` tags |
| MyHeritage Family Tree Builder | 8 (desktop) | ✅ | Real export tested (vendored corpus); empty `HEAD.SOUR` handled, `MH:` RINs, level-0 `_PUBLISH` record preserved |
| Ancestry.com | 2025.08 | ✅ | Real export tested; `_TREE` parsed, long XRefs, nickname handling |
| FamilySearch | 2025 | ✅ | Real export tested (GEDCOM 5.5.1); `_HASH`/`_LHASH` tags, standardizer note |
| Ancestris | 11 (2025) | ✅ | Real export tested (vendored corpus); 5.5.1 UTF-8, French data, OBJE FILE refs |
| PAF (Personal Ancestral File) | 5.2.18.0 | ✅ | Real export tested (vendored corpus); decodes clean, zero diagnostics |
| Family Origins | 5.0 | ✅ | Real export tested (vendored corpus); AFN tags preserved, DATE trailing qualifiers flagged with diagnostics |
| The Master Genealogist | 1.2a | ✅ | Real export tested (vendored corpus); `CHAR IBMPC` (ASCII payload), custom NUMB tags |
| My Roots (Palm OS) | 4.00 | ✅ | Real export tested (vendored corpus); ANSEL encoding |
| webtreeprint.com | 1.0 | ✅ | Real export tested (vendored corpus); ALIA usage preserved, free-text month DATE flagged |
| EasyTree | V1.0 | ✅ | Real export tested (vendored corpus); nonstandard `CHAR IBM WINDOWS` preserved verbatim |
| Brother's Keeper | 5.2 | ❌ | Real export (vendored corpus); CP437 encoding unsupported — strict decode fails, lenient mode recovers a partial document |

### Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Verified with a committed fixture and passing test (Notes say whether it is a real export or spec examples) |
| ⚠️ | Tested with older software version; current versions may differ |
| ❌ | Known failure; behavior documented and pinned by a test |
| 🧪 | Synthetic test file only; not a real export |
| ❓ | Vendor detection exists but no test data available |

### Important Notes

- **Export column intentionally omitted**: This library produces standard GEDCOM output, not vendor-specific formats. All encoding is spec-compliant.
- **Older versions**: Test files for FTM 22.2.5, Family Historian, and HEREDIS are from 2015-2016 era software. Current versions may have different export patterns. (Legacy is covered by both a current 10.0 export and the older 8.0 file.)
- **"Vendored corpus" rows**: Real exports vendored byte-identical (aside from submitter-contact redaction) from the [D-Jeffrey/gedcom-samples](https://github.com/D-Jeffrey/gedcom-samples) corpus (MIT OR CC0-1.0); several are vintage platforms (PAF, Family Origins, TMG, My Roots, EasyTree, Brother's Keeper) valuable for exercising historical export quirks. Exercised by `decoder/corpus_vendor_test.go`.
- **FamilySearch**: The ✅ is backed by a real GEDCOM **5.5.1** export from FamilySearch.org (2025). FamilySearch has since retired its own tree-download tool — [familysearch.org/innovate/export](https://www.familysearch.org/innovate/export) states "GEDCOM export is no longer supported using this resource" (verified 2026-08-15) — so no newer first-party export is obtainable. Our GEDCOM 7.0 coverage comes from [official spec examples](https://gedcom.io/tools/).

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

Detection is implemented in [`gedcom/vendor.go`](../../../gedcom/vendor.go).

Currently detected vendors:
- Ancestry.com (including FamilyTreeMaker)
- FamilySearch
- RootsMagic
- Legacy Family Tree
- Gramps
- MyHeritage

## Test Data Sources

All test files are documented in [`testdata/README.md`](../../../testdata/README.md) with:

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
| [D-Jeffrey/gedcom-samples](https://github.com/D-Jeffrey/gedcom-samples) | 12 vendor exports (modern: Legacy 10, Ancestris 11, MyHeritage FTB 8; the rest vintage) in `edge-cases/`+`encoding/`, `gedcom-5.5.1/longsword.ged` | MIT OR CC0-1.0 |
| [gedcom7code/test-files](https://github.com/gedcom7code/test-files) | `edge-cases/atsign-55.ged`, `xref-case.ged`, `age-keywords-551.ged`, `date-dual-years.ged` | Unlicense (public domain) |
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

**Note**: Our FamilySearch fixture is GEDCOM 5.5.1, and the GEDCOM 7.0 spec examples in our test suite are from gedcom.io documentation, not real FamilySearch exports.

**Note (2026+)**: FamilySearch has retired its own family-tree download. [familysearch.org/innovate/export](https://www.familysearch.org/innovate/export) now states "GEDCOM export is no longer supported using this resource" and directs users to third-party apps in its Solution Gallery; the tool produced **GEDCOM 7** before it was withdrawn. The 5.5.1 fixture above is therefore the newest first-party FamilySearch export we can obtain, and a real FamilySearch GEDCOM 7 export is no longer acquirable directly — only indirectly, via a Solution Gallery app re-exporting FamilySearch-sourced data (which is what `edge-cases/legacy10-2025-export.ged` is).

### Encoding and Grammar Limitations

- **CP437 (IBMPC) is not supported.** Files declaring `CHAR IBMPC` with genuine CP437 bytes fail strict decoding ("error reading input"); lenient mode recovers a partial document. Pinned by `encoding/ibmpc-cp437-broskeep.ged` and `TestCorpusVendorCP437KnownFailure` in `decoder/corpus_vendor_test.go`.
- **UTF-16 without a BOM is not detected.** The encoding cascade (ADR 0004) is BOM → header declaration → UTF-8 fallback; BOM-less UTF-16 falls through. No fixture yet — a known follow-up from the #301 research.
- **Some standard tags are recognized but not decoded into typed fields.** At the `INDI` level, `ALIA`, `AFN`, `RFN`, `RIN`, `ANCI`, `DESI`, `SUBM`, `RESN`, and `INIL` are preserved in `Individual.Tags` only — no typed accessor yet. `INIL` stays that way by decision ([#386](https://github.com/cacack/gedcom-go/issues/386)): a typed initiatory ordinance needs a sixth `gedcom.LDSOrdinanceType` constant that no consumer reads. At the `FAM` level, `RESN`, `SUBM`, `ASSO`, and `FACT` are the same case — `gedcom.Family` has no `Attributes` field, so 7.0's `FAM`-level `FACT` is recognized but only reachable via `Family.Tags`. None of these produce `UNKNOWN_TAG`, which is reserved for genuinely nonstandard tags ([#375](https://github.com/cacack/gedcom-go/issues/375)). The recognition tables for subordinate structures still classify some standard tags as `UNKNOWN_TAG`.
- **Levels above 99 are rejected, not clamped.** The GEDCOM level field is at most two digits, so the parser accepts levels 0-99 (`parser.MaxNestingDepth-1`) and drops anything deeper with a `CodeInvalidLevel` error — the line's content is lost. `CodeBadLevelJump` always means the opposite: the line was clamped and *kept*.
- **The nesting ceiling is not configurable.** `decoder.DecodeOptions.MaxNestingDepth` has no effect and is deprecated: the decoder never reads it, and the ceiling is fixed at `parser.MaxNestingDepth-1` (99) by the grammar's two-digit level field, so setting the option neither lowers nor raises it. The field is kept for source compatibility through the v2 series and will be removed in v3 ([#383](https://github.com/cacack/gedcom-go/issues/383)).
- **`INVALID_XREF` covers three different outcomes.** An XRef containing a space (`0 @NoTe ref@ NOTE x`), and a level-0 XRef with no closing `@` followed by a tag (`0 @I1 INDI`), are recovered and the record *kept* with a usable `Record.XRef`. A well-formed XRef with no tag (`0 @I1@`) is *dropped*, line and subordinates alike. And `0 @I1` — unterminated **and** with no tag — is neither: the line keeps its pre-existing parse, so the record is kept but with `Type == "@I1"`, `XRef == ""` and `Entity == nil`, absent from `Document.XRefMap`, and `GetIndividual("@I1")` returns nil. The same three-way split applies to `0 @I1 HEAD` and `0 @I1 TRLR`, whose tags the decoder reserves for structural use; promoting them would silently overwrite the real header or drop the record. All of these surface as `CodeInvalidXRef` at `SeverityError`, so severity does not distinguish them — check `Document.XRefMap` if you need to know which happened.
- **A recovered XRef is stored verbatim, so it may not be pointer-shaped.** `0 @I1 INDI` is recovered as `Record.XRef == "@I1"` — no closing `@` is invented, so re-encoding is byte-exact but `gedcom.IsPointerXRef` (which requires both delimiters) rejects the identifier. Every `@I1@` pointer elsewhere in the file therefore stays unresolved. `Document.GetIndividual("@I1")` works; `GetIndividual("@I1@")` does not. What the recovery buys is the diagnostic and a correctly typed record, not pointer resolution.
- **`merge` rejects a document containing a recovered non-pointer-shaped XRef.** `merge.RemapXRefs` validates every rewritten identifier with `gedcom.IsPointerXRef`, so a `Record.XRef` of `@I1` fails it and the call returns a `*RemapError` for the *whole document* — not just that record. `merge.Combine` with `PrefixDoc2` or `RenumberDoc2` fails the same way, taking the other document down with it. This strictness is not specific to unterminated XRefs: a spaced identifier recovered per #377 fails identically ([#397](https://github.com/cacack/gedcom-go/issues/397)).
- **An unterminated XRef whose line holds a second `@` is not recovered.** Recovery requires exactly one `@` on the line, because every competing reading of a lone `@` carries a second one. `0 @N1 NOTE write to a@b.com` is genuinely ambiguous — the email's `@` could be the terminator — so it keeps its pre-existing parse (`@N1` becomes the tag) with no diagnostic. Widening the rule would let a later pointer pose as the terminator ([#385](https://github.com/cacack/gedcom-go/issues/385)).
- **Payload-grammar strictness is deliberately out of scope for the decoder.** Invalid enum casing, malformed media types, out-of-range date parts, and similar payload-grammar violations are tolerated and preserved losslessly per ADR 0007 (error transparency) rather than rejected. Enforcing them is validator territory (ADR 0008 pluggable rules); peer libraries' `*-invalid.ged` strictness fixtures were reviewed and intentionally not vendored.

## How We Test Compatibility

This section explains how compatibility claims in this document are verified, enabling you to audit our process or reproduce tests locally.

### Where Sample GEDCOMs Live

Test files are organized under `testdata/` by GEDCOM version and purpose:

```
testdata/
├── gedcom-5.5/          # GEDCOM 5.5 samples
│   └── torture-test/    # Comprehensive TGC55* validation suite
├── gedcom-5.5.1/        # GEDCOM 5.5.1 samples (EMAIL/FAX/WWW tags; 46 MiB scale fixture longsword.ged)
├── gedcom-7.0/          # GEDCOM 7.0 samples
│   └── familysearch-examples/  # Official FamilySearch edge cases
├── encoding/            # Character encoding tests (UTF-8, UTF-16, ANSEL, CP1252/CP437 vintage exports)
├── edge-cases/          # Structural edge cases, vendor-specific exports, structural torture fixtures
│   └── vendor-*.ged     # Vendor-specific custom tag tests
└── malformed/           # Invalid files for error handling tests
```

See [`testdata/README.md`](../../../testdata/README.md) for complete attribution, licensing, and descriptions of each file.

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

- [Vendor Extensions](../../guides/vendor-extensions.md) - Per-tag reference for vendor-specific extensions, typed accessors, and validation
- [GEDCOM Version Differences](../../reference/gedcom-versions.md) - Detailed spec differences
- [Test Data README](../../../testdata/README.md) - Complete test file documentation
- [API Stability](api-stability.md) - What APIs are stable vs experimental
