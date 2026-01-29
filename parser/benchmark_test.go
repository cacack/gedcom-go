package parser

import (
	"strings"
	"testing"
)

// BenchmarkParseLine benchmarks parsing a single GEDCOM line
func BenchmarkParseLine(b *testing.B) {
	tests := []struct {
		name string
		line string
	}{
		{"simple tag", "0 HEAD"},
		{"tag with value", "1 NAME John /Doe/"},
		{"tag with xref", "0 @I1@ INDI"},
		{"nested tag", "2 DATE 1 JAN 1980"},
		{"long value", "1 NOTE This is a very long note that contains a lot of text and should test the performance of parsing longer values"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			p := NewParser()
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = p.ParseLine(tt.line)
			}
		})
	}
}

// BenchmarkParse benchmarks parsing entire GEDCOM files
func BenchmarkParse(b *testing.B) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "minimal file",
			content: `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
0 TRLR`,
		},
		{
			name:    "file with 100 individuals",
			content: generateLargeGEDCOM(100),
		},
		{
			name:    "file with 1000 individuals",
			content: generateLargeGEDCOM(1000),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			reader := strings.NewReader(tt.content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader.Reset(tt.content)
				p := NewParser()
				_, _ = p.Parse(reader)
			}
		})
	}
}

// BenchmarkParseLineTypes benchmarks different line types
func BenchmarkParseLineTypes(b *testing.B) {
	benchmarks := []struct {
		name string
		line string
	}{
		{"level 0 no xref", "0 HEAD"},
		{"level 0 with xref", "0 @I1@ INDI"},
		{"level 1 simple", "1 NAME John Doe"},
		{"level 2 nested", "2 DATE 1 JAN 1980"},
		{"level 3 deeply nested", "3 SOUR @S1@"},
		{"tag with pointer", "1 FAMC @F1@"},
		{"long tag value", "1 NOTE " + strings.Repeat("A", 200)},
	}

	p := NewParser()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = p.ParseLine(bm.line)
			}
		})
	}
}

// BenchmarkRecordIterator benchmarks the RecordIterator with various file sizes
func BenchmarkRecordIterator(b *testing.B) {
	tests := []struct {
		name string
		size int
	}{
		{"10 records", 10},
		{"100 records", 100},
		{"1000 records", 1000},
	}

	for _, tt := range tests {
		content := generateLargeGEDCOM(tt.size)
		b.Run(tt.name, func(b *testing.B) {
			reader := strings.NewReader(content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader.Reset(content)
				it := NewRecordIterator(reader)
				for it.Next() {
					_ = it.Record()
				}
			}
		})
	}
}

// BenchmarkRecordIteratorWithOffset benchmarks the offset-tracking variant
func BenchmarkRecordIteratorWithOffset(b *testing.B) {
	tests := []struct {
		name string
		size int
	}{
		{"10 records", 10},
		{"100 records", 100},
		{"1000 records", 1000},
	}

	for _, tt := range tests {
		content := generateLargeGEDCOM(tt.size)
		b.Run(tt.name, func(b *testing.B) {
			reader := strings.NewReader(content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader.Reset(content)
				it := NewRecordIteratorWithOffset(reader)
				for it.Next() {
					_ = it.Record()
				}
			}
		})
	}
}

// BenchmarkRecordsRangeFunc benchmarks the Go 1.23 range-over-func iterator
func BenchmarkRecordsRangeFunc(b *testing.B) {
	tests := []struct {
		name string
		size int
	}{
		{"10 records", 10},
		{"100 records", 100},
		{"1000 records", 1000},
	}

	for _, tt := range tests {
		content := generateLargeGEDCOM(tt.size)
		b.Run(tt.name, func(b *testing.B) {
			reader := strings.NewReader(content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader.Reset(content)
				for record, err := range Records(reader) {
					if err != nil {
						b.Fatal(err)
					}
					_ = record
				}
			}
		})
	}
}

// BenchmarkRecordsWithOffsetRangeFunc benchmarks the offset-tracking range-over-func iterator
func BenchmarkRecordsWithOffsetRangeFunc(b *testing.B) {
	tests := []struct {
		name string
		size int
	}{
		{"10 records", 10},
		{"100 records", 100},
		{"1000 records", 1000},
	}

	for _, tt := range tests {
		content := generateLargeGEDCOM(tt.size)
		b.Run(tt.name, func(b *testing.B) {
			reader := strings.NewReader(content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader.Reset(content)
				for record, err := range RecordsWithOffset(reader) {
					if err != nil {
						b.Fatal(err)
					}
					_ = record
				}
			}
		})
	}
}

// Helper function to generate a large GEDCOM file for benchmarking
func generateLargeGEDCOM(numIndividuals int) string {
	var sb strings.Builder

	// Header
	sb.WriteString("0 HEAD\n")
	sb.WriteString("1 GEDC\n")
	sb.WriteString("2 VERS 5.5\n")
	sb.WriteString("1 CHAR UTF-8\n")

	// Generate individuals
	for i := 0; i < numIndividuals; i++ {
		sb.WriteString("0 @I")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString("@ INDI\n")
		sb.WriteString("1 NAME Person")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString(" /Surname/\n")
		sb.WriteString("1 SEX M\n")
		sb.WriteString("1 BIRT\n")
		sb.WriteString("2 DATE 1 JAN 1980\n")
		sb.WriteString("2 PLAC New York, NY\n")
	}

	// Trailer
	sb.WriteString("0 TRLR\n")

	return sb.String()
}
