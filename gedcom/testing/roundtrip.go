package testing

import (
	"bytes"
	"io"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/encoder"
)

// AssertRoundTrip decodes input, encodes it, decodes again, and asserts equality.
// It fails the test if the round-trip produces semantic differences.
//
// This is the primary function for testing GEDCOM round-trip fidelity in unit tests.
//
// Example:
//
//	func TestMyGEDCOM(t *testing.T) {
//	    data, _ := os.ReadFile("family.ged")
//	    gedcomtesting.AssertRoundTrip(t, data)
//	}
func AssertRoundTrip(t *testing.T, input []byte, opts ...Option) {
	t.Helper()

	report, err := CheckRoundTrip(bytes.NewReader(input), opts...)
	if err != nil {
		t.Fatalf("round-trip check failed: %v", err)
	}

	if !report.Equal {
		t.Errorf("round-trip produced differences:\n%s", report.String())
	}
}

// CheckRoundTrip performs a round-trip and returns a detailed comparison report.
// This is for non-test usage, such as validation tools or CI pipelines.
//
// The function:
//  1. Decodes the input GEDCOM
//  2. Encodes it back to bytes
//  3. Decodes the encoded result
//  4. Compares the two documents semantically
//
// Example:
//
//	report, err := gedcomtesting.CheckRoundTrip(file)
//	if err != nil {
//	    log.Fatalf("round-trip error: %v", err)
//	}
//	if !report.Equal {
//	    log.Printf("Differences found: %s", report.String())
//	}
func CheckRoundTrip(input io.Reader, opts ...Option) (*RoundTripReport, error) {
	_ = applyOptions(opts...) // Reserved for future options

	// Step 1: Decode original
	originalDoc, err := decoder.Decode(input)
	if err != nil {
		return nil, err
	}

	// Step 2: Encode to bytes
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, originalDoc); err != nil {
		return nil, err
	}

	// Step 3: Decode the encoded result
	roundTrippedDoc, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}

	// Step 4: Compare documents
	report := &RoundTripReport{Equal: true}
	compareDocuments(originalDoc, roundTrippedDoc, report)

	return report, nil
}
