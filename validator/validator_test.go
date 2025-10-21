package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
)

func TestValidateBrokenXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John Smith
1 FAMS @F999@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errors := v.Validate(doc)

	if len(errors) == 0 {
		t.Fatal("Expected validation errors for broken XRef")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "BROKEN_XREF") {
			found = true
			t.Logf("Found expected error: %v", err)
		}
	}

	if !found {
		t.Error("Expected BROKEN_XREF error")
	}
}

func TestValidateMissingName(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errors := v.Validate(doc)

	if len(errors) == 0 {
		t.Fatal("Expected validation errors for missing NAME")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "MISSING_REQUIRED_FIELD") {
			found = true
			t.Logf("Found expected error: %v", err)
		}
	}

	if !found {
		t.Error("Expected MISSING_REQUIRED_FIELD error")
	}
}

func TestValidateValidFile(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errors := v.Validate(doc)

	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid file, got %d errors:", len(errors))
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}
}
