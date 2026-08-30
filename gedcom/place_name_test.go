package gedcom

import "testing"

// TestPlaceNameAccessors covers the nil-safe read path added by issue #506.
// Reading a place used to be `ev.Place`, a scalar with a safe zero value;
// PlaceDetail is a pointer, so migrating that read without an accessor turns a
// one-liner into a nil check at every call site.
//
// PlaceName prefers PlaceDetail.Name so a call site written against it stays
// correct once the legacy scalar is removed. That is the opposite of the
// encoder's precedence, which prefers the scalar to keep re-encoding a decoded
// document byte-identical; decode sets both from the same line, so the two
// orders only differ for a hand-built value whose carriers disagree.
func TestPlaceNameAccessors(t *testing.T) {
	tests := []struct {
		name   string
		place  string
		detail *PlaceDetail
		want   string
	}{
		{
			name: "no place recorded",
			want: "",
		},
		{
			name:  "legacy scalar only",
			place: "Boston, MA",
			want:  "Boston, MA",
		},
		{
			name:   "PlaceDetail only",
			detail: &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA"},
			want:   "Boston, Suffolk, Massachusetts, USA",
		},
		{
			name:   "PlaceDetail with empty Name falls back to the scalar",
			place:  "Boston, MA",
			detail: &PlaceDetail{Form: "City, State"},
			want:   "Boston, MA",
		},
		{
			name:   "PlaceDetail with empty Name and no scalar",
			detail: &PlaceDetail{Coordinates: &Coordinates{Latitude: "N42.3601"}},
			want:   "",
		},
		{
			name:   "PlaceDetail wins when both are set",
			place:  "Scalar Place",
			detail: &PlaceDetail{Name: "Detail Place"},
			want:   "Detail Place",
		},
	}

	for _, tt := range tests {
		t.Run("Event/"+tt.name, func(t *testing.T) {
			ev := &Event{Type: EventBirth, Place: tt.place, PlaceDetail: tt.detail}
			if got := ev.PlaceName(); got != tt.want {
				t.Errorf("Event.PlaceName() = %q, want %q", got, tt.want)
			}
		})

	}

	// An Attribute has a single place carrier in v2, so its accessor reads the
	// scalar. The v3 removal replaces that scalar with a PlaceDetail and the
	// accessor reads the name out of it, which is why a call site migrated to
	// PlaceName now needs no further change then.
	for _, tt := range []struct {
		name  string
		place string
		want  string
	}{
		{name: "no place recorded", want: ""},
		{name: "place recorded", place: "Boston, MA", want: "Boston, MA"},
	} {
		t.Run("Attribute/"+tt.name, func(t *testing.T) {
			attr := &Attribute{Type: "OCCU", Place: tt.place}
			if got := attr.PlaceName(); got != tt.want {
				t.Errorf("Attribute.PlaceName() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("nil Event is safe", func(t *testing.T) {
		var ev *Event
		if got := ev.PlaceName(); got != "" {
			t.Errorf("Event.PlaceName() on nil = %q, want empty", got)
		}
	})

	t.Run("nil Attribute is safe", func(t *testing.T) {
		var attr *Attribute
		if got := attr.PlaceName(); got != "" {
			t.Errorf("Attribute.PlaceName() on nil = %q, want empty", got)
		}
	})
}
