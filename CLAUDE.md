# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gedcom-go` is a pure Go library for processing GEDCOM files. GEDCOM (GEnealogical Data COMmunication) is a standard file format for exchanging genealogical data between different genealogy software.

## Quick Start

**Required**: Go 1.24+

```bash
make setup            # Downloads deps, installs tools, sets up hooks
```

This installs pre-commit hooks that enforce:
- gofmt formatting
- go vet checks
- golangci-lint (v2)
- Per-package test coverage ≥85%

## Development Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (with race detector) |
| `make preflight` | Run all CI checks locally before pushing |
| `make test-coverage` | Run tests with per-package coverage report |
| `make fmt` | Format code |
| `make vet` | Run go vet |
| `make lint` | Run staticcheck |
| `make security` | Run gosec + govulncheck |
| `make api-check` | Check for breaking API changes vs latest release |
| `make bench` | Run benchmarks |
| `make watch-test` | Watch for changes and re-run tests (requires `entr`) |

**Running a single test:**
```bash
go test ./parser -run TestParseLine -v       # Run specific test
go test ./decoder -run TestDecode -v         # Another example
go test ./gedcom -run TestDate -v -count=1   # Skip cache
```

## Architecture

### Package Structure

```
gedcom/     # Core data types (Document, Individual, Family, Source, etc.)
decoder/    # High-level GEDCOM decoding with automatic version detection
encoder/    # GEDCOM document writing with configurable line endings
parser/     # Low-level line parsing with detailed error reporting
validator/  # Document validation with error categorization
charset/    # Character encoding (UTF-8, ANSEL) with BOM detection
version/    # GEDCOM version detection (5.5, 5.5.1, 7.0)
```

### Data Flow

```
GEDCOM file → charset.NewReader() → parser.Parse() → decoder.buildDocument() → gedcom.Document
                  ↓                      ↓                    ↓
            UTF-8 validation      []*parser.Line      Typed entities (Individual, Family, etc.)
```

### Key Types

- **`gedcom.Document`**: Root container with `Header`, `Records`, `XRefMap` for O(1) lookup
- **`gedcom.Record`**: Wrapper holding typed data (`Individual`, `Family`, `Source`, etc.)
- **`gedcom.Individual`**: Person with `Names`, `Events`, `Attributes`, `FamilyChild`, `FamilySpouse`
- **`gedcom.Family`**: Family unit with `Husband`, `Wife`, `Children` (as XRef strings)
- **`parser.Line`**: Raw parsed line with `Level`, `XRef`, `Tag`, `Value`, `LineNumber`

### Cross-Reference Resolution

Records link via XRef strings (e.g., `"@I1@"`). Use `Document.GetIndividual(xref)` for typed lookup:
```go
family := doc.GetFamily("@F1@")
husband := doc.GetIndividual(family.Husband)  // family.Husband is "@I1@"
```

## Test Data

```
testdata/
├── edge-cases/      # Boundary condition and unusual structure samples
├── encoding/        # Character encoding test files (ANSEL, UTF-16, BOM)
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

## Strategic Context

- **[docs/ETHOS.md](docs/ETHOS.md)** — Vision, core principles, differentiators, anti-patterns
- **[docs/ROADMAP.md](docs/ROADMAP.md)** — Phased feature plan (Phase 1 is current priority)

Two principles are **NON-NEGOTIABLE**: Test Coverage (≥85%) and Lossless Representation. See ETHOS.md for all six core principles.

## Documentation Structure

- **README.md**: Project overview, quick start, installation
- **FEATURES.md**: Exhaustive list of implemented features
- **IDEAS.md**: Unvetted ideas and rough concepts (create when needed)
- **GitHub Issues**: Single source of truth for planned work
- **docs/**: Implementation reference material
  - `ETHOS.md` - Vision, core principles, differentiators, anti-patterns
  - `ROADMAP.md` - Phased feature plan (Phase 1 is current priority)
  - `API_STABILITY.md` - API compatibility guarantees and versioning policy
  - `TESTING.md` - Test coverage requirements and critical paths
  - `GEDCOM_VERSIONS.md` - GEDCOM version differences (5.5, 5.5.1, 7.0)
  - `ENCODING_IMPLEMENTATION_PLAN.md` - UTF-16/ANSEL implementation guide
  - `GEDCOM_DATE_FORMATS_RESEARCH.md` - Date format specification research
  - `PERFORMANCE.md` - Benchmarks and optimization notes
  - `adr/` - Architecture Decision Records (see below)

### Architecture Decision Records

Key design decisions are documented in `docs/adr/`:

| ADR | Decision |
|-----|----------|
| 001 | Custom Date struct for lossless GEDCOM dates |
| 002 | XRef resolution via strings + O(1) map lookup |
| 003 | Lossless dual storage (raw tags + typed entity) |
| 004 | Encoding detection cascade (BOM → Header → UTF-8) |
| 005 | Version detection (header-first with tag fallback) |
| 006 | Line continuation handling (CONT/CONC) |
| 007 | Error transparency (line numbers, context, never panic) |
| 008 | Validator architecture (pluggable, configurable) |

### Workflow for New Ideas

1. **Quick idea**: Add to `IDEAS.md` for later consideration
2. **Vetted feature**: Create GitHub issue with appropriate labels
3. **Implementation reference**: Add detailed docs in `docs/` folder

### Issue Labels

Effort: `effort:low`, `effort:medium`, `effort:high`
Value: `value:low`, `value:medium`, `value:high`
Area: `area:encoding`, `area:parsing`, `area:validation`, `area:api`, `area:tooling`, `area:dx`

## Git Conventions

### Commit Messages
Use conventional commits with defined types. See [CONTRIBUTING.md](CONTRIBUTING.md#5-commit-your-changes) for the full list.

Key distinction: `feat`/`fix` are for **library changes** (what users consume), not development tooling.
- `feat(parser): add GEDCOM 7.0 date support` — library feature
- `ci: add PR title validation` — development tooling (not `feat`)

### PR Titles
PR titles must **NOT** use conventional commit format:
- **Good**: `Add GEDCOM 7.0 header parsing`
- **Bad**: `feat(parser): add GEDCOM 7.0 header parsing`

This prevents duplicate changelog entries (release-please picks up both PR titles and commits).

### Branch Strategy
- Rebase feature branches on `main` before merging
- Use merge commits (not squash) to preserve commit history
- CI enforces PR title format (release-please PRs are exempt)

## Downstream Consumer

This library is used by `github.com/cacack/my-family` (at `/Users/chris/devel/home/my-family`) via a `replace` directive during development. When adding features:

1. **Driven by real usage**: Features should be added when my-family needs them, not speculatively
2. **Self-contained commits**: Each enhancement should be a complete, testable unit with its own tests
3. **Run consumer tests**: After changes, verify my-family still works: `cd /Users/chris/devel/home/my-family && go test ./...`
4. **API stability**: Follow `docs/API_STABILITY.md`; prefer additive changes over breaking ones
5. **Document in commit**: Note which my-family feature drove the change (e.g., "feat(decoder): add entity parsing for GEDCOM import")
