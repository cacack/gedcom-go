package decoder

import (
	"bytes"
	"os"
	"testing"
)

// scaleFixturePath is the one deliberately large real-world fixture
// (Legacy 10.0 export, GEDCOM 5.5.1, UTF-8 with BOM, ~46 MiB,
// 203,154 individuals / 90,085 families). It exercises scale behavior
// (XRef map growth, memory, throughput) that the smaller fixtures cannot.
const scaleFixturePath = "../testdata/gedcom-5.5.1/longsword.ged"

// TestScaleFixture decodes the large scale fixture and verifies the
// individual/family counts match the corpus audit. Skipped in short mode
// (matches the TestLargeRealWorldFiles convention for large files) and
// under the race detector: race instrumentation multiplies the decode's
// ~930 MB peak RSS several-fold, which risks OOM on CI runners. The
// non-race coverage job still exercises it.
func TestScaleFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale fixture test in short mode")
	}
	if raceDetectorEnabled {
		t.Skip("Skipping scale fixture test under the race detector (memory amplification)")
	}

	f, err := os.Open(scaleFixturePath)
	if err != nil {
		t.Skipf("Test file not found: %s", scaleFixturePath)
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if doc == nil {
		t.Fatal("Decode() returned nil document")
	}

	const (
		wantIndividuals = 203154
		wantFamilies    = 90085
	)
	if got := len(doc.Individuals()); got != wantIndividuals {
		t.Errorf("Individuals() = %d, want %d", got, wantIndividuals)
	}
	if got := len(doc.Families()); got != wantFamilies {
		t.Errorf("Families() = %d, want %d", got, wantFamilies)
	}

	t.Logf("Successfully parsed scale fixture: %d total records, %d individuals, %d families, %d XRefs",
		len(doc.Records), len(doc.Individuals()), len(doc.Families()), len(doc.XRefMap))
}

// BenchmarkScaleDecode benchmarks parse+decode of the ~46 MiB scale fixture.
// b.SetBytes reports throughput in MB/s.
//
//	go test -bench BenchmarkScaleDecode -run '^$' -benchmem ./decoder/
func BenchmarkScaleDecode(b *testing.B) {
	data, err := os.ReadFile(scaleFixturePath)
	if err != nil {
		b.Skip("Test file not found:", err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decode(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}
