---
name: check-phase
description: Assess alignment with the current roadmap phase
user_invocable: true
context: fork
agent: Explore
---

# Phase Alignment Assessment

Check whether recent work aligns with the current roadmap phase. Phases and their milestone mapping are defined in the Phasing table in `CONSTITUTION.md`; the live issue list and exit criteria live in GitHub milestones. Phase 1 (Real-World Compatibility & API Polish, milestones `v2.1.0`/`v2.2.0`) is the current priority.

## Checks to Perform

### 1. Recent Commit Categorization
Review the last 20 commits on main:
- Run `git log --oneline --max-count=20 main`
- Categorize each as Phase 1, Phase 2, Phase 3, or Maintenance using the Phasing table in CONSTITUTION.md (map by milestone: v2.1.0/v2.2.0 → Phase 1, v2.3.0 → Phase 2, unscheduled advanced work → Phase 3)
- Calculate percentage of work in each phase

### 2. Phase 1 Issue Progress
Check status of the current-phase milestones against their exit criteria:
- Read all issues grouped by milestone (include closed, so open/closed counts are accurate): `gh issue list --state all --limit 1000 --json number,title,milestone,state,labels`
- Read milestone descriptions (which carry exit criteria): `gh api repos/cacack/gedcom-go/milestones`
- For each Phase 1 milestone (`v2.1.0`, `v2.2.0`), report open vs. closed counts and whether exit criteria are met

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

### Phase 1 Milestone Status
| Milestone | Open | Closed | Exit criteria met? |
|-----------|------|--------|--------------------|
| v2.1.0 | ... | ... | ... |
| v2.2.0 | ... | ... | ... |

### Phase Leakage
[Any premature Phase 2/3 work detected]

### Recommendations
[Suggestions for staying focused on the current phase]
```
