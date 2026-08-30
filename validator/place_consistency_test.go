package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

func TestPlaceConsistencyValidator(t *testing.T) {
	tests := []struct {
		name   string
		place  string
		detail *gedcom.PlaceDetail
		want   bool // want an issue
	}{
		{
			name:   "carriers disagree",
			place:  "Boston, MA",
			detail: &gedcom.PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA"},
			want:   true,
		},
		{
			name:   "carriers agree",
			place:  "Boston, MA",
			detail: &gedcom.PlaceDetail{Name: "Boston, MA"},
			want:   false,
		},
		{
			name:   "scalar only",
			place:  "Boston, MA",
			detail: nil,
			want:   false,
		},
		{
			name:   "detail only",
			detail: &gedcom.PlaceDetail{Name: "Boston, MA"},
			want:   false,
		},
		{
			name:   "detail with empty name",
			place:  "Boston, MA",
			detail: &gedcom.PlaceDetail{Coordinates: &gedcom.Coordinates{Latitude: "N42.3601"}},
			want:   false,
		},
		{
			name:  "no place at all",
			place: "",
			want:  false,
		},
	}

	v := NewPlaceConsistencyValidator()

	for _, tt := range tests {
		t.Run("Event/"+tt.name, func(t *testing.T) {
			ind := &gedcom.Individual{
				XRef:   "@I1@",
				Events: []*gedcom.Event{{Type: gedcom.EventBirth, Place: tt.place, PlaceDetail: tt.detail}},
			}
			issues := v.ValidateIndividual(ind)
			if got := len(issues) > 0; got != tt.want {
				t.Fatalf("ValidateIndividual() issues = %d, want issue: %v", len(issues), tt.want)
			}
			if tt.want {
				if issues[0].Code != CodePlaceCarrierMismatch {
					t.Errorf("Code = %q, want %q", issues[0].Code, CodePlaceCarrierMismatch)
				}
				if issues[0].Severity != SeverityWarning {
					t.Errorf("Severity = %v, want SeverityWarning", issues[0].Severity)
				}
				if issues[0].RecordXRef != "@I1@" {
					t.Errorf("XRef = %q, want @I1@", issues[0].RecordXRef)
				}
				// Both values belong in the message so the caller can see which
				// one the file will get.
				for _, want := range []string{tt.place, tt.detail.Name} {
					if !strings.Contains(issues[0].Message, want) {
						t.Errorf("Message %q missing %q", issues[0].Message, want)
					}
				}
			}
		})

		t.Run("Attribute/"+tt.name, func(t *testing.T) {
			ind := &gedcom.Individual{
				XRef:       "@I1@",
				Attributes: []*gedcom.Attribute{{Type: "OCCU", Place: tt.place, PlaceDetail: tt.detail}},
			}
			issues := v.ValidateIndividual(ind)
			if got := len(issues) > 0; got != tt.want {
				t.Errorf("ValidateIndividual() issues = %d, want issue: %v", len(issues), tt.want)
			}
		})
	}
}

func TestPlaceConsistencyValidatorNilSafety(t *testing.T) {
	v := NewPlaceConsistencyValidator()

	if got := v.Validate(nil); got != nil {
		t.Errorf("Validate(nil) = %v, want nil", got)
	}
	if got := v.ValidateIndividual(nil); got != nil {
		t.Errorf("ValidateIndividual(nil) = %v, want nil", got)
	}

	// Nil elements inside the slices must not panic.
	ind := &gedcom.Individual{
		XRef:       "@I1@",
		Events:     []*gedcom.Event{nil},
		Attributes: []*gedcom.Attribute{nil},
	}
	if got := v.ValidateIndividual(ind); len(got) != 0 {
		t.Errorf("ValidateIndividual() with nil elements = %v, want none", got)
	}
}

// TestPlaceConsistencyValidatorDocument walks a document, which is the entry
// point a caller actually uses.
func TestPlaceConsistencyValidatorDocument(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{
					XRef: "@I1@",
					Events: []*gedcom.Event{{
						Type:        gedcom.EventBirth,
						Place:       "Stale Place",
						PlaceDetail: &gedcom.PlaceDetail{Name: "Updated Place"},
					}},
				},
			},
		},
	}

	issues := NewPlaceConsistencyValidator().Validate(doc)
	if len(issues) != 1 {
		t.Fatalf("Validate() issues = %d, want 1", len(issues))
	}
	if issues[0].RecordXRef != "@I1@" {
		t.Errorf("XRef = %q, want @I1@", issues[0].RecordXRef)
	}
}
