# Performance Characteristics

Measured benchmark figures for gedcom-go, and guidance on choosing between the
batch and streaming APIs.

## Benchmark Environment

Every benchmark table in this guide comes from **one run** of the suite:

- **Machine**: Apple M2, 8 cores, 16 GB (darwin/arm64)
- **Go**: 1.27.4
- **Date**: 2026-11-24
- **Command**: `go test -run '^$' -bench . -benchmem -count=6 ./parser ./decoder ./encoder ./validator`

Each figure is the median of the six samples, and each row names the benchmark
it came from, so `go test -bench '<name>' -benchmem ./<package>` reproduces it.
Samples stayed within ±4% of the median, except `BenchmarkFindDuplicates/1000`
(±7%). Sizes are decimal (1 MB = 10⁶ bytes). B/op is the cumulative bytes
allocated per operation, not the memory held at any one moment.

This guide is the single source of truth for gedcom-go benchmark figures. They
are specific to the environment above; run `make bench` to measure your own
hardware. The figures under [Large-File Measurements](#large-file-measurements)
are the exception: they are one-off measurements on files too large for the
suite, and say so.

## Decoder Performance

`decoder.Decode` of a whole file, and the streaming paths over the largest
ordinary fixture. Throughput is file size over time.

| Benchmark | Input | Time/op | B/op | Allocs/op | Throughput |
|-----------|-------|---------|------|-----------|------------|
| `BenchmarkDecodeMinimal` | `minimal.ged`, 170 B, 1 individual | 11.0 µs | 89.8 KB | 118 | 15 MB/s |
| `BenchmarkDecodeSmall` | `maximal70.ged`, 14.9 KB, 4 individuals | 193 µs | 328 KB | 4,204 | 77 MB/s |
| `BenchmarkDecodeMedium` | `royal92.ged`, 469 KB, 3,010 individuals | 14.6 ms | 23.8 MB | 189,859 | 32 MB/s |
| `BenchmarkDecodeLarge` | `pres2020.ged`, 1.1 MB, 2,322 individuals | 14.8 ms | 23.8 MB | 277,934 | 75 MB/s |
| `BenchmarkScaleDecode` | `longsword.ged`, 48.3 MB, 203,154 individuals | 999 ms | 1.69 GB | 18.5M | 48 MB/s |
| `BenchmarkDecodeCitationHeavy` | synthetic, citation-heavy, 2.5 MB | 35.5 ms | 61.0 MB | 735,780 | 70 MB/s |
| `BenchmarkStreamingParseLarge` | `pres2020.ged` through `parser.Records` | 7.91 ms | 9.77 MB | 157,513 | 140 MB/s |
| `BenchmarkStreamingDecodeRecordLarge` | `pres2020.ged` through `parser.Records` and `decoder.DecodeRecord` | 13.6 ms | 25.5 MB | 280,977 | 81 MB/s |

Throughput depends on the file's shape more than its size: `royal92.ged` is
less than half the size of `pres2020.ged` and takes as long to decode. Budget
13–31 ms per MB of input.

Cumulative allocation is 21–51× the file size on the four larger real files. That is
churn, not residency: [Batch vs Streaming](#decoder-batch-vs-streaming-parse)
measures what stays on the heap, and
[Decoding Untrusted Input](untrusted-input.md#unbounded-dimensions) measures
peak memory.

### Scale Fixture (`testdata/gedcom-5.5.1/longsword.ged`)

A deliberately large real-world export (Legacy 10.0, GEDCOM 5.5.1, UTF-8 with
BOM): 48.3 MB (46.1 MiB), 203,154 individuals, 90,085 families, 293,241 XRefs.
`BenchmarkScaleDecode` in the table above is its decode;
`BenchmarkValidateAllStrictLongsword` under
[Duplicate Detection](#duplicate-detection) is its validation. Reproduce with:

```bash
go test -bench BenchmarkScaleDecode -run '^$' -benchmem ./decoder/
go test ./decoder -run TestScaleFixture -v
```

## Parser Performance

### Line Parsing

| Benchmark | Line | Time/op | B/op | Allocs/op |
|-----------|------|---------|------|-----------|
| `BenchmarkParseLine/simple_tag` | `0 HEAD` | 44 ns | 96 B | 2 |
| `BenchmarkParseLine/tag_with_value` | `1 NAME John /Doe/` | 72 ns | 128 B | 2 |
| `BenchmarkParseLine/tag_with_xref` | `0 @I1@ INDI` | 49 ns | 112 B | 2 |
| `BenchmarkParseLine/nested_tag` | `2 DATE 1 JAN 1980` | 76 ns | 144 B | 2 |
| `BenchmarkParseLine/long_value` | a 116-character `NOTE` line | 241 ns | 448 B | 2 |

Every line costs two allocations, whatever its shape.
`BenchmarkParseLineTypes` covers further line shapes.

### File Parsing

The synthetic input is a header, six lines per individual and a trailer.
"Records" below are individuals.

| Benchmark | Input | Time/op | B/op | Allocs/op |
|-----------|-------|---------|------|-----------|
| `BenchmarkParse/minimal_file` | 8 lines | 1.09 µs | 5.30 KB | 29 |
| `BenchmarkParse/file_with_100_individuals` | 605 lines | 55.0 µs | 98.1 KB | 1,826 |
| `BenchmarkParse/file_with_1000_individuals` | 6,005 lines | 564 µs | 1.00 MB | 18,032 |
| `BenchmarkRecordIterator/10_records` | 65 lines | 7.16 µs | 20.2 KB | 229 |
| `BenchmarkRecordIterator/100_records` | 605 lines | 59.6 µs | 115 KB | 2,029 |
| `BenchmarkRecordIterator/1000_records` | 6,005 lines | 581 µs | 1.06 MB | 20,029 |
| `BenchmarkRecordIteratorWithOffset/10_records` | 65 lines | 12.3 µs | 34.6 KB | 357 |
| `BenchmarkRecordIteratorWithOffset/100_records` | 605 lines | 111 µs | 280 KB | 3,237 |
| `BenchmarkRecordIteratorWithOffset/1000_records` | 6,005 lines | 1.10 ms | 2.74 MB | 32,037 |

Parsing is linear in the number of lines, about 10.7 million lines per second
at 1,000 individuals. Iterating record by record costs about what `Parse`
does; tracking byte offsets, which indexing needs, roughly doubles it.
`BenchmarkRecordsRangeFunc` and `BenchmarkRecordsWithOffsetRangeFunc` measure
the range-over-func forms (`parser.Records`, `parser.RecordsWithOffset`) on the
same inputs and match the iterator rows within noise.

## Encoder Performance

| Benchmark | Input | Time/op | B/op | Allocs/op |
|-----------|-------|---------|------|-----------|
| `BenchmarkEncodeMinimal` | 1 individual | 934 ns | 744 B | 26 |
| `BenchmarkEncodeSmall` | 10 individuals | 10.8 µs | 4.79 KB | 275 |
| `BenchmarkEncodeMedium` | 100 individuals | 101 µs | 40.8 KB | 2,525 |
| `BenchmarkEncodeLarge` | 1,000 individuals | 1.02 ms | 401 KB | 25,026 |
| `BenchmarkEncodeLineEndings/LF_(Unix)` | 100 individuals | 100 µs | 40.8 KB | 2,525 |
| `BenchmarkEncodeLineEndings/CRLF_(Windows)` | 100 individuals | 101 µs | 40.8 KB | 2,525 |
| `BenchmarkStreamEncodeLarge` | 1,000 individuals, each built inside the loop | 1.41 ms | 2.06 MB | 41,519 |
| `BenchmarkStreamEncoder_LargeFile` | one prebuilt 7-line record written 1,000 times | 944 µs | 631 KB | 23,031 |

Encoding is linear in document size, about 25 allocations per individual. The
line ending makes no measurable difference.

`BenchmarkStreamEncodeLarge` is not a like-for-like comparison with
`BenchmarkEncodeLarge`: it builds each record inside the timed loop, as a
streaming producer would, while the batch benchmark is handed a finished
`Document`. The streaming encoder's advantage is the memory it does not hold;
see [Streaming Encoder](#streaming-encoder).

## Validator Performance

`Validator.Validate` (and `ValidateAll`, which it wraps) at the default
options, duplicate detection included, on a synthetic document that reports no
issues: unique XRefs, a submitter, and surname groups of four.
`TestValidFixtureShape` pins that shape.

| Benchmark | Input | Time/op | B/op | Allocs/op |
|-----------|-------|---------|------|-----------|
| `BenchmarkValidateMinimal` | 1 individual | 11.4 µs | 6.34 KB | 49 |
| `BenchmarkValidateSmall` | 10 individuals | 20.2 µs | 10.3 KB | 111 |
| `BenchmarkValidateMedium` | 100 individuals | 102 µs | 47.3 KB | 626 |
| `BenchmarkValidateLarge` | 1,000 individuals | 927 µs | 495 KB | 5,603 |
| `BenchmarkValidateWithErrors` | 100 individuals and 10 families with broken references | 127 µs | 84.5 KB | 883 |
| `BenchmarkStreamingValidate/100` | 100 individuals, record by record | 19.6 µs | 54.4 KB | 121 |
| `BenchmarkStreamingValidate/1000` | 1,000 individuals, record by record | 134 µs | 107 KB | 1,031 |

Validation of a clean document is linear in its size: 1,000 individuals cost
9.1× what 100 do, about 0.9 µs and 5.6 allocations per individual. The
validator allocates on every document, issues or not.

The `BenchmarkStreamingValidate` rows are a whole `StreamingValidator` pass
(`ValidateRecord` for each record, then `Finalize`) over the same documents.
It is cheaper because it runs fewer checks, with no duplicate detection among
them, so the two are not interchangeable; see
[Streaming Validator](#streaming-validator).

A real file costs more per individual than the synthetic one, because its
records carry events, dates and references to check: the 2,322 individuals of
`pres2020.ged` take 6.1 ms with duplicate detection off (next table).

### Opt-in Checks

`ValidateAll` on `pres2020.ged` (2,322 individuals, decoded, so every record
has `Tags`), with duplicate detection off so that each pair of rows differs by
the check alone. The two `without-check` rows are the same configuration
measured by two benchmarks.

| Benchmark | `ValidateAll` | Time/op | B/op | Allocs/op |
|-----------|---------------|---------|------|-----------|
| `BenchmarkValidateAllEntityTags/without-check` | default | 6.14 ms | 2.03 MB | 27,329 |
| `BenchmarkValidateAllEntityTags/with-check` | `CheckEntityTags` | 47.9 ms | 59.8 MB | 1,138,079 |
| `BenchmarkValidateAllGrammar/without-check` | default | 6.10 ms | 2.03 MB | 27,329 |
| `BenchmarkValidateAllGrammar/with-check` | `CheckGrammar` | 17.7 ms | 4.24 MB | 104,523 |

```bash
go test ./validator -bench 'BenchmarkValidateAll(EntityTags|Grammar)' -run '^$' -benchmem -count=6
```

#### Entity Tags Check

`ENTITY_TAGS_DIVERGED` (opt-in via `ValidateOptions.CheckEntityTags`) compares
each record's typed fields with its raw `Tags` by running `encoder.SyncTags`
on a copy, then reading the tags back through the decoder. On this file it
makes `ValidateAll` 7.8× slower, adding 42 ms and 58 MB: between two and
three times what decoding the file costs (`BenchmarkDecodeLarge`). The cost grows with
file size. A hand-built record with no `Tags` costs nothing. A group of more
than 512 changed structures of one tag in one record is rewritten rather than
aligned line by line, bounding the alignment's memory.

#### Grammar Check

The grammar conformance check (opt-in via `ValidateOptions.CheckGrammar`)
walks every line of the header and records once against the version's
structure tables, so its cost is linear in the number of lines. On this file
it makes `ValidateAll` 2.9× slower, adding 12 ms and 2.2 MB.

### Duplicate Detection

`DuplicateDetector.FindDuplicates` is the most expensive validator, and
`Validator.ValidateAll` runs it unless `ValidateOptions.SkipDuplicateDetection`
is set. Its cost is driven by the size of each *surname group*, not by the
document size directly: individuals are blocked by normalized surname, and only
pairs within a group are compared.

Blocking is what makes this tractable. The scale fixture's 203,154 individuals
fall into 55,801 surname groups, the largest of 519, so detection compares
about 3.06 million candidate pairs rather than 20 billion, and returns 55,259
(`TestDuplicateCapPreservesCorpusResults` logs the groups and the pairs
returned). Two consequences follow:

- **Individuals with no surname are excluded entirely.** They share no surname
  with each other, so no pair among them could match; comparing them would be
  pure waste.
- **Cost is quadratic within a group, so the group is capped.** A document
  where many individuals share one surname would degrade to O(k²) over that
  group. Natural data rarely does this; adversarial input can, because the input
  controls the surnames. `DuplicateConfig.MaxGroupSize` bounds it — see below.

| Benchmark | Input | Time/op | B/op | Allocs/op |
|-----------|-------|---------|------|-----------|
| `BenchmarkFindDuplicates/100` | 100 individuals, 5 surname groups of 16 | 118 µs | 273 KB | 1,452 |
| `BenchmarkFindDuplicates/1000` | 1,000 individuals, 5 groups of 160 | 6.44 ms | 7.20 MB | 120,025 |
| `BenchmarkFindDuplicates/5000` | 5,000 individuals, 5 groups of 800 | 143 ms | 145 MB | 2,956,142 |
| `BenchmarkFindDuplicatesAdversarial/capped` | 1,001 individuals in one group, default cap: skipped whole | 68.9 µs | 62.3 KB | 1,046 |
| `BenchmarkFindDuplicatesAdversarial/uncapped` | the same group, `UnlimitedGroupSize`: 500,500 pairs | 899 ms | 297 MB | 3,506,037 |
| `BenchmarkValidateAllStrictLongsword` | Strict `ValidateAll` of the scale fixture, detection included | 1.54 s | 1.02 GB | 10.8M |

The `FindDuplicates` rows grow with the square of the group size, not with the
document. The adversarial fixture is the worst case per pair: 26-rune given
names that are all near misses, so every pair runs the full Levenshtein
comparison and then loses. It costs 1.8 µs and about 590 B per pair, and one
group just under the default cap costs nearly what the `uncapped` row shows.
The allocation is transient; none of it outlives the pair.

Names are normalized once per individual (lowercased, diacritics removed).
All-ASCII names skip the Unicode transform, which is the common case.

Reproduce with:

```bash
go test ./validator -bench BenchmarkFindDuplicates -run '^$' -benchmem -count=6
go test ./validator -bench BenchmarkValidateAllStrictLongsword -run '^$' -benchmem -count=6
```

`FindDuplicatesBetween` compares two documents using the same surname blocking,
but only across them: a surname with *a* individuals in one document and *b* in
the other costs *a·b* comparisons, and a surname found in only one document costs
nothing beyond grouping. Since *a·b ≤ (a² + b²)/2*, that is about what two
single-document runs spend on the shared surnames, and never more than one run on
the documents combined. `MaxGroupSize` applies to *a + b*, so each shared surname
costs at most `MaxGroupSize²/4` comparisons.

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
100,000-individual upload cannot exceed ~50M comparisons. For scale,
`BenchmarkFindDuplicatesAdversarial/uncapped` above spends 0.9 s on 500,500
pairs of 26-rune names. The name bound matters when the names are long, because
a pair's cost is the product of the two lengths:
[ADR 0009](../decisions/0009-bounded-duplicate-detection.md) records the
measurements behind it, and the worst case that remains at the defaults —
seconds for each 1,000 same-surname individuals with distinct 64-rune names. A
service exposed to hostile uploads at scale should set both caps lower than
the defaults.

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

- **Cumulative allocated bytes** (B/op, or a `TotalAlloc` delta): total bytes ever allocated during the operation.
- **Retained heap after return** (`HeapAlloc` delta with `KeepAlive`): bytes still on the heap when the call completes. **This is the streaming win.** Batch holds the full `Document` afterwards; streaming holds essentially nothing.

The retained-heap metric is what matters when the operation runs inside a longer-lived process and you care about steady-state memory. No benchmark reports it, so the figures come from two tests that log it, `TestMemoryBatchVsStreaming` and `TestMemoryBatchVsStreamingEncode`, each run six times in the session described under [Benchmark Environment](#benchmark-environment). The heap is read without a collection in between, so the streaming figures include garbage not yet collected and vary from run to run.

### Decoder (Batch) vs Streaming Parse

On `pres2020.ged` (1.1 MB, 2,322 individuals), streaming parse takes 0.53× the
time of batch decode and allocates 0.41× the bytes: the
`BenchmarkStreamingParseLarge` and `BenchmarkDecodeLarge` rows under
[Decoder Performance](#decoder-performance). What it retains is smaller still:

| `TestMemoryBatchVsStreaming` | `decoder.Decode` (batch) | `parser.Records` (streaming) |
|---|---|---|
| Retained heap after return, median of 6 | 17.2 MB | 1.1 MB |
| Range over the 6 runs | 16.8–17.5 MB | 0.9–1.2 MB |

Streaming parse retains about 6% of what batch decode does. Reproduce with:

```bash
go test -bench='DecodeLarge|StreamingParseLarge' -run '^$' -benchmem ./decoder/
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

On `pres2020.ged` (3,844 records), `DecodeRecord` adds about 1.5 µs, 4.1 KB
and 32 allocations per record to the raw parse
(`BenchmarkStreamingDecodeRecordLarge` against `BenchmarkStreamingParseLarge`).
Parse and decode together cost about what `Decode` does for the file (13.6 ms
and 25.5 MB against 14.8 ms and 23.8 MB), but none of it outlives the record.
[Large-File Measurements](#large-file-measurements) shows the memory staying
flat on a 1.5 M-record file.

```bash
go test -bench='DecodeLarge|StreamingParseLarge|StreamingDecodeRecordLarge' -run '^$' -benchmem ./decoder/
```

### Streaming Encoder

`BenchmarkStreamEncodeLarge` and `BenchmarkEncodeLarge` under
[Encoder Performance](#encoder-performance) give the throughput, and why the
two are not like for like. For a memory comparison where both paths do the
same total work, encoding 10,000 generated individuals:

| `TestMemoryBatchVsStreamingEncode` | `encoder.Encode` (batch) | `encoder.StreamEncoder` |
|---|---|---|
| Cumulative bytes allocated | 20.6 MB | 20.6 MB |
| Retained heap after return, median of 6 | 20.5 MB | 0.4 MB |
| Range over the 6 runs | 20.5–20.6 MB | 0.4–3.4 MB |

Batch keeps the entire `Document` (all 10,000 records) live throughout, while
streaming generates and discards each record as it writes, retaining about 2%
as much (16% in the one run where a collection had not yet happened).

```bash
go test -bench='EncodeLarge|StreamEncodeLarge' -run '^$' -benchmem ./encoder/
go test -v -run TestMemory ./encoder/
```

### Streaming Validator

`BenchmarkStreamingValidate` under
[Validator Performance](#validator-performance) gives the time and allocation
of a streaming pass.

`ValidateRecord` needs each record's typed `Entity`: feed it records from
`decoder.DecodeRecord` (see [Typed Streaming](#typed-streaming-decoderdecoderecord)).
A record built from raw lines alone is registered as declared but nothing in
it is checked, so a pipeline that skips `DecodeRecord` reports no issues.

Stream validator memory scales with the number of declared XRefs, plus the references to XRefs not yet declared (needed for orphan-reference detection at `Finalize()`), not with total record count. Each distinct XRef is stored once, in a few dozen bytes; a reference to a record already seen is checked at once and not kept; and a forward reference is kept as three integers until its record arrives. A file that writes all individuals before all families therefore holds every individual-to-family reference until the families come. [Large-File Measurements](#large-file-measurements) has the figures for a 1.5 M-record file.

Streaming validation currently covers Individual and Family records fully and collects cross-references from Source records; Repository, Note, Multimedia Object, and Submitter records are not validated in streaming mode — use batch `validator.Validate()` if you need coverage for those types.

### Incremental Parser

`parser.LazyParser` builds an index of record offsets in one pass and then
finds a record by XRef in O(1); iteration holds one record at a time. The
`BenchmarkRecordIteratorWithOffset` rows under
[File Parsing](#file-parsing) measure offset-tracking iteration, the pass an
index is built from. The index is held in memory whole and can be saved and
loaded to skip the scan; [Large-File Measurements](#large-file-measurements)
has its size on a 1.5 M-record file. Records the index finds are raw: decode
them with `decoder.DecodeRecord` for typed access.

### Large-File Measurements

These figures are **not** from the benchmark run above. They are one-off
measurements, taken while the features were built, on synthetic GEDCOM 5.5.1
files too large to keep in the repository, on an 8-core arm64 machine with
16 GB. They have not been re-measured since; treat them as the order of
magnitude.

A file of 1,000,000 individuals and 500,000 families (1.5 M records, 481 MB).
CPU time is reported because wall time varied with machine load:

- `parser.Records` alone: 8.0 s CPU, 34 MB peak RSS.
- `parser.Records` → `DecodeRecord`: 16.4 s CPU, 64 MB peak RSS, about 5.6 µs
  of CPU per record for `DecodeRecord`. That is CPU time summed over all
  cores, garbage collection included, on a different file, so it is not
  comparable with the 1.5 µs of wall-clock time per record under
  [Typed Streaming](#typed-streaming-decoderdecoderecord).
- `LazyParser.FindRecord` followed by `DecodeRecord`: about 2 ms for one
  record of the 1.5 M.
- The `LazyParser` index: 167 MB of heap retained after building (about 117 B
  per entry), 625 MB peak RSS while building, 52 MB saved to disk.
- `decoder.Decode` (batch) of a 100,000-individual, 47 MB subset: 823 MB peak
  RSS. Batch peak RSS was about 13–17× the file size, up to 8.6 GB (73 s) for
  a 676 MB file of the same shape; above a few hundred megabytes, typed access
  is practical only through `DecodeRecord`.
- `decoder.Decode` of the scale fixture (48.3 MB, a real export): about
  930 MB peak RSS for a single decode.

`StreamingValidator` on two files of the same record counts, every record
declared and referenced, all individuals first. The first (459 MB) gives each
individual one FAMS and each family a HUSB and a WIFE; the second (345 MB)
adds a FAMC to nearly every individual and two CHIL to half the families,
doubling the forward references. Peak RSS varied by about ±10 MB between runs:

- `parser.Records` → `DecodeRecord`, no validator, first file: 23 MB peak RSS.
- With `StreamingValidator`, first file: 136 MB peak RSS, 53 MB of heap before
  `Finalize` (829 MB and 413 MB before #778).
- With `StreamingValidator`, second file: 159 MB peak RSS, 64 MB of heap
  before `Finalize` (1,251 MB and 546 MB before #778).

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

   // Streaming validate - O(declared XRefs) memory. ValidateRecord needs
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
   - Batch APIs: memory grows with the input; size the limit from
     [Decoding Untrusted Input](untrusted-input.md#sizing-iolimitreader)
   - Streaming APIs: O(1) to O(declared XRefs) regardless of file size

4. **Processing Time**: Budget 13–31 ms per MB for `Decode`
   ([Decoder Performance](#decoder-performance))

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

1. **Set Timeouts**: Give every stage the same deadline, not just the decode.
   `DecodeOptions`, `ValidateOptions`, `ConvertOptions`, `EncodeOptions`,
   `merge.CombineOptions` and `merge.RecordMergeOptions` each take a `Context`, and `LazyParser` has
   `BuildIndexContext`; each stops shortly after the context is done and
   returns its error wrapped, so `errors.Is(err, context.DeadlineExceeded)`
   works. On the validator, call `ValidateAllWithError`: `ValidateAll` has no
   error to report the cancellation with
2. **Limit File Sizes**: Memory grows with input, up to about 35× for
   `Decode`; cap uploads with `io.LimitReader` sized per API as in
   [Decoding Untrusted Input](untrusted-input.md#sizing-iolimitreader)
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
go test -run '^$' -bench=. -benchmem ./...

# With profiling
go test -run '^$' -bench=BenchmarkDecodeLarge -cpuprofile=cpu.prof -memprofile=mem.prof ./decoder
go tool pprof -http=:8080 cpu.prof
```

## Comparing Against a Baseline

Nothing gates performance automatically: no CI job runs the benchmarks, and
the tables above go stale unless someone re-runs them. To check a change,
compare it with a baseline on the same machine, using
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```bash
# Baseline from the latest release tag (benchmarks an export of the tag in a
# temporary directory), or `make bench-save` on a checkout of your choosing
make bench-baseline

# Benchmark the working tree and print benchstat's comparison
make bench-compare
```

Both write to gitignored files (`perf-baseline.txt`, `perf-current.txt`), and
each takes about ten minutes. benchstat marks a difference only when it is
statistically significant; read its `vs base` column rather than a fixed
threshold. A benchmark whose fixture changed since the baseline, or that the
baseline does not have, is not a comparison.

## Questions?

- **GitHub Issues**: https://github.com/cacack/gedcom-go/issues
- **Discussions**: https://github.com/cacack/gedcom-go/discussions
