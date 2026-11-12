package converter

import (
	"errors"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertErrorSentinels checks each ConvertWithOptions failure wraps its
// sentinel and keeps the message it had before the sentinels existed.
func TestConvertErrorSentinels(t *testing.T) {
	lossy := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{{
			XRef: "@I1@",
			Tags: []*gedcom.Tag{{Level: 0, Tag: "INDI"}, {Level: 1, Tag: "EXID", Value: "external-id"}},
		}},
	}
	tests := []struct {
		name    string
		doc     *gedcom.Document
		target  gedcom.Version
		opts    *ConvertOptions
		want    error
		message string
	}{
		{"nil document", nil, gedcom.Version70, nil, ErrNilDocument, "document is nil"},
		{"invalid target", &gedcom.Document{}, "6.0", nil, ErrInvalidVersion, "invalid target version: 6.0"},
		{"unknown source", &gedcom.Document{Header: &gedcom.Header{Version: "4.0"}}, gedcom.Version70, nil,
			ErrUnsupportedConversion, "unsupported conversion: 4.0 to 7.0"},
		{"strict data loss", lossy, gedcom.Version55, &ConvertOptions{StrictDataLoss: true}, ErrDataLoss,
			"conversion would result in data loss (strict mode enabled)"},
	}
	sentinels := []error{ErrNilDocument, ErrInvalidVersion, ErrUnsupportedConversion, ErrDataLoss}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ConvertWithOptions(tt.doc, tt.target, tt.opts)
			if err == nil {
				t.Fatalf("error = nil, want %v", tt.want)
			}
			for _, s := range sentinels {
				if got := errors.Is(err, s); got != (s == tt.want) {
					t.Errorf("errors.Is(%q, %v) = %t, want %t", err, s, got, s == tt.want)
				}
			}
			if err.Error() != tt.message {
				t.Errorf("error = %q, want %q", err, tt.message)
			}
		})
	}
}
