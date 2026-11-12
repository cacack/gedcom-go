# Decoding Untrusted Input

What `gedcom-go` guarantees when the input is hostile, and what it leaves to
the caller. Read this before decoding uploads in a server.

Two guarantees hold for any input:

- **No panic.** Malformed input is an error or a diagnostic, never a panic
  ([ADR 0007](../decisions/0007-error-transparency.md)). The parser, line
  parser, charset reader, date parser and encoder round-trip are fuzzed
  nightly.
- **Bounded per-line work.** No single line, diagnostic or charset buffer can
  grow without limit; the bounds are listed below.

What is **not** bounded is the total: how many lines, records and
continuation lines a file has. Memory grows with the input, by the multipliers
measured below, so the caller has to cap the input size. Like `archive/zip`
and `encoding/xml`, the library reads what it is given; the size limit is
yours to set.

## Bounds

| Bound | Value | Configurable | When it is hit |
|-------|-------|--------------|----------------|
| Line length | `parser.MaxLineBytes` (1 MiB, excluding the terminator) | No | The whole read fails, in strict and lenient mode. The error wraps `bufio.ErrTooLong`, or `parser.ErrLineTooLong` from the offset-reporting iterators and `LazyParser.BuildIndex`. See [Line Length Limit](decoding.md#line-length-limit). |
| Nesting depth | `parser.MaxNestingDepth` (100: levels 0–99) | No; fixed by the two-digit level field | The line is a parse error. Strict mode fails; lenient mode drops the line with an `INVALID_LEVEL` diagnostic. |
| Charset header probe | 64 KiB | No | `CHAR` is looked for in the first 64 KiB only. A `CHAR` line past that is not seen, so the encoding falls back to detection from the bytes ([ADR 0004](../decisions/0004-encoding-detection-cascade.md)). The rest of the file is still read. |
| ANSEL combining marks held | `charset.MaxPendingCombining` (16) | No | A longer run of marks is written out standalone in groups of 16, not rejected. |
| Diagnostics returned | `DecodeOptions.MaxDiagnostics`, default `decoder.DefaultMaxDiagnostics` (100,000) | Yes; a negative value (`decoder.UnlimitedDiagnostics`) removes it | The first ones are kept and one `DIAGNOSTICS_TRUNCATED` diagnostic counts the rest. Errors for an XRef containing a space are kept past the cap. See [Bounds on Diagnostics](decoding.md#bounds-on-diagnostics). |
| Unknown-tag diagnostics per tag name | `decoder.MaxUnknownTagDiagnosticsPerTag` (10) | No | The rest of that tag's lines are counted in one closing diagnostic. Every line stays in `Record.Tags`. Unknown level-0 record types are not capped. |
| Invalid-UTF-8 diagnostics | `decoder.MaxInvalidEncodingDiagnostics` (100) | No | Every invalid byte is still replaced; the rest are counted in one closing diagnostic. |
| Input echoed per diagnostic or `ParseError` | `parser.MaxContextBytes` (256) | No | `Message` and `Context` are clipped at a UTF-8 boundary and end in `…`. Values in the document are never clipped. |
| Parse errors kept by `parser.ParseWithOptions` | `ParseOptions.MaxErrors` | Yes; the default, 0, is unlimited | Parsing continues; further errors are not collected. |
| Duplicate-detection group size | `DuplicateConfig.MaxGroupSize`, default 1000 | Yes; 0 means the default, negative is unlimited | The surname group is skipped whole, with one `DUPLICATE_DETECTION_LIMITED` warning. See [Bounding Untrusted Input](performance.md#bounding-untrusted-input). |
| Given-name runes compared | `DuplicateConfig.MaxNameRunes`, default 64 | Yes; as above | Longer names are compared on their first 64 runes and counted on the same warning. |
| Time | `Context` on `DecodeOptions`, `ValidateOptions`, `ConvertOptions`, `EncodeOptions`, `merge.CombineOptions`; `LazyParser.BuildIndexContext` | Yes; none by default | The call stops shortly after the context is done and returns its error wrapped. |

## Unbounded Dimensions

Each of these grows with the input. The costs were measured on synthetic
files with Go 1.27 on arm64: peak is the highest in-use heap sampled during
the call, retained is the heap still held after it returns. Real exports fall
between the typical and worst cases;
[performance](performance.md) has benchmarks on real files.

| Dimension | Limited only by | Measured cost |
|-----------|-----------------|---------------|
| File size and record count, batch | Your input limit | `Decode`: peak about 18× the input and 15× retained for typical records (98 MB → 1.8 GB peak). Worst case, short lines: about 35× peak (20 MB of 4-byte lines → 696 MB). |
| Lines in one record | Your input limit | Streaming APIs hold one whole record, and one record can be the whole file. `parser.Records` peaked at 11× the record's size for a 2 M-line record (18 MB → 206 MB), and 20× for 4-byte lines; adding `DecodeRecord` raised those to 35× and 45×. |
| `CONC`/`CONT` chain length | Your input limit | Each line is under 1 MiB, but a chain has no length limit, and its joined value is held next to its raw lines. `Decode` of one note with 2 M `CONC` lines: 36 MB → 453 MB peak, 214 MB retained. |
| Name and value length | `MaxLineBytes`, or a `CONC` chain | Duplicate detection compares only `MaxNameRunes`; nothing else shortens them. |
| `LazyParser` index entries | Your input limit | About 170 bytes per record with an XRef. 2 M minimal records (27 MB) → 466 MB peak, 333 MB retained. Records without an XRef are not indexed. |
| Diagnostics for XRefs containing a space | Your input limit | Kept past `MaxDiagnostics`, as the parser keeps their lines. |

Validation, conversion, encoding and merging work on a decoded `Document`, so
their memory follows the document's; give them a `Context` deadline.

## Sizing `io.LimitReader`

Choose a memory budget per request, then cap the input it may read. The rule
differs by API because what is held differs:

| API | Holds | Worst case measured | Input limit for a memory budget *M* |
|-----|-------|---------------------|-------------------------------------|
| `decoder.Decode`, `DecodeWithOptions`, `DecodeWithDiagnostics` | The whole document | 35× | *M* / 40 |
| `parser.Parse`, `ParseWithOptions` | Every line | 19× | *M* / 25 |
| `parser.Records`, `RecordIterator`, `RecordsWithOffset`, `LazyParser.FindRecord` | One record at a time | 20× | *M* / 25 |
| The same, each record passed to `decoder.DecodeRecord` | One record and its typed form | 45× | *M* / 50 |
| `parser.LazyParser.BuildIndex` | One index entry per record with an XRef | 17× | *M* / 20 |

The streaming rows need a limit as strict as the batch ones. They hold one
record at a time, but nothing stops one record being the whole input, so only
the input limit bounds the largest record. Their advantage is the typical
case: a 24 MB file of ordinary records peaked at about 4 MB through
`parser.Records` and 89 MB through `parser.Records` and `DecodeRecord`,
against 443 MB through `Decode`.

```go
const budget = 512 << 20 // 512 MiB per request
const limit = budget / 40

lr := &io.LimitedReader{R: upload, N: limit + 1}
doc, err := decoder.DecodeWithOptions(lr, opts)
if lr.N == 0 { // read past the limit
    return nil, errors.New("upload too large")
}
if err != nil {
    return nil, err
}
```

Read one byte past the limit, as above, so a file that is too large is
reported as such rather than decoded truncated: `io.LimitReader` alone stops
silently, and a truncated GEDCOM file usually still decodes.
