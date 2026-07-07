# ADR-007: Error Transparency

**Status**: Accepted
**Date**: 2025-01-19
**Context**: Error handling philosophy in gedcom-go library
**Constitution**: Implements Principle V (Error Transparency)

## Decision

All errors include line numbers, context (actual content), and structured data for programmatic access. The library never panics; all error conditions are recoverable and return descriptive errors.

## Context

Genealogical data is often messy:
- Legacy software exports with quirks
- Hand-edited files with typos
- Encoding issues from multiple conversions
- Vendor-specific extensions

When parsing fails, users need to:
1. Locate the exact problem in their file
2. Understand what went wrong
3. Fix it or report it upstream

The question: how do we make errors actionable rather than frustrating?

## Decision Drivers

1. **Actionable errors** - Users can locate and fix problems
2. **Never panic** - Library consumers shouldn't need recovery handlers
3. **Programmatic access** - Errors can be inspected, not just printed
4. **Context preservation** - Show what was being processed

## Considered Options

### Option A: Simple Error Strings

```go
return fmt.Errorf("invalid tag")
```

- **Pros**: Simple implementation
- **Cons**: No location info, no context, frustrating to debug
- **Verdict**: Rejected - not actionable

### Option B: Wrapped Errors Only

```go
return fmt.Errorf("line 42: %w", err)
```

- **Pros**: Some context, standard wrapping
- **Cons**: No structured access, inconsistent formatting
- **Verdict**: Rejected - insufficient structure

### Option C: Structured Error Types (Selected)

```go
type ParseError struct {
    Line    int
    Column  int
    Message string
    Context string  // Actual line content
    Err     error
}
```

- **Pros**: Full context, structured access, implements error interface
- **Verdict**: Accepted

## Consequences

### Positive

- Error messages show exact location: `line 42, column 15`
- Context shows what was parsed: `Context: "1 NAME John /Smith"`
- Programmatic inspection: `if pe, ok := err.(*ParseError); ok { ... }`
- Consistent formatting across all error types

### Negative

- More complex error construction
- Slightly larger error values (acceptable)

## Implementation

### Parse Errors

```go
type ParseError struct {
    Line    int     // 1-based line number
    Column  int     // 1-based column (where applicable)
    Message string  // Human-readable description
    Context string  // Actual line content
    Err     error   // Underlying error (for wrapping)
}

func (e *ParseError) Error() string {
    if e.Context != "" {
        return fmt.Sprintf("line %d: %s (context: %q)", e.Line, e.Message, e.Context)
    }
    return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}
```

### Encoding Errors

```go
type ErrInvalidUTF8 struct {
    Line   int
    Column int
}

func (e *ErrInvalidUTF8) Error() string {
    return fmt.Sprintf("invalid UTF-8 sequence at line %d, column %d", e.Line, e.Column)
}
```

### Validation Issues

```go
type Issue struct {
    Severity    Severity  // Error, Warning, Info
    Code        string    // "ORPHANED_FAMC", "DEATH_BEFORE_BIRTH"
    Message     string    // Human-readable description
    RecordXRef  string    // Affected record
    RelatedXRef string    // Related record (if applicable)
    LineNumber  int       // Source line (if known)
}
```

### Error Examples

```
Parse error:
  line 42: invalid level value "abc" (context: "abc NAME John /Smith/")

Encoding error:
  invalid UTF-8 sequence at line 156, column 23

Validation issue:
  [ERROR] DEATH_BEFORE_BIRTH: Individual @I42@ has death date (1900) before birth date (1950)
```

## Never Panic

The library follows Go's convention that panics are for programmer errors, not runtime conditions:

- Invalid input returns error, never panics
- Nil checks return early with descriptive errors
- Index bounds checked before access
- Type assertions use comma-ok form

```go
// Good: return error
if level < 0 {
    return nil, &ParseError{Line: lineNum, Message: "negative level"}
}

// Bad: panic
if level < 0 {
    panic("negative level")  // Never do this
}
```

## References

- `parser/errors.go` - ParseError implementation
- `charset/charset.go` - ErrInvalidUTF8
- `validator/issue.go` - Validation issue structure
- CLAUDE.md - Principle V (Error Transparency)
