<!--
Sync Impact Report
==================
Version Change: 1.0.0 → 1.1.0
Modified Principles: None
Added Sections:
  - Core Principles VI: Lossless Representation

Previous versions:
- 1.0.0 (2025-10-16): Initial constitution with 5 principles

Templates Status:
  ✅ plan-template.md - reviewed, Constitution Check section compatible
  ✅ spec-template.md - reviewed, requirement structure compatible
  ✅ tasks-template.md - reviewed, testing guidance compatible
  ✅ agent-file-template.md - reviewed, generic structure compatible

Follow-up TODOs: None
-->

# go-gedcom Constitution

## Core Principles

### I. Library-First Design

Every feature MUST be implemented as a well-defined library component with clear boundaries.
Library components MUST be:
- Self-contained with explicit dependencies
- Independently testable without external systems
- Documented with godoc comments following Go conventions
- Usable as importable packages (not just through CLI)

**Rationale**: GEDCOM processing is a reusable capability. Library-first design enables
integration into diverse applications (web servers, desktop apps, batch processors) while
enforcing modularity and testability.

### II. API Clarity

Public APIs MUST prioritize simplicity and discoverability over implementation convenience.

API design requirements:
- Exported types, functions, and methods MUST have comprehensive godoc comments
- Function signatures MUST use standard Go idioms (e.g., `(result, error)` returns)
- APIs MUST accept `io.Reader`/`io.Writer` for file operations (not just filenames)
- Breaking changes MUST follow semantic versioning (documented in CHANGELOG)
- Examples MUST be provided in `examples/` directory and as godoc examples

**Rationale**: Library users discover functionality through godoc and code exploration.
Clear APIs reduce support burden and increase adoption.

### III. Test Coverage (NON-NEGOTIABLE)

Comprehensive testing is MANDATORY for all code paths.

Testing requirements:
- Unit tests MUST cover all public APIs
- Table-driven tests MUST be used for parsing/validation logic
- Integration tests MUST use real GEDCOM files from `testdata/`
- Test coverage MUST be ≥85% (measured via `go test -cover`)
- Tests MUST be written BEFORE implementation (TDD approach)
- All tests MUST pass before any commit

**Rationale**: GEDCOM parsing involves complex edge cases, encoding variations, and
spec ambiguities. Without rigorous testing, silent data corruption is likely.
Test-first development catches design flaws early.

### IV. Version Support

The library MUST support multiple GEDCOM specification versions (5.5, 5.5.1, 7.0).

Version handling requirements:
- Parser MUST auto-detect version from GEDCOM header
- Version-specific validation rules MUST be encapsulated
- Decoder/Encoder MUST support roundtrip fidelity per version
- Breaking changes between GEDCOM versions MUST be documented
- Users MUST be able to specify target version explicitly

**Rationale**: Real-world genealogy data spans decades and multiple GEDCOM versions.
Supporting only one version makes the library impractical for most use cases.

### V. Error Transparency

Errors MUST be informative, actionable, and preserve context.

Error handling requirements:
- Parsing errors MUST include line number and content snippet
- Validation errors MUST cite specific GEDCOM spec violations
- Encoding errors MUST report invalid data with context
- Errors MUST use Go 1.13+ error wrapping for context chains
- Malformed input MUST fail gracefully (never panic in parser)

**Rationale**: GEDCOM files often contain errors from legacy software. Users need
precise diagnostics to fix data issues. Panics are unacceptable in a library.

### VI. Lossless Representation (NON-NEGOTIABLE)

Source GEDCOM data MUST be preserved without information loss.

Lossless requirements:
- Original values MUST be retained and recoverable
- Partial/incomplete data MUST be representable (year-only dates, missing fields)
- GEDCOM-specific semantics MUST NOT be flattened into lossy formats
- Calendar-specific dates MUST preserve their calendar system
- Modifiers, ranges, and uncertainty MUST be captured
- Conversion to external formats MUST be opt-in, not automatic

**Rationale**: GEDCOM represents genealogical data spanning centuries with varying
precision and calendar systems. Normalizing to modern precise formats destroys
historical nuance. A date recorded as "about 1850" is fundamentally different
from "January 1, 1850" - both must be representable.

See project ADRs for implementation-specific decisions that fulfill this principle.

## Quality Standards

### Code Standards
- All code MUST pass `go fmt`, `go vet`, and `golint` (if available)
- Exported APIs MUST have godoc comments (enforced by linters)
- Cyclomatic complexity MUST be kept low (prefer small functions)
- No magic numbers or unexplained constants

### Performance Standards
- Large file handling MUST use streaming where possible (avoid full in-memory parse)
- Memory allocations MUST be minimized in hot paths
- Benchmarks MUST be provided for parsing/encoding operations (`go test -bench`)
- Performance regressions MUST be investigated before merge

### Documentation Standards
- README.md MUST include quick start guide and examples
- Each package MUST have package-level godoc
- Complex algorithms (e.g., cross-reference resolution) MUST have inline comments
- CHANGELOG.md MUST document all public API changes

## Development Workflow

### Feature Development
1. Create feature specification in `/specs/[###-feature-name]/spec.md`
2. Define acceptance criteria as Given/When/Then scenarios
3. Write failing tests that exercise acceptance criteria
4. Implement minimum code to pass tests
5. Refactor while keeping tests green
6. Update documentation (godoc, examples, README if needed)
7. Run full test suite: `go test ./...`
8. Verify code quality: `go vet ./...`

### Testing Workflow
- Use `testdata/` for sample GEDCOM files (various versions, edge cases)
- Name test files `*_test.go` following Go conventions
- Use `t.Helper()` in test utility functions
- Use `testing.Short()` to skip slow integration tests in CI
- Run benchmarks periodically: `go test -bench=. ./...`

### Review Checklist
Before any merge:
- [ ] All tests pass: `go test ./...`
- [ ] Test coverage ≥85%: `go test -cover ./...`
- [ ] Code formatted: `go fmt ./...`
- [ ] No vet issues: `go vet ./...`
- [ ] Godoc comments on all exports
- [ ] Examples updated if API changed
- [ ] CHANGELOG.md updated for user-facing changes

## Governance

### Amendment Process
1. Propose amendment in GitHub issue with rationale
2. Discuss impact on existing code/features
3. Update constitution.md with version bump
4. Update dependent templates (plan, spec, tasks) as needed
5. Commit with message: `docs: amend constitution to vX.Y.Z (description)`

### Versioning Policy
- **MAJOR**: Removed/redefined principles that invalidate existing code
- **MINOR**: New principles or expanded requirements
- **PATCH**: Clarifications, typo fixes, non-semantic changes

### Compliance Review
- All feature plans MUST include Constitution Check section
- PRs violating principles MUST justify exception or be rejected
- Technical debt that violates principles MUST be tracked and remediated

### Runtime Guidance
Development guidance for AI assistants and developers is in:
- Global: `~/.claude/CLAUDE.md` (if exists)
- Project: `/CLAUDE.md` (operational commands, architecture details)

The constitution defines WHAT; CLAUDE.md defines HOW.

**Version**: 1.1.0 | **Ratified**: 2025-10-16 | **Last Amended**: 2025-12-21
