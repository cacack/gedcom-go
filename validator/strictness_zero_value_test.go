package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestStrictnessZeroValueIsNormal pins the renumbering from issue #489.
// StrictnessRelaxed used to be the zero value while three doc comments
// promised Normal, so any option struct that left Strictness unset silently
// discarded every warning and info issue. apidiff cannot see a constant's
// value change, so this test is the guard.
func TestStrictnessZeroValueIsNormal(t *testing.T) {
	var s Strictness
	if s != StrictnessNormal {
		t.Errorf("zero Strictness = %d, want StrictnessNormal (%d)", s, StrictnessNormal)
	}
	if StrictnessRelaxed == StrictnessNormal || StrictnessStrict == StrictnessNormal ||
		StrictnessRelaxed == StrictnessStrict {
		t.Errorf("Strictness constants collided: Normal=%d Relaxed=%d Strict=%d",
			StrictnessNormal, StrictnessRelaxed, StrictnessStrict)
	}
}

// warningFixture returns a 5.5.1 document that trips MISSING_SUBM (no header
// SUBM) and XREF_TOO_LONG (a 22-character identifier, against the 20-character
// MaxXRefLength), both at SeverityWarning.
func warningFixture() *gedcom.Document {
	const longXRef = "@I012345678901234567890@"
	rec := &gedcom.Record{
		XRef:   longXRef,
		Type:   gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: longXRef},
		Tags:   []*gedcom.Tag{{Level: 1, Tag: "_VENDOR", Value: "x"}},
	}
	return &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{rec},
		XRefMap: map[string]*gedcom.Record{longXRef: rec},
	}
}

func codesReported(issues []Issue) map[string]bool {
	seen := make(map[string]bool, len(issues))
	for _, issue := range issues {
		seen[issue.Code] = true
	}
	return seen
}

// TestUnsetStrictnessReportsWarnings asserts the behaviour the renumbering
// buys: an option struct that never mentions Strictness reports warnings.
// Under the old numbering every code below was silently dropped.
func TestUnsetStrictnessReportsWarnings(t *testing.T) {
	doc := warningFixture()

	t.Run("bare ValidateOptions", func(t *testing.T) {
		got := codesReported(NewWithOptions(&ValidateOptions{}).ValidateAll(doc))
		for _, code := range []string{CodeMissingSUBM, CodeXRefTooLong} {
			if !got[code] {
				t.Errorf("%s not reported with Strictness unset", code)
			}
		}
	})

	// CodeUnknownCustomTag additionally needs a registry and ValidateCustomTags;
	// Strictness stays unset, which is what this asserts.
	t.Run("custom tag validation enabled, Strictness unset", func(t *testing.T) {
		opts := &ValidateOptions{
			TagRegistry:        NewTagRegistry(),
			ValidateCustomTags: true,
		}
		got := codesReported(NewWithOptions(opts).ValidateAll(doc))
		if !got[CodeUnknownCustomTag] {
			t.Error("UNKNOWN_CUSTOM_TAG not reported with Strictness unset")
		}
	})

	// The package doc example at the top of streaming.go constructs the
	// streaming validator from a bare StreamingOptions. It stores opts
	// verbatim and defaults nothing, so the zero value must itself be Normal.
	// The streaming checks only emit SeverityError today, so the filter is
	// exercised directly rather than through a fixture that cannot trip it.
	t.Run("bare StreamingOptions", func(t *testing.T) {
		sv := NewStreamingValidator(StreamingOptions{})
		if sv.opts.Strictness != StrictnessNormal {
			t.Fatalf("zero StreamingOptions.Strictness = %d, want StrictnessNormal", sv.opts.Strictness)
		}

		warning := NewIssue(SeverityWarning, CodeMissingSUBM, "synthetic", "")
		if got := sv.filterByStrictness([]Issue{warning}); len(got) != 1 {
			t.Errorf("a zero-value StreamingOptions dropped a warning: %v", got)
		}
	})
}

// TestStrictnessRelaxedStillSuppressesWarnings keeps the renumbering honest:
// Relaxed must still mean errors only, it just is no longer what you get by
// omission.
func TestStrictnessRelaxedStillSuppressesWarnings(t *testing.T) {
	doc := warningFixture()
	got := codesReported(NewWithOptions(&ValidateOptions{Strictness: StrictnessRelaxed}).ValidateAll(doc))
	for _, code := range []string{CodeMissingSUBM, CodeXRefTooLong} {
		if got[code] {
			t.Errorf("%s reported under StrictnessRelaxed, want suppressed", code)
		}
	}
}
