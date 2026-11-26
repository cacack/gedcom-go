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

// TestSamePersonalName covers what SamePersonalName ignores (case, spacing,
// punctuation, the slashes) and what it does not (words, their order).
func TestSamePersonalName(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", "John /Smith/", "John /Smith/", true},
		{"case, spacing, punctuation, slashes", "John /Smith/ Jr.", "john  smith, jr", true},
		{"slashes moved", "/John/ Smith", "John /Smith/", true},
		{"no space before slash", "John/Smith/", "John Smith", true},
		{"tab and surrounding spaces", " John\t/Smith/ ", "John /Smith/", true},
		{"case folded beyond ASCII", "José /Ñúñez/", "JOSÉ /ÑÚÑEZ/", true},
		{"different word", "John /Smith/", "Jon /Smith/", false},
		{"word order", "John Smith", "Smith John", false},
		{"extra word", "John /Smith/", "John /Smith/ Jr.", false},
		{"apostrophe splits a word", "O'Brien", "OBrien", false},
		{"curly and straight apostrophe", "O'Brien", "O’Brien", true},
		{"accent kept", "José", "Jose", false},
		{"combining mark is part of a word", "José Smith", "JOSÉ smith", true},
		{"digits", "Henry 8", "Henry VIII", false},
		{"both empty", "", "", false},
		{"only punctuation", "//", "//", false},
		{"one empty", "John", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SamePersonalName(tt.a, tt.b); got != tt.want {
				t.Errorf("SamePersonalName(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if got := SamePersonalName(tt.b, tt.a); got != tt.want {
				t.Errorf("SamePersonalName(%q, %q) = %v, want %v", tt.b, tt.a, got, tt.want)
			}
		})
	}
}
