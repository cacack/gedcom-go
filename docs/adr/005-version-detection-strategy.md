# ADR-005: GEDCOM Version Detection Strategy

**Status**: Accepted
**Date**: 2025-01-19
**Context**: GEDCOM version detection in gedcom-go library

## Decision

Use header-first detection (`HEAD/GEDC/VERS`) with tag-based fallback. When the header version is missing or unclear, detect version by presence of version-specific tags.

## Context

GEDCOM has evolved through multiple versions with different capabilities:

| Version | Key Differences |
|---------|-----------------|
| 5.5 | Legacy, limited structure |
| 5.5.1 | Added MAP coordinates, multimedia improvements |
| 7.0 | EXID, SCHMA, enhanced associations, no CONC |

Files should declare their version:
```gedcom
0 HEAD
1 GEDC
2 VERS 5.5.1
```

However, many files have:
- Missing version declaration
- Incorrect version (claims 5.5 but uses 7.0 features)
- Version in non-standard location

The question: how do we reliably determine which version rules to apply?

## Decision Drivers

1. **Respect declarations** - Honor explicit version statements
2. **Handle malformed files** - Many real-world files are imperfect
3. **Enable appropriate validation** - Version affects valid tags/structures
4. **Support version conversion** - Need accurate source version

## Detection Strategy

### Primary: Header Version Tag

```gedcom
0 HEAD
1 GEDC
2 VERS 7.0
```

Parsed and trusted when present and valid.

### Fallback: Tag-Based Detection

When header is missing/ambiguous, detect by presence of version-specific tags:

| Version | Indicator Tags |
|---------|---------------|
| 7.0 | `EXID`, `SCHMA`, `PHRASE` on associations |
| 5.5.1 | `MAP`, `LATI`, `LONG`, `EMAIL`, `FAX`, `WWW` |
| 5.5 | Default when no distinguishing tags found |

## Considered Options

### Option A: Strict Header Only

- **Pros**: Simple, respects declarations
- **Cons**: Fails on files without version, can't detect incorrect declarations
- **Verdict**: Rejected - too many files lack proper headers

### Option B: Tag-Based Only (Ignore Header)

- **Pros**: Detects actual features used
- **Cons**: Ignores explicit declarations, may misclassify
- **Verdict**: Rejected - should respect explicit version

### Option C: Header-First with Tag Fallback (Selected)

- **Pros**: Respects explicit declarations, handles missing headers, can detect version mismatches
- **Cons**: Two-pass detection for some files
- **Verdict**: Accepted

## Consequences

### Positive

- Files with proper headers processed quickly (single check)
- Files without headers still work (fallback detection)
- Can warn when declared version doesn't match detected features
- Enables version-specific validation rules

### Negative

- Some ambiguous files may be misclassified (5.5 vs 5.5.1 edge cases)
- Detection requires scanning record tags for fallback

## Implementation

```go
func Detect(doc *Document) Version {
    // Check header first
    if v := doc.Header.Version; v != "" {
        if parsed, ok := parseVersion(v); ok {
            return parsed
        }
    }

    // Fallback: scan for version-specific tags
    if hasTag(doc, "EXID") || hasTag(doc, "SCHMA") {
        return Version70
    }
    if hasTag(doc, "MAP") || hasTag(doc, "EMAIL") {
        return Version551
    }
    return Version55  // Default
}
```

## Version-Specific Features

### GEDCOM 7.0 Unique
- `EXID` (external ID)
- `SCHMA` (schema definitions)
- `PHRASE` on ASSO (human-readable description)
- No `CONC` tags (values can contain embedded newlines)

### GEDCOM 5.5.1 Additions (over 5.5)
- `MAP` with `LATI`/`LONG` coordinates
- `EMAIL`, `FAX`, `WWW` contact fields
- Enhanced multimedia structure

### GEDCOM 5.5 Baseline
- Core tags: INDI, FAM, SOUR, REPO, NOTE
- CONC/CONT for line continuation
- Basic multimedia (BLOB)

## References

- `version/version.go` - Version detection implementation
- `version/v55.go`, `v551.go`, `v70.go` - Version-specific tag definitions
- `docs/GEDCOM_VERSIONS.md` - Detailed version differences
