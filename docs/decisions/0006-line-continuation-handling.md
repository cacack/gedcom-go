# ADR-006: Line Continuation Handling (CONT/CONC)

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Long text and multiline handling in gedcom-go library

## Decision

Automatically handle CONT (continuation) and CONC (concatenation) tags during both decoding and encoding. CONT preserves logical line breaks, CONC handles physical line length limits with word-boundary splitting.

## Context

GEDCOM has historically limited line length (255 characters) and uses two mechanisms for longer text:

```gedcom
1 NOTE This is a long note that needs
2 CONC  to be concatenated (no line break)
2 CONT This is a new line (with line break)
2 CONT Another line
```

- **CONT**: Logical line break (newline in content)
- **CONC**: Physical split (continues same logical line)

The question: how do we handle these transparently for library consumers?

## Decision Drivers

1. **Transparent handling** - Consumers shouldn't deal with CONT/CONC
2. **Round-trip fidelity** - Encoded output should be valid GEDCOM
3. **GEDCOM 7.0 compatibility** - 7.0 removed CONC and has no line-length limit
4. **Reader tolerance** - 5.5.1 requires a CONC split between two non-space characters, since readers that trim lines lose a space at a split

## Considered Options

### Option A: Preserve CONT/CONC as Separate Tags

```go
type Note struct {
    Lines []string  // Each CONT is separate element
}
```

- **Pros**: Exact preservation
- **Cons**: Terrible API, consumers must reassemble
- **Verdict**: Rejected - poor ergonomics

### Option B: Merge on Decode, Raw on Encode

- **Pros**: Easy read access
- **Cons**: Can't round-trip, encoding requires manual splitting
- **Verdict**: Rejected - incomplete solution

### Option C: Full Automatic Handling (Selected)

- **Decode**: Merge CONT/CONC into single string with embedded `\n`
- **Encode**: Split on `\n` to CONT; for 5.x, split long lines to CONC between two non-space characters and never inside a UTF-8 character
- **Pros**: Transparent API, valid output, round-trip works
- **Verdict**: Accepted

## Consequences

### Positive

- Consumers work with plain strings containing `\n`
- Encoded output respects GEDCOM line limits
- No space is lost to readers that trim a line at a CONC split
- GEDCOM 7.0 output never contains CONC

### Negative

- Original CONT/CONC structure not preserved exactly (acceptable)
- Slight encoding overhead for long text
- A 5.x split usually falls mid-word, which reads worse in a raw file than a word-boundary split

## Implementation

### Decoding

```go
func mergeTextTags(tags []*Tag) string {
    var result strings.Builder
    for _, tag := range tags {
        switch tag.Tag {
        case "CONT":
            result.WriteString("\n")
            result.WriteString(tag.Value)
        case "CONC":
            result.WriteString(tag.Value)
        default:
            result.WriteString(tag.Value)
        }
    }
    return result.String()
}
```

### Encoding

```go
func writeTextValue(level int, tag, value string, opts *EncodeOptions) []*Tag {
    // Split on newlines first (become CONT)
    lines := strings.Split(value, "\n")

    var tags []*Tag
    for i, line := range lines {
        if i == 0 {
            // First line is the main tag
            tags = append(tags, splitForLength(level, tag, line, opts)...)
        } else {
            // Subsequent lines are CONT
            tags = append(tags, splitForLength(level+1, "CONT", line, opts)...)
        }
    }
    return tags
}

func splitForLength(level int, tag, value string, opts *EncodeOptions) []*Tag {
    maxLen := opts.MaxLineLength  // Default: 248
    if len(value) <= maxLen {
        return []*Tag{{Level: level, Tag: tag, Value: value}}
    }

    // Split between two non-space characters, on a rune boundary
    var tags []*Tag
    for len(value) > maxLen {
        splitPoint := findSplitPoint(value, maxLen)
        tags = append(tags, &Tag{Level: level, Tag: tag, Value: value[:splitPoint]})
        value = value[splitPoint:]
        tag = "CONC"  // Subsequent segments are CONC
        level++       // CONC is subordinate
    }
    tags = append(tags, &Tag{Level: level, Tag: tag, Value: value})
    return tags
}
```

### Configuration

```go
type EncodeOptions struct {
    MaxLineLength   int   // Default: 248 (GEDCOM spec: 255 minus overhead)
    DisableLineWrap bool  // Skip CONC generation (implied for GEDCOM 7.0)
}
```

## GEDCOM 7.0 Considerations

GEDCOM 7.0 removed CONC and has no line-length limit. When the output version (`TargetVersion`, else `Header.Version`) is 7.0:

- No CONC is generated, whatever `MaxLineLength` and `DisableLineWrap` say (issue #565)
- CONT is still used for explicit line breaks
- Values may exceed traditional line limits

## References

- `encoder/entity_writer.go` - Text splitting implementation
- `encoder/options.go` - Configuration options
- `decoder/decoder.go` - CONT/CONC merging during decode
