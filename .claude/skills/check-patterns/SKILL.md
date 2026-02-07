---
name: check-patterns
description: Check code pattern consistency across the codebase
user_invocable: true
context: fork
agent: Explore
---

# Code Pattern Consistency Check

Verify that the codebase follows consistent Go patterns and project conventions.

## Checks to Perform

### 1. Error Handling Patterns
Spot-check 3 .go files (non-test) for error wrapping:
- Look for `fmt.Errorf` with `%w` verb (preferred pattern)
- Flag any bare `errors.New()` that should include context
- Flag any errors that swallow context (no wrapping)

### 2. Table-Driven Tests
Spot-check 3 _test.go files for table-driven test patterns:
- Look for `[]struct` test case definitions
- Look for `t.Run(` subtest execution
- Flag test files with repetitive non-table patterns

### 3. Godoc Coverage
Check that all exported types and functions have doc comments:
- Search for `^func [A-Z]` and `^type [A-Z]` declarations
- Verify the preceding line is a `//` comment
- Report any undocumented exports (sample 2-3 packages)

### 4. Package doc.go Files
Verify every package has a doc.go file:
- List all Go packages (directories with .go files)
- Check for doc.go in each
- Flag any missing doc.go files

### 5. Interface Usage in Public APIs
Spot-check that public APIs use io.Reader/io.Writer interfaces:
- Check decoder's main entry point accepts io.Reader
- Check encoder's main entry point accepts io.Writer
- Flag any public functions that take concrete types (e.g., *os.File)

### 6. No Panics in Library Code
Grep for `panic(` in all non-test .go files:
- Exclude _test.go files
- Exclude any in main packages (if CLI exists)
- Report any found with file:line references

## Output Format

```
## Code Pattern Consistency

### Summary
| Pattern | Status | Findings |
|---------|--------|----------|
| Error handling | ... | ... |
| Table-driven tests | ... | ... |
| Godoc coverage | ... | ... |
| doc.go files | ... | ... |
| Interface APIs | ... | ... |
| No panics | ... | ... |

### Details
[Findings per pattern with code references]

### Recommendations
[Inconsistencies to address, ordered by importance]
```
