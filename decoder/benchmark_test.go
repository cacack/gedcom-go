package decoder

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/cacack/gedcom-go/charset"
	"github.com/cacack/gedcom-go/parser"
)

// BenchmarkDecodeMinimal benchmarks parsing a minimal GEDCOM file (~170 bytes)
func BenchmarkDecodeMinimal(b *testing.B) {
	f, err := os.Open("../testdata/gedcom-5.5/minimal.ged")
	if err != nil {
		b.Skip("Test file not found:", err)
	}
	defer f.Close()

	// Read file into memory once
	data, err := os.ReadFile("../testdata/gedcom-5.5/minimal.ged")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decode(newBytesReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeSmall benchmarks parsing a small GEDCOM file (~15KB, GEDCOM 7.0 maximal)
func BenchmarkDecodeSmall(b *testing.B) {
	data, err := os.ReadFile("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		b.Skip("Test file not found:", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decode(newBytesReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeMedium benchmarks parsing a medium GEDCOM file (~458KB, British Royal Family)
func BenchmarkDecodeMedium(b *testing.B) {
	data, err := os.ReadFile("../testdata/gedcom-5.5/royal92.ged")
	if err != nil {
		b.Skip("Test file not found:", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decode(newBytesReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeLarge benchmarks parsing a large GEDCOM file (~1.1MB, US Presidents)
func BenchmarkDecodeLarge(b *testing.B) {
	data, err := os.ReadFile("../testdata/gedcom-5.5/pres2020.ged")
	if err != nil {
		b.Skip("Test file not found:", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decode(newBytesReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper to create a fresh bytes.Reader for each iteration
func newBytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

// BenchmarkStreamingParseLarge benchmarks streaming parse of the same large
// file used by BenchmarkDecodeLarge. Each iteration touches rec.Type so the
// benchmark models a real consumer that at least inspects each record,
// matching the example pattern.
//
//	go test -bench='DecodeLarge|StreamingParseLarge' -benchmem ./decoder/
func BenchmarkStreamingParseLarge(b *testing.B) {
	data, err := os.ReadFile("../testdata/gedcom-5.5/pres2020.ged")
	if err != nil {
		b.Skip("Test file not found:", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := charset.NewReader(bytes.NewReader(data))
		var iterErr error
		for rec, perr := range parser.Records(r) {
			if perr != nil {
				iterErr = perr
				break
			}
			_ = rec.Type
		}
		if iterErr != nil {
			b.Fatal(iterErr)
		}
	}
}

// TestMemoryBatchVsStreaming reports two complementary memory metrics for
// batch-decode vs streaming-parse over the large fixture:
//
//  1. Cumulative allocated bytes (TotalAlloc delta) — total work done.
//     Stable, GC-immune. Tells you "how many bytes did the operation churn
//     through in aggregate".
//  2. Retained heap after return (HeapAlloc delta with KeepAlive) — what's
//     still resident when the call completes. Tells you "if I do this in
//     the middle of a long-running process, what's the steady-state cost?"
//
// The streaming win shows up in (2): batch holds the full Document; streaming
// holds essentially nothing. Cumulative work in (1) is similar because both
// paths parse the same bytes and allocate similar per-record buffers.
//
//	go test -v -run TestMemory ./decoder/
func TestMemoryBatchVsStreaming(t *testing.T) {
	data, err := os.ReadFile("../testdata/gedcom-5.5/pres2020.ged")
	if err != nil {
		t.Skip("Test file not found:", err)
	}

	batchAllocs, batchRetained := measureMemory(t, func() {
		doc, err := Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		runtime.KeepAlive(doc)
	})

	streamAllocs, streamRetained := measureMemory(t, func() {
		r := charset.NewReader(bytes.NewReader(data))
		var iterErr error
		for rec, perr := range parser.Records(r) {
			if perr != nil {
				iterErr = perr
				break
			}
			_ = rec.Type
		}
		if iterErr != nil {
			t.Fatal(iterErr)
		}
	})

	t.Logf("Large fixture: %d bytes", len(data))
	t.Logf("Batch decode:    cumulative=%d KB retained=%d KB", batchAllocs/1024, batchRetained/1024)
	t.Logf("Streaming parse: cumulative=%d KB retained=%d KB", streamAllocs/1024, streamRetained/1024)
	if batchRetained > 0 {
		t.Logf("Streaming retains ~%.1f%% of batch retained heap", 100*float64(streamRetained)/float64(batchRetained))
	}
}

// measureMemory runs fn and returns two values:
//   - cumulative: bytes allocated during fn (TotalAlloc delta — monotonic,
//     GC-immune; "total work done")
//   - retained: heap-resident bytes after fn returns (HeapAlloc delta;
//     "what's still in memory afterwards")
//
// Together they capture two distinct aspects of memory cost. Call sites
// that want the post-call heap to include something specific must use
// runtime.KeepAlive on it inside fn so the GC can't reclaim it before
// the second ReadMemStats.
//
// This helper is intentionally duplicated in encoder/benchmark_test.go
// (Go test helpers can't be shared across packages without an internal
// package, and the duplication is small enough that adding one isn't
// worth it). Keep the two copies in sync.
func measureMemory(t *testing.T, fn func()) (cumulative, retained uint64) {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	cumulative = after.TotalAlloc - before.TotalAlloc
	if after.HeapAlloc > before.HeapAlloc {
		retained = after.HeapAlloc - before.HeapAlloc
	}
	return cumulative, retained
}
