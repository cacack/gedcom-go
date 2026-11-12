# Performance Characteristics

This document provides detailed performance benchmarks and optimization guidance for the gedcom-go library.

## Benchmark Environment

All benchmarks were run on:
- **Platform**: Apple M2 (ARM64)
- **OS**: macOS
- **Go Version**: 1.25
- **Test Files**: Real-world GEDCOM files from `testdata/`

This document is the single source of truth for gedcom-go benchmark figures. The numbers
below are specific to the environment above — absolute values are hardware- and
warmup-dependent, so treat them as indicative rather than guaranteed. Run `make bench` to
measure on your own hardware.

## Performance Summary

| Operation | Target | Current | Status |
|-----------|--------|---------|--------|
| Parse 1000 individuals | < 1ms | 697 µs | Pass |
| Decode 1000 individuals | < 15ms | 13.0 ms | Pass |
| Encode 1000 individuals | < 2ms | 1.15 ms | Pass |
| Validate 1000 individuals | < 10µs | 5.91 µs | Pass |

## Decoder Performance

### Throughput Benchmarks

| File Size | Time/Operation | Memory | Allocations | Throughput |
|-----------|----------------|--------|-------------|------------|
| 170B (minimal) | 7 µs | 7 KB | 69 | ~24 MB/s |
| 15KB (small) | 381 µs | 214 KB | 3,530 | ~39 MB/s |
| 458KB (medium) | 17 ms | 8.5 MB | 139K | ~27 MB/s |
| 1.1MB (large) | 32 ms | 14 MB | 214K | ~34 MB/s |
| 46.1MB (scale) | 854 ms | 1.5 GB | 18.1M | ~57 MB/s |

**Average Throughput**: ~32 MB/s for large files (the 46 MB scale fixture sustains ~57 MB/s)

**Memory Usage** (Memory column above): cumulative bytes allocated per decode (`B/op`),
roughly 12-33x file size due to per-line and per-record churn (the 46 MB scale
fixture measures ~33x). Retained heap after decode
is much lower — the batch-vs-streaming section below measures it directly.

### Scale Fixture (`testdata/gedcom-5.5.1/longsword.ged`)

A deliberately large real-world export (Legacy 10.0, GEDCOM 5.5.1, UTF-8 with BOM):
46.1 MiB, 203,154 individuals, 90,085 families, 293,241 XRefs. Measured on the
environment above:

| Metric | Value |
|---|---|
| Decode time | 854 ms/op |
| Throughput | 56.6 MB/s |
| Cumulative bytes/op | 1.50 GB |
| Allocs/op | 18.1M |
| Peak process RSS (single decode) | ~930 MB (~20x file size, includes GC headroom) |

Throughput *improves* at this scale versus the 1 MB class — per-file fixed costs
amortize and XRefMap lookups remain average O(1), while map storage grows with
the number of cross-references — but a fully-decoded 46 MB document costs
roughly 0.9 GB of peak process memory. For files this size and beyond, weigh the
streaming APIs (next section). Reproduce with:

```bash
go test -bench BenchmarkScaleDecode -run '^$' -benchmem ./decoder/
go test ./decoder -run TestScaleFixture -v
```

### Hot Paths (CPU Profile)

From profiling the decoder with large files:

1. **Parser Operations (48%)**: Line parsing (31%), string splitting (13%)
2. **Garbage Collection (30%)**: Expected with 214K allocations per MB
3. **Record Building (22%)**: Creating gedcom.Record structures

## Parser Performance

### Line Parsing Benchmarks

| Operation | Time/Op | Memory | Allocations |
|-----------|---------|--------|-------------|
| Simple tag (`0 HEAD`) | 143 ns | 96 B | 2 |
| Tag with value | 179 ns | 128 B | 2 |
| Tag with XRef | 139 ns | 112 B | 2 |
| Nested tag | 168 ns | 144 B | 2 |
| Long value (>100 chars) | 459 ns | 448 B | 2 |

**Key Insight**: Consistent 2 allocations per line regardless of complexity

### File Parsing Benchmarks

| Input Size | Time | Memory | Throughput |
|------------|------|--------|------------|
| Minimal (8 lines) | 2.2 µs | 5 KB | - |
| 100 individuals (~1800 lines) | 114 µs | 98 KB | ~15M lines/sec |
| 1000 individuals (~18K lines) | 1.3 ms | 1 MB | ~13M lines/sec |

**Scaling**: Linear with file size

## Encoder Performance

| Document Size | Time/Op | Memory | Allocations |
|---------------|---------|--------|-------------|
| 1 individual | 1.0 µs | 208 B | 13 |
| 10 individuals | 18 µs | 4.1 KB | 259 |
| 100 individuals | 176 µs | 40 KB | 2.5K |
| 1000 individuals | 1.7 ms | 400 KB | 25K |

**Scaling**: Linear with document size (~25 allocs per individual)

Line ending format has negligible impact (<1% overhead for CRLF vs LF).

## Validator Performance

| Document Size | Time/Op | Memory | Allocations |
|---------------|---------|--------|-------------|
| 1 individual | 15 ns | 0 B | 0 |
| 10 individuals | 52 ns | 0 B | 0 |
| 100 individuals | 526 ns | 0 B | 0 |
| 1000 individuals | 6.6 µs | 0 B | 0 |

**Key Insight**: Zero allocations when no errors found!

### Entity Tags Check

`ENTITY_TAGS_DIVERGED` (opt-in via `ValidateOptions.CheckEntityTags`) compares
each record's typed fields with its raw `Tags` by running `encoder.SyncTags`
on a copy, which encodes and decodes the document. On `pres2020.ged`
(2,322 individuals, every record decoded with `Tags`), with duplicate
detection off:

| `ValidateAll` | Time/Op | Memory | Allocations |
|---------------|---------|--------|-------------|
| Without the check (default) | 34 ms | 9.4 MB | 403k |
| With the check | 73 ms | 63 MB | 1.44M |

The extra cost is about one encode-decode round trip of the document, so it
grows with file size. A hand-built record with no `Tags` costs nothing. A
group of more than 512 changed structures of one tag in one record is
rewritten rather than aligned line by line, bounding the alignment's memory.

```bash
go test ./validator -bench BenchmarkValidateAllEntityTags -run '^$' -benchmem -count=3
```

### Duplicate Detection

`DuplicateDetector.FindDuplicates` is the most expensive validator, and
`Validator.ValidateAll` runs it unless `ValidateOptions.SkipDuplicateDetection`
is set. Its cost is driven by the size of each *surname group*, not by the
document size directly: individuals are blocked by normalized surname, and only
pairs within a group are compared.

On the scale fixture (203,154 individuals, 55,801 non-empty surname groups):

| Metric | Value |
|---|---|
| Candidate pairs compared | ~3.1M |
| Time | ~1.1 s |
| Pairs returned | 86,235 |

Blocking is what makes this tractable — the fixture's surnames spread across
tens of thousands of buckets, so no single bucket is large. Two consequences
follow:

- **Individuals with no surname are excluded entirely.** They share no surname
  with each other, so no pair among them could match; comparing them would be
  pure waste. On this fixture that skips ~4,950 individuals and ~12.3M pairs.
- **Cost is quadratic within a bucket, so the bucket is capped.** A document
  where many individuals share one surname would degrade to O(k²) over that
  group. Natural data rarely does this; adversarial input can, because the input
  controls the surnames. `DuplicateConfig.MaxGroupSize` bounds it — see below.

Synthetic duplicate-heavy fixture, measured on the environment above with
`-count=3`:

| Individuals | Time/Op | Memory | Allocations |
|---|---|---|---|
| 100 | 0.23 ms | 1.5 MB | 2.3K |
| 1000 | 8.5 ms | 19 MB | 128K |
| 5000 | 152 ms | 205 MB | 3.0M |

Reproduce with:

```bash
go test ./validator -bench BenchmarkFindDuplicates -run '^$' -benchmem -count=3
```

#### Bounding Untrusted Input

Three knobs control the worst case, so callers validating untrusted uploads do not
have to bound document size at their own layer:

| Knob | Default | Effect |
|---|---|---|
| `DuplicateConfig.MaxGroupSize` | `validator.DefaultMaxGroupSize` (1000) | Any normalized-surname group larger than the cap is skipped whole, never truncated |
| `DuplicateConfig.MaxNameRunes` | `validator.DefaultMaxNameRunes` (64) | A given name longer than the bound is compared on its first 64 runes; the similarity is Levenshtein, quadratic in name length, so this bounds what one pair costs |
| `ValidateOptions.SkipDuplicateDetection` | `false` | Drops duplicate detection from the `ValidateAll` sweep. `QualityReport` and `FindPotentialDuplicates` still run it — they are explicit requests — bounded by the two caps above |

`MaxGroupSize` and `MaxNameRunes` are tri-state, and zero does **not** mean unlimited:

| Value | Meaning |
|---|---|
| `0` | Use the default (`DefaultMaxGroupSize` 1000, `DefaultMaxNameRunes` 64) |
| negative (`validator.UnlimitedGroupSize`, `validator.UnlimitedNameRunes`) | No cap — compare every group however large, or every name at full length |
| positive | Use that value |

A partial struct literal such as `&validator.DuplicateConfig{MinConfidence: 0.7}`
leaves the fields at zero, so it stays bounded rather than silently opting back
into unbounded work. Opting out has to be explicit, and `UnlimitedGroupSize` and
`UnlimitedNameRunes` exist so it reads as a decision rather than a magic `-1`.

The caps turn the worst case from quadratic into linear in document size. With
*n* individuals, the work is bounded by:

```
worst case = n × MaxGroupSize / 2 pair comparisons × MaxNameRunes² cells each
```

At the defaults that is 500 × *n* pairs, whatever the surnames are: a
100,000-individual upload cannot exceed ~50M comparisons. For scale, the fixture
row above compares ~3.1M pairs in ~1.1 s. The name bound matters when the names
are long: two individuals with 100,000-character given names took 51 s before
it (the pair cost is the product of the lengths) and under a millisecond with
it. The fully adversarial shape — 1000 same-surname individuals with distinct
64-rune names — still costs about ten seconds, so a service exposed to hostile
uploads at scale should set both caps lower than the defaults.

Skipping and prefix comparison are reported rather than silent. When any group
exceeds the cap, or any given name is compared on a prefix, the detector emits
**one** aggregate `DUPLICATE_DETECTION_LIMITED` issue at Warning severity, with
details for the number of groups and individuals skipped, the largest group, the
resolved caps, the number of names cut, and a sample of skipped surnames. It
surfaces through `Validator.ValidateAll` and `Validator.QualityReport`, and
through `DuplicateDetector.FindDuplicatesReport` as `DuplicateReport.LimitIssues`.
`FindDuplicates` returns pairs only, so use the report form when you need to know
whether the sweep was complete — an empty result from a capped document means
"unknown", not "none".

The defaults are set from the corpus. The largest normalized-surname group
anywhere in `testdata/` is **519**, in the scale fixture (203,154 individuals,
55,801 groups); the next largest anywhere is 70. A cap of 1000 leaves roughly 2x
headroom over the worst real-world observation, and running the whole corpus
through the detector at the default skips no group. Of 216,186 individuals, 64
have a given name over 64 runes, all titles and epithets stored in `GIVN` in two
files; comparing them on their first 64 runes changes no pair those files report,
so the two files gain the notice and lose no detection.

See [ADR 0009](../decisions/0009-bounded-duplicate-detection.md) for the
alternatives considered and why the cap is per-group rather than global.

## Streaming APIs Performance

The streaming APIs provide memory-efficient alternatives for very large files. Two memory metrics matter:

- **Cumulative allocated bytes** (`TotalAlloc` delta): total bytes ever allocated during the operation. Both streaming and batch do similar parsing/encoding work, so cumulative allocations are roughly similar.
- **Retained heap after return** (`HeapAlloc` delta with `KeepAlive`): bytes still resident when the call completes. **This is the streaming win.** Batch holds the full `Document` afterwards; streaming holds essentially nothing.

The retained-heap metric is what matters when the operation runs inside a longer-lived process and you care about steady-state memory.

### Decoder (Batch) vs Streaming Parse — `pres2020.ged` (1.1MB, 2,322 individuals)

| Metric | `decoder.Decode` (batch) | `parser.Records` (streaming) | Ratio |
|---|---|---|---|
| Time/op | 14.2 ms | 7.9 ms | 0.56x |
| Cumulative bytes/op | 22 MB | 12 MB | 0.55x |
| Allocs/op | 261,258 | 170,263 | 0.65x |
| **Retained heap after** | **9,545 KB** | **1,611 KB** | **0.17x** |

Streaming parse retains ~17% of batch decode's heap after the call returns. Reproduce with:

```bash
go test -bench='DecodeLarge|StreamingParseLarge' -benchmem ./decoder/
go test -v -run TestMemory ./decoder/
```

### Typed Streaming: `decoder.DecodeRecord`

`parser.Records`, `parser.RecordIterator` and `parser.LazyParser` (iteration
and `FindRecord`) are raw: each yields a `*parser.RawRecord` of parsed lines,
with no `Individual`, `Family` or other typed entity. `decoder.DecodeRecord`
turns one raw record into the `*gedcom.Record` `decoder.Decode` gives it, so
typed access no longer needs a whole `Document`:

```go
var ver gedcom.Version
for raw, err := range parser.Records(charset.NewReader(f)) {
    if err != nil {
        return err
    }
    if raw.Type == "HEAD" {
        ver = version.DeclaredVersion(raw.Lines) // the header comes first
    }
    record, err := decoder.DecodeRecord(raw, ver, nil)
    if err != nil {
        return err
    }
    // record.Entity is typed, as in a decoded Document
}
```

Measured on a synthetic GEDCOM 5.5.1 file of 1,000,000 individuals and 500,000
families (1.5 M records, 481 MB) on an 8-core arm64 machine with 16 GB. CPU
time is reported because wall time varied with machine load:

| Pipeline | CPU time | Peak RSS |
|---|---|---|
| `parser.Records` only | 8.0 s | 34 MB |
| `parser.Records` → `DecodeRecord` | 16.4 s | 64 MB |
| `parser.Records` → `DecodeRecord` → `StreamingValidator` | 30.4 s | 897 MB (see [Streaming Validator](#streaming-validator)) |
| `decoder.Decode` (batch), 100,000-individual 47 MB subset | — | 823 MB |

`DecodeRecord` costs about 5.6 µs of CPU per record on that file, and peak
memory stays flat. On `pres2020.ged` it allocates about 3.7 KB in 30
allocations per record (`BenchmarkStreamingDecodeRecordLarge` against
`BenchmarkStreamingParseLarge`). Parse and decode together allocate about what
`Decode` does for the file (23.8 MB against 22.1 MB), but none of it outlives
the record. A `LazyParser.FindRecord` lookup
followed by `DecodeRecord` took about 2 ms for one record of the 1.5 M.

Batch `Decode` peak RSS is about 13–17× the file size: 823 MB for the 47 MB
subset above, and 8.6 GB (73 s) for a 676 MB file of the same shape in an
earlier measurement. Above a few hundred megabytes, typed access is practical
only through `DecodeRecord`.

```bash
go test -bench='DecodeLarge|StreamingParseLarge|StreamingDecodeRecordLarge' -benchmem ./decoder/
```

### Streaming Encoder

The throughput benchmarks below construct records inside the loop for the streaming path (matching how real streaming consumers produce records on the fly) and use a pre-built `Document` for the batch path (because batch *requires* the full Document to exist before it can run):

| Metric (1,000 individuals) | `encoder.Encode` (batch, pre-built doc) | `encoder.StreamEncoder` (records produced in loop) |
|---|---|---|
| Time/op | 1.27 ms | 1.77 ms |
| Bytes/op | 400 KB | 1.67 MB |
| Allocs/op | 25,009 | 41,505 |

Streaming throughput appears worse because the streaming benchmark includes record-construction cost while the batch benchmark excludes it (the cost is paid up front, before `b.ResetTimer`). For a memory comparison where both paths do the same total work, see the retained-heap test:

| Metric (10,000 individuals) | Batch | Streaming | Ratio |
|---|---|---|---|
| Cumulative bytes | 16 MB | 16 MB | 0.99x |
| **Retained heap after** | **14,813 KB** | **2,492 KB** | **0.17x** |

Streaming encode retains ~17% of batch encode's heap because batch keeps the entire `Document` (all 10K records) live throughout, while streaming generates and discards each record as it writes.

```bash
go test -bench='EncodeLarge|StreamEncodeLarge' -benchmem ./encoder/
go test -v -run TestMemory ./encoder/
```

### Streaming Validator

| Document Size | Batch Validator | Stream Validator | Memory (Stream) |
|---------------|-----------------|------------------|-----------------|
| 1K individuals | 6.6 µs / 0 B | 7.2 µs | O(unique XRefs) |
| 10K individuals | 66 µs / 0 B | 72 µs | O(unique XRefs) |
| 100K individuals | 660 µs / 0 B | 720 µs | O(unique XRefs) |

`ValidateRecord` needs each record's typed `Entity`: feed it records from
`decoder.DecodeRecord` (see [Typed Streaming](#typed-streaming-decoderdecoderecord)).
A record built from raw lines alone is registered as declared but nothing in
it is checked, so a pipeline that skips `DecodeRecord` reports no issues.

Stream validator memory scales with unique cross-references (needed for orphan-reference detection at `Finalize()`), not total record count. That is still large for a large file: on the 1.5 M-record file above, where every record is declared and referenced, the validator's cross-reference state reached about 500 MB of heap (897 MB peak RSS), against 64 MB for the parse-and-decode pipeline alone. Note that streaming validation currently covers Individual and Family records fully and collects cross-references from Source records; Repository, Note, Multimedia Object, and Submitter records are not validated in streaming mode — use batch `validator.Validate()` if you need coverage for those types.

### Incremental Parser

| Operation | Time | Memory | Notes |
|-----------|------|--------|-------|
| Build index (1K records) | ~1.5 ms | O(records) | One-time cost |
| Build index (10K records) | ~15 ms | O(records) | One-time cost |
| Lookup by XRef | O(1) | O(1) | After index built |
| Iterate (per record) | ~1.5 µs | O(1) | Constant memory |
| Save/Load index | ~0.5 ms | O(records) | Persist for reuse |

**Key Insight**: Index enables O(1) random access; iteration uses O(1) memory per record.

The index is held in memory whole. On the 1.5 M-record file above it retained
167 MB of heap after building (about 117 B per entry), with peak RSS of 625 MB
while building; an earlier measurement saw 337 MB of heap (about 230 B per
entry) before collection. Saved, it took 52 MB on disk. Records the index finds
are raw: decode them with `decoder.DecodeRecord` for typed access.

### When to Use Streaming APIs

| Scenario | Recommended API |
|----------|-----------------|
| Files < 10 MB | Standard batch APIs (simpler) |
| Files 10-100 MB | Either (based on memory constraints) |
| Files > 100 MB | Streaming APIs recommended |
| Memory-constrained | Always use streaming APIs |
| Random access needed | LazyParser with index |
| Single-pass processing | RecordIterator |

Encoding detection in `charset.NewReader` reads only the header (at most 64 KiB) before streaming starts, and replays those bytes. When the encoding is known in advance, `charset.NewReaderWithEncoding` skips detection.

## Performance Recommendations

### For Large Files (>10 MB)

1. **Use Streaming APIs**: For very large files, use the streaming APIs instead of batch processing:
   ```go
   // Streaming encode - O(1) memory
   enc := encoder.NewStreamEncoder(writer)
   enc.WriteHeader(header)
   for record := range recordChannel {
       enc.WriteRecord(record)
   }
   enc.WriteTrailer()

   // Streaming validate - O(unique XRefs) memory. ValidateRecord needs
   // typed records: decode each raw record first.
   v := validator.NewStreamingValidator(opts)
   for raw, err := range parser.Records(reader) {
       if err != nil {
           return err
       }
       record, err := decoder.DecodeRecord(raw, ver, nil)
       if err != nil {
           return err
       }
       issues := v.ValidateRecord(record)
   }
   finalIssues := v.Finalize()

   // Incremental parse with range-over-func (Go 1.23+) - O(1) memory per record
   for record, err := range parser.Records(reader) {
       if err != nil {
           return err
       }
       // Process one record at a time
   }

   // Or with classic iterator API (pre-Go 1.23)
   it := parser.NewRecordIterator(reader)
   for it.Next() {
       record := it.Record()
       // Process one record at a time
   }
   if err := it.Err(); err != nil {
       return err
   }
   ```

2. **For Random Access**: Use LazyParser with a persisted index:
   ```go
   lp := parser.NewLazyParser(file)
   lp.LoadIndex(indexFile)  // Skip O(n) scan if index exists
   raw, _ := lp.FindRecord("@I12345@")  // O(1) lookup, raw lines
   record, _ := decoder.DecodeRecord(raw, ver, nil)  // typed
   ```

3. **Memory Budget**:
   - Batch APIs: budget ~13-20x file size peak RSS (measured: 46MB file -> ~930MB peak; 676MB -> 8.6GB)
   - Streaming APIs: O(1) to O(unique XRefs) regardless of file size

4. **Processing Time**: Budget ~30ms per MB (100MB file -> ~3 seconds)

### For Batch Processing

1. **Reuse Validators**: Create once, use multiple times:
   ```go
   v := validator.New()
   for _, doc := range documents {
       errs := v.Validate(doc)
   }
   ```

2. **Consider Concurrent Processing** for multiple files using goroutines with semaphore limiting.

### For Web Services

1. **Set Timeouts**: Use `context` for long-running operations
2. **Limit File Sizes**: Prevent DoS attacks with `io.LimitReader`
3. **Memory Profiling**: Monitor for memory leaks
4. **Bound Duplicate Detection**: It is capped by default
   (`DuplicateConfig.MaxGroupSize` = 1000), so validation stays linear in
   document size. Lower the cap, or set
   `ValidateOptions.SkipDuplicateDetection` to drop the costliest validator from
   `ValidateAll` — see [Bounding Untrusted Input](#bounding-untrusted-input)
5. **Per-Line Ceiling**: Every parse path rejects a single line longer than
   `parser.MaxLineBytes` (1 MiB), so one line cannot force an unbounded buffer
   — see [Line Length Limit](decoding.md#line-length-limit)

## Running Benchmarks

```bash
# All packages
make bench

# Or manually
go test -bench=. -benchmem ./...

# With profiling
go test -bench=BenchmarkDecodeLarge -cpuprofile=cpu.prof -memprofile=mem.prof ./decoder
go tool pprof -http=:8080 cpu.prof
```

## Regression Testing

```bash
# Save baseline
make bench-save

# Compare after changes
make bench-compare

# Run regression tests
make perf-regression
```

## Troubleshooting

### Slow Parsing
- **Symptoms**: ParseLine > 200ns/op, allocations > 500B/op
- **Diagnosis**: `go test -bench=BenchmarkParseLine -cpuprofile=cpu.prof ./parser`
- **Common causes**: String allocation in hot path, regex compilation per line

### High Memory Usage
- **Symptoms**: Decode using > 20MB for 1000 individuals
- **Diagnosis**: `go test -bench=BenchmarkDecodeLarge -memprofile=mem.prof ./decoder`
- **Common causes**: Duplicate string storage, unnecessary slice growth

## Future Optimization Opportunities

If performance becomes critical:

1. **Parser Optimization**: Replace `strings.Fields` with custom parser (-20% allocations)
2. **Object Pooling**: Pool Record and Tag objects for batch processing
3. **Profile-Guided Optimization (PGO)**: Collect profiles, let Go compiler optimize hot paths (10-20% improvement)

## Library Design Priorities

1. **Correctness**: Full GEDCOM 5.5, 5.5.1, and 7.0 support
2. **Safety**: Comprehensive validation and error reporting
3. **Simplicity**: minimal dependencies (only `golang.org/x/text`), clean API
4. **Reasonable Performance**: 32 MB/s is fast enough for most use cases

## Questions?

- **GitHub Issues**: https://github.com/cacack/gedcom-go/issues
- **Discussions**: https://github.com/cacack/gedcom-go/discussions
