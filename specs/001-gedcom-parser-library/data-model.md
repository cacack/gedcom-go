# Data Model: GEDCOM Parser Library

**Feature**: 001-gedcom-parser-library
**Date**: 2025-10-16
**Status**: Complete

## Overview

This document defines the core data structures for representing GEDCOM files in Go. All entities are extracted from the feature specification's "Key Entities" section and functional requirements.

## Core Principles

- **Immutability**: Structs are designed to be created once and not modified (supports concurrent access)
- **Zero-value Safety**: All fields have sensible zero values
- **Version-aware**: Structs accommodate all GEDCOM versions (5.5, 5.5.1, 7.0)
- **Stream-friendly**: Can be processed individually without full document in memory

---

## Entity Definitions

### 1. Document

**Purpose**: Represents a complete GEDCOM file

**Package**: `gedcom`

**Go Definition**:
```go
type Document struct {
    Header  Header              // File header with metadata
    Records []Record            // All records in order
    Trailer Trailer             // File trailer
    Version Version             // Detected/explicit GEDCOM version
    XRefMap map[string]*Record  // Cross-reference lookup index
}
```

**Fields**:
- `Header`: Contains version, encoding, source system, date
- `Records`: Ordered slice of all records (INDI, FAM, SOUR, etc.)
- `Trailer`: End-of-file marker (mostly empty in GEDCOM)
- `Version`: Enum indicating 5.5, 5.5.1, or 7.0
- `XRefMap`: Index for fast cross-reference resolution (built during decoding)

**Relationships**:
- Contains many `Record`
- Owns the `XRefMap` for cross-reference resolution

**Validation Rules**:
- Must have exactly one Header
- Must have exactly one Trailer
- Version must be valid (5.5, 5.5.1, 7.0)
- All XRefs in Records must be unique within document
- XRefMap must contain every record with XRef

**State Transitions**: None (immutable after decoding)

---

### 2. Record

**Purpose**: Base entity for any GEDCOM record type

**Package**: `gedcom`

**Go Definition**:
```go
type Record struct {
    XRef       string          // Cross-reference ID (e.g., "@I1@")
    Type       RecordType      // INDI, FAM, SOUR, REPO, NOTE, OBJE, SUBM, SUBN
    Tags       []Tag           // Hierarchical tag-value pairs
    LineNumber int             // Original line number in file
}

type RecordType string

const (
    TypeIndividual  RecordType = "INDI"
    TypeFamily      RecordType = "FAM"
    TypeSource      RecordType = "SOUR"
    TypeRepository  RecordType = "REPO"
    TypeNote        RecordType = "NOTE"
    TypeMediaObject RecordType = "OBJE"
    TypeSubmitter   RecordType = "SUBM"
    TypeSubmission  RecordType = "SUBN"
)
```

**Fields**:
- `XRef`: Unique identifier (may be empty for some record types)
- `Type`: Record type enum for type-safe handling
- `Tags`: All tag-value pairs in hierarchical order
- `LineNumber`: For error reporting (preserved from parser)

**Relationships**:
- Contained by `Document`
- References other `Record` via XRef strings in tag values
- Contains many `Tag`

**Validation Rules**:
- XRef must be unique if present
- XRef format: `@[A-Za-z0-9_-]+@` (from clarifications)
- Type must be one of the defined RecordType constants
- Tags must form valid hierarchy (levels consistent)

**State Transitions**: None (immutable)

---

### 3. Individual (INDI)

**Purpose**: Represents a person with biographical information

**Package**: `gedcom`

**Go Definition**:
```go
type Individual struct {
    Record                      // Embedded base Record
    Names        []NameParts   // Personal names (can have multiple)
    Gender       string        // M, F, U, or custom
    Events       []Event       // Birth, death, marriage, etc.
    Attributes   []Attribute   // Occupation, education, etc.
    ChildInFamily   []string   // XRefs to Family records (as child)
    SpouseInFamily []string    // XRefs to Family records (as spouse)
}

type NameParts struct {
    Full       string  // Full name as written
    Given      string  // Given names
    Surname    string  // Surname/family name
    Prefix     string  // Name prefix (Dr., Sir, etc.)
    Suffix     string  // Name suffix (Jr., III, etc.)
}
```

**Fields**:
- `Names`: Multiple names (maiden name, married name, etc.)
- `Gender`: From GEDCOM SEX tag
- `Events`: Life events (birth, death, marriage, immigration, etc.)
- `Attributes`: Non-event facts (occupation, education, religion, etc.)
- `ChildInFamily`: References to FAM records where this person is a child
- `SpouseInFamily`: References to FAM records where this person is a spouse

**Relationships**:
- Referenced by `Family` (as spouse or child)
- References `Family` via XRefs
- Contains many `Event`
- Contains many `Attribute`

**Validation Rules** (from FR-010):
- NAME tag is required (GEDCOM 5.5 spec)
- Gender must be M, F, U, or custom string (version-specific)
- Family XRefs must exist in document
- Cannot create circular relationships (person as own ancestor) - detected in validation, not parsing

**State Transitions**: None (immutable)

---

### 4. Family (FAM)

**Purpose**: Represents a family unit linking spouses and children

**Package**: `gedcom`

**Go Definition**:
```go
type Family struct {
    Record                    // Embedded base Record
    Husband       string     // XRef to husband Individual
    Wife          string     // XRef to wife Individual
    Children      []string   // XRefs to child Individuals
    Events        []Event    // Marriage, divorce, annulment
}
```

**Fields**:
- `Husband`: XRef to Individual (optional - may be unknown)
- `Wife`: XRef to Individual (optional - may be unknown)
- `Children`: List of XRefs to Individual records (ordered by birth)
- `Events`: Family events (marriage, divorce, annulment, etc.)

**Relationships**:
- References `Individual` for husband, wife, children
- Contains many `Event`

**Validation Rules**:
- At least one spouse or one child must be present
- Spouse and child XRefs must exist in document
- Cannot have duplicate children
- Marriage event should precede divorce/annulment events (warning, not error)

**State Transitions**: None (immutable)

---

### 5. Source (SOUR)

**Purpose**: Represents a source of genealogical information

**Package**: `gedcom`

**Go Definition**:
```go
type Source struct {
    Record                       // Embedded base Record
    Title          string        // Source title
    Author         string        // Author/originator
    Publication    string        // Publication facts
    RepositoryRefs []RepositoryRef // References to repositories
}

type RepositoryRef struct {
    XRef       string  // XRef to Repository
    CallNumber string  // Call number/identifier within repository
}
```

**Fields**:
- `Title`: Descriptive title of source
- `Author`: Person/organization who created source
- `Publication`: Where/when published
- `RepositoryRefs`: Physical/digital locations where source is held

**Relationships**:
- Referenced by `Event` (as source citation)
- References `Repository` via XRefs

**Validation Rules**:
- Title is recommended (warning if missing)
- Repository XRefs must exist if specified

**State Transitions**: None (immutable)

---

### 6. Repository (REPO)

**Purpose**: Physical or digital location where sources are stored

**Package**: `gedcom`

**Go Definition**:
```go
type Repository struct {
    Record                // Embedded base Record
    Name    string       // Repository name
    Address Address      // Physical/mailing address
}
```

**Fields**:
- `Name`: Name of institution/location
- `Address`: Contact information

**Relationships**:
- Referenced by `Source`

**Validation Rules**:
- Name is required

**State Transitions**: None (immutable)

---

### 7. Event

**Purpose**: Represents a life event (birth, death, marriage, etc.)

**Package**: `gedcom`

**Go Definition**:
```go
type Event struct {
    Type       EventType  // BIRT, DEAT, MARR, etc.
    Date       Date       // When event occurred
    Place      Place      // Where event occurred
    Sources    []string   // XRefs to Source records
    Notes      []string   // XRefs to Note records
    LineNumber int        // For error reporting
}

type EventType string

const (
    EventBirth       EventType = "BIRT"
    EventDeath       EventType = "DEAT"
    EventMarriage    EventType = "MARR"
    EventDivorce     EventType = "DIV"
    EventImmigration EventType = "IMMI"
    // ... more event types
)

type Date struct {
    Raw      string  // Original date string from GEDCOM
    Parsed   time.Time // Parsed Go time (may be zero if unparseable)
    Calendar string  // Gregorian, Hebrew, French Republican, Julian
    Modifier string  // ABT (about), BEF (before), AFT (after), etc.
}

type Place struct {
    Name       string  // Full place name
    Components []string // Hierarchical components (city, county, state, country)
}
```

**Fields**:
- `Type`: Event type enum
- `Date`: When event occurred (complex structure to handle GEDCOM date formats)
- `Place`: Where event occurred (hierarchical location)
- `Sources`: Citations supporting this event
- `Notes`: Additional notes about event

**Relationships**:
- Contained by `Individual` or `Family`
- References `Source` and `Note` via XRefs

**Validation Rules**:
- Date format must conform to GEDCOM spec (from FR-010)
- Invalid dates like "32 JAN 2020" trigger validation error
- Calendar must be recognized (Gregorian, Hebrew, French Republican, Julian)
- Place components separated by commas

**State Transitions**: None (immutable)

---

### 8. Note

**Purpose**: Extended textual information attached to records or events

**Package**: `gedcom`

**Go Definition**:
```go
type Note struct {
    Record        // Embedded base Record
    Text   string // Note text (may be multi-line)
}
```

**Fields**:
- `Text`: Free-form text content

**Relationships**:
- Referenced by any record type or event

**Validation Rules**: None

**State Transitions**: None (immutable)

---

### 9. Media Object (OBJE)

**Purpose**: Multimedia files (photos, documents, audio, video)

**Package**: `gedcom`

**Go Definition**:
```go
type MediaObject struct {
    Record                // Embedded base Record
    Files  []MediaFile   // File references
    Title  string        // Descriptive title
}

type MediaFile struct {
    FilePath string  // Path or URL to file
    Format   string  // File format (JPG, PDF, etc.)
    MediaType string // photo, audio, video, document
}
```

**Fields**:
- `Files`: One or more file references (primary + alternates)
- `Title`: Human-readable description

**Relationships**:
- Referenced by `Individual`, `Family`, `Event`, etc.

**Validation Rules**:
- At least one file must be specified
- Format should be recognized media type (warning if unknown)

**State Transitions**: None (immutable)

---

### 10. Cross-Reference

**Purpose**: Represents a link between records using unique identifiers

**Note**: Not a separate struct - implemented as string XRef fields in other structs

**Pattern**:
```go
// In Individual
SpouseInFamily []string  // XRefs like "@F1@", "@F2@"

// In Family
Husband string  // XRef like "@I1@"
```

**Resolution**: Via `Document.XRefMap[xref]` returns `*Record`

**Validation Rules** (from FR-005, FR-022):
- Format: `@[A-Za-z0-9_-]+@`
- Must be unique within document
- Referenced XRefs must exist (broken references trigger validation error)
- Non-standard formats (with dashes, underscores) accepted but warned

---

### 11. Tag-Value Pair

**Purpose**: Fundamental GEDCOM structure representing hierarchical data element

**Package**: `gedcom`

**Go Definition**:
```go
type Tag struct {
    Level   int     // Hierarchy level (0-99)
    Tag     string  // Tag name (HEAD, INDI, NAME, etc.)
    Value   string  // Tag value (may be empty)
    XRef    string  // Cross-reference if present
    SubTags []Tag   // Child tags (recursive structure)
    LineNumber int  // Original line number
}
```

**Fields**:
- `Level`: Nesting depth (0 = root)
- `Tag`: GEDCOM tag name (standardized or custom)
- `Value`: Associated value
- `XRef`: Pointer to another record (if this tag references one)
- `SubTags`: Nested tags (forms tree structure)
- `LineNumber`: For error reporting

**Relationships**:
- Contained by `Record`
- Forms recursive tree structure via `SubTags`

**Validation Rules** (from FR-008):
- Level must be consistent with parent (parent level + 1)
- Tag must be recognized for version (unknown tags allowed but warned for custom extensions)
- XRef must match pattern if present

**State Transitions**: None (immutable)

---

## Entity Relationship Diagram

```
Document
├── Header
├── Records[]
│   ├── Individual (INDI)
│   │   ├── Names[]
│   │   ├── Events[]
│   │   │   ├── Date
│   │   │   ├── Place
│   │   │   └── Sources[] (XRef → Source)
│   │   ├── Attributes[]
│   │   ├── ChildInFamily[] (XRef → Family)
│   │   └── SpouseInFamily[] (XRef → Family)
│   ├── Family (FAM)
│   │   ├── Husband (XRef → Individual)
│   │   ├── Wife (XRef → Individual)
│   │   ├── Children[] (XRef → Individual)
│   │   └── Events[]
│   ├── Source (SOUR)
│   │   └── RepositoryRefs[] (XRef → Repository)
│   ├── Repository (REPO)
│   ├── Note (NOTE)
│   └── MediaObject (OBJE)
│       └── Files[]
├── Trailer
└── XRefMap (index: XRef → *Record)
```

## Data Volume Estimates

Based on success criteria and assumptions:

| Entity Type | Typical File | Large File |
|-------------|--------------|------------|
| Individuals | 1K-10K | 100K-1M |
| Families | 500-5K | 50K-500K |
| Sources | 100-1K | 10K-100K |
| Events | 5K-50K | 500K-5M |
| Total Records | 2K-20K | 200K-2M |
| File Size | 1-50MB | 50-100MB |
| Peak Memory | 10-100MB | 100-200MB |

## Implementation Notes

### Type Safety

All enums (RecordType, EventType) are typed strings in Go:
```go
type RecordType string
```

This provides:
- Type safety (can't pass arbitrary string)
- String operations (comparison, printing)
- Switch exhaustiveness checking (with golangci-lint)

### Zero Values

All structs designed for safe zero-value usage:
- Slices: `nil` slice is valid empty slice
- Strings: `""` empty string is valid
- Maps: Created lazily or during initialization

### Pointer vs Value

- `Record`, `Individual`, `Family` etc.: Used by value (copied when passed)
- `Document.XRefMap`: Map values are pointers (`*Record`) to avoid copying large structs

### Custom vs Standard Tags

- Standard tags defined as constants: `TagName`, `TagBirth`, etc.
- Custom tags (extension tags) stored as-is
- FR-008: Library provides access to all tags including custom

## Validation Summary

| Entity | Required Fields | Unique Fields | Cross-Ref Fields |
|--------|----------------|---------------|------------------|
| Document | Header, Trailer, Version | - | XRefMap |
| Record | Type | XRef (if present) | - |
| Individual | Names (at least one NAME tag) | XRef | ChildInFamily, SpouseInFamily |
| Family | At least one spouse or child | XRef | Husband, Wife, Children |
| Source | - | XRef | RepositoryRefs |
| Repository | Name | XRef | - |
| Event | Type | - | Sources, Notes |
| Note | - | XRef | - |
| MediaObject | Files (at least one) | XRef | - |

All validation performed by `validator/` package, not during parsing. Parser builds structure; validator checks semantics.
