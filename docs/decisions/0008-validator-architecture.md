# ADR-008: Validator Architecture

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Validation system design in gedcom-go library

## Decision

Use a pluggable, composition-based validator architecture with specialized validators for different concern areas. Support configurable strictness levels and lazy initialization of validator components.

## Context

GEDCOM validation has multiple dimensions:
- **Structural**: Valid line format, proper hierarchy, required tags
- **Referential**: XRefs point to existing records
- **Semantic**: Death after birth, reasonable ages, date logic
- **Quality**: Source coverage, duplicate detection

Different users have different needs:
- Data entry: lenient, show warnings
- Publication: strict, require sources
- Migration: permissive, preserve everything

The question: how do we support these varied needs without a monolithic validator?

## Decision Drivers

1. **Separation of concerns** - Each validation type is independent
2. **Configurability** - Users enable/disable specific checks
3. **Performance** - Only run requested validations
4. **Extensibility** - Easy to add new validators

## Considered Options

### Option A: Monolithic Validator

```go
func Validate(doc *Document) []Issue {
    // All validation logic in one function
}
```

- **Pros**: Simple to call
- **Cons**: Can't customize, all-or-nothing, hard to maintain
- **Verdict**: Rejected - not flexible enough

### Option B: Separate Functions

```go
func ValidateStructure(doc *Document) []Issue
func ValidateReferences(doc *Document) []Issue
func ValidateDates(doc *Document) []Issue
// Caller combines as needed
```

- **Pros**: Flexible composition
- **Cons**: No shared state, repeated work, no unified config
- **Verdict**: Rejected - lacks cohesion

### Option C: Pluggable Composition (Selected)

```go
type Validator struct {
    config       *ValidatorConfig
    dateLogic    *DateLogicValidator    // Lazy initialized
    references   *ReferenceValidator
    duplicates   *DuplicateDetector
    tagValidator *TagValidator
}
```

- **Pros**: Unified interface, lazy init, shared config, extensible
- **Verdict**: Accepted

## Consequences

### Positive

- Single entry point with configurable behavior
- Validators only instantiated when used
- Shared configuration (strictness, thresholds)
- Easy to add new validator types
- Issues aggregated with consistent structure

### Negative

- More complex than simple function
- Configuration requires understanding options

## Implementation

### Configuration

```go
type ValidatorConfig struct {
    // Strictness level
    Strictness Strictness  // Relaxed, Normal, Strict

    // Date validation settings
    DateLogic *DateLogicConfig

    // Duplicate detection settings
    Duplicates *DuplicateConfig

    // Custom tag validation
    TagRegistry        *TagRegistry
    ValidateCustomTags bool
}

type Strictness int
const (
    StrictnessRelaxed Strictness = iota  // Errors only
    StrictnessNormal                      // Errors + Warnings
    StrictnessStrict                      // All including Info
)
```

### Validator Structure

```go
type Validator struct {
    config *ValidatorConfig

    // Lazy-initialized specialized validators
    dateLogic    *DateLogicValidator
    references   *ReferenceValidator
    duplicates   *DuplicateDetector
    quality      *QualityAnalyzer
    tagValidator *TagValidator
}

func (v *Validator) ValidateAll(doc *Document) []Issue {
    var issues []Issue
    issues = append(issues, v.ValidateStructure(doc)...)
    issues = append(issues, v.ValidateReferences(doc)...)
    issues = append(issues, v.ValidateDateLogic(doc)...)
    // ... etc
    return v.filterBySeverity(issues)
}
```

### Specialized Validators

| Validator | Responsibility |
|-----------|----------------|
| `DateLogicValidator` | Death before birth, impossible ages |
| `ReferenceValidator` | Orphaned XRefs, missing targets |
| `DuplicateDetector` | Potential duplicate individuals |
| `QualityAnalyzer` | Source coverage, data completeness |
| `TagValidator` | Custom tag validity, placement |

### Streaming Validator

For large files, a streaming variant validates records incrementally:

```go
type StreamingValidator struct {
    seenXRefs map[string]bool  // Declarations
    usedXRefs map[string]bool  // References
}

func (v *StreamingValidator) ValidateRecord(record *Record) []Issue {
    // Immediate: date logic, structure
    return issues
}

func (v *StreamingValidator) Finalize() []Issue {
    // Deferred: cross-reference validation
    return v.checkOrphanedReferences()
}
```

### Issue Structure

```go
type Issue struct {
    Severity    Severity  // Error, Warning, Info
    Code        string    // Machine-readable code
    Message     string    // Human-readable description
    RecordXRef  string    // Affected record
    RelatedXRef string    // Related record
    LineNumber  int       // Source location
    Details     map[string]string  // Additional context
}
```

### Usage Examples

```go
// Default validation
v := validator.New()
issues := v.ValidateAll(doc)

// Strict validation with custom config
v := validator.NewWithConfig(&validator.ValidatorConfig{
    Strictness: validator.StrictnessStrict,
    DateLogic: &validator.DateLogicConfig{
        MaxAge: 120,
    },
})
issues := v.ValidateAll(doc)

// Specific validation only
issues := v.ValidateDateLogic(doc)
issues := v.FindPotentialDuplicates(doc)
```

## References

- `validator/validator.go` - Main validator implementation
- `validator/date_logic.go` - Date validation
- `validator/references.go` - Reference validation
- `validator/duplicates.go` - Duplicate detection
- `validator/streaming.go` - Streaming validator
