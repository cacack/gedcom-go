# Handoff: Add "build vs reuse" check to planning workflows

**From**: gedcom-go session (2025-12-21)
**Status**: Idea captured

## The Idea

Inject a "build vs reuse" consideration into Claude Code planning workflows to prevent over-engineering. Before implementation planning begins, force the question:

> "What existing stdlib or library functionality can be leveraged? What is truly domain-specific?"

This should be added to:
- `/cacack:play` - during issue review and planning phase
- Plan subagent - during any implementation planning

## Context

This came up while reviewing issue #32 (date validation). The original plan proposed implementing custom `IsLeapYear()`, `daysInMonth()`, and other calendar math—when Go's `time.Date()` already handles all of this. The detailed research doc (`GEDCOM_DATE_FORMATS_RESEARCH.md`) described *what* was needed but led to over-engineering by not asking "does this already exist?"

Pattern identified: detailed requirements → detailed implementation, skipping the "do we need to build this?" question.

Already added guidance to this project's CLAUDE.md under "Implementation Philosophy", but the plugin work would make this automatic across all projects.

## Next Steps

1. Review `/cacack:play` skill - add build-vs-reuse question to the planning output
2. Review Plan subagent prompt - inject the consideration before implementation steps
3. Consider adding to other planning-related skills (speckit.plan, etc.)

## Related

- CLAUDE.md now has "Implementation Philosophy" section with this guidance
- Issue #32 needs to be updated to reflect simplified approach (use stdlib for validation)
