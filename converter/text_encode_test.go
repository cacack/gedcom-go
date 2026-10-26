package converter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// The continuation transformations in the ConversionReport must describe the
// converted output, not the converter's internal steps (#422). These tests
// check the report against a line count of the actual input and encoded output
// so the two cannot drift: GEDCOM 7.0 removed CONC, but kept CONT.

// countContinuations counts the CONC and CONT lines in GEDCOM text.
func countContinuations(text string) (conc, cont int) {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[1] {
		case "CONC":
			conc++
		case "CONT":
			cont++
		}
	}
	return conc, cont
}

// assertContinuationReportMatchesOutput converts input to 7.0, encodes it and
// checks each continuation transformation against the input/output diff.
func assertContinuationReportMatchesOutput(t *testing.T, input string) string {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, report, err := Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()

	inCONC, inCONT := countContinuations(input)
	outCONC, outCONT := countContinuations(out)

	if outCONC != 0 {
		t.Errorf("7.0 output has %d CONC lines, want 0", outCONC)
	}
	if outCONT != inCONT {
		t.Errorf("7.0 output has %d CONT lines, input had %d", outCONT, inCONT)
	}

	var reportedCONC int
	for _, tr := range report.Transformations {
		switch tr.Type {
		case "CONC_REMOVED":
			reportedCONC = tr.Count
		case "CONT_CONVERTED":
			t.Errorf("report claims %d CONT converted, but output CONT lines %d -> %d", tr.Count, inCONT, outCONT)
		}
	}
	if removed := inCONC - outCONC; reportedCONC != removed {
		t.Errorf("CONC_REMOVED count = %d, output diff removed %d", reportedCONC, removed)
	}
	for _, n := range report.Normalized {
		if strings.Contains(n.Reason, "CONT") {
			t.Errorf("normalized note claims a CONT change: %+v", n)
		}
	}
	return out
}

func TestContinuationReportMatchesOutput(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n" +
		"1 NOTE aaa\n2 CONC bbb\n" +
		"1 NAME John /Doe/\n2 NOTE Line one\n3 CONT Line two\n3 CONT Line three\n" +
		"1 NOTE Start\n2 CONC  continued\n2 CONT New line\n" +
		// Level-0 NOTE record values, folded into Record.Value (#567).
		"0 @N1@ NOTE First\n1 CONT Second\n1 CONT Third\n" +
		"0 @N2@ NOTE Begin\n1 CONC ning\n1 CONT Next\n" +
		"0 TRLR\n"

	out := assertContinuationReportMatchesOutput(t, input)

	for _, want := range []string{
		"1 NOTE aaabbb\n",
		"2 NOTE Line one\n3 CONT Line two\n3 CONT Line three\n",
		"1 NOTE Start continued\n2 CONT New line\n",
		"0 @N1@ SNOTE First\n1 CONT Second\n1 CONT Third\n",
		"0 @N2@ SNOTE Beginning\n1 CONT Next\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestContinuationReportMatchesOutputCorpus(t *testing.T) {
	for _, path := range []string{
		"../testdata/gedcom-5.5.1/comprehensive.ged",
		"../testdata/gedcom-5.5/royal92.ged",
		"../testdata/gedcom-5.5/pres2020.ged",
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			assertContinuationReportMatchesOutput(t, string(data))
		})
	}
}
