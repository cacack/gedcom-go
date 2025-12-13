# go-gedcom TODO List

This document tracks planned improvements and next steps for the go-gedcom project.

## High Priority

### 1. ✅ Improve Test Coverage - COMPLETED
**Goal**: Achieve 90%+ coverage across all packages

Final status:
- ✅ `encoder`: 58.7% → 95.7% ⭐
- ✅ `gedcom`: 64.1% → 100.0% ⭐⭐
- ✅ `charset`: 82.2% → 100.0% ⭐⭐
- ✅ `decoder`: 92.1%
- ✅ `parser`: 94.3%
- ✅ `validator`: 94.4%
- ✅ `version`: 87.5%

**Completed Tasks**:
- [x] Add comprehensive encoder tests (edge cases, error conditions, all GEDCOM versions)
- [x] Add gedcom package tests (all record types, tag handling, data structures)
- [x] Add charset tests (all encoding types, conversion edge cases)

### 2. ✅ Complete Missing Documentation - COMPLETED
**Goal**: Professional, complete documentation for open-source release

**Completed Tasks**:
- [x] Create CONTRIBUTING.md with:
  - Code of conduct
  - How to submit issues/PRs
  - Development setup
  - Testing requirements
  - Code style guidelines
- [x] Add godoc comments to all public APIs:
  - Package-level documentation
  - All exported types, functions, methods
  - Examples in godoc format
- [x] Create more examples in `examples/`:
  - Basic parsing example (already exists)
  - Validation example ✨
  - Encoding/writing GEDCOM ✨
  - Querying individuals/families ✨
  - Error handling patterns
  - Working with different GEDCOM versions

### 3. ✅ Publish the Package - COMPLETED
**Goal**: Make package publicly available and installable

**Completed Tasks**:
- [x] Verify module path matches intended repository
  - Current go.mod: `github.com/cacack/gedcom-go`
  - ✅ Updated to match actual GitHub repository
- [x] Create initial release (v0.1.0)
- [x] Tag release in git
- [x] Verify `go get` works
- [x] Submit to pkg.go.dev for documentation indexing
- [x] Update README with correct installation instructions
- [x] Add release badges to README

### 4. PEDI (Pedigree Linkage) Support
**Goal**: Capture child relationship types (biological, adopted, foster, etc.)

**Problem**: Currently `Individual.ChildInFamilies` is `[]string` (XRefs only), losing the PEDI subordinate tag that indicates relationship type:

```gedcom
0 @I13@ INDI
1 FAMC @F2@
2 PEDI adopted   <-- Currently not captured
```

**Impact**: Applications using gedcom-go (e.g., my-family) cannot distinguish adopted from biological children.

**Tasks**:
- [ ] Create `FamilyLink` struct to replace `[]string`:
  ```go
  type FamilyLink struct {
      FamilyXRef string
      Pedigree   string // "birth", "adopted", "foster", "sealing"
  }
  ```
- [ ] Update `Individual.ChildInFamilies` to `[]FamilyLink`
- [ ] Update decoder to parse PEDI tags under FAMC
- [ ] Add `EventAdoption` constant (`ADOP`) to EventType
- [ ] Add tests with comprehensive GEDCOM containing PEDI tags
- [ ] Update documentation

**Reference**: GEDCOM 5.5.1 spec section on INDIVIDUAL_EVENT_STRUCTURE

---

## Medium Priority

### 5. Performance Benchmarking & Optimization
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

### 6. Enhanced Features
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

### 7. Developer Experience
**Goal**: Make contributing and maintenance easier

**Tasks**:
- [ ] Add pre-commit hooks (as outlined in CLAUDE.md)
  - Format checking
  - Vet
  - Test execution
  - Coverage validation
- [x] ✅ Set up CI/CD (GitHub Actions) - COMPLETED
  - ✅ Run tests on all PRs
  - ✅ Check code coverage (≥85% enforced, achieving 96.5%)
  - ✅ Run linters (golangci-lint with 15 linters)
  - ✅ Test on multiple Go versions (1.21, 1.22, 1.23)
  - ✅ Test on multiple platforms (Ubuntu, macOS, Windows)
  - ✅ Automated release workflow
- [ ] Add more real-world test data
  - Famous genealogy files (with proper licensing)
  - Edge cases from different software
  - Multi-language examples
- [ ] Create issue templates
  - Bug report template
  - Feature request template
  - Question template

### 8. Additional Tooling
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
