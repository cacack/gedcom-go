# ADR-003: Lossless Dual Storage (Raw Tags + Typed Entity)

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Record representation in gedcom-go library
**Constitution**: Implements Principle VI (Lossless Representation)

## Decision

Each `Record` maintains both raw GEDCOM tags and a typed entity. The encoder prefers raw tags when present, falling back to entity-to-tags conversion only when tags are absent.

## Context

GEDCOM files contain both standard tags and vendor-specific custom tags (prefixed with underscore):

```gedcom
0 @I1@ INDI
1 NAME John /Smith/
1 _FSFTID KWCJ-QN7
1 _CUSTOM Some vendor data
```

The question: how do we provide typed access to known structures while preserving unknown/custom tags?

## Decision Drivers

1. **Lossless representation** - Custom tags must survive decode→encode cycles
2. **Type safety** - Known structures should have typed access
3. **Round-trip fidelity** - Output should match input when no modifications made
4. **API ergonomics** - Common operations should be convenient

## Considered Options

### Option A: Tags Only (No Typed Entities)

```go
type Record struct {
    XRef string
    Tags []*Tag
}
// Access via: record.FindTag("NAME").Value
```

- **Pros**: Perfect fidelity, simple model
- **Cons**: Terrible ergonomics, no type safety, complex navigation
- **Verdict**: Rejected - unusable API for common operations

### Option B: Typed Entities Only (Discard Unknown Tags)

```go
type Record struct {
    XRef   string
    Entity interface{}  // *Individual, *Family, etc.
}
```

- **Pros**: Clean typed API
- **Cons**: Loses custom/vendor tags, violates lossless principle
- **Verdict**: Rejected - data loss

### Option C: Dual Storage with Tag Preference (Selected)

```go
type Record struct {
    XRef   string
    Tags   []*Tag       // Raw tags (preserved)
    Entity interface{}  // Typed entity (parsed)
}
```

- **Pros**: Lossless storage, typed access, round-trip fidelity
- **Cons**: Memory overhead (acceptable for correctness)
- **Verdict**: Accepted

## Consequences

### Positive

- Unknown/custom tags preserved through decode→encode
- Typed access via `record.Entity.(*Individual)`
- Modifications to Entity reflected in output (when Tags empty)
- Vendor extensions (Ancestry, FamilySearch) survive round-trips

### Negative

- Higher memory usage (both representations stored)
- Encoder must check Tags first, then Entity
- Modifications to Entity ignored if Tags present (by design)

## Implementation

**Decode**: Parser populates Tags, decoder builds Entity from Tags

```go
record := &Record{
    XRef:   "@I1@",
    Tags:   parsedTags,        // All tags preserved
    Entity: buildIndividual(parsedTags),  // Typed extraction
}
```

**Encode**: Prefer Tags, fall back to Entity conversion

```go
func encodeRecord(record *Record, opts *EncodeOptions) {
    tags := record.Tags
    if len(tags) == 0 && record.Entity != nil {
        tags = entityToTags(record, opts)  // Convert only if needed
    }
    writeTags(tags)
}
```

**Modification Pattern**: Clear Tags to use Entity changes

```go
// To modify and re-encode:
individual := record.Entity.(*Individual)
individual.Names[0].Given = "Jonathan"
record.Tags = nil  // Clear to force entity-to-tags conversion
```

## References

- `gedcom/record.go` - Record struct definition
- `encoder/encoder.go` - Tag preference logic
- `encoder/entity_writer.go` - Entity-to-tags conversion
- CLAUDE.md - Principle VI (Lossless Representation)
