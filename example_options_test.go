package gedcomgo_test

import (
	"bytes"
	"fmt"
	"strings"

	gedcomgo "github.com/cacack/gedcom-go"
	"github.com/cacack/gedcom-go/gedcom"
)

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

	doc, _ := gedcomgo.Decode(strings.NewReader(gedcomData))

	opts := gedcomgo.DefaultValidateOptions()
	opts.MaxErrors = 5

	issues := gedcomgo.ValidateAllWithOptions(doc, opts)
	fmt.Printf("Issues: %d\n", len(issues))

	// Output:
	// Issues: 1
}
