package gedcom

import "testing"

// Dedicated tests for tag.go so a failure points directly at the Tag
// helpers rather than at a decode pipeline several layers up. The
// HasValue/HasXRef assertions previously lived inside types_test.go's
// TestTag; they are relocated here in table-driven form.

func TestTag_HasValue(t *testing.T) {
	tests := []struct {
		name string
		tag  Tag
		want bool
	}{
		{"value present", Tag{Level: 1, Tag: "NAME", Value: "John /Doe/"}, true},
		{"empty value", Tag{Level: 1, Tag: "NAME"}, false},
		{"whitespace counts as a value", Tag{Level: 1, Tag: "NOTE", Value: " "}, true},
		{"xref set but no value", Tag{Level: 0, Tag: "INDI", XRef: "@I1@"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag.HasValue(); got != tt.want {
				t.Errorf("HasValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTag_HasXRef(t *testing.T) {
	tests := []struct {
		name string
		tag  Tag
		want bool
	}{
		{"xref present", Tag{Level: 0, Tag: "INDI", XRef: "@I1@"}, true},
		{"empty xref", Tag{Level: 1, Tag: "NAME"}, false},
		{"value set but no xref", Tag{Level: 1, Tag: "NAME", Value: "John /Doe/"}, false},
		{"both xref and value", Tag{Level: 0, Tag: "SUBM", XRef: "@U1@", Value: "x"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag.HasXRef(); got != tt.want {
				t.Errorf("HasXRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
