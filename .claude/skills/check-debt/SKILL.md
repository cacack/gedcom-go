---
name: check-debt
description: Inventory technical debt across the codebase
user_invocable: true
context: fork
agent: Explore
---

# Technical Debt Inventory

Scan for technical debt indicators and prioritize them by impact.

## Checks to Perform

### 1. TODO/FIXME/HACK Comments
Search all .go files for debt markers:
- `TODO`, `FIXME`, `HACK`, `XXX`, `NOCOMMIT`
- Group by package with counts
- Include the comment text for context

### 2. Long Functions
Find functions exceeding 100 lines (refactoring candidates):
- Search for `func ` declarations and measure line counts
- List functions over 100 lines with their package and line count
- Note any over 200 lines as high-priority

### 3. Open GitHub Issues
Summarize open issues by effort/value labels:
- Run `gh issue list --state open --json number,title,labels`
- Group by effort label (low/medium/high)
- Highlight any high-value items that haven't been started

### 4. Coverage Gaps
Check for packages below the 85% threshold:
- Run `go test -cover ./...` and parse output
- Flag any package below 85%
- Note packages close to the threshold (85-88%)

### 5. IDEAS.md Staleness
Review IDEAS.md for items that may have been implemented:
- Check if any ideas already exist as features in the codebase
- Flag ideas that duplicate open issues
- Note any ideas with no clear path forward

## Output Format

```
## Technical Debt Inventory

### Summary
| Category | Count | Severity |
|----------|-------|----------|
| TODO/FIXME comments | ... | ... |
| Long functions | ... | ... |
| Open issues | ... | ... |
| Coverage gaps | ... | ... |
| Stale ideas | ... | ... |

### Details
[Findings grouped by category]

### Top 5 Items to Address
[Prioritized by impact and effort]
```
