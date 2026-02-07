---
name: check-invariants
description: Spot-check that code matches ADR invariants
user_invocable: true
context: fork
agent: Explore
---

# ADR Invariant Spot-Checks

Verify that the codebase honors the architectural decisions documented in docs/adr/. Each ADR establishes invariants that must hold.

## Checks to Perform

### ADR-001: Custom Date Struct
- Verify `gedcom.Date` struct has fields for partial dates (Day, Month, Year can be zero)
- Verify modifier fields exist (Modifier, IsBC, DualYear, Phrase)
- Verify calendar type field exists (Calendar)
- Check that `ParseDate()` preserves the original string

### ADR-002: XRef Resolution via Strings
- Verify family linkage fields (Husband, Wife, Children, FamilyChild, FamilySpouse) are string types, not pointer types
- Verify `Document.XRefMap` exists for O(1) lookup
- Verify `GetIndividual()`, `GetFamily()` etc. use the map

### ADR-003: Lossless Dual Storage
- Verify `Record` has both a typed `Data` field and raw `Children`/`Tags` field
- Verify entities (Individual, Family, etc.) preserve raw tags
- Spot-check that unknown/vendor tags survive a decode

### ADR-004: Encoding Detection Cascade
- Verify charset package implements BOM detection
- Verify header-based encoding detection exists
- Verify UTF-8 fallback is the default

### ADR-005: Version Detection Strategy
- Verify version package checks header first
- Verify tag-based heuristic fallback exists
- Verify all three versions (5.5, 5.5.1, 7.0) are detected

### ADR-006: Line Continuation Handling
- Verify CONT handling in parser or decoder (newline insertion)
- Verify CONC handling (concatenation without newline)
- Check that multiline text round-trips correctly

### ADR-007: Error Transparency
- Verify error types include LineNumber or line context
- Grep for `panic(` in non-test library code — should be zero occurrences
- Verify errors use `fmt.Errorf` with `%w` for wrapping

### ADR-008: Validator Architecture
- Verify validators are composable (not one monolithic function)
- Check for configurable validation (strictness levels, options)
- Verify individual validation functions can be called independently

## Output Format

```
## ADR Invariant Check

### Summary
| ADR | Status | Notes |
|-----|--------|-------|
| 001 - Custom Date | ... | ... |
| 002 - XRef Strings | ... | ... |
| 003 - Dual Storage | ... | ... |
| 004 - Encoding Cascade | ... | ... |
| 005 - Version Detection | ... | ... |
| 006 - Line Continuation | ... | ... |
| 007 - Error Transparency | ... | ... |
| 008 - Validator Architecture | ... | ... |

### Details
[Findings per ADR with code references]

### Violations Found
[Any invariant violations, with file:line references]
```
