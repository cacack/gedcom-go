# Decoding Contract

This document describes the strict vs lenient decoding behavior of gedcom-go, helping you predict what happens when parsing messy vendor GEDCOMs.

## Strict Mode

Enable strict mode when you need files to be fully valid or rejected entirely:

```go
opts := &decoder.DecodeOptions{StrictMode: true}
doc, err := decoder.DecodeWithOptions(r, opts)
if err != nil {
    // File rejected on first parse error
}
```

### Behavior

- **First error stops parsing**: Any syntax error returns immediately
- **No partial documents**: Either full success or complete failure
- **No diagnostics collected**: Use the returned `error` for failure details

### What Causes Rejection

| Issue | Example |
|-------|---------|
| Invalid level number | `XYZ NAME John` (non-numeric level) |
| Missing tag | `0 @I1@` (XRef without tag) |
| Malformed XRef | `0 @BAD XREF@ INDI` (space inside the identifier), `0 @I1 INDI` (no closing `@`) |
| Invalid level jump | Level 0 to level 2 (skipping level 1) |
| Empty lines | Blank lines in the GEDCOM stream |

### When to Use Strict Mode

- Validating files before archival
- Testing spec compliance
- Processing files from known-good sources
- When partial data is worse than no data

## Lenient Mode (Default)

Lenient mode continues parsing after errors, collecting diagnostics:

```go
result, err := decoder.DecodeWithDiagnostics(r, nil)
if err != nil {
    // Only returned if no valid records could be parsed
}

// Access partial document
doc := result.Document

// Check what went wrong
for _, d := range result.Diagnostics {
    fmt.Printf("Line %d: %s - %s\n", d.Line, d.Code, d.Message)
}
```

### Behavior

- **Skips or recovers malformed lines**: Invalid lines are recorded as diagnostics; where a line can be recovered (see the table below) it is kept, so a diagnostic does not always mean the data is gone
- **Preserves valid records**: All parseable data is kept
- **Collects all issues**: Every problem is recorded with line number and context
- **Returns error only if completely unparseable**: Empty or fully-corrupt files

### Diagnostic Codes

Parse-level errors (severity: ERROR):

| Code | Meaning |
|------|---------|
| `SYNTAX_ERROR` | General syntax problem |
| `INVALID_LEVEL` | Level number could not be parsed, is negative, or exceeds 99 (line dropped) |
| `INVALID_XREF` | Cross-reference identifier that is malformed (contains a space, or has no closing `@`) or that the grammar forbids where it appears (`0 @X1@ HEAD`, `0 @X1@ TRLR`); the line is kept as a record wherever recovery is possible — see [FEATURES.md](../../FEATURES.md#lenient-parsing--diagnostics) |
| `EMPTY_LINE` | Unexpected blank line |

Entity-level warnings (severity: WARNING):

| Code | Meaning |
|------|---------|
| `BAD_LEVEL_JUMP` | Invalid level increment (e.g., 0 to 2); line clamped to `prev + 1` and kept |
| `UNKNOWN_TAG` | Unrecognized tag (preserved in raw form) |
| `INVALID_VALUE` | Value doesn't match expected format |
| `MISSING_REQUIRED` | Required subordinate tag is missing |
| `SKIPPED_RECORD` | Entire record skipped due to errors |

### Working with Diagnostics

```go
result, _ := decoder.DecodeWithDiagnostics(r, nil)

// Check if any errors occurred
if result.Diagnostics.HasErrors() {
    // At least one ERROR-level diagnostic
}

// Filter by severity
errors := result.Diagnostics.Errors()     // ERROR only
warnings := result.Diagnostics.Warnings() // WARNING only

// Human-readable summary
fmt.Println(result.Diagnostics.String())
```

### When to Use Lenient Mode

- Importing files from unknown sources
- Processing vendor GEDCOMs with quirks
- Maximizing data recovery from corrupt files
- Production systems where some data is better than none

## Round-trip Expectations

When encoding a decoded document back to GEDCOM format, here's what to expect.

### Preserved Exactly

| Element | Notes |
|---------|-------|
| Record hierarchy | All records and their nested structure |
| XRef identifiers | `@I1@`, `@F1@`, etc. |
| Tag names | Including vendor custom tags (`_APID`, `_MHID`) |
| Tag values | Original content preserved |
| Custom tags | Stored in `CustomTags` on each entity |

### Normalized (May Change)

| Element | Default Behavior |
|---------|------------------|
| Line endings | Converted to `\n` (configurable via `EncodeOptions.LineEnding`) |
| Character encoding | Output as UTF-8 (input ANSEL/UTF-16 converted) |
| Long lines | May be split with CONC tags at 248 chars (configurable) |

### Known Transformations

**CONT/CONC Handling**

The decoder processes CONT (continuation with newline) and CONC (concatenation without newline) tags and joins the text. On re-encoding:

- Multi-line text is split with CONT tags at newlines
- Lines exceeding `MaxLineLength` (default 248) are split with CONC tags
- Original CONT/CONC boundaries are not preserved

Example decode:
```
0 @N1@ NOTE First line
1 CONT Second line
1 CONC  with more text
```

Becomes `Note.Text`: `"First line\nSecond line with more text"`

`Note.FullText()` returns the same string. It exists for the deprecated
`Note.Continuation` slice, which the decoder no longer populates — read `Text`.

The same folding now applies to notes on substructures, not only to the `NOTE`
record. `Event`, `Attribute`, `SourceCitation`, `LDSOrdinance` and `ChangeDate`
each expose `NoteXRefs` (pointers to `NOTE`/`SNOTE` records) and `InlineNotes`
(text, with `CONT`/`CONC` folded), plus the deprecated interleaved
`Notes []string`.

**Behaviour change.** `Event.Notes` previously received the raw `NOTE` value
with no folding, so a multi-line event note read back as its first line only;
it now carries the whole folded text. A `SNOTE` pointer on an event is also no
longer reported as an unknown tag, so an event carrying both `NOTE` and `SNOTE`
has two `Notes` entries where it previously had one. Notes on `SourceCitation`,
`LDSOrdinance` and `ChangeDate` were previously dropped entirely and are now
kept.

Re-encoded (may differ from original):
```
0 @N1@ NOTE First line
1 CONT Second line with more text
```

**Value Whitespace**

GEDCOM separates a tag from its value with exactly one space. The decoder consumes
that one space and keeps everything after it, so a value that an exporter padded
with extra spaces arrives padded:

```
1 ADDR   123 Main St     ->  Line1 = "  123 Main St"
2 CONC  and then left    ->  the leading space is the word separator
```

That padding is payload, not formatting — dropping it merges words across a `CONC`
continuation. Callers that want tidy display strings should `strings.TrimSpace`
the fields they render; the library does not trim on your behalf, because it
cannot tell a cosmetic space from a significant one.

A value that is nothing but spaces is reported as empty: `1 CONT ` and `1 CONT`
both yield `""`.

**Header Reconstruction**

The header is rebuilt from `Document.Header` fields. Non-standard header tags not mapped to typed fields may not round-trip.

**Empty Values**

Tags with empty values are preserved: `1 BIRT` becomes `1 BIRT` (not `1 BIRT ` with trailing space).

## API Summary

| Function | Strict Mode | Lenient Mode |
|----------|-------------|--------------|
| `Decode(r)` | N/A | Default behavior, no diagnostics access |
| `DecodeWithOptions(r, opts)` | `opts.StrictMode=true` | `opts.StrictMode=false` (no diagnostics) |
| `DecodeWithDiagnostics(r, opts)` | Returns error on first issue | Returns `DecodeResult` with diagnostics |

## Related Documentation

- [compatibility](../governance/policies/compatibility.md) - Vendor compatibility matrix
- [gedcom-versions](../reference/gedcom-versions.md) - Version-specific differences
- [api-stability](../governance/policies/api-stability.md) - API compatibility guarantees
