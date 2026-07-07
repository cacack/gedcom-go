# ADR-001: Custom Date Struct for GEDCOM Dates

**Status**: Accepted
**Date**: 2025-12-21
**Context**: Date parsing implementation for go-gedcom library
**Constitution**: Implements Principle VI (Lossless Representation)

## Decision

Use a custom `Date` struct for representing GEDCOM dates rather than normalizing to Go's `time.Time`. Use stdlib `time.Date()` only for validation of complete Gregorian dates.

## Context

GEDCOM dates have representations that differ fundamentally from precise timestamps:

| GEDCOM Concept | Example | `time.Time` Support |
|----------------|---------|---------------------|
| Partial dates | `1850`, `JAN 1850` | No - requires complete date |
| Modifiers | `ABT 1850`, `BEF 1900` | No - represents precise moment |
| Ranges | `BET 1850 AND 1860` | No - single point in time |
| Periods | `FROM 1880 TO 1920` | No |
| Dual dating | `21 FEB 1750/51` | No |
| B.C. dates | `44 BC` | Limited support |
| Date phrases | `(unknown)` | No |
| Non-Gregorian | Hebrew, Julian, French Republican | No - Gregorian only |

**Note**: These loose representations apply to ALL calendar systems, not just Gregorian:
```
@#DHEBREW@ 5765                    # Year only (Hebrew)
@#DJULIAN@ 1750                    # Year only (Julian)
ABT @#DHEBREW@ 15 NSN 5765         # Approximate (Hebrew)
BET @#DJULIAN@ 1700 AND 1750       # Range (Julian)
```

The question: normalize GEDCOM dates to `time.Time` or maintain custom structures?

## Decision Drivers

1. **Lossless representation** - This library should not lose information from source GEDCOM files
2. **Leverage stdlib where appropriate** - Don't reimplement calendar math (leap years, days per month)
3. **Domain-appropriate modeling** - Data structures should represent GEDCOM semantics faithfully

## Considered Options

### Option A: Normalize to `time.Time`

- **Pros**: Leverage stdlib for comparison, arithmetic, formatting
- **Cons**: Lossy - cannot represent partial dates, modifiers become disconnected metadata, ranges require separate handling
- **Verdict**: Rejected - violates lossless requirement

### Option B: Custom struct, reimplement calendar math

- **Pros**: Full control, no dependencies
- **Cons**: Reimplements well-tested stdlib functionality, risk of bugs in edge cases (leap years, month lengths)
- **Verdict**: Rejected - unnecessary reimplementation

### Option C: Custom struct, stdlib for validation (Selected)

- **Pros**: Lossless GEDCOM representation, leverages stdlib for validation, no calendar math reimplementation
- **Cons**: Custom comparison logic needed (acceptable - handles partial dates correctly)
- **Verdict**: Accepted

## Consequences

### Positive

- GEDCOM dates represented faithfully without information loss
- Partial dates, modifiers, ranges, and phrases all supported
- Validation of complete Gregorian dates uses battle-tested `time.Date()`
- No custom leap year or days-per-month logic to maintain

### Negative

- Custom `Compare()` method required (already implemented)
- Consumers needing `time.Time` must use `ToTime()` conversion, understanding it's lossy for partial dates

## Implementation

**Custom `Date` struct provides:**
- Storage: Day, Month, Year (0 = unknown), Modifier, EndDate, Calendar, Phrase
- Original string preservation for round-trip fidelity
- Custom `Compare()` handling partial dates (missing = earliest possible)
- `ToTime()` for optional conversion when consumer needs it

**Stdlib `time.Date()` used for:**
- Validation of complete Gregorian dates (non-lossy check)
- Example: Detect that `30 FEB 2020` is invalid without reimplementing calendar math

```go
// Validation using stdlib (non-lossy - just checking)
func (d *Date) Validate() error {
    if d.Day == 0 || d.Month == 0 || d.Year == 0 {
        return nil // Partial dates skip day/month validation
    }
    t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
    if t.Day() != d.Day || int(t.Month()) != d.Month {
        return fmt.Errorf("invalid date: %d %s %d", d.Day, monthName(d.Month), d.Year)
    }
    return nil
}
```

## References

- `.specify/memory/constitution.md` - Project constitution, Principle VI (Lossless Representation)
- `docs/GEDCOM_DATE_FORMATS_RESEARCH.md` - Comprehensive date format specifications
- `gedcom/date.go` - Current implementation
