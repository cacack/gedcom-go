---
name: check-tests
description: Analyze test quality and coverage patterns
user_invocable: true
context: fork
agent: Explore
---

# Test Quality Analysis

Assess test quality beyond simple coverage numbers — edge cases, patterns, and completeness.

## Checks to Perform

### 1. Edge Case Coverage
Spot-check 3 test files for coverage of error and boundary conditions:
- Do tests include invalid/malformed inputs?
- Are nil, empty, and zero values tested?
- Are boundary conditions covered (max values, overflow)?

### 2. Table-Driven Test Usage
Check that multi-case functions use table-driven patterns:
- Search for test functions with more than 3 similar assertion blocks
- These are candidates for table-driven refactoring
- Report files that would benefit from the pattern

### 3. Round-Trip Tests
Verify decode-encode-decode cycles exist:
- Search for round-trip or roundtrip in test files
- Check that encoder tests verify output can be re-decoded
- Flag any encode/decode paths without round-trip verification

### 4. Error Message Quality
Spot-check 3 test files for error message assertions:
- Do tests check error message content (not just `!= nil`)?
- Are error types checked with `errors.Is()` or `errors.As()`?
- Flag tests that only check `err != nil` without message validation

### 5. Critical Path Coverage
Cross-reference docs/TESTING.md critical paths with actual tests:
- For each critical function listed, verify a dedicated test exists
- Check that the test covers the "Required Tests" column
- Flag any critical paths without sufficient test coverage

### 6. Missing Test Files
Find .go files without corresponding _test.go files:
- List all non-test .go files (excluding doc.go, generated files)
- Check for corresponding _test.go
- Flag any missing test files (excluding trivially small files)

## Output Format

```
## Test Quality Analysis

### Summary
| Dimension | Status | Findings |
|-----------|--------|----------|
| Edge cases | ... | ... |
| Table-driven | ... | ... |
| Round-trip tests | ... | ... |
| Error messages | ... | ... |
| Critical paths | ... | ... |
| Missing test files | ... | ... |

### Details
[Findings per dimension with file references]

### Recommendations
[Top items to improve test quality]
```
