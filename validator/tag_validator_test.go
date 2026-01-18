package validator

import (
	"regexp"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestNewTagValidator(t *testing.T) {
	t.Run("with nil registry and validateUnknown false", func(t *testing.T) {
		tv := NewTagValidator(nil, false)
		if tv == nil {
			t.Fatal("expected non-nil TagValidator")
		}
		if tv.registry != nil {
			t.Error("expected nil registry")
		}
		if tv.validateUnknown {
			t.Error("expected validateUnknown to be false")
		}
	})

	t.Run("with registry and validateUnknown true", func(t *testing.T) {
		registry := NewTagRegistry()
		tv := NewTagValidator(registry, true)
		if tv == nil {
			t.Fatal("expected non-nil TagValidator")
		}
		if tv.registry != registry {
			t.Error("expected registry to be set")
		}
		if !tv.validateUnknown {
			t.Error("expected validateUnknown to be true")
		}
	})
}

func TestTagValidator_Validate_NilDocument(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, true)

	issues := tv.Validate(nil)
	if issues != nil {
		t.Errorf("expected nil issues for nil document, got %v", issues)
	}
}

func TestTagValidator_Validate_NoCustomTags(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for document without custom tags, got %d", len(issues))
	}
}

func TestTagValidator_Validate_UnknownCustomTag(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_CUSTOM", Value: "some value", LineNumber: 5},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeUnknownCustomTag {
		t.Errorf("expected code %s, got %s", CodeUnknownCustomTag, issue.Code)
	}
	if issue.Severity != SeverityWarning {
		t.Errorf("expected severity Warning, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.Details["tag"] != "_CUSTOM" {
		t.Errorf("expected tag detail _CUSTOM, got %s", issue.Details["tag"])
	}
}

func TestTagValidator_Validate_UnknownCustomTag_Disabled(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, false)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_UNKNOWN", Value: "value"},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues when validateUnknown is false, got %d", len(issues))
	}
}

func TestTagValidator_Validate_KnownCustomTag_Valid(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"},
		Description:    "Military service",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MILT", Value: "Army"},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid custom tag, got %d: %v", len(issues), issues)
	}
}

func TestTagValidator_Validate_InvalidParent(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"}, // Only allowed under INDI
		Description:    "Military service",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MILT", Value: "Army", LineNumber: 10},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeInvalidTagParent {
		t.Errorf("expected code %s, got %s", CodeInvalidTagParent, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("expected severity Error, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@F1@" {
		t.Errorf("expected RecordXRef @F1@, got %s", issue.RecordXRef)
	}
}

func TestTagValidator_Validate_InvalidValue(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_PRIM", TagDefinition{
		Tag:          "_PRIM",
		ValuePattern: YesNoPattern, // Only Y or N allowed
		Description:  "Primary indicator",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_PRIM", Value: "INVALID", LineNumber: 7},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeInvalidTagValue {
		t.Errorf("expected code %s, got %s", CodeInvalidTagValue, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("expected severity Error, got %s", issue.Severity)
	}
}

func TestTagValidator_Validate_NestedCustomTags(t *testing.T) {
	registry := NewTagRegistry()
	// Register a tag that should only appear under BIRT
	err := registry.Register("_CERT", TagDefinition{
		Tag:            "_CERT",
		AllowedParents: []string{"BIRT"},
		Description:    "Certificate reference",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "_CERT", Value: "BC-1234"}, // Valid: under BIRT
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid nested custom tag, got %d: %v", len(issues), issues)
	}
}

func TestTagValidator_Validate_NestedCustomTags_InvalidParent(t *testing.T) {
	registry := NewTagRegistry()
	// Register a tag that should only appear under BIRT
	err := registry.Register("_CERT", TagDefinition{
		Tag:            "_CERT",
		AllowedParents: []string{"BIRT"},
		Description:    "Certificate reference",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "DEAT"},                    // Death event
					{Level: 2, Tag: "_CERT", Value: "DC-5678"}, // Invalid: under DEAT, not BIRT
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeInvalidTagParent {
		t.Errorf("expected code %s, got %s", CodeInvalidTagParent, issue.Code)
	}
}

func TestTagValidator_Validate_MultipleIssues(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"},
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_UNKNOWN1"}, // Unknown
					{Level: 1, Tag: "_UNKNOWN2"}, // Unknown
					{Level: 1, Tag: "_MILT"},     // Valid
				},
			},
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MILT"}, // Invalid parent
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 3 {
		t.Errorf("expected 3 issues, got %d: %v", len(issues), issues)
	}

	// Count issue types
	unknownCount := 0
	invalidParentCount := 0
	for _, issue := range issues {
		switch issue.Code {
		case CodeUnknownCustomTag:
			unknownCount++
		case CodeInvalidTagParent:
			invalidParentCount++
		}
	}

	if unknownCount != 2 {
		t.Errorf("expected 2 unknown tag issues, got %d", unknownCount)
	}
	if invalidParentCount != 1 {
		t.Errorf("expected 1 invalid parent issue, got %d", invalidParentCount)
	}
}

func TestTagValidator_Validate_EmptyRecords(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for empty records, got %d", len(issues))
	}
}

func TestTagValidator_Validate_RecordWithNoTags(t *testing.T) {
	registry := NewTagRegistry()
	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: nil,
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for record with no tags, got %d", len(issues))
	}
}

func TestTagValidator_Validate_XRefPattern(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_ASSO", TagDefinition{
		Tag:          "_ASSO",
		ValuePattern: XRefPattern,
		Description:  "Association to another individual",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	tests := []struct {
		name       string
		value      string
		wantIssues int
	}{
		{"valid XRef", "@I123@", 0},
		{"invalid XRef - no @", "I123", 1},
		{"invalid XRef - missing closing @", "@I123", 1},
		{"valid XRef with underscore", "@I_1@", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{Level: 1, Tag: "_ASSO", Value: tt.value},
						},
					},
				},
			}

			issues := tv.Validate(doc)
			if len(issues) != tt.wantIssues {
				t.Errorf("expected %d issues, got %d: %v", tt.wantIssues, len(issues), issues)
			}
		})
	}
}

func TestTagValidator_Validate_DeepNesting(t *testing.T) {
	registry := NewTagRegistry()
	err := registry.Register("_DEEP", TagDefinition{
		Tag:            "_DEEP",
		AllowedParents: []string{"DATE"},
		Description:    "Deep nested tag",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
					{Level: 3, Tag: "_DEEP", Value: "data"}, // Valid: under DATE
				},
			},
		},
	}

	issues := tv.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid deep nesting, got %d: %v", len(issues), issues)
	}
}

func TestTagValidator_Validate_NilRegistry(t *testing.T) {
	tv := NewTagValidator(nil, true) // nil registry, but validateUnknown = true

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_CUSTOM", Value: "value"},
				},
			},
		},
	}

	issues := tv.Validate(doc)
	// With nil registry, all custom tags are unknown
	if len(issues) != 1 {
		t.Errorf("expected 1 unknown tag issue, got %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Code != CodeUnknownCustomTag {
		t.Errorf("expected code %s, got %s", CodeUnknownCustomTag, issues[0].Code)
	}
}

func TestTagValidator_Validate_CustomValuePattern(t *testing.T) {
	registry := NewTagRegistry()
	// Custom pattern for year values
	yearPattern := regexp.MustCompile(`^\d{4}$`)
	err := registry.Register("_YEAR", TagDefinition{
		Tag:          "_YEAR",
		ValuePattern: yearPattern,
		Description:  "Year-only value",
	})
	if err != nil {
		t.Fatalf("failed to register tag: %v", err)
	}

	tv := NewTagValidator(registry, true)

	tests := []struct {
		name       string
		value      string
		wantIssues int
	}{
		{"valid year", "1985", 0},
		{"invalid - too short", "85", 1},
		{"invalid - has letters", "198X", 1},
		{"invalid - too long", "19850", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{Level: 1, Tag: "_YEAR", Value: tt.value},
						},
					},
				},
			}

			issues := tv.Validate(doc)
			if len(issues) != tt.wantIssues {
				t.Errorf("expected %d issues, got %d: %v", tt.wantIssues, len(issues), issues)
			}
		})
	}
}
