---
name: check-phase
description: Assess alignment with the current roadmap phase
user_invocable: true
context: fork
agent: Explore
---

# Phase Alignment Assessment

Check whether recent work aligns with the current roadmap phase (Phase 1: API Polish & Stability).

## Checks to Perform

### 1. Recent Commit Categorization
Review the last 20 commits on main:
- Run `git log --oneline -20`
- Categorize each as Phase 1, Phase 2, Phase 3, or Maintenance based on docs/ROADMAP.md
- Calculate percentage of work in each phase

### 2. Phase 1 Issue Progress
Check status of Phase 1 issues from docs/ROADMAP.md:
- #156 — Options types for Encode and Validate
- #135 — Place parsing helpers
- #141 — Vendor test data (Legacy Family Tree)
- #44 — CLI tool for GEDCOM validation
- Report which are open, closed, or in progress

### 3. Phase Leakage Detection
Look for signs of Phase 2/3 work happening before Phase 1 is complete:
- Check recent commits for merge/diff/export features (Phase 2)
- Check for GEDZip, builder API, or plugin work (Phase 3)
- Flag any premature work with context

### 4. Downstream Consumer Alignment
Check if work is being driven by real downstream usage:
- Look at recent commits for references to my-family or consumer needs
- Check if features added map to downstream requirements
- Flag any speculative features not tied to real needs

## Output Format

```
## Phase Alignment Assessment

### Current Phase: Phase 1 — API Polish & Stability

### Recent Work Distribution
| Phase | Commits | Percentage |
|-------|---------|------------|
| Phase 1 | ... | ... |
| Phase 2 | ... | ... |
| Phase 3 | ... | ... |
| Maintenance | ... | ... |

### Phase 1 Issue Status
| Issue | Title | Status |
|-------|-------|--------|
| #156 | Options types | ... |
| #135 | Place parsing | ... |
| #141 | Vendor test data | ... |
| #44 | CLI tool | ... |

### Phase Leakage
[Any premature Phase 2/3 work detected]

### Recommendations
[Suggestions for staying focused on the current phase]
```
