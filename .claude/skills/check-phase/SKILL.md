---
name: check-phase
description: Assess whether recent work aligns with the open milestones and the evidence bar
user_invocable: true
context: fork
agent: Explore
---

# Roadmap Alignment Assessment

Check whether recent work aligns with what is planned. Planned work and its order live only in GitHub milestones and issues; `CONSTITUTION.md` holds the principles, including what counts as evidence for a feature. Do not assume any milestone names: read them live.

## Checks to Perform

### 1. Milestone Status
- Read milestones with their descriptions (which carry exit criteria): `gh api 'repos/cacack/gedcom-go/milestones?state=open'`
- Read their issues: `gh issue list --state all --limit 1000 --json number,title,milestone,state,labels`
- For each open milestone, report open vs. closed counts and whether its exit criteria are met
- List open issues with no milestone

### 2. Recent Commit Categorization
Review the last 20 commits on main:
- Run `git log --oneline --max-count=20 main`
- Map each to the issue and milestone it served (from the merge commit or PR), or to Maintenance (ci, docs, chore, deps)
- Flag feature work tied to no issue

### 3. Evidence Check
Per `CONSTITUTION.md` ("What counts as evidence"), a feature needs corpus behavior, spec coverage, or a downstream request:
- For each recent `feat` commit, find the evidence its issue or commit body cites
- Flag speculative features with none

## Output Format

```
## Roadmap Alignment Assessment

### Milestone Status
| Milestone | Open | Closed | Exit criteria met? |
|-----------|------|--------|--------------------|
| ... | ... | ... | ... |

### Recent Work Distribution
| Milestone / Maintenance | Commits | Percentage |
|-------------------------|---------|------------|
| ... | ... | ... |

### Unplanned or Speculative Work
[Feature work with no issue or no cited evidence]

### Recommendations
[Suggestions for staying aligned with the open milestones]
```
