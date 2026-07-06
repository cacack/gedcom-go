package decoder

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestFoldContinuation documents the CONT/CONC fold contract shared by
// appendRecordNote and parseSharedNote (issue #331).
func TestFoldContinuation(t *testing.T) {
	tests := []struct {
		name string
		base string
		tag  *gedcom.Tag
		want string
	}{
		{"CONT joins with newline", "a", &gedcom.Tag{Tag: "CONT", Value: "b"}, "a\nb"},
		{"CONT onto empty base keeps leading newline", "", &gedcom.Tag{Tag: "CONT", Value: "b"}, "\nb"},
		{"CONC concatenates directly", "a", &gedcom.Tag{Tag: "CONC", Value: "b"}, "ab"},
		{"other tag is a no-op", "a", &gedcom.Tag{Tag: "NOTE", Value: "b"}, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := foldContinuation(tt.base, tt.tag); got != tt.want {
				t.Errorf("foldContinuation(%q, %s) = %q, want %q", tt.base, tt.tag.Tag, got, tt.want)
			}
		})
	}
}
