// encoding.go provides GEDCOM 7.0 encoding validation.
//
// This module validates GEDCOM 7.0 encoding requirements:
// - UTF-8 is the only allowed encoding for GEDCOM 7.0
// - C0 control characters (U+0000-U+001F) are banned except TAB, LF, CR

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// EncodingValidator validates GEDCOM 7.0 encoding requirements.
type EncodingValidator struct{}

// NewEncodingValidator creates a new EncodingValidator.
func NewEncodingValidator() *EncodingValidator {
	return &EncodingValidator{}
}

// Validate performs all encoding validations on the document.
// It combines results from ValidateEncoding and ValidateControlCharacters.
func (e *EncodingValidator) Validate(doc *gedcom.Document) []Issue {
	var issues []Issue
	issues = append(issues, e.ValidateEncoding(doc)...)
	issues = append(issues, e.ValidateControlCharacters(doc)...)
	return issues
}

// ValidateEncoding validates that GEDCOM 7.0 files use UTF-8 encoding.
// For GEDCOM 5.5 and 5.5.1, all encodings are allowed (returns no issues).
//
// GEDCOM 7.0 requires UTF-8 exclusively. The following encodings are rejected:
// - ANSEL
// - UTF-16LE, UTF-16BE
// - LATIN1 (ISO-8859-1)
func (e *EncodingValidator) ValidateEncoding(doc *gedcom.Document) []Issue {
	if doc == nil || doc.Header == nil {
		return nil
	}

	// Only validate encoding for GEDCOM 7.0
	if doc.Header.Version != gedcom.Version70 {
		return nil
	}

	encoding := doc.Header.Encoding

	// Empty encoding or UTF-8 are valid for GEDCOM 7.0
	if encoding == "" || encoding == gedcom.EncodingUTF8 || encoding == gedcom.EncodingASCII {
		return nil
	}

	// UNICODE is an alias for UTF-8
	if encoding == gedcom.EncodingUNICODE {
		return nil
	}

	// All other encodings are invalid for GEDCOM 7.0
	var encodingName string
	switch encoding {
	case gedcom.EncodingANSEL:
		encodingName = "ANSEL"
	case gedcom.EncodingLATIN1:
		encodingName = "LATIN1"
	default:
		// Handle string-based encoding values (e.g., "UTF-16LE", "UTF-16BE")
		encodingStr := strings.ToUpper(string(encoding))
		if strings.Contains(encodingStr, "UTF-16") || strings.Contains(encodingStr, "UTF16") {
			encodingName = encodingStr
		} else {
			encodingName = string(encoding)
		}
	}

	return []Issue{
		NewIssue(
			SeverityError,
			CodeInvalidEncodingForVersion,
			fmt.Sprintf("GEDCOM 7.0 requires UTF-8 encoding, found %s", encodingName),
			"",
		).WithDetail("encoding", encodingName).
			WithDetail("version", string(doc.Header.Version)),
	}
}

// ValidateControlCharacters scans all tag values for banned C0 control characters.
// GEDCOM 7.0 bans U+0000-U+001F except:
// - TAB (U+0009)
// - LF (U+000A)
// - CR (U+000D)
//
// For GEDCOM 5.5 and 5.5.1, no control character restrictions are enforced.
func (e *EncodingValidator) ValidateControlCharacters(doc *gedcom.Document) []Issue {
	if doc == nil || doc.Header == nil {
		return nil
	}

	// Only validate control characters for GEDCOM 7.0
	if doc.Header.Version != gedcom.Version70 {
		return nil
	}

	var issues []Issue

	// Scan header string fields
	headerFields := []struct {
		value string
		field string
	}{
		{doc.Header.SourceSystem, "SOUR"},
		{doc.Header.Language, "LANG"},
		{doc.Header.Copyright, "COPR"},
		{doc.Header.Submitter, "SUBM"},
		{doc.Header.AncestryTreeID, "_TREE"},
	}
	for _, hf := range headerFields {
		if hf.value != "" {
			if issue := e.checkControlChars(hf.value, "", hf.field); issue != nil {
				issues = append(issues, *issue)
			}
		}
	}

	// Scan header tags
	if doc.Header.Tags != nil {
		e.scanTagsForControlChars(doc.Header.Tags, "", &issues)
	}

	// Scan all records
	for _, record := range doc.Records {
		e.scanTagsForControlChars(record.Tags, record.XRef, &issues)

		// Also check the record's value field
		if record.Value != "" {
			if issue := e.checkControlChars(record.Value, record.XRef, string(record.Type)); issue != nil {
				issues = append(issues, *issue)
			}
		}
	}

	return issues
}

// scanTagsForControlChars recursively scans tags for banned control characters.
func (e *EncodingValidator) scanTagsForControlChars(tags []*gedcom.Tag, recordXRef string, issues *[]Issue) {
	for _, tag := range tags {
		if tag.Value != "" {
			if issue := e.checkControlChars(tag.Value, recordXRef, tag.Tag); issue != nil {
				*issues = append(*issues, *issue)
			}
		}
	}
}

// checkControlChars checks a string for banned C0 control characters.
// Returns an Issue if a banned character is found, nil otherwise.
func (e *EncodingValidator) checkControlChars(value, recordXRef, field string) *Issue {
	for i, r := range value {
		if e.isBannedControlChar(r) {
			issue := NewIssue(
				SeverityError,
				CodeBannedControlCharacter,
				fmt.Sprintf("banned C0 control character U+%04X in %s field", r, field),
				recordXRef,
			).WithDetail("character", fmt.Sprintf("U+%04X", r)).
				WithDetail("field", field).
				WithDetail("position", fmt.Sprintf("%d", i))
			return &issue
		}
	}
	return nil
}

// isBannedControlChar returns true if the rune is a banned C0 control character.
// Banned: U+0000-U+001F except TAB (U+0009), LF (U+000A), CR (U+000D)
func (e *EncodingValidator) isBannedControlChar(r rune) bool {
	// Allow TAB, LF, CR
	if r == 0x09 || r == 0x0A || r == 0x0D {
		return false
	}
	// Ban U+0000-U+001F
	return r >= 0x00 && r <= 0x1F
}
