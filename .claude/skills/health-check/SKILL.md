---
name: health-check
description: Run all project health checks and produce a unified report
user_invocable: true
---

# Project Health Check

Orchestrate all six health check dimensions and aggregate results into a unified report.

## Execution

Launch all six checks as parallel subagents using the Task tool with `subagent_type: Explore`:

1. **audit-docs** — Documentation drift audit
2. **check-debt** — Technical debt inventory
3. **check-invariants** — ADR invariant spot-checks
4. **check-patterns** — Code pattern consistency
5. **check-phase** — Phase alignment assessment
6. **check-tests** — Test quality analysis

For each subagent, use the Skill tool to invoke the corresponding skill (e.g., `/audit-docs`, `/check-debt`, etc.).

Wait for all six to complete, then aggregate their findings.

## Output Format

```
# Project Health Report

## Dashboard
| Dimension | Status | Key Finding |
|-----------|--------|-------------|
| Documentation | ... | ... |
| Technical Debt | ... | ... |
| ADR Invariants | ... | ... |
| Code Patterns | ... | ... |
| Phase Alignment | ... | ... |
| Test Quality | ... | ... |

## Detailed Findings

### Documentation
[Summary from audit-docs]

### Technical Debt
[Summary from check-debt]

### ADR Invariants
[Summary from check-invariants]

### Code Patterns
[Summary from check-patterns]

### Phase Alignment
[Summary from check-phase]

### Test Quality
[Summary from check-tests]

## Top Recommendations
1. ...
2. ...
3. ...
4. ...
5. ...

## What's Going Well
- ...
- ...
- ...
```

## Status Indicators

Use these status indicators in the dashboard:
- **PASS** — No issues found
- **WARN** — Minor issues that should be addressed
- **FAIL** — Significant issues requiring attention
