---
name: audit-docs
description: Audit documentation for drift, dead links, and stale content
user_invocable: true
context: fork
agent: Explore
---

# Documentation Drift Audit

Audit project documentation for accuracy, completeness, and internal consistency.

## Checks to Perform

### 1. Package Tree Accuracy
Compare the package list in CLAUDE.md's "Package Structure" section against the actual Go packages:
- Run `ls -d */` in the repo root to find actual packages (directories with .go files)
- Flag any packages in CLAUDE.md that don't exist
- Flag any real packages missing from CLAUDE.md

### 2. FEATURES.md Spot-Check
Pick 3 features claimed in FEATURES.md and verify they exist:
- Search for the exported symbols mentioned (types, functions)
- Check that code examples reference real API signatures
- Flag any features that appear to be missing or renamed

### 3. TESTING.md Critical Paths
Verify critical path functions listed in docs/TESTING.md exist:
- Search for each function name in the corresponding package
- Flag functions that have been renamed or removed
- Check that test files exist for the listed functions

### 4. ADR List Accuracy
Compare the ADR table in CLAUDE.md against actual files in docs/adr/:
- List files in docs/adr/
- Flag any ADRs in CLAUDE.md not backed by a file
- Flag any ADR files not listed in CLAUDE.md

### 5. README.md Version Claims
Check that version support claims match actual code:
- Search for version constants in the codebase
- Verify version detection code handles claimed versions

### 6. Dead Internal Links
Scan all markdown files for internal links and verify targets exist:
- Check relative links like `[text](./path)` and `[text](path)`
- Check anchor links like `[text](#section)`
- Flag any broken links

## Output Format

```
## Documentation Drift Audit

### Summary
| Check | Status | Findings |
|-------|--------|----------|
| Package tree | ... | ... |
| FEATURES.md | ... | ... |
| TESTING.md | ... | ... |
| ADR list | ... | ... |
| Version claims | ... | ... |
| Dead links | ... | ... |

### Details
[Detailed findings per check]

### Recommendations
[Top items to fix, ordered by severity]
```
