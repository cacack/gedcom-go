package validator

import (
	"fmt"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestNewEncodingValidator(t *testing.T) {
	v := NewEncodingValidator()
	if v == nil {
		t.Error("NewEncodingValidator() returned nil")
	}
}

func TestEncodingValidator_ValidateEncoding_NilDocument(t *testing.T) {
	v := NewEncodingValidator()
	issues := v.ValidateEncoding(nil)
	if issues != nil {
		t.Errorf("ValidateEncoding(nil) should return nil, got %d issues", len(issues))
	}
}

func TestEncodingValidator_ValidateEncoding_NilHeader(t *testing.T) {
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header:  nil,
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}
	issues := v.ValidateEncoding(doc)
	if issues != nil {
		t.Errorf("ValidateEncoding with nil Header should return nil, got %d issues", len(issues))
	}
}

func TestEncodingValidator_ValidateEncoding(t *testing.T) {
	tests := []struct {
		name         string
		version      gedcom.Version
		encoding     gedcom.Encoding
		wantIssues   int
		wantCode     string
		wantSeverity Severity
	}{
		{
			name:       "GEDCOM 7.0 with UTF-8 encoding - pass",
			version:    gedcom.Version70,
			encoding:   gedcom.EncodingUTF8,
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with empty encoding - pass (defaults to UTF-8)",
			version:    gedcom.Version70,
			encoding:   "",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with ASCII encoding - pass (subset of UTF-8)",
			version:    gedcom.Version70,
			encoding:   gedcom.EncodingASCII,
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with UNICODE encoding - pass (alias for UTF-8)",
			version:    gedcom.Version70,
			encoding:   gedcom.EncodingUNICODE,
			wantIssues: 0,
		},
		{
			name:         "GEDCOM 7.0 with ANSEL encoding - fail",
			version:      gedcom.Version70,
			encoding:     gedcom.EncodingANSEL,
			wantIssues:   1,
			wantCode:     CodeInvalidEncodingForVersion,
			wantSeverity: SeverityError,
		},
		{
			name:         "GEDCOM 7.0 with LATIN1 encoding - fail",
			version:      gedcom.Version70,
			encoding:     gedcom.EncodingLATIN1,
			wantIssues:   1,
			wantCode:     CodeInvalidEncodingForVersion,
			wantSeverity: SeverityError,
		},
		{
			name:         "GEDCOM 7.0 with UTF-16LE encoding - fail",
			version:      gedcom.Version70,
			encoding:     gedcom.Encoding("UTF-16LE"),
			wantIssues:   1,
			wantCode:     CodeInvalidEncodingForVersion,
			wantSeverity: SeverityError,
		},
		{
			name:         "GEDCOM 7.0 with UTF-16BE encoding - fail",
			version:      gedcom.Version70,
			encoding:     gedcom.Encoding("UTF-16BE"),
			wantIssues:   1,
			wantCode:     CodeInvalidEncodingForVersion,
			wantSeverity: SeverityError,
		},
		{
			name:         "GEDCOM 7.0 with UTF16 encoding - fail",
			version:      gedcom.Version70,
			encoding:     gedcom.Encoding("UTF16"),
			wantIssues:   1,
			wantCode:     CodeInvalidEncodingForVersion,
			wantSeverity: SeverityError,
		},
		{
			name:       "GEDCOM 5.5 with ANSEL encoding - pass (no restriction)",
			version:    gedcom.Version55,
			encoding:   gedcom.EncodingANSEL,
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5.1 with ANSEL encoding - pass (no restriction)",
			version:    gedcom.Version551,
			encoding:   gedcom.EncodingANSEL,
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5 with UTF-16LE encoding - pass (no restriction)",
			version:    gedcom.Version55,
			encoding:   gedcom.Encoding("UTF-16LE"),
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5.1 with UTF-16BE encoding - pass (no restriction)",
			version:    gedcom.Version551,
			encoding:   gedcom.Encoding("UTF-16BE"),
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5 with LATIN1 encoding - pass (no restriction)",
			version:    gedcom.Version55,
			encoding:   gedcom.EncodingLATIN1,
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewEncodingValidator()
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version:  tt.version,
					Encoding: tt.encoding,
				},
				Records: []*gedcom.Record{},
				XRefMap: make(map[string]*gedcom.Record),
			}

			issues := v.ValidateEncoding(doc)

			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateEncoding() returned %d issues, want %d", len(issues), tt.wantIssues)
				for _, issue := range issues {
					t.Logf("  Issue: %s", issue.String())
				}
				return
			}

			if tt.wantIssues > 0 {
				issue := issues[0]

				if issue.Code != tt.wantCode {
					t.Errorf("issue.Code = %q, want %q", issue.Code, tt.wantCode)
				}

				if issue.Severity != tt.wantSeverity {
					t.Errorf("issue.Severity = %v, want %v", issue.Severity, tt.wantSeverity)
				}

				if issue.Details["version"] != string(tt.version) {
					t.Errorf("issue.Details[\"version\"] = %q, want %q", issue.Details["version"], tt.version)
				}
			}
		})
	}
}

func TestEncodingValidator_ValidateControlCharacters_NilDocument(t *testing.T) {
	v := NewEncodingValidator()
	issues := v.ValidateControlCharacters(nil)
	if issues != nil {
		t.Errorf("ValidateControlCharacters(nil) should return nil, got %d issues", len(issues))
	}
}

func TestEncodingValidator_ValidateControlCharacters_NilHeader(t *testing.T) {
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header:  nil,
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}
	issues := v.ValidateControlCharacters(doc)
	if issues != nil {
		t.Errorf("ValidateControlCharacters with nil Header should return nil, got %d issues", len(issues))
	}
}

func TestEncodingValidator_ValidateControlCharacters(t *testing.T) {
	tests := []struct {
		name         string
		version      gedcom.Version
		tagValue     string
		wantIssues   int
		wantCode     string
		wantSeverity Severity
		wantCharCode string
	}{
		{
			name:       "GEDCOM 7.0 with normal text - pass",
			version:    gedcom.Version70,
			tagValue:   "John Smith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with TAB (U+0009) - pass (allowed)",
			version:    gedcom.Version70,
			tagValue:   "John\tSmith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with LF (U+000A) - pass (allowed)",
			version:    gedcom.Version70,
			tagValue:   "John\nSmith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with CR (U+000D) - pass (allowed)",
			version:    gedcom.Version70,
			tagValue:   "John\rSmith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with CRLF - pass (allowed)",
			version:    gedcom.Version70,
			tagValue:   "John\r\nSmith",
			wantIssues: 0,
		},
		{
			name:         "GEDCOM 7.0 with NUL (U+0000) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x00Smith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+0000",
		},
		{
			name:         "GEDCOM 7.0 with SOH (U+0001) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x01Smith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+0001",
		},
		{
			name:         "GEDCOM 7.0 with BEL (U+0007) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x07Smith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+0007",
		},
		{
			name:         "GEDCOM 7.0 with BS (U+0008) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x08Smith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+0008",
		},
		{
			name:         "GEDCOM 7.0 with VT (U+000B) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x0BSmith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+000B",
		},
		{
			name:         "GEDCOM 7.0 with FF (U+000C) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x0CSmith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+000C",
		},
		{
			name:         "GEDCOM 7.0 with SO (U+000E) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x0ESmith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+000E",
		},
		{
			name:         "GEDCOM 7.0 with US (U+001F) - fail",
			version:      gedcom.Version70,
			tagValue:     "John\x1FSmith",
			wantIssues:   1,
			wantCode:     CodeBannedControlCharacter,
			wantSeverity: SeverityError,
			wantCharCode: "U+001F",
		},
		{
			name:       "GEDCOM 7.0 with space (U+0020) - pass (not a control char)",
			version:    gedcom.Version70,
			tagValue:   "John Smith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5 with NUL (U+0000) - pass (no restriction)",
			version:    gedcom.Version55,
			tagValue:   "John\x00Smith",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5.1 with control chars - pass (no restriction)",
			version:    gedcom.Version551,
			tagValue:   "John\x01\x02\x03Smith",
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewEncodingValidator()
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version: tt.version,
				},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{
								Tag:   "NAME",
								Value: tt.tagValue,
							},
						},
					},
				},
				XRefMap: make(map[string]*gedcom.Record),
			}

			issues := v.ValidateControlCharacters(doc)

			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateControlCharacters() returned %d issues, want %d", len(issues), tt.wantIssues)
				for _, issue := range issues {
					t.Logf("  Issue: %s", issue.String())
				}
				return
			}

			if tt.wantIssues > 0 {
				issue := issues[0]

				if issue.Code != tt.wantCode {
					t.Errorf("issue.Code = %q, want %q", issue.Code, tt.wantCode)
				}

				if issue.Severity != tt.wantSeverity {
					t.Errorf("issue.Severity = %v, want %v", issue.Severity, tt.wantSeverity)
				}

				if issue.RecordXRef != "@I1@" {
					t.Errorf("issue.RecordXRef = %q, want %q", issue.RecordXRef, "@I1@")
				}

				if issue.Details["character"] != tt.wantCharCode {
					t.Errorf("issue.Details[\"character\"] = %q, want %q", issue.Details["character"], tt.wantCharCode)
				}

				if issue.Details["field"] != "NAME" {
					t.Errorf("issue.Details[\"field\"] = %q, want %q", issue.Details["field"], "NAME")
				}
			}
		})
	}
}

func TestEncodingValidator_ValidateControlCharacters_RecordValue(t *testing.T) {
	// Test that control characters in record.Value are also detected
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
		},
		Records: []*gedcom.Record{
			{
				XRef:  "@N1@",
				Type:  gedcom.RecordTypeNote,
				Value: "Note with\x00null",
				Tags:  []*gedcom.Tag{},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateControlCharacters(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeBannedControlCharacter {
		t.Errorf("issue.Code = %q, want %q", issue.Code, CodeBannedControlCharacter)
	}
	if issue.Details["field"] != "NOTE" {
		t.Errorf("issue.Details[\"field\"] = %q, want %q", issue.Details["field"], "NOTE")
	}
}

func TestEncodingValidator_ValidateControlCharacters_HeaderTags(t *testing.T) {
	// Test that control characters in header tags are detected
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
			Tags: []*gedcom.Tag{
				{
					Tag:   "SOUR",
					Value: "Software\x07Name",
				},
			},
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateControlCharacters(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeBannedControlCharacter {
		t.Errorf("issue.Code = %q, want %q", issue.Code, CodeBannedControlCharacter)
	}
	if issue.Details["character"] != "U+0007" {
		t.Errorf("issue.Details[\"character\"] = %q, want %q", issue.Details["character"], "U+0007")
	}
}

func TestEncodingValidator_Validate(t *testing.T) {
	// Test that Validate combines both encoding and control char validation
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version70,
			Encoding: gedcom.EncodingANSEL,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{
						Tag:   "NAME",
						Value: "John\x00Smith",
					},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.Validate(doc)

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues (encoding + control char), got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}

	// Check that we have both types of issues
	foundEncoding := false
	foundControlChar := false
	for _, issue := range issues {
		if issue.Code == CodeInvalidEncodingForVersion {
			foundEncoding = true
		}
		if issue.Code == CodeBannedControlCharacter {
			foundControlChar = true
		}
	}

	if !foundEncoding {
		t.Error("Expected to find encoding validation issue")
	}
	if !foundControlChar {
		t.Error("Expected to find control character validation issue")
	}
}

func TestValidator_ValidateEncoding(t *testing.T) {
	// Test the public ValidateEncoding method on Validator
	v := New()

	// Test nil document
	issues := v.ValidateEncoding(nil)
	if issues != nil {
		t.Errorf("ValidateEncoding(nil) should return nil, got %d issues", len(issues))
	}

	// Test GEDCOM 7.0 with invalid encoding
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version70,
			Encoding: gedcom.EncodingANSEL,
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues = v.ValidateEncoding(doc)
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
}

func TestValidator_ValidateAll_IncludesEncoding(t *testing.T) {
	// Test that ValidateAll includes encoding validation for GEDCOM 7.0
	v := New()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version70,
			Encoding: gedcom.EncodingANSEL,
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateAll(doc)

	foundEncodingIssue := false
	for _, issue := range issues {
		if issue.Code == CodeInvalidEncodingForVersion {
			foundEncodingIssue = true
			break
		}
	}

	if !foundEncodingIssue {
		t.Error("ValidateAll should include encoding validation issues for GEDCOM 7.0")
	}
}

func TestEncodingValidator_IsBannedControlChar(t *testing.T) {
	v := NewEncodingValidator()

	tests := []struct {
		char   rune
		banned bool
	}{
		{0x00, true},  // NUL
		{0x01, true},  // SOH
		{0x07, true},  // BEL
		{0x08, true},  // BS
		{0x09, false}, // TAB - allowed
		{0x0A, false}, // LF - allowed
		{0x0B, true},  // VT
		{0x0C, true},  // FF
		{0x0D, false}, // CR - allowed
		{0x0E, true},  // SO
		{0x1F, true},  // US
		{0x20, false}, // Space - not a control char
		{0x41, false}, // 'A' - not a control char
		{0x7F, false}, // DEL - not in C0 range
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("U+%04X", tt.char), func(t *testing.T) {
			got := v.isBannedControlChar(tt.char)
			if got != tt.banned {
				t.Errorf("isBannedControlChar(0x%02X) = %v, want %v", tt.char, got, tt.banned)
			}
		})
	}
}

func TestEncodingValidator_MultipleControlCharsOnlyReportsFirst(t *testing.T) {
	// Test that we report only the first control character per field
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{
						Tag:   "NAME",
						Value: "John\x01\x02\x03Smith",
					},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateControlCharacters(doc)

	// We only report the first control character per field
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue (first control char only), got %d", len(issues))
	}

	if len(issues) > 0 && issues[0].Details["character"] != "U+0001" {
		t.Errorf("Expected first control char U+0001, got %s", issues[0].Details["character"])
	}
}

func TestEncodingValidator_MultipleFieldsWithControlChars(t *testing.T) {
	// Test that we report control chars in multiple fields
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{
						Tag:   "NAME",
						Value: "John\x01Smith",
					},
					{
						Tag:   "OCCU",
						Value: "Farmer\x02",
					},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateControlCharacters(doc)

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues (one per field), got %d", len(issues))
	}
}

func TestEncodingValidator_EmptyTagValue(t *testing.T) {
	// Test that empty tag values don't cause issues
	v := NewEncodingValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{
						Tag:   "NAME",
						Value: "",
					},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateControlCharacters(doc)

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for empty value, got %d", len(issues))
	}
}
