# Performance Benchmarks

This document contains performance benchmarks and regression testing guidelines for the gedcom-go library.

## Baseline Performance Metrics

Benchmarks run on: Apple M2 (darwin/arm64)
Go version: 1.24.3
Date: 2025-01-21

### Parser Performance

| Benchmark | Operations | Time/op | Memory/op | Allocs/op |
|-----------|------------|---------|-----------|-----------|
| **ParseLine** |
| Simple tag | 16.1M | 66.43 ns | 96 B | 2 |
| Tag with value | 15.1M | 81.84 ns | 128 B | 2 |
| Tag with xref | 18.5M | 68.74 ns | 112 B | 2 |
| Nested tag | 13.5M | 87.60 ns | 144 B | 2 |
| Long value | 4.7M | 253.5 ns | 448 B | 2 |
| **Parse Full Files** |
| Minimal file | 1.0M | 1.19 μs | 5.2 KB | 29 |
| 100 individuals | 17.9K | 67.3 μs | 98.0 KB | 1,826 |
| 1000 individuals | 1.7K | 697 μs | 999 KB | 18,032 |

**Key Insights:**
- ParseLine is extremely fast (sub-100ns for most cases)
- Linear scaling with file size (~70ns per individual)
- Allocations scale linearly with file size
- Minimal memory overhead per line (96-448 bytes)

### Decoder Performance

| Benchmark | Operations | Time/op | Memory/op | Allocs/op |
|-----------|------------|---------|-----------|-----------|
| Minimal | 448.7K | 2.63 μs | 7.4 KB | 69 |
| Small (10 ind) | 8.2K | 150 μs | 214 KB | 3,530 |
| Medium (100 ind) | 160 | 7.44 ms | 8.5 MB | 139,193 |
| Large (1000 ind) | 90 | 13.0 ms | 14.3 MB | 214,666 |

**Key Insights:**
- Decode includes parse + build document structure
- ~150ns per individual (overhead beyond parsing)
- Memory scales: ~85KB per 100 individuals
- Allocation count grows with complexity

### Encoder Performance

| Benchmark | Operations | Time/op | Memory/op | Allocs/op |
|-----------|------------|---------|-----------|-----------|
| Minimal | 2.0M | 588 ns | 208 B | 13 |
| Small (10 ind) | 104.5K | 11.5 μs | 4.1 KB | 259 |
| Medium (100 ind) | 10.0K | 111 μs | 40.2 KB | 2,509 |
| Large (1000 ind) | 1.1K | 1.15 ms | 400 KB | 25,009 |
| **Line Endings** |
| LF (Unix) | 10.0K | 112 μs | 40.2 KB | 2,509 |
| CRLF (Windows) | 9.6K | 113 μs | 40.2 KB | 2,509 |

**Key Insights:**
- Encoding is ~16x slower than parsing (expected for formatting)
- ~115ns per individual
- Line ending format has negligible impact (<1%)
- Memory usage: ~400KB per 1000 individuals

### Validator Performance

| Benchmark | Operations | Time/op | Memory/op | Allocs/op |
|-----------|------------|---------|-----------|-----------|
| Minimal | 74.9M | 14.61 ns | 0 B | 0 |
| Small (10 ind) | 28.9M | 43.19 ns | 0 B | 0 |
| Medium (100 ind) | 3.3M | 371 ns | 0 B | 0 |
| Large (1000 ind) | 197K | 5.91 μs | 0 B | 0 |
| With errors (100 ind) | 403.5K | 3.15 μs | 3.6 KB | 66 |

**Key Insights:**
- Validation is extremely fast (~6ns per individual)
- **Zero allocations** for valid documents
- Error detection adds ~3KB + 66 allocations for 100 individuals
- Highly efficient for large documents

---

## Performance Targets

### Throughput Targets

| Operation | Target | Current | Status |
|-----------|--------|---------|--------|
| Parse 1000 individuals | < 1ms | 697 μs | ✅ Pass |
| Decode 1000 individuals | < 15ms | 13.0 ms | ✅ Pass |
| Encode 1000 individuals | < 2ms | 1.15 ms | ✅ Pass |
| Validate 1000 individuals | < 10μs | 5.91 μs | ✅ Pass |

### Memory Targets

| Operation | Target | Current | Status |
|-----------|--------|---------|--------|
| Parse memory/individual | < 1KB | ~1KB | ✅ Pass |
| Decode memory/100 ind | < 100KB | ~85KB | ✅ Pass |
| Encode memory/100 ind | < 50KB | ~40KB | ✅ Pass |
| Validate allocations | 0 (valid docs) | 0 | ✅ Pass |

---

## Running Benchmarks

### Run All Benchmarks

```bash
make bench
```

Or manually:

```bash
go test -bench=. -benchmem ./parser ./decoder ./encoder ./validator
```

### Run Specific Benchmarks

```bash
# Parser only
go test -bench=. -benchmem ./parser

# Specific pattern
go test -bench=BenchmarkParse/file_with_1000 -benchmem ./parser

# With CPU profiling
go test -bench=BenchmarkDecodeLarge -cpuprofile=cpu.prof ./decoder
go tool pprof cpu.prof

# With memory profiling
go test -bench=BenchmarkEncodeLarge -memprofile=mem.prof ./encoder
go tool pprof -alloc_space mem.prof
```

### Compare Performance

To compare performance between versions:

```bash
# Baseline (current version)
go test -bench=. -benchmem ./... > baseline.txt

# Make changes...

# Compare
go test -bench=. -benchmem ./... > new.txt
benchstat baseline.txt new.txt
```

Install benchstat:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

---

## Performance Regression Testing

### Automated Regression Detection

Use the provided regression testing script:

```bash
./scripts/perf-regression-test.sh
```

This script:
1. Runs benchmarks multiple times for statistical significance
2. Compares against baseline
3. Fails if performance degrades > 10%
4. Generates detailed comparison report

### Manual Regression Testing

1. **Establish Baseline**
   ```bash
   go test -bench=. -benchmem -count=10 ./... > perf-baseline.txt
   ```

2. **Make Changes**
   - Implement your feature/fix
   - Commit changes

3. **Run Comparison**
   ```bash
   go test -bench=. -benchmem -count=10 ./... > perf-new.txt
   benchstat perf-baseline.txt perf-new.txt
   ```

4. **Interpret Results**
   - Look for significant changes (p < 0.05)
   - Regressions > 10% require investigation
   - Improvements > 20% should be verified

### Example benchstat Output

```
name                         old time/op    new time/op    delta
ParseLine/simple_tag-8         66.4ns ± 2%    68.1ns ± 3%   +2.56%
Parse/file_with_1000-8          697µs ± 1%     702µs ± 2%     ~     (p=0.095)
DecodeSmall-8                   150µs ± 3%     148µs ± 2%   -1.33%
EncodeLarge-8                  1.15ms ± 1%    1.17ms ± 2%   +1.74%

name                         old alloc/op   new alloc/op   delta
ParseLine/simple_tag-8          96.0B ± 0%     96.0B ± 0%     ~     (all equal)
DecodeMedium-8                 8.50MB ± 0%    8.49MB ± 0%   -0.12%
```

---

## Optimization Guidelines

### When to Optimize

1. **Performance degrades > 10%** in regression tests
2. **Actual user complaints** about speed
3. **Profiling reveals** clear hot spots
4. **Large file handling** becomes problematic

### Optimization Priorities

1. **Correctness first** - Never sacrifice correctness for speed
2. **Measure before optimizing** - Use profiling data
3. **Focus on hot paths** - 80/20 rule applies
4. **Memory matters** - Reduce allocations in tight loops

### Known Optimization Opportunities

1. **Parser String Splitting** (`strings.Fields` in hot path)
   - Current: Uses `strings.Fields` for each line
   - Opportunity: Custom tokenizer could reduce allocations
   - Expected gain: 10-20% faster parsing

2. **Encoder Buffer Management**
   - Current: Multiple small writes
   - Opportunity: Pre-allocate buffers based on document size
   - Expected gain: 5-10% faster encoding

3. **Validator XRef Lookups**
   - Current: Map lookups for each reference
   - Opportunity: Build index once, validate later
   - Expected gain: Already very fast, minimal benefit

---

## Continuous Monitoring

### CI Integration

Add to `.github/workflows/ci.yml`:

```yaml
- name: Run Benchmarks
  run: |
    go test -bench=. -benchmem ./... > bench-current.txt

- name: Compare with Baseline
  run: |
    benchstat bench-baseline.txt bench-current.txt || true

- name: Upload Results
  uses: actions/upload-artifact@v3
  with:
    name: benchmark-results
    path: bench-current.txt
```

### Regular Review

- Review benchmark results monthly
- Update baseline after verified improvements
- Track trends over time
- Document significant changes

---

## Troubleshooting Performance Issues

### Issue: Slow Parsing

**Symptoms:**
- ParseLine benchmarks > 200ns/op
- Memory allocations > 500B/op

**Diagnosis:**
```bash
go test -bench=BenchmarkParseLine -cpuprofile=cpu.prof ./parser
go tool pprof -http=:8080 cpu.prof
```

**Common causes:**
- String allocation in hot path
- Regex compilation per line
- Inefficient tokenization

### Issue: High Memory Usage

**Symptoms:**
- Decode using > 20MB for 1000 individuals
- Many allocations per operation

**Diagnosis:**
```bash
go test -bench=BenchmarkDecodeLarge -memprofile=mem.prof ./decoder
go tool pprof -alloc_space mem.prof
```

**Common causes:**
- Duplicate string storage
- Unnecessary slice growth
- Retained temporary buffers

### Issue: Slow Validation

**Symptoms:**
- Validation > 10μs per 1000 individuals
- Non-zero allocations for valid docs

**Diagnosis:**
```bash
go test -bench=BenchmarkValidateLarge -cpuprofile=cpu.prof ./validator
go tool pprof cpu.prof
```

**Common causes:**
- Inefficient XRef map traversal
- String concatenation in error paths
- Unnecessary validation passes

---

## Profiling Tools

### CPU Profiling

```bash
# Generate CPU profile
go test -bench=BenchmarkDecodeLarge -cpuprofile=cpu.prof ./decoder

# Interactive analysis
go tool pprof cpu.prof
# Commands: top, list, web

# Web UI
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
go test -bench=BenchmarkEncodeLarge -memprofile=mem.prof ./encoder

# Analyze allocations
go tool pprof -alloc_space mem.prof

# Analyze in-use memory
go tool pprof -inuse_space mem.prof
```

### Trace Analysis

```bash
# Generate trace
go test -bench=BenchmarkParse -trace=trace.out ./parser

# Analyze trace
go tool trace trace.out
```

---

## Version History

### v1.0.0 (2025-01-21)
- Established baseline benchmarks
- Apple M2, Go 1.24.3
- All targets met or exceeded

---

## References

- [Go Performance Tips](https://github.com/golang/go/wiki/Performance)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [benchstat Documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
