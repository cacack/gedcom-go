package decoder

import (
	"strings"
	"testing"
)

// TestDecodeHeaderSubmitter verifies that HEAD.SUBM populates
// Header.Submitter and that the pointer resolves to the submitter record.
// Regression test for #503: buildHeader had no SUBM case, so the field was
// permanently empty and validator's MISSING_SUBM warned on every document.
// No version gate — HEAD.SUBM is valid in 7.0 too (optional there).
func TestDecodeHeaderSubmitter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "5.5",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR ANSEL
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 TRLR`,
		},
		{
			name: "5.5.1",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 TRLR`,
		},
		{
			name: "7.0",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 TRLR`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if doc.Header.Submitter != "@U1@" {
				t.Fatalf("Header.Submitter = %q, want %q", doc.Header.Submitter, "@U1@")
			}

			subm := doc.GetSubmitter(doc.Header.Submitter)
			if subm == nil {
				t.Fatalf("GetSubmitter(%q) = nil, want submitter record", doc.Header.Submitter)
			}
			if subm.Name != "John Researcher" {
				t.Errorf("Submitter.Name = %q, want %q", subm.Name, "John Researcher")
			}
		})
	}
}

// TestDecodeHeaderSubmitterAbsent verifies that a header without SUBM leaves
// Header.Submitter empty rather than picking up a submitter from elsewhere.
func TestDecodeHeaderSubmitterAbsent(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if doc.Header.Submitter != "" {
		t.Errorf("Header.Submitter = %q, want empty", doc.Header.Submitter)
	}
}

// TestDecodeHeaderSubmitterIgnoresSubordinate verifies the level-1 guard does
// something. No GEDCOM version nests SUBM below HEAD, so a subordinate one is
// malformed input -- but buildHeader's switch runs on every line inside the
// header regardless of depth (LANG and COPR have no level check at all), so
// without the guard a "2 SUBM" under another structure would overwrite the
// header's real submitter. The raw tag is preserved either way, since lossless
// storage happens before the switch.
func TestDecodeHeaderSubmitterIgnoresSubordinate(t *testing.T) {
	input := `0 HEAD
1 SOUR TESTAPP
2 SUBM @BOGUS@
1 SUBM @U1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME John Researcher
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if doc.Header.Submitter != "@U1@" {
		t.Errorf("Header.Submitter = %q, want @U1@ -- a subordinate SUBM must not win",
			doc.Header.Submitter)
	}
	if doc.GetSubmitter(doc.Header.Submitter) == nil {
		t.Errorf("Header.Submitter %q resolves to no submitter record", doc.Header.Submitter)
	}

	// Lossless representation: the malformed line is still in the raw tags.
	var found bool
	for _, tag := range doc.Header.Tags {
		if tag.Level == 2 && tag.Tag == "SUBM" && tag.Value == "@BOGUS@" {
			found = true
		}
	}
	if !found {
		t.Error("subordinate SUBM must still be preserved in Header.Tags")
	}
}
