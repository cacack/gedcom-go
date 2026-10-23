package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// A dangling pointer under INDI.NAME is reported (issue #643). The raw-tag scan
// covered it before PersonalName typed these fields; this pins that typing them
// did not hide them.
func TestBrokenXRefUnderPersonalName(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
2 SOUR @S9@
2 NOTE @N9@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	broken := FilterByCode(New().ValidateAll(doc), CodeBrokenXRef)
	got := map[string]int{}
	for _, issue := range broken {
		got[issue.RelatedXRef] = issue.LineNumber
	}
	if got["@S9@"] != 6 || got["@N9@"] != 7 || len(got) != 2 {
		t.Errorf("BROKEN_XREF = %v, want @S9@ on line 6 and @N9@ on line 7", broken)
	}
}
