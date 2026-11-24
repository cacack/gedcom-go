package validator

import (
	"slices"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestFindDuplicates_MissingValueCostsNoMore pins the #932 scenario: a value
// recorded on one side only costs no more than one neither side records, and
// neither costs more than two values that agree. Every pair shares the
// full name John Paul /Example/ and nothing contradicts it.
func TestFindDuplicates_MissingValueCostsNoMore(t *testing.T) {
	person := func(xref, sex string, l life) *gedcom.Individual {
		ind := lifePerson(t, xref, l)
		ind.Names = []*gedcom.PersonalName{{Full: "John Paul /Example/"}}
		ind.Sex = sex
		return ind
	}

	tests := []struct {
		name        string
		a, b        *gedcom.Individual
		want        float64
		wantReasons []string
	}{
		{
			// The issue's pair: one record has sex and a full birth date, the
			// other neither (a birth place only). It scored 0.6 before #932.
			name:        "sex and birth on one side only",
			a:           person("@I1@", "M", life{birth: "3 MAR 1850"}),
			b:           person("@I2@", "", life{birthPlace: "Springfield"}),
			want:        0.7,
			wantReasons: []string{"exact surname match", "exact given name match", "sex not recorded on both"},
		},
		{
			name:        "sex and birth on neither side",
			a:           person("@I1@", "", life{}),
			b:           person("@I2@", "U", life{}),
			want:        0.7,
			wantReasons: []string{"exact surname match", "exact given name match", "sex not recorded on both"},
		},
		{
			// The issue's comparison pair, unchanged at 0.7.
			name:        "same sex, birth on neither side",
			a:           person("@I1@", "M", life{}),
			b:           person("@I2@", "M", life{}),
			want:        0.7,
			wantReasons: []string{"exact surname match", "exact given name match", "same sex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDuplicateDetector(nil).FindDuplicates(twoIndividualDoc(tt.a, tt.b))
			if len(got) != 1 {
				t.Fatalf("got %d pairs at the default MinConfidence, want 1", len(got))
			}
			assertConfidence(t, got[0], tt.want)
			if !slices.Equal(got[0].MatchReasons, tt.wantReasons) {
				t.Errorf("reasons = %v, want %v", got[0].MatchReasons, tt.wantReasons)
			}
		})
	}
}
