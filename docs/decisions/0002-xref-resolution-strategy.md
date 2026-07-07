# ADR-002: XRef Resolution Strategy

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Cross-reference handling in gedcom-go library

## Decision

Use string-based cross-references with a document-level `XRefMap` providing O(1) lookup to typed `Record` objects, rather than embedding direct pointers between entities.

## Context

GEDCOM files use cross-references (XRefs) like `@I1@`, `@F1@` to link records. For example, a Family record references its members:

```gedcom
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
```

The question: how should the library represent these relationships in memory?

## Decision Drivers

1. **Lossless representation** - Preserve original XRef strings for round-trip fidelity
2. **API simplicity** - Easy navigation between related records
3. **Memory efficiency** - Avoid circular reference issues
4. **Consistency** - All record types handled uniformly

## Considered Options

### Option A: Direct Typed Pointers

```go
type Family struct {
    Husband  *Individual  // Direct pointer
    Wife     *Individual
    Children []*Individual
}
```

- **Pros**: Type-safe navigation, no lookup required
- **Cons**: Circular dependencies, complex initialization order, loses original XRef strings, GC complexity
- **Verdict**: Rejected - loses GEDCOM fidelity, creates tight coupling

### Option B: Lazy Resolution on Access

```go
func (f *Family) GetHusband(doc *Document) *Individual {
    return doc.resolveIndividual(f.husbandRef)  // Resolve every time
}
```

- **Pros**: Simple storage, deferred resolution
- **Cons**: Repeated resolution overhead, caching complexity
- **Verdict**: Rejected - unnecessary complexity

### Option C: String XRefs with O(1) Map Lookup (Selected)

```go
type Family struct {
    Husband  string  // "@I1@"
    Wife     string  // "@I2@"
    Children []string
}

type Document struct {
    XRefMap map[string]*Record  // Built during decode
}
```

- **Pros**: Preserves original XRefs, O(1) lookup, simple memory model, no circular refs
- **Cons**: Requires explicit lookup call
- **Verdict**: Accepted

## Consequences

### Positive

- Original XRef strings preserved for encoding
- O(1) lookup via `doc.GetIndividual(xref)`
- No circular reference issues
- Simple serialization/deserialization
- Uniform handling across all record types

### Negative

- Consumers must call lookup methods rather than traversing pointers
- Broken references return `nil` (must handle gracefully)

## Implementation

**Storage**: XRefs stored as strings in all relationship fields

```go
type Individual struct {
    FamilyChild  []FamilyLink  // FamilyLink.FamilyXRef = "@F1@"
    FamilySpouse []string      // ["@F1@", "@F2@"]
}
```

**Lookup**: Document provides typed accessor methods

```go
func (d *Document) GetIndividual(xref string) *Individual
func (d *Document) GetFamily(xref string) *Family
func (d *Document) GetSource(xref string) *Source
// ... etc
```

**Convenience**: Relationship traversal methods take Document parameter

```go
func (i *Individual) Parents(doc *Document) []*Individual
func (i *Individual) Children(doc *Document) []*Individual
func (f *Family) HusbandIndividual(doc *Document) *Individual
```

## References

- `gedcom/document.go` - XRefMap and lookup methods
- `gedcom/individual.go` - Relationship traversal methods
- `decoder/decoder.go` - XRefMap construction during decode
