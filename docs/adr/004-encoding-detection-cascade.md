# ADR-004: Encoding Detection Cascade

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Character encoding detection in gedcom-go library

## Decision

Use a three-tier cascade for encoding detection: (1) BOM detection, (2) GEDCOM CHAR header tag, (3) UTF-8 default. Each tier provides fallback for when the previous tier cannot determine encoding.

## Context

GEDCOM files can be encoded in various character sets:
- UTF-8 (modern standard)
- UTF-16 LE/BE (Windows exports)
- ANSEL (legacy genealogy standard)
- LATIN1/ISO-8859-1 (European legacy)
- ASCII (subset)

The encoding may be indicated by:
- Byte Order Mark (BOM) at file start
- `HEAD.CHAR` tag declaring encoding
- Nothing (implicit)

The question: how do we reliably detect encoding without data corruption?

## Decision Drivers

1. **Correctness** - Never misinterpret character data
2. **Compatibility** - Handle files from various genealogy software
3. **Graceful degradation** - Work even with incomplete/missing declarations
4. **Performance** - Minimize buffering and re-reading

## Detection Cascade

### Tier 1: BOM Detection (Highest Priority)

```
FF FE       → UTF-16 LE
FE FF       → UTF-16 BE
EF BB BF    → UTF-8 (with BOM)
```

- BOM is unambiguous physical evidence
- Takes precedence over any header declaration
- UTF-16 requires BOM for reliable detection

### Tier 2: GEDCOM Header Tag

```gedcom
0 HEAD
1 CHAR UTF-8
```

Recognized values: `UTF-8`, `UNICODE`, `ANSEL`, `ANSI`, `ASCII`, `UTF-16`

- Only checked if BOM didn't indicate UTF-16
- Respects file's self-declared encoding
- `UNICODE` maps to UTF-16 (common in older exports)

### Tier 3: UTF-8 Default (Fallback)

- Modern standard, backward compatible with ASCII
- Most new GEDCOM files are UTF-8
- Invalid UTF-8 sequences detected during parsing

## Considered Options

### Option A: Require Explicit Declaration

- **Pros**: No guessing
- **Cons**: Many files lack proper declarations, would reject valid files
- **Verdict**: Rejected - too strict

### Option B: Heuristic Detection (chardet-style)

- **Pros**: Works without declarations
- **Cons**: Can misdetect, adds complexity/dependencies, slower
- **Verdict**: Rejected - unnecessary for GEDCOM's limited encoding set

### Option C: Cascade with UTF-8 Default (Selected)

- **Pros**: Handles all cases, respects explicit declarations, safe default
- **Cons**: May misinterpret rare legacy files (acceptable)
- **Verdict**: Accepted

## Consequences

### Positive

- UTF-16 files (common from Windows genealogy software) handled correctly
- Legacy ANSEL files decoded properly when declared
- Modern UTF-8 files work without declarations
- No external dependencies for detection

### Negative

- Undeclared ANSEL files may be misread (rare, legacy)
- Requires buffering to peek at header (minimal overhead)

## Implementation

```go
func NewReader(r io.Reader) (io.Reader, error) {
    // Tier 1: Check for BOM
    detectedReader, bomEnc, err := DetectBOM(r)
    if bomEnc == EncodingUTF16LE || bomEnc == EncodingUTF16BE {
        return NewReaderWithEncoding(detectedReader, bomEnc)
    }

    // Tier 2: Check GEDCOM header
    headerReader, headerEnc, err := DetectEncodingFromHeader(detectedReader)
    if headerEnc != EncodingUnknown {
        return NewReaderWithEncoding(headerReader, headerEnc)
    }

    // Tier 3: Default to UTF-8
    return NewReaderWithEncoding(headerReader, EncodingUTF8)
}
```

## Supported Encodings

| Encoding | BOM | Header Values | Notes |
|----------|-----|---------------|-------|
| UTF-8 | `EF BB BF` | `UTF-8` | Default |
| UTF-16 LE | `FF FE` | `UNICODE`, `UTF-16` | Windows common |
| UTF-16 BE | `FE FF` | `UNICODE`, `UTF-16` | Rare |
| ANSEL | - | `ANSEL` | Legacy genealogy |
| LATIN1 | - | `ANSI` | European legacy |
| ASCII | - | `ASCII` | Subset of UTF-8 |

## References

- `charset/charset.go` - Detection implementation
- `charset/bom.go` - BOM detection
- `charset/ansel.go` - ANSEL decoding tables
- `docs/ENCODING_IMPLEMENTATION_PLAN.md` - Detailed encoding notes
