package decoder

import (
	"bytes"
	"io"
	"os"
	"testing"
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
