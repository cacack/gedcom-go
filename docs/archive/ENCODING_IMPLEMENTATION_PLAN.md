# Character Encoding Implementation Plan

**Status:** 📋 Planning Document
**Priority:** Medium (6 test files waiting)

---

## Current State

### ✅ Supported Encodings
- **UTF-8** - Fully implemented with BOM detection and validation
- **ASCII** - Works as subset of UTF-8
- **LATIN1 (ISO-8859-1)** - Recognized but not converted

### ⏭️ Pending Encodings

| Encoding | Files Waiting | Complexity | Priority |
|----------|---------------|------------|----------|
| **UTF-16 LE/BE** | 2 files | Low | High |
| **ANSEL** | 4 torture test files | High | Medium |
| **LATIN1** | 0 files currently | Low | Low |

---

## Implementation Requirements

### 1. UTF-16 Support (UNICODE in GEDCOM)

**Status:** 2 test files ready (utf16le.ged, utf16be.ged)

#### What's Needed

**A. BOM Detection & Endianness**
```
UTF-16 LE BOM: 0xFF 0xFE
UTF-16 BE BOM: 0xFE 0xFF
```

**B. Dependencies**
- Standard library: `golang.org/x/text/encoding/unicode`
- OR third-party: `github.com/dimchansky/utfbom`

**C. Implementation Approach**

**Option 1: Using golang.org/x/text (Recommended)**
```go
import (
    "golang.org/x/text/encoding/unicode"
    "golang.org/x/text/transform"
)

// Detect BOM and create appropriate transformer
func NewUTF16Reader(r io.Reader) io.Reader {
    // Read first 2 bytes for BOM detection
    // Create unicode.UTF16(endianness, unicode.IgnoreBOM)
    // Wrap with transform.NewReader for conversion
    // Return transformer that converts UTF-16 → UTF-8
}
```

**Option 2: Using github.com/dimchansky/utfbom (Simpler)**
```go
import "github.com/dimchansky/utfbom"

func NewUTF16Reader(r io.Reader) io.Reader {
    // utfbom.Skip() automatically detects and handles BOM
    // Returns reader with encoding detected
    reader, encoding := utfbom.Skip(r)
    // Then convert based on encoding
}
```

**D. Integration Points**

1. **charset/charset.go** - Add UTF16 detection
   ```go
   func NewReader(r io.Reader, declaredEncoding Encoding) io.Reader {
       // Peek at BOM
       // If UTF-16 BOM detected, return UTF16Reader
       // Otherwise, existing UTF-8 handling
   }
   ```

2. **decoder/decoder.go** - Pass encoding from CHAR tag
   ```go
   // After parsing HEAD.CHAR, determine encoding
   reader := charset.NewReader(input, header.Encoding)
   ```

**E. Complexity Assessment**

- **Effort:** 4-6 hours
- **Testing:** Use existing utf16le.ged, utf16be.ged
- **Risk:** Low (well-supported in Go ecosystem)

**F. Files to Modify**
- `charset/charset.go` - Add UTF16Reader type
- `charset/charset_test.go` - Add UTF-16 conversion tests
- `decoder/integration_test.go` - Un-skip UTF-16 tests

---

### 2. ANSEL Support (ANSI Z39.47)

**Status:** 4 torture test files waiting (TGC55*.ged)

#### What's Needed

**A. Understanding ANSEL**

ANSEL is a complex character encoding used primarily in library systems:
- Standard: ANSI Z39.47-1985 (now withdrawn as of 2013)
- Used extensively in GEDCOM 5.5 files
- No official Unicode mapping provided by FamilySearch
- **Multi-byte encoding with combining diacriticals**

**B. Character Set Composition**

1. **ASCII subset** (0x00-0x7F) - Direct mapping
2. **Extended Latin** (0xA0-0xFF) - Requires mapping table
3. **Combining diacriticals** (0xE0-0xFF) - Multi-byte sequences

**C. Implementation Challenges**

**Challenge 1: No Standard Mapping**
- ANSEL → Unicode mapping not officially defined
- Different implementations use different mappings
- Must consult multiple sources (MARC-21, various GEDCOM tools)

**Challenge 2: Combining Characters**
- ANSEL uses combining diacriticals BEFORE base character
- Unicode uses combining marks AFTER base character
- Requires character reordering

**Challenge 3: Multi-byte Sequences**
- Some ANSEL characters span 2-3 bytes
- Need stateful decoder

**D. Research Required**

Sources to consult:
1. **MARC-21 Character Sets**: https://www.loc.gov/marc/specifications/
   - Library of Congress maintains ANSEL mapping for MARC records
2. **Tamura Jones articles**: https://www.tamurajones.net/GEDCOMCharacterEncodings.xhtml
   - Genealogy expert with detailed ANSEL research
3. **Existing implementations**:
   - python-ansel: https://python-ansel.readthedocs.io/
   - Gramps genealogy software (Python)
   - GEDCOM.NET (C#)

**E. Implementation Approach**

**Option 1: Full Implementation (High Effort)**
```go
package charset

// ANSELReader converts ANSEL encoding to UTF-8
type anselReader struct {
    reader       io.Reader
    buffer       []byte
    combiningBuf []rune // Buffer for combining diacriticals
}

// Conversion table: ANSEL byte → Unicode code point
var anselToUnicode = map[byte]rune{
    0xA1: 0x0141, // Ł (L with stroke)
    0xA2: 0x00D8, // Ø (O with stroke)
    // ... 200+ mappings needed
}

// Combining characters require special handling
var combiningDiacriticals = map[byte]rune{
    0xE0: 0x0309, // Combining hook above
    0xE1: 0x0300, // Combining grave
    // ... 50+ combining marks
}
```

**Option 2: Use Existing Library (If Available)**
- Check if golang ANSEL library exists
- Port python-ansel to Go
- Use cgo to wrap C implementation

**Option 3: Partial Implementation (Pragmatic)**
```go
// Support common ANSEL characters only
// Log warning for unsupported sequences
// Provide "best effort" conversion
// Document known limitations
```

**F. Complexity Assessment**

- **Research:** 8-16 hours (mapping table creation)
- **Implementation:** 16-24 hours (decoder + tests)
- **Testing:** 8 hours (torture test validation)
- **Total Effort:** 32-48 hours
- **Risk:** High (edge cases, compatibility issues)

**G. Files to Modify**
- `charset/ansel.go` - New file with ANSEL decoder
- `charset/ansel_table.go` - Conversion tables
- `charset/ansel_test.go` - Comprehensive ANSEL tests
- `charset/charset.go` - Integrate ANSEL detection
- `decoder/integration_test.go` - Un-skip torture tests

**H. Recommended Approach**

**Phase 1: Research & Validation**
1. Create ANSEL → Unicode mapping table from authoritative sources
2. Validate against known GEDCOM files
3. Compare with other implementations (python-ansel, Gramps)

**Phase 2: Basic Implementation**
1. Implement ASCII pass-through (0x00-0x7F)
2. Implement extended Latin mapping (0xA0-0xDF)
3. Test with simple ANSEL files

**Phase 3: Combining Characters**
1. Implement combining diacriticals (0xE0-0xFF)
2. Handle character reordering
3. Test with torture test files

**Phase 4: Edge Cases**
1. Handle invalid sequences
2. Multi-byte character support
3. Comprehensive error handling

---

### 3. LATIN1 (ISO-8859-1) Support

**Status:** No test files waiting, but recognized encoding

#### What's Needed

**A. Complexity**
- **Effort:** 2-4 hours
- **Risk:** Low (straightforward 1:1 mapping)

**B. Implementation**

LATIN1 is simpler than ANSEL:
- Bytes 0x00-0x7F: Same as ASCII
- Bytes 0x80-0xFF: Direct Unicode mapping (U+0080 to U+00FF)

```go
// LATIN1 → UTF-8 conversion
func decodeLatin1(b byte) rune {
    return rune(b) // Direct mapping!
}
```

**C. Can use golang.org/x/text**
```go
import "golang.org/x/text/encoding/charmap"

reader := transform.NewReader(input, charmap.ISO8859_1.NewDecoder())
```

---

## Implementation Priority

### Recommended Order

**1. UTF-16 Support (High Priority, Low Complexity)**
- ✅ Clear specification
- ✅ Well-supported in Go
- ✅ 2 test files ready
- ⏱️ Effort: 4-6 hours
- 📊 Impact: Immediate test coverage improvement

**2. LATIN1 Support (Medium Priority, Low Complexity)**
- ✅ Simple implementation
- ✅ Standardized mapping
- ⚠️ No test files (create synthetic)
- ⏱️ Effort: 2-4 hours
- 📊 Impact: Completeness

**3. ANSEL Support (Medium Priority, High Complexity)**
- ⚠️ Complex specification
- ⚠️ No official mapping
- ⚠️ Requires extensive research
- ✅ 4 test files ready (torture tests)
- ⏱️ Effort: 32-48 hours
- 📊 Impact: GEDCOM 5.5 completeness

---

## Dependencies to Add

### Required
```bash
go get golang.org/x/text/encoding/unicode
go get golang.org/x/text/transform
go get golang.org/x/text/encoding/charmap
```

### Optional (for simpler UTF-16 BOM handling)
```bash
go get github.com/dimchansky/utfbom
```

---

## Testing Strategy

### UTF-16
- ✅ Test files ready: utf16le.ged, utf16be.ged
- Add unit tests for BOM detection
- Add edge cases: no BOM, invalid BOM

### ANSEL
- ✅ Test files ready: TGC551.ged, TGC551LF.ged, TGC55C.ged, TGC55CLF.ged
- Create unit tests for:
  - ASCII pass-through
  - Extended Latin characters
  - Combining diacriticals
  - Multi-byte sequences
  - Invalid sequences

### LATIN1
- ⚠️ Need to create test file
- Test all 256 byte values
- Verify UTF-8 output

---

## Architecture Changes

### Current Architecture
```
Input → charset.NewReader() → UTF-8 validation → Parser
                  ↓
         (Only handles UTF-8 BOM)
```

### Proposed Architecture
```
Input → charset.NewReader(declaredEncoding) → Encoding detection
                                    ↓
                    ┌───────────────┼───────────────┐
                    ↓               ↓               ↓
                UTF-8          UTF-16          ANSEL
                Reader         Reader          Reader
                    ↓               ↓               ↓
                    └───────────────┼───────────────┘
                                    ↓
                            UTF-8 validated stream
                                    ↓
                                 Parser
```

### Key Changes

1. **charset.NewReader() signature change**
   ```go
   // Before
   func NewReader(r io.Reader) io.Reader

   // After
   func NewReader(r io.Reader, encoding Encoding) io.Reader
   ```

2. **Decoder integration**
   ```go
   // In decoder.Decode()
   // Parse header to get CHAR tag
   // Create charset reader with declared encoding
   reader := charset.NewReader(input, doc.Header.Encoding)
   ```

3. **BOM takes precedence**
   - If BOM detected, override declared encoding
   - UTF-8 BOM → UTF-8
   - UTF-16 LE BOM → UTF-16 LE
   - UTF-16 BE BOM → UTF-16 BE

---

## Risks & Mitigations

### Risk 1: ANSEL Mapping Incompatibility
**Impact:** High
**Probability:** Medium
**Mitigation:**
- Use multiple authoritative sources
- Provide configuration option for mapping variant
- Document known differences
- Provide "strict" vs "lenient" modes

### Risk 2: Performance Impact
**Impact:** Medium
**Probability:** Low
**Mitigation:**
- Benchmark conversion overhead
- Use buffered I/O
- Consider lazy conversion
- Profile with large files

### Risk 3: Breaking Changes
**Impact:** Medium
**Probability:** Medium
**Mitigation:**
- Keep existing NewReader() as default UTF-8
- Add NewReaderWithEncoding(r, enc)
- Maintain backward compatibility
- Clear migration guide

---

## Success Criteria

### UTF-16 Implementation
- ✅ Both test files parse correctly
- ✅ BOM correctly detected (LE and BE)
- ✅ Content matches UTF-8 versions
- ✅ Integration tests pass
- ✅ No performance regression (<10% overhead)

### ANSEL Implementation
- ✅ All 4 torture test files parse
- ✅ Special characters render correctly
- ✅ Combining diacriticals properly ordered
- ✅ Content matches expected Unicode
- ✅ Error handling for invalid sequences

### LATIN1 Implementation
- ✅ All 256 bytes correctly mapped
- ✅ Test file with extended characters parses
- ✅ Integration tests pass

---

## Quick Start Guide

### Implementing UTF-16 (Fastest Win)

**Step 1: Add dependency**
```bash
go get golang.org/x/text/encoding/unicode
go get golang.org/x/text/transform
```

**Step 2: Create UTF-16 reader**
```go
// charset/utf16.go
package charset

import (
    "io"
    "golang.org/x/text/encoding/unicode"
    "golang.org/x/text/transform"
)

func NewUTF16Reader(r io.Reader, bigEndian bool) io.Reader {
    endian := unicode.LittleEndian
    if bigEndian {
        endian = unicode.BigEndian
    }

    decoder := unicode.UTF16(endian, unicode.IgnoreBOM).NewDecoder()
    return transform.NewReader(r, decoder)
}
```

**Step 3: Add BOM detection**
```go
func DetectBOM(r io.Reader) (io.Reader, Encoding, error) {
    buf := make([]byte, 4)
    n, _ := io.ReadFull(r, buf)

    // UTF-16 LE: FF FE
    if n >= 2 && buf[0] == 0xFF && buf[1] == 0xFE {
        return io.MultiReader(bytes.NewReader(buf[2:n]), r), EncodingUTF16LE, nil
    }

    // UTF-16 BE: FE FF
    if n >= 2 && buf[0] == 0xFE && buf[1] == 0xFF {
        return io.MultiReader(bytes.NewReader(buf[2:n]), r), EncodingUTF16BE, nil
    }

    // UTF-8 BOM: EF BB BF
    if n >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
        return io.MultiReader(bytes.NewReader(buf[3:n]), r), EncodingUTF8, nil
    }

    // No BOM
    return io.MultiReader(bytes.NewReader(buf[:n]), r), EncodingUnknown, nil
}
```

**Step 4: Update integration tests**
```go
// In decoder/integration_test.go
// Remove skip reason, run tests
```

**Estimated time:** Half day

---

## Next Steps

1. **Decision**: Prioritize UTF-16 or ANSEL?
   - Recommend: UTF-16 first (quick win)

2. **Create GitHub issue** for tracking

3. **ANSEL Research Phase**:
   - Study python-ansel implementation
   - Build conversion table
   - Validate against MARC-21 spec

4. **Implementation Phases**:
   - Phase 1: UTF-16 (1 week)
   - Phase 2: LATIN1 (few days)
   - Phase 3: ANSEL (3-4 weeks)

---

## References

### UTF-16
- Go text encoding: https://pkg.go.dev/golang.org/x/text/encoding/unicode
- UTF-16 specification: https://datatracker.ietf.org/doc/html/rfc2781

### ANSEL
- GEDCOM Character Encodings: https://www.tamurajones.net/GEDCOMCharacterEncodings.xhtml
- MARC-21 Character Sets: https://www.loc.gov/marc/specifications/
- python-ansel: https://python-ansel.readthedocs.io/
- ANSI Z39.47 (withdrawn): Historical reference

### GEDCOM Specifications
- GEDCOM 5.5: Character set specification
- GEDCOM 5.5.1: UTF-8 support added
- GEDCOM 7.0: UTF-8 only (simplified)
