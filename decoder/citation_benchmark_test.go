package decoder

import (
	"bytes"
	"fmt"
	"testing"
)

// citationHeavyGEDCOM synthesizes a GEDCOM 5.5.1 document whose weight is in
// source citations carrying DATA.TEXT: every individual has a birth and a death
// event, and each event cites the same source twice, each citation with a DATA
// block holding a DATE and one TEXT line. That is four SourceCitationData
// values per individual, the shape a vendor export such as pres2020.ged
// (2,587 DATA.TEXT lines) produces, scaled up so the per-citation cost of
// SourceCitationData.Text dominates the measurement.
func citationHeavyGEDCOM(individuals int) []byte {
	var b bytes.Buffer
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n")
	b.WriteString("0 @S1@ SOUR\n1 TITL Parish register\n")
	for i := 0; i < individuals; i++ {
		fmt.Fprintf(&b, "0 @I%d@ INDI\n1 NAME Person /Number%d/\n", i, i)
		for _, event := range []string{"BIRT", "DEAT"} {
			fmt.Fprintf(&b, "1 %s\n2 DATE 1 JAN 1850\n", event)
			for c := 0; c < 2; c++ {
				fmt.Fprintf(&b, "2 SOUR @S1@\n3 PAGE Folio %d\n3 DATA\n4 DATE 2 JAN 1850\n", i)
				fmt.Fprintf(&b, "4 TEXT Entry %d for person %d in the register\n", c, i)
			}
		}
	}
	b.WriteString("0 TRLR\n")
	return b.Bytes()
}

// BenchmarkDecodeCitationHeavy measures decode cost on a document dominated by
// source citations with DATA.TEXT (issue #497). It is the benchmark the
// SourceCitationData.Text element type was chosen against: compare allocs/op
// and B/op across a change to that type.
//
//	go test -bench DecodeCitationHeavy -run '^$' -benchmem ./decoder/
func BenchmarkDecodeCitationHeavy(b *testing.B) {
	data := citationHeavyGEDCOM(5000)

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := Decode(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}
