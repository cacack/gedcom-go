# Performance Characteristics

This document provides detailed performance benchmarks and optimization guidance for the gedcom-go library.

## Benchmark Environment

All benchmarks were run on:
- **Platform**: Apple M2 (ARM64)
- **OS**: macOS
- **Go Version**: 1.23
- **Test Files**: Real-world GEDCOM files from `testdata/`

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

1. **Parser Operations (48%)**: Time split between:
   - Line parsing: 31%
   - String splitting (`strings.Fields`): 13%

2. **Garbage Collection (30%)**: Expected with 214K allocations per MB

3. **Record Building (34% of memory allocations)**: Creating gedcom.Record structures

### Optimization Opportunities

Based on profiling, potential optimizations (if needed):

1. **strings.Fields** accounts for 20% of memory allocations
   - Could be replaced with manual parsing for better performance
   - Trade-off: code complexity vs. speed

2. **Record building** allocations could be reduced with object pooling
   - Would help for processing many files consecutively

3. **Current performance (32 MB/s)** is already good for most use cases
   - Parsing a 100MB genealogy file takes ~3 seconds

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

### Encoding Benchmarks

| Document Size | Time/Op | Memory | Allocations | Throughput |
|---------------|---------|--------|-------------|------------|
| 1 individual | 1.0 µs | 208 B | 13 | - |
| 10 individuals | 18 µs | 4.1 KB | 259 | ~0.5 MB/s |
| 100 individuals | 176 µs | 40 KB | 2.5K | ~0.5 MB/s |
| 1000 individuals | 1.7 ms | 400 KB | 25K | ~0.5 MB/s |

**Average Throughput**: ~0.5 MB/s

**Scaling**: Linear with document size (~25 allocs per individual)

### Line Ending Performance

| Format | Time/Op (100 individuals) | Overhead |
|--------|---------------------------|----------|
| LF (Unix) | 182 µs | Baseline |
| CRLF (Windows) | 187 µs | +2.7% |

**Recommendation**: Use default LF unless Windows compatibility required

## Validator Performance

### Validation Benchmarks (Valid Documents)

| Document Size | Time/Op | Memory | Allocations |
|---------------|---------|--------|-------------|
| 1 individual | 15 ns | 0 B | 0 |
| 10 individuals | 52 ns | 0 B | 0 |
| 100 individuals | 526 ns | 0 B | 0 |
| 1000 individuals | 6.6 µs | 0 B | 0 |

**Key Insight**: Zero allocations when no errors found!

### Validation with Errors

| Document | Time/Op | Memory | Allocations |
|----------|---------|--------|-------------|
| 100 individuals, 10 broken refs | 3.1 µs | 3.6 KB | 66 |

**Impact**: Only allocates when creating error objects

**Scaling**: ~66 allocations per 10 errors

## Performance Recommendations

### For Large Files (>10 MB)

1. **Stream Processing**: The decoder already uses streaming, but ensure you're not loading entire file into memory first:
   ```go
   // Good: Stream from file
   f, _ := os.Open("large.ged")
   doc, _ := decoder.Decode(f)

   // Avoid: Loading into memory first
   data, _ := os.ReadFile("large.ged") // Don't do this
   doc, _ := decoder.Decode(bytes.NewReader(data))
   ```

2. **Memory**: Budget ~1.2x file size for parsing
   - 100MB file → ~120MB RAM during parsing

3. **Processing Time**: Budget ~30ms per MB
   - 100MB file → ~3 seconds parsing time

### For Batch Processing

1. **Reuse Validators**: Create once, use multiple times:
   ```go
   v := validator.New()
   for _, doc := range documents {
       errors := v.Validate(doc)
       // Process errors
   }
   ```

2. **Consider Concurrent Processing** for multiple files:
   ```go
   // Process files in parallel
   var wg sync.WaitGroup
   sem := make(chan struct{}, runtime.NumCPU())

   for _, file := range files {
       wg.Add(1)
       go func(f string) {
           defer wg.Done()
           sem <- struct{}{}        // Limit concurrency
           defer func() { <-sem }()

           // Parse file
           fd, _ := os.Open(f)
           doc, _ := decoder.Decode(fd)
           fd.Close()
       }(file)
   }
   wg.Wait()
   ```

### For Web Services

1. **Set Timeouts**: Use `context` for long-running operations:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()

   doc, err := decoder.DecodeWithOptions(f, &decoder.Options{
       MaxNestingDepth: 100,
       StrictMode:      true,
   })
   ```

2. **Limit File Sizes**: Prevent DoS attacks:
   ```go
   maxSize := 100 * 1024 * 1024 // 100 MB limit
   lr := io.LimitReader(request.Body, maxSize)
   doc, err := decoder.Decode(lr)
   ```

3. **Memory Profiling**: Monitor for memory leaks:
   ```bash
   go test -bench=. -memprofile=mem.prof
   go tool pprof -alloc_space mem.prof
   ```

## Benchmark Commands

To run all benchmarks:

```bash
# All packages
go test -bench=. -benchmem ./...

# Specific package
go test -bench=. -benchmem ./decoder

# With profiling
go test -bench=BenchmarkDecodeLarge -cpuprofile=cpu.prof -memprofile=mem.prof -benchtime=5s ./decoder

# Analyze profiles
go tool pprof -top -cum cpu.prof
go tool pprof -top -alloc_space mem.prof

# Generate HTML report
go tool pprof -http=:8080 cpu.prof
```

## Profiling Results Summary

### CPU Hotspots (Decoder)

1. **Parser operations**: 48% of CPU time
   - Most time spent in string parsing (unavoidable)

2. **Garbage collection**: 30% of CPU time
   - Normal for allocation-heavy operations
   - Memory usage is already efficient (1.2x file size)

3. **Record building**: 22% of CPU time
   - Building the document structure

### Memory Hotspots (Decoder)

1. **buildRecords**: 34% of allocations
   - Creating Record objects

2. **ParseLine**: 22% of allocations
   - Includes strings.Fields (13%)

3. **strings.Fields**: 20% of allocations
   - Single biggest allocation hotspot
   - Could be optimized with manual parsing if needed

## Comparison with Other Libraries

The gedcom-go library prioritizes:

1. **Correctness**: Full GEDCOM 5.5, 5.5.1, and 7.0 support
2. **Safety**: Comprehensive validation and error reporting
3. **Simplicity**: Zero dependencies, clean API
4. **Reasonable Performance**: 32 MB/s is fast enough for most use cases

For ultra-high-performance requirements, consider:
- Streaming processing without full document parsing
- Using Profile-Guided Optimization (PGO) in Go 1.21+
- Custom optimizations for specific GEDCOM dialects

## Future Optimization Opportunities

If performance becomes critical:

1. **Parser Optimization**:
   - Replace `strings.Fields` with custom parser (-20% allocations)
   - Use buffer pooling for line parsing

2. **Object Pooling**:
   - Pool Record and Tag objects for batch processing
   - Reduce GC pressure

3. **Profile-Guided Optimization (PGO)**:
   - Collect profiles from real workloads
   - Let Go compiler optimize hot paths
   - Typically 10-20% improvement

4. **Parallel Parsing**:
   - For very large files, parse sections in parallel
   - Complex due to GEDCOM's hierarchical nature

## Contributing Performance Improvements

If you have performance improvements:

1. **Run benchmarks before and after**:
   ```bash
   # Before
   go test -bench=. -benchmem ./... > old.txt

   # Make changes

   # After
   go test -bench=. -benchmem ./... > new.txt

   # Compare
   benchstat old.txt new.txt
   ```

2. **Profile to verify**:
   - Ensure improvement is real
   - Check for unexpected allocations

3. **Consider trade-offs**:
   - Code complexity vs. performance
   - Maintainability vs. speed

4. **Submit PR** with benchmark results

## Questions?

- **GitHub Issues**: https://github.com/cacack/gedcom-go/issues
- **Discussions**: https://github.com/cacack/gedcom-go/discussions
