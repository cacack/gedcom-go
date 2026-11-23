package gedcom

import (
	"reflect"
	"testing"
)

// TestPlaceParts covers the form-aware place accessor added by issue #874:
// parts in file order, labeled only when the file's form has a label for
// every part.
func TestPlaceParts(t *testing.T) {
	const headerForm = "Town, District, Region, Nation"

	tests := []struct {
		name        string
		detail      *PlaceDetail
		defaultForm string
		want        PlaceParts
	}{
		{
			name:   "own form",
			detail: &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA", Form: "City, County, State, Country"},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston", Label: "City"},
					{Name: "Suffolk", Label: "County"},
					{Name: "Massachusetts", Label: "State"},
					{Name: "USA", Label: "Country"},
				},
				Labels: []string{"City", "County", "State", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:        "header fallback",
			detail:      &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA"},
			defaultForm: headerForm,
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston", Label: "Town"},
					{Name: "Suffolk", Label: "District"},
					{Name: "Massachusetts", Label: "Region"},
					{Name: "USA", Label: "Nation"},
				},
				Labels: []string{"Town", "District", "Region", "Nation"},
				Source: PlaceFormHeader,
			},
		},
		{
			name:        "own form wins over header",
			detail:      &PlaceDetail{Name: "Boston, USA", Form: "City, Country"},
			defaultForm: headerForm,
			want: PlaceParts{
				Parts:  []PlacePart{{Name: "Boston", Label: "City"}, {Name: "USA", Label: "Country"}},
				Labels: []string{"City", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:        "whitespace-only own form falls back to header",
			detail:      &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA", Form: " \t "},
			defaultForm: headerForm,
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston", Label: "Town"},
					{Name: "Suffolk", Label: "District"},
					{Name: "Massachusetts", Label: "Region"},
					{Name: "USA", Label: "Nation"},
				},
				Labels: []string{"Town", "District", "Region", "Nation"},
				Source: PlaceFormHeader,
			},
		},
		{
			name:        "whitespace-only default form is no form",
			detail:      &PlaceDetail{Name: "Boston, USA"},
			defaultForm: "   ",
			want: PlaceParts{
				Parts: []PlacePart{{Name: "Boston"}, {Name: "USA"}},
			},
		},
		{
			name:   "empty parts kept",
			detail: &PlaceDetail{Name: ", , Utah, USA", Form: "City, County, State, Country"},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "", Label: "City"},
					{Name: "", Label: "County"},
					{Name: "Utah", Label: "State"},
					{Name: "USA", Label: "Country"},
				},
				Labels: []string{"City", "County", "State", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:   "empty label kept at its position",
			detail: &PlaceDetail{Name: "Boston, Suffolk, USA", Form: "City, , Country"},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston", Label: "City"},
					{Name: "Suffolk", Label: ""},
					{Name: "USA", Label: "Country"},
				},
				Labels: []string{"City", "", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:   "mismatch: more parts than labels",
			detail: &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA", Form: "City, Country"},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston"}, {Name: "Suffolk"}, {Name: "Massachusetts"}, {Name: "USA"},
				},
				Labels:   []string{"City", "Country"},
				Source:   PlaceFormOwn,
				Mismatch: true,
			},
		},
		{
			name:        "mismatch: fewer parts than labels",
			detail:      &PlaceDetail{Name: "Massachusetts, USA"},
			defaultForm: headerForm,
			want: PlaceParts{
				Parts:    []PlacePart{{Name: "Massachusetts"}, {Name: "USA"}},
				Labels:   []string{"Town", "District", "Region", "Nation"},
				Source:   PlaceFormHeader,
				Mismatch: true,
			},
		},
		{
			name:   "no form at all",
			detail: &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA"},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston"}, {Name: "Suffolk"}, {Name: "Massachusetts"}, {Name: "USA"},
				},
			},
		},
		{
			name:   "empty name without a form",
			detail: &PlaceDetail{},
			want:   PlaceParts{},
		},
		{
			name:        "empty name with a form",
			detail:      &PlaceDetail{Name: "  ", Form: "City, Country"},
			defaultForm: headerForm,
			want: PlaceParts{
				Labels: []string{"City", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:   "whitespace trimmed from parts and labels",
			detail: &PlaceDetail{Name: "  Boston ,Suffolk\t,  USA  ", Form: " City,County ,\tCountry "},
			want: PlaceParts{
				Parts: []PlacePart{
					{Name: "Boston", Label: "City"},
					{Name: "Suffolk", Label: "County"},
					{Name: "USA", Label: "Country"},
				},
				Labels: []string{"City", "County", "Country"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:        "nil PlaceDetail",
			detail:      nil,
			defaultForm: headerForm,
			want:        PlaceParts{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.detail.Parts(tt.defaultForm)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parts(%q) = %+v, want %+v", tt.defaultForm, got, tt.want)
			}
		})
	}
}

// TestDocumentPlaceParts covers the Document wrapper, which supplies
// Header.PlaceForm as the default form and tolerates a missing document or
// header.
func TestDocumentPlaceParts(t *testing.T) {
	place := &PlaceDetail{Name: "Boston, USA"}
	unlabeled := PlaceParts{Parts: []PlacePart{{Name: "Boston"}, {Name: "USA"}}}

	tests := []struct {
		name  string
		doc   *Document
		place *PlaceDetail
		want  PlaceParts
	}{
		{
			name:  "nil document",
			doc:   nil,
			place: place,
			want:  unlabeled,
		},
		{
			name:  "nil header",
			doc:   &Document{},
			place: place,
			want:  unlabeled,
		},
		{
			name:  "header without a form",
			doc:   &Document{Header: &Header{}},
			place: place,
			want:  unlabeled,
		},
		{
			name:  "header form",
			doc:   &Document{Header: &Header{PlaceForm: "City, Country"}},
			place: place,
			want: PlaceParts{
				Parts:  []PlacePart{{Name: "Boston", Label: "City"}, {Name: "USA", Label: "Country"}},
				Labels: []string{"City", "Country"},
				Source: PlaceFormHeader,
			},
		},
		{
			name:  "own form wins over header form",
			doc:   &Document{Header: &Header{PlaceForm: "City, Country"}},
			place: &PlaceDetail{Name: "Boston, USA", Form: "Town, Nation"},
			want: PlaceParts{
				Parts:  []PlacePart{{Name: "Boston", Label: "Town"}, {Name: "USA", Label: "Nation"}},
				Labels: []string{"Town", "Nation"},
				Source: PlaceFormOwn,
			},
		},
		{
			name:  "nil place",
			doc:   &Document{Header: &Header{PlaceForm: "City, Country"}},
			place: nil,
			want:  PlaceParts{},
		},
		{
			name:  "nil document and nil place",
			doc:   nil,
			place: nil,
			want:  PlaceParts{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.doc.PlaceParts(tt.place)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PlaceParts() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
