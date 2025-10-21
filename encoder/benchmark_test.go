package encoder

import (
	"bytes"
	"io"
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

// Helper function to generate a document with N individuals
func generateDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals)

	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + string(rune('0'+i%10)) + "@"
		records = append(records, &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('0'+i%10)) + " /Surname/"},
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
					{Full: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				},
				Sex: "M",
			},
		})
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
