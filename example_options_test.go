package gedcomgo_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	gedcomgo "github.com/cacack/gedcom-go"
	"github.com/cacack/gedcom-go/gedcom"
)

func TestDefaultOptionsHelpers(t *testing.T) {
	t.Parallel()

	if opts := gedcomgo.DefaultDecodeOptions(); opts == nil || opts.MaxNestingDepth == 0 {
		t.Errorf("DefaultDecodeOptions returned unexpected value: %+v", opts)
	}
	if opts := gedcomgo.DefaultEncodeOptions(); opts == nil || opts.LineEnding == "" {
		t.Errorf("DefaultEncodeOptions returned unexpected value: %+v", opts)
	}
	if opts := gedcomgo.DefaultValidateOptions(); opts == nil {
		t.Error("DefaultValidateOptions returned nil")
	}
}

func TestDecodeWithOptionsFacade(t *testing.T) {
	t.Parallel()

	data := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Test /User/
0 TRLR`

	doc, err := gedcomgo.DecodeWithOptions(strings.NewReader(data), gedcomgo.DefaultDecodeOptions())
	if err != nil {
		t.Fatalf("DecodeWithOptions: %v", err)
	}
	if doc == nil || len(doc.Records) != 1 {
		t.Errorf("expected 1 record, got %+v", doc)
	}

	doc, err = gedcomgo.DecodeWithOptions(strings.NewReader(data), nil)
	if err != nil || doc == nil {
		t.Errorf("DecodeWithOptions(nil opts) failed: %v", err)
	}
}

func TestValidateWithOptionsFacade(t *testing.T) {
	t.Parallel()

	data := `0 HEAD
1 GEDC
2 VERS 5.5
0 @F1@ FAM
0 TRLR`

	doc, err := gedcomgo.Decode(strings.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	errs := gedcomgo.ValidateWithOptions(doc, gedcomgo.DefaultValidateOptions())
	if len(errs) == 0 {
		t.Error("ValidateWithOptions: expected at least one error for empty family")
	}

	if errs := gedcomgo.ValidateWithOptions(doc, nil); len(errs) == 0 {
		t.Error("ValidateWithOptions(nil opts): expected at least one error")
	}
}

// ExampleEncodeWithOptions shows encoding via the facade with custom options.
func ExampleEncodeWithOptions() {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Trailer: &gedcom.Trailer{},
	}

	opts := gedcomgo.DefaultEncodeOptions()
	opts.LineEnding = "\r\n"

	var buf bytes.Buffer
	if err := gedcomgo.EncodeWithOptions(&buf, doc, opts); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("CRLF lines: %d\n", strings.Count(buf.String(), "\r\n"))

	// Output:
	// CRLF lines: 4
}

// ExampleValidateAllWithOptions shows validating via the facade with custom options.
func ExampleValidateAllWithOptions() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 2000
1 DEAT
2 DATE 1 JAN 1900
0 TRLR`

	doc, err := gedcomgo.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Decode: %v\n", err)
		return
	}

	opts := gedcomgo.DefaultValidateOptions()
	opts.MaxErrors = 5

	issues := gedcomgo.ValidateAllWithOptions(doc, opts)
	fmt.Printf("Issues: %d\n", len(issues))

	// Output:
	// Issues: 1
}
