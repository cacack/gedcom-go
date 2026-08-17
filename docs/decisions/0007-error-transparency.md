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

func (e *ErrInvalidUTF8) ErrorLine() int {
    return e.Line
}
```

### Reader Errors Own Their Line Number

A reader-layer error knows the physical line of the byte that failed; the
line-oriented layers above it do not. The parser's counter only reaches the last
line the reader handed over, so it lags whenever a read is rejected part-way
through a chunk the reader had already consumed.

The convention: **a reader error that knows its own physical line exposes
`ErrorLine() int`, and the layer above adopts it in preference to its own
counter.** `parser.readErrorLine` applies this when wrapping `scanner.Err()`,
treating a zero return as "unknown" and falling back.

Reader errors carrying a line, and their conformance to the contract:

| Error | Package | `ErrorLine()` |
|-------|---------|---------------|
| `ErrInvalidUTF8` | `charset` | yes |
| `ErrInvalidANSEL` | `charset` | yes |

Each implementer pins the method with a `var _ interface{ ErrorLine() int }`
assertion, since the parser matches on the method rather than the concrete type.
Any new reader-layer error carrying a line (a future `ErrUnsupportedEncoding`)
belongs in this table, or it will reintroduce the drift.

Line numbering must also match the parser's line splitting, which ends a line on
LF, CRLF, or a bare CR (old Macintosh). A counter that recognizes only LF
reports line 1 for an entire CR-only file. Each `charset` reader implements this
rule in its own `advance` method — two copies of the same switch, so a change to
one must be mirrored in the other.

The fallback counter cannot reach the line the bad byte is on. Nothing after
that byte is delivered, and the truncated head of its line is dropped rather
than tokenized (see the next section), so the counter stops at the last
*complete* line — one short — and reports line 0 when the bad byte is the first
in the file. `ErrorLine()` is therefore the only source of the right answer, not
a refinement of a usually-correct guess.

### A Reader Error Outranks the Truncated Line It Leaves Behind

Reads are byte-aligned, never line-aligned, so a read that fails lands part-way
through a line as often as not. The line-oriented layer above is then holding
the head of a line whose remaining bytes will never arrive, and `bufio.Scanner`
offers no way to tell that fragment from a line: it treats *any* reader error as
EOF, so it emits the residue as a token.

Parsing that token is wrong either way it goes. With fewer than two fields it
produces a syntax error against text the file never contained, and that error is
reported instead of the read failure that caused it. With two or more fields
(`1 NAME Fr`) it parses cleanly and a silently shortened value enters the
document — the outcome this ADR exists to forbid.

The convention: **a reader failure outranks any parse result derived from the
line it truncated.** The fragment is dropped, contributes no diagnostic, and the
wrapped reader error is reported — located, per the rule above, at the physical
line the reader names. Nothing is hidden by the drop: the error identifies the
exact line and column, and the fragment is by construction the last line, so no
subordinate lines can be reparented by its absence (contrast the malformed-XRef
recovery in `parser.ParseLine`, which keeps its line for exactly that reason).

The counterpart obligation runs the other way. A reader validates in chunks
whose boundaries are byte-aligned, so the chunk holding a bad byte usually holds
good lines too — 189 of them in the CP437 corpus fixture. **A reader that
rejects a chunk still delivers the bytes ahead of the offending one**, so the
partial document a lenient caller recovers reaches the last complete line before
the failure rather than stopping a chunk short of the line the error names
(Constitution Principle 6). Because that makes a failing reader hand back data,
the failure is also sticky: a second `Read` re-reports it instead of resuming
after the rejected bytes, which would turn a reported error into an unreported
hole in the middle of the document.

Completeness is decided by the terminator, not by the failure. A token that
reached LF, CRLF, or a bare CR is a whole line and survives the read error that
follows it — including a CRLF pair split at the failure point, where the CR ends
the line and only the LF is lost. Only a token emitted with no terminator at all
is a fragment, and only when the scan ended in a reader error rather than at
end of input, where an unterminated final line is ordinary and valid.
`parser.lineScanner.Truncated` is the single place that predicate lives.

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

- `parser/errors.go` - ParseError implementation, readErrorLine
- `parser/parser.go` - lineScanner.Truncated, the truncated-tail predicate
- `charset/charset.go` - ErrInvalidUTF8
- `charset/ansel.go` - ErrInvalidANSEL
- `validator/issue.go` - Validation issue structure
- CLAUDE.md - Principle V (Error Transparency)
