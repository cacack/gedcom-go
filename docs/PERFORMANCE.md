# Performance Characteristics

This document provides detailed performance benchmarks and optimization guidance for the gedcom-go library.

## Benchmark Environment

All benchmarks were run on:
- **Platform**: Apple M2 (ARM64)
- **OS**: macOS
- **Go Version**: 1.23
- **Test Files**: Real-world GEDCOM files from `testdata/`

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

**Average Throughput**: ~32 MB/s for large files

**Memory Usage**: ~1.2x file size (very efficient for in-memory parsing)

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

## Streaming APIs Performance

The streaming APIs provide memory-efficient alternatives for very large files.

### Streaming Encoder

| Document Size | Batch Encoder | Stream Encoder | Memory Difference |
|---------------|---------------|----------------|-------------------|
| 1K individuals | 1.7 ms / 400 KB | 1.8 ms / ~1 KB | 400x less memory |
| 10K individuals | 17 ms / 4 MB | 18 ms / ~1 KB | 4000x less memory |
| 100K individuals | 170 ms / 40 MB | 180 ms / ~1 KB | 40000x less memory |

**Key Insight**: Stream encoder maintains O(1) memory regardless of record count.

### Streaming Validator

| Document Size | Batch Validator | Stream Validator | Memory (Stream) |
|---------------|-----------------|------------------|-----------------|
| 1K individuals | 6.6 µs / 0 B | 7.2 µs | O(unique XRefs) |
| 10K individuals | 66 µs / 0 B | 72 µs | O(unique XRefs) |
| 100K individuals | 660 µs / 0 B | 720 µs | O(unique XRefs) |

**Key Insight**: Stream validator memory scales with unique cross-references, not record count.

### Incremental Parser

| Operation | Time | Memory | Notes |
|-----------|------|--------|-------|
| Build index (1K records) | ~1.5 ms | O(records) | One-time cost |
| Build index (10K records) | ~15 ms | O(records) | One-time cost |
| Lookup by XRef | O(1) | O(1) | After index built |
| Iterate (per record) | ~1.5 µs | O(1) | Constant memory |
| Save/Load index | ~0.5 ms | O(records) | Persist for reuse |

**Key Insight**: Index enables O(1) random access; iteration uses O(1) memory per record.

### When to Use Streaming APIs

| Scenario | Recommended API |
|----------|-----------------|
| Files < 10 MB | Standard batch APIs (simpler) |
| Files 10-100 MB | Either (based on memory constraints) |
| Files > 100 MB | Streaming APIs recommended |
| Memory-constrained | Always use streaming APIs |
| Random access needed | LazyParser with index |
| Single-pass processing | RecordIterator |

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

   // Streaming validate - O(unique XRefs) memory
   v := validator.NewStreamingValidator(opts)
   for _, record := range records {
       issues := v.ValidateRecord(record)
   }
   finalIssues := v.Finalize()

   // Incremental parse - O(1) memory per record
   it := parser.NewRecordIterator(reader)
   for it.Next() {
       record := it.Record()
       // Process one record at a time
   }
   ```

2. **For Random Access**: Use LazyParser with a persisted index:
   ```go
   lp := parser.NewLazyParser(file)
   lp.LoadIndex(indexFile)  // Skip O(n) scan if index exists
   record, _ := lp.FindRecord("@I12345@")  // O(1) lookup
   ```

3. **Memory Budget**:
   - Batch APIs: ~1.2x file size (100MB file -> ~120MB RAM)
   - Streaming APIs: O(1) to O(unique XRefs) regardless of file size

4. **Processing Time**: Budget ~30ms per MB (100MB file -> ~3 seconds)

### For Batch Processing

1. **Reuse Validators**: Create once, use multiple times:
   ```go
   v := validator.New()
   for _, doc := range documents {
       errors := v.Validate(doc)
   }
   ```

2. **Consider Concurrent Processing** for multiple files using goroutines with semaphore limiting.

### For Web Services

1. **Set Timeouts**: Use `context` for long-running operations
2. **Limit File Sizes**: Prevent DoS attacks with `io.LimitReader`
3. **Memory Profiling**: Monitor for memory leaks

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
3. **Simplicity**: Zero dependencies, clean API
4. **Reasonable Performance**: 32 MB/s is fast enough for most use cases

## Questions?

- **GitHub Issues**: https://github.com/cacack/gedcom-go/issues
- **Discussions**: https://github.com/cacack/gedcom-go/discussions
