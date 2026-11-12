package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestHeaderVersionNormalization pins how a GEDC.VERS value is read (#741): a
// later GEDCOM 7 minor as 7.0 keeping its declared minor, with no diagnostic,
// and a vendor spelling of a known version as that version, with an
// INVALID_VALUE warning on the VERS line saying it was normalized; an
// unrecognized value falls back to tag detection with no diagnostic.
func TestHeaderVersionNormalization(t *testing.T) {
	tests := []struct {
		vers         string
		wantVersion  gedcom.Version
		wantDeclared gedcom.Version
		wantMessage  string // "" for no diagnostic
	}{
		{"7.1", gedcom.Version70, "7.1", ""},
		{"7.1.0", gedcom.Version70, "7.1", ""},
		{"7.0.14", gedcom.Version70, gedcom.Version70, ""},
		{"5.5.1", gedcom.Version551, gedcom.Version551, ""},
		{"5.51", gedcom.Version551, gedcom.Version551, `"5.51" is not a GEDCOM version string; normalized to 5.5.1`},
		{"551", gedcom.Version551, gedcom.Version551, `"551" is not a GEDCOM version string; normalized to 5.5.1`},
		{"5.5.5", gedcom.Version551, gedcom.Version551, `"5.5.5" is not a GEDCOM version string; normalized to 5.5.1`},
		{"5.5.1 (GEDCOM)", gedcom.Version551, gedcom.Version551, `"5.5.1 (GEDCOM)" is not a GEDCOM version string; normalized to 5.5.1`},
		{"7", gedcom.Version70, gedcom.Version70, `"7" is not a GEDCOM version string; normalized to 7.0`},
		// Unrecognized: detected by tag fallback, deliberately with no
		// INVALID_VALUE (there is no version to say it was normalized to).
		{"7.0.x", gedcom.Version55, "", ""},
		{"8.0", gedcom.Version55, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.vers, func(t *testing.T) {
			data := "0 HEAD\n1 GEDC\n2 VERS " + tt.vers + "\n0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n"
			result, err := DecodeWithDiagnostics(strings.NewReader(data), nil)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			h := result.Document.Header
			if h.Version != tt.wantVersion || h.DeclaredVersion != tt.wantDeclared {
				t.Errorf("Version, DeclaredVersion = %q, %q; want %q, %q", h.Version, h.DeclaredVersion, tt.wantVersion, tt.wantDeclared)
			}
			if !headerHasVers(h.Tags, tt.vers) {
				t.Errorf("Header.Tags lost GEDC.VERS %q", tt.vers)
			}
			var got []Diagnostic
			for _, d := range result.Diagnostics {
				if d.Code == CodeInvalidValue {
					got = append(got, d)
				}
			}
			if tt.wantMessage == "" {
				if len(got) != 0 {
					t.Errorf("unexpected INVALID_VALUE diagnostics: %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("INVALID_VALUE diagnostics = %+v, want one", got)
			}
			d := got[0]
			if d.Line != 3 || d.Severity != SeverityWarning || d.Context != tt.vers || !strings.Contains(d.Message, tt.wantMessage) {
				t.Errorf("diagnostic = %+v, want a warning on line 3 with context %q and message containing %q", d, tt.vers, tt.wantMessage)
			}
		})
	}
}

// headerHasVers reports whether tags hold a level-2 VERS with value vers.
func headerHasVers(tags []*gedcom.Tag, vers string) bool {
	for _, tag := range tags {
		if tag.Level == 2 && tag.Tag == "VERS" && tag.Value == vers {
			return true
		}
	}
	return false
}
