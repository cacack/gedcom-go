# go-gedcom TODO List

This document tracks planned improvements and next steps for the go-gedcom project.

## High Priority

### 1. Improve Test Coverage
**Goal**: Achieve 90%+ coverage across all packages

Current status:
- ✗ `encoder`: 58.7% → Target: 90%+
- ✗ `gedcom`: 64.1% → Target: 90%+
- ✗ `charset`: 82.2% → Target: 90%+
- ✓ `decoder`: 92.1%
- ✓ `parser`: 94.3%
- ✓ `validator`: 94.4%
- ✓ `version`: 87.5%

**Tasks**:
- [ ] Add comprehensive encoder tests (edge cases, error conditions, all GEDCOM versions)
- [ ] Add gedcom package tests (all record types, tag handling, data structures)
- [ ] Add charset tests (all encoding types, conversion edge cases)

### 2. Complete Missing Documentation
**Goal**: Professional, complete documentation for open-source release

**Tasks**:
- [ ] Create CONTRIBUTING.md with:
  - Code of conduct
  - How to submit issues/PRs
  - Development setup
  - Testing requirements
  - Code style guidelines
- [ ] Add godoc comments to all public APIs:
  - Package-level documentation
  - All exported types, functions, methods
  - Examples in godoc format
- [ ] Create more examples in `examples/`:
  - Basic parsing example (already exists)
  - Validation example
  - Encoding/writing GEDCOM
  - Querying individuals/families
  - Error handling patterns
  - Working with different GEDCOM versions

### 3. Publish the Package
**Goal**: Make package publicly available and installable

**Tasks**:
- [x] Verify module path matches intended repository
  - Current go.mod: `github.com/cacack/gedcom-go`
  - ✅ Updated to match actual GitHub repository
- [ ] Create initial release (v0.1.0 or v1.0.0)
- [ ] Tag release in git
- [ ] Verify `go get` works
- [ ] Submit to pkg.go.dev for documentation indexing
- [ ] Update README with correct installation instructions

## Medium Priority

### 4. Performance Benchmarking & Optimization
**Goal**: Ensure library performs well with large GEDCOM files

**Tasks**:
- [ ] Add comprehensive benchmarks:
  - Small files (< 1KB)
  - Medium files (100KB - 1MB)
  - Large files (10MB+)
  - Memory allocation benchmarks
- [ ] Profile hot paths using pprof
- [ ] Optimize identified bottlenecks
- [ ] Consider Profile-Guided Optimization (PGO) for Go 1.21+
- [ ] Document performance characteristics in README

### 5. Enhanced Features
**Goal**: Add useful functionality beyond basic parsing

**Possible features**:
- [ ] Query/Search API
  - Find individuals by name
  - Find families by ID
  - Filter records by criteria
  - Navigate relationships (parents, children, spouses)
- [ ] Merge/Diff capabilities
  - Compare two GEDCOM files
  - Merge genealogy data
  - Detect conflicts
- [ ] Export to other formats
  - JSON export
  - XML export
  - Custom format support
- [ ] Data validation helpers
  - Detect orphaned references
  - Find duplicate individuals
  - Validate date ranges

## Low Priority

### 6. Developer Experience
**Goal**: Make contributing and maintenance easier

**Tasks**:
- [ ] Add pre-commit hooks (as outlined in CLAUDE.md)
  - Format checking
  - Vet
  - Test execution
  - Coverage validation
- [ ] Set up CI/CD (GitHub Actions)
  - Run tests on all PRs
  - Check code coverage
  - Run linters (go vet, staticcheck)
  - Test on multiple Go versions
  - Auto-deploy documentation
- [ ] Add more real-world test data
  - Famous genealogy files (with proper licensing)
  - Edge cases from different software
  - Multi-language examples
- [ ] Create issue templates
  - Bug report template
  - Feature request template
  - Question template

### 7. Additional Tooling
**Goal**: Provide useful command-line tools

**Possible tools**:
- [ ] CLI tool for validating GEDCOM files
- [ ] CLI tool for converting between GEDCOM versions
- [ ] CLI tool for querying GEDCOM data
- [ ] Web-based GEDCOM viewer/validator

## Future Considerations

### Standards Compliance
- [ ] Track GEDCOM 7.0 specification updates
- [ ] Add FamilySearch GEDCOM extensions support
- [ ] Support for vendor-specific extensions (Ancestry, MyHeritage, etc.)

### Advanced Features
- [ ] GEDCOM-X support (next-gen standard)
- [ ] Streaming encoder for large files
- [ ] Incremental parsing for partial file access
- [ ] Schema validation with custom rules

## Completed
- ✓ Phase 1-8: Complete GEDCOM parser implementation
- ✓ Parser with multi-version support (5.5, 5.5.1, 7.0)
- ✓ Validator implementation
- ✓ Encoder implementation
- ✓ Basic examples
- ✓ Initial documentation (README, CLAUDE.md)
