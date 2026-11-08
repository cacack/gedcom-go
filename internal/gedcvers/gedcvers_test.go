package gedcvers

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestParse pins that VERS 7.0.N is GEDCOM 7.0 when N is one or more ASCII
// digits (leading zeros accepted), and that malformed values stay unrecognized
// (#587).
func TestParse(t *testing.T) {
	tests := map[string]gedcom.Version{
		"5.5":      gedcom.Version55,
		"5.5.1":    gedcom.Version551,
		"7.0":      gedcom.Version70,
		"7.0.0":    gedcom.Version70,
		"7.0.1":    gedcom.Version70,
		"7.0.14":   gedcom.Version70,
		"7.0.999":  gedcom.Version70,
		"7.0.01":   gedcom.Version70, // leading zeros accepted deliberately
		" 7.0.14 ": gedcom.Version70,
		"":         "",
		"7.0.":     "",
		"7.0.x":    "",
		"7.0.-1":   "",
		"7.0.+1":   "",
		"7.0.1a":   "",
		"7.0.1.2":  "",
		"7.0. 1":   "",
		"7.0.١":    "", // non-ASCII digit
		"7.1":      "",
		"7.1.0":    "",
		"7":        "",
	}
	for value, want := range tests {
		t.Run(value, func(t *testing.T) {
			if got := Parse(value); got != want {
				t.Errorf("Parse(%q) = %q, want %q", value, got, want)
			}
		})
	}
}
