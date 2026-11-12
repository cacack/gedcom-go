package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestValidateAll_DuplicateXRef pins issue #724: two records sharing an XRef
// are reported as DUPLICATE_XREF on the second, with its line, even when the
// hand-built XRefMap holds only one of them.
func TestValidateAll_DuplicateXRef(t *testing.T) {
	first := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, LineNumber: 4,
		Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "John /Smith/"}}}}
	second := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, LineNumber: 6,
		Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "Jane /Doe/"}}}}
	other := &gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, LineNumber: 8,
		Entity: &gedcom.Individual{XRef: "@I2@", Names: []*gedcom.PersonalName{{Full: "Ann /Lee/"}}}}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{first, second, other},
		XRefMap: map[string]*gedcom.Record{"@I1@": second, "@I2@": other},
	}

	var dups []Issue
	for _, issue := range New().ValidateAll(doc) {
		if issue.Code == CodeDuplicateXRef {
			dups = append(dups, issue)
		}
	}
	if len(dups) != 1 {
		t.Fatalf("DUPLICATE_XREF issues = %v, want exactly one", dups)
	}
	d := dups[0]
	if d.Severity != SeverityError || d.RecordXRef != "@I1@" || d.LineNumber != 6 ||
		!strings.Contains(d.Message, "line 4") {
		t.Errorf("issue = %+v, want an error on @I1@ at line 6 naming line 4", d)
	}
}

// TestStreamingValidator_DuplicateXRef pins the same report in streaming mode.
func TestStreamingValidator_DuplicateXRef(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})
	if issues := sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, LineNumber: 3}); len(issues) != 0 {
		t.Fatalf("first definition issues = %v, want none", issues)
	}
	issues := sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeNote, LineNumber: 9})
	if len(issues) != 1 || issues[0].Code != CodeDuplicateXRef || issues[0].LineNumber != 9 ||
		!strings.Contains(issues[0].Message, "line 3") {
		t.Fatalf("redefinition issues = %v, want one DUPLICATE_XREF at line 9 naming line 3", issues)
	}
	if sv.xrefTypes["@S1@"] != gedcom.RecordTypeSource {
		t.Errorf("xrefTypes[@S1@] = %q, want the first definition's type SOUR", sv.xrefTypes["@S1@"])
	}
}
