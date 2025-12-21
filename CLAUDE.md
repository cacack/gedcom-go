# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`go-gedcom` is a pure Go library for processing GEDCOM files. GEDCOM (GEnealogical Data COMmunication) is a standard file format for exchanging genealogical data between different genealogy software.

## Quick Start

**Required**: Go 1.21+

```bash
make setup-dev-env    # Downloads deps, installs tools, sets up hooks
```

This installs pre-commit hooks that enforce:
- gofmt formatting
- go vet checks
- golangci-lint (v2)
- Per-package test coverage ≥85%

## Development Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-coverage` | Run tests with per-package coverage report |
| `make fmt` | Format code |
| `make vet` | Run go vet |
| `make lint` | Run staticcheck |
| `make bench` | Run benchmarks |

## Test Data

```
testdata/
├── gedcom-5.5/      # GEDCOM 5.5 samples
├── gedcom-5.5.1/    # GEDCOM 5.5.1 samples
├── gedcom-7.0/      # GEDCOM 7.0 samples
└── malformed/       # Invalid files for error testing
```

## Coverage Requirements

| Level | Threshold | Enforcement |
|-------|-----------|-------------|
| Per-package | ≥85% | Pre-commit hook, CI |
| Critical paths | 100% | Code review |

See `docs/TESTING.md` for critical paths that require 100% coverage.

## Architecture Guidelines

### GEDCOM Format Basics
- GEDCOM files use a line-based format with hierarchical levels (0, 1, 2, etc.)
- Each line format: `LEVEL TAG [VALUE] [XREF]`
- Main record types: Individual (INDI), Family (FAM), Source (SOUR), Repository (REPO), etc.
- Records are linked via cross-references (e.g., `@I1@`)

### Expected Code Structure
The library should be organized around:

1. **Parser/Lexer**: Low-level tokenization and parsing of GEDCOM line format
2. **Data Model**: Go structs representing GEDCOM records (Individual, Family, Event, etc.)
3. **Decoder**: High-level API to decode GEDCOM files into Go data structures
4. **Encoder**: Convert Go data structures back to GEDCOM format
5. **Validator**: Validate GEDCOM data against specification versions (5.5, 5.5.1, 7.0)

### Key Design Considerations
- Handle large GEDCOM files efficiently (streaming where possible)
- Support multiple GEDCOM versions (5.5, 5.5.1, 7.0)
- Character encoding handling (GEDCOM supports ANSEL, ASCII, UNICODE, etc.)
- Preserve original line numbers for error reporting
- Handle malformed GEDCOM files gracefully with clear error messages

### Implementation Philosophy

Before implementing, ask:
1. Does Go stdlib solve this? (`time`, `strings`, `encoding/json`, etc.)
2. Is there a well-maintained library? (Check pkg.go.dev)
3. What is truly domain-specific requiring custom code?

Only build custom implementations for domain-specific logic (GEDCOM parsing, genealogy concepts). Detailed specs describe *what* is needed, not *how* to build it—leverage existing solutions for common problems.

### Standard Go Library Structure
Follow standard Go project layout:
- Root: library code (parser, types, decoder, encoder)
- `internal/`: private implementation details
- `examples/`: example usage code
- `testdata/`: sample GEDCOM files for testing

## Project Constitution

@.specify/memory/constitution.md

The constitution defines core principles (the WHAT). This CLAUDE.md defines operational guidance (the HOW). When in doubt, constitution principles take precedence.

Key principles: Library-First Design, API Clarity, Test Coverage (≥85%), Version Support, Error Transparency, **Lossless Representation**.

## Documentation Structure

- **README.md**: Project overview, quick start, installation
- **FEATURES.md**: Exhaustive list of implemented features
- **IDEAS.md**: Unvetted ideas and rough concepts (create when needed)
- **GitHub Issues**: Single source of truth for planned work
- **docs/**: Implementation reference material
  - `adr/` - Architecture Decision Records
  - `ENCODING_IMPLEMENTATION_PLAN.md` - UTF-16/ANSEL implementation guide
  - `GEDCOM_DATE_FORMATS_RESEARCH.md` - Date format specification research
  - `PERFORMANCE.md` - Benchmarks and optimization notes

### Workflow for New Ideas

1. **Quick idea**: Add to `IDEAS.md` for later consideration
2. **Vetted feature**: Create GitHub issue with appropriate labels
3. **Implementation reference**: Add detailed docs in `docs/` folder

### Issue Labels

Priority: `priority:high`, `priority:medium`, `priority:low`, `priority:future`
Area: `area:encoding`, `area:parsing`, `area:validation`, `area:api`, `area:tooling`, `area:dx`

## Git Conventions

### Commit Messages
Use [conventional commits](https://www.conventionalcommits.org/): `type(scope): description`
- `feat(parser): add GEDCOM 7.0 header parsing`
- `fix(decoder): handle empty CONC values`
- `docs: update API examples`

### PR Titles (IMPORTANT)
PR titles must **NOT** use conventional commit format. Use plain descriptive titles:
- **Good**: `Add GEDCOM 7.0 header parsing`
- **Bad**: `feat(parser): add GEDCOM 7.0 header parsing`

**Why?** We use merge commits (not squash) for semi-linear history. Release-please picks up both PR titles and commit messages. If both use conventional format, changelog entries are duplicated.

### Branch Strategy
- Rebase feature branches on `main` before merging
- Use merge commits (not squash) to preserve commit history
- CI enforces PR title format via `.github/workflows/pr-title.yml`

## Downstream Consumer

This library is used by `github.com/cacack/my-family` (at `/Users/chris/devel/home/my-family`) via a `replace` directive during development. When adding features:

1. **Driven by real usage**: Features should be added when my-family needs them, not speculatively
2. **Self-contained commits**: Each enhancement should be a complete, testable unit with its own tests
3. **Run consumer tests**: After changes, verify my-family still works: `cd /Users/chris/devel/home/my-family && go test ./...`
4. **API stability**: Consider how changes affect the public API; prefer additive changes over breaking ones
5. **Document in commit**: Note which my-family feature drove the change (e.g., "feat(decoder): add entity parsing for GEDCOM import")
