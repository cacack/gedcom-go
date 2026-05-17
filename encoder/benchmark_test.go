package encoder

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// BenchmarkEncodeMinimal benchmarks encoding a minimal GEDCOM document
func BenchmarkEncodeMinimal(b *testing.B) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  "5.5",
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Encode(io.Discard, doc)
	}
}

// BenchmarkEncodeSmall benchmarks encoding a small document (10 individuals)
func BenchmarkEncodeSmall(b *testing.B) {
	doc := generateDocument(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Encode(io.Discard, doc)
	}
}

// BenchmarkEncodeMedium benchmarks encoding a medium document (100 individuals)
func BenchmarkEncodeMedium(b *testing.B) {
	doc := generateDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Encode(io.Discard, doc)
	}
}

// BenchmarkEncodeLarge benchmarks encoding a large document (1000 individuals)
func BenchmarkEncodeLarge(b *testing.B) {
	doc := generateDocument(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Encode(io.Discard, doc)
	}
}

// BenchmarkEncodeWithBuffer benchmarks encoding with actual buffer allocation
func BenchmarkEncodeWithBuffer(b *testing.B) {
	doc := generateDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = Encode(&buf, doc)
	}
}

// BenchmarkEncodeLineEndings benchmarks different line ending options
func BenchmarkEncodeLineEndings(b *testing.B) {
	doc := generateDocument(100)

	tests := []struct {
		name    string
		options *EncodeOptions
	}{
		{"LF (Unix)", &EncodeOptions{LineEnding: "\n"}},
		{"CRLF (Windows)", &EncodeOptions{LineEnding: "\r\n"}},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = EncodeWithOptions(io.Discard, doc, tt.options)
			}
		})
	}
}

// BenchmarkStreamEncodeLarge benchmarks streaming encode of 1000 individuals
// where records are generated *inside* the loop, mirroring how a real
// streaming consumer produces records on the fly without materializing the
// full document.
//
// The comparison with BenchmarkEncodeLarge is intentionally asymmetric:
// batch encoding requires the full Document to exist before it can run, so
// that allocation cost is part of the batch workload. Use
// TestAllocDeltaBatchVsStreamingEncode for a more direct memory comparison.
//
//	go test -bench='EncodeLarge|StreamEncodeLarge' -benchmem ./encoder/
func BenchmarkStreamEncodeLarge(b *testing.B) {
	const n = 1000
	header := &gedcom.Header{Version: "5.5", Encoding: "UTF-8", SourceSystem: "benchmark"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		enc := NewStreamEncoder(io.Discard)
		_ = enc.WriteHeader(header)
		for j := 0; j < n; j++ {
			_ = enc.WriteRecord(newIndividual(j))
		}
		_ = enc.WriteTrailer()
		_ = enc.Close()
	}
}

// TestMemoryBatchVsStreamingEncode reports two complementary memory metrics
// for batch encode vs streaming encode of 10,000 generated individuals:
//
//  1. Cumulative allocated bytes (TotalAlloc delta) — total work done.
//  2. Retained heap after return (HeapAlloc delta with KeepAlive) — what's
//     still resident when the call completes.
//
// The streaming win is in (2): the batch path constructs a full Document
// (held alive during Encode and after, via KeepAlive), while streaming
// generates records on the fly and discards them. Cumulative work in (1)
// is similar because both paths construct the same records — they just
// retain them differently.
//
//	go test -v -run TestMemory ./encoder/
func TestMemoryBatchVsStreamingEncode(t *testing.T) {
	const n = 10000

	batchAllocs, batchRetained := measureMemory(t, func() {
		doc := generateDocument(n)
		_ = Encode(io.Discard, doc)
		runtime.KeepAlive(doc)
	})

	streamAllocs, streamRetained := measureMemory(t, func() {
		header := &gedcom.Header{Version: "5.5", Encoding: "UTF-8", SourceSystem: "benchmark"}
		enc := NewStreamEncoder(io.Discard)
		_ = enc.WriteHeader(header)
		for i := 0; i < n; i++ {
			_ = enc.WriteRecord(newIndividual(i))
		}
		_ = enc.WriteTrailer()
		_ = enc.Close()
	})

	t.Logf("Individuals: %d", n)
	t.Logf("Batch encode:    cumulative=%d KB retained=%d KB", batchAllocs/1024, batchRetained/1024)
	t.Logf("Streaming encode: cumulative=%d KB retained=%d KB", streamAllocs/1024, streamRetained/1024)
	if batchRetained > 0 {
		t.Logf("Streaming retains ~%.1f%% of batch retained heap", 100*float64(streamRetained)/float64(batchRetained))
	}
}

// newIndividual produces a representative individual record at index i with
// a unique XRef. The shape mirrors generateDocument so a per-record streaming
// loop produces semantically equivalent output to the batch path.
func newIndividual(i int) *gedcom.Record {
	xref := fmt.Sprintf("@I%d@", i)
	name := fmt.Sprintf("Person %d /Surname/", i)
	return &gedcom.Record{
		XRef: xref,
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: name},
			{Level: 1, Tag: "SEX", Value: "M"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "DATE", Value: "1 JAN 1980"},
			{Level: 2, Tag: "PLAC", Value: "New York, NY"},
			{Level: 1, Tag: "DEAT"},
			{Level: 2, Tag: "DATE", Value: "1 JAN 2050"},
			{Level: 2, Tag: "PLAC", Value: "Boston, MA"},
		},
		Entity: &gedcom.Individual{
			XRef: xref,
			Names: []*gedcom.PersonalName{
				{Full: name},
			},
			Sex: "M",
		},
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
// This helper is intentionally duplicated in decoder/benchmark_test.go
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

// Helper function to generate a document with N individuals
func generateDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals)

	for i := 0; i < numIndividuals; i++ {
		records = append(records, newIndividual(i))
	}

	return &gedcom.Document{
		Header: &gedcom.Header{
			Version:      "5.5",
			Encoding:     "UTF-8",
			SourceSystem: "benchmark",
		},
		Records: records,
	}
}
