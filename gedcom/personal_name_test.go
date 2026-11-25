package gedcom

import "testing"

// TestSplitPersonalName covers each slash shape SplitPersonalName documents:
// none, one, a closed pair, and more than two.
func TestSplitPersonalName(t *testing.T) {
	tests := []struct {
		name, full             string
		given, surname, suffix string
	}{
		{"empty", "", "", "", ""},
		{"no slashes", " John Smith ", "John Smith", "", ""},
		{"one slash", "John /Smith ", "John", "Smith", ""},
		{"closed pair", "John /Smith/", "John", "Smith", ""},
		{"suffix after surname", "John /Smith/ Jr.", "John", "Smith", "Jr."},
		{"no given name", "/Kamanawa/ I", "", "Kamanawa", "I"},
		{"padded surname kept as written", "John / Smith /", "John", " Smith ", ""},
		{"more than two slashes", "John /de la /Smith/", "John", "Smith", ""},
		{"more than two slashes with suffix", "John /de la /Smith/ III", "John", "Smith", "III"},
		{"slash in suffix", "John /Smith/ Jr./Sr", "John", " Jr.", "Sr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			given, surname, suffix := SplitPersonalName(tt.full)
			if given != tt.given || surname != tt.surname || suffix != tt.suffix {
				t.Errorf("SplitPersonalName(%q) = %q/%q/%q, want %q/%q/%q",
					tt.full, given, surname, suffix, tt.given, tt.surname, tt.suffix)
			}
		})
	}
}
