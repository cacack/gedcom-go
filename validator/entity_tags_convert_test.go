package validator_test

import (
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/validator"
)

// Issue #725: converting to 7.0 sets Header.Encoding and removes CHAR, which
// is not a header divergence; 7.0 has no CHAR to write.
func TestEntityTagsDivergedConverted70Header(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-5.5/royal92.ged")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range validator.New().ValidateAll(out) {
		if issue.Code == validator.CodeEntityTagsDiverged {
			t.Errorf("converted 7.0 document reported %v", issue)
		}
	}
}
