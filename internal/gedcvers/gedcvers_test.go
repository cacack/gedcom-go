package gedcvers

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestResolve pins what each GEDC.VERS value names: conforming 5.5, 5.5.1 and
// GEDCOM 7 minor/patch versions (#587, #741), the spellings vendors write for a
// known version, normalized (#741), and malformed values left unrecognized.
func TestResolve(t *testing.T) {
	v55, v551, v70 := gedcom.Version55, gedcom.Version551, gedcom.Version70
	tests := map[string]Resolution{
		"5.5":      {Version: v55, Declared: v55},
		"5.5.1":    {Version: v551, Declared: v551},
		"7.0":      {Version: v70, Declared: v70},
		"7.0.0":    {Version: v70, Declared: v70},
		"7.0.1":    {Version: v70, Declared: v70},
		"7.0.14":   {Version: v70, Declared: v70},
		"7.0.999":  {Version: v70, Declared: v70},
		"7.0.01":   {Version: v70, Declared: v70}, // leading zeros accepted deliberately
		" 7.0.14 ": {Version: v70, Declared: v70},
		"7.00":     {Version: v70, Declared: v70},
		"7.1":      {Version: v70, Declared: "7.1"},
		"7.1.0":    {Version: v70, Declared: "7.1"},
		"7.12.3":   {Version: v70, Declared: "7.12"},
		"7.01":     {Version: v70, Declared: "7.1"},

		// Vendor spellings: the obvious version, normalized.
		"5.51":           {Version: v551, Declared: v551, Normalized: true},
		"551":            {Version: v551, Declared: v551, Normalized: true},
		"5.5.5":          {Version: v551, Declared: v551, Normalized: true},
		"55":             {Version: v55, Declared: v55, Normalized: true},
		"7":              {Version: v70, Declared: v70, Normalized: true},
		"5.5.1 (GEDCOM)": {Version: v551, Declared: v551, Normalized: true},
		"5.5 EL":         {Version: v55, Declared: v55, Normalized: true},
		"5.51\tfoo":      {Version: v551, Declared: v551, Normalized: true},
		"7.1 draft":      {Version: v70, Declared: "7.1", Normalized: true},

		// Unrecognized.
		"":           {},
		"7.0.":       {},
		"7.0.x":      {},
		"7.0.-1":     {},
		"7.0.+1":     {},
		"7.0.1a":     {},
		"7.0.1.2":    {},
		"7.0. 1":     {},
		"7.0.١":      {}, // non-ASCII digit
		"7.":         {},
		"7.x":        {},
		"8.0":        {},
		"5.6":        {},
		"(GEDCOM)":   {},
		"GEDCOM 5.5": {},
	}
	for value, want := range tests {
		t.Run(value, func(t *testing.T) {
			if got := Resolve(value); got != want {
				t.Errorf("Resolve(%q) = %+v, want %+v", value, got, want)
			}
			if got := Parse(value); got != want.Version {
				t.Errorf("Parse(%q) = %q, want %q", value, got, want.Version)
			}
		})
	}
}
