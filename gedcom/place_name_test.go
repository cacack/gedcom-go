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

// TestSetPlaceName covers the write path that lets a caller migrate an
// `ev.Place = name` assignment ahead of v3, where the scalar is gone and
// PlaceDetail is the only carrier.
func TestSetPlaceName(t *testing.T) {
	t.Run("allocates PlaceDetail on an event that has none", func(t *testing.T) {
		ev := &Event{Type: EventBirth}
		ev.SetPlaceName("Boston, MA")

		if ev.PlaceDetail == nil {
			t.Fatal("PlaceDetail was not allocated")
		}
		if got := ev.PlaceName(); got != "Boston, MA" {
			t.Errorf("PlaceName() = %q, want %q", got, "Boston, MA")
		}
	})

	t.Run("writes both carriers so the encoder cannot emit a stale name", func(t *testing.T) {
		// The shape a decoded event has: both carriers filled from one PLAC
		// line. The encoder prefers the scalar, so a setter that touched only
		// PlaceDetail would re-encode the old name -- and v3, with one carrier,
		// would write the new one. Same call site, two results.
		ev := &Event{Type: EventBirth, Place: "Old", PlaceDetail: &PlaceDetail{Name: "Old"}}
		ev.SetPlaceName("New")

		if ev.Place != "New" {
			t.Errorf("Place = %q, want %q", ev.Place, "New")
		}
		if ev.PlaceDetail.Name != "New" {
			t.Errorf("PlaceDetail.Name = %q, want %q", ev.PlaceDetail.Name, "New")
		}
	})

	t.Run("preserves Form and Coordinates", func(t *testing.T) {
		coords := &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"}
		ev := &Event{
			Type:        EventBirth,
			PlaceDetail: &PlaceDetail{Name: "Old", Form: "City, State", Coordinates: coords},
		}
		ev.SetPlaceName("New")

		if ev.PlaceDetail.Form != "City, State" {
			t.Errorf("Form = %q, want it preserved", ev.PlaceDetail.Form)
		}
		if ev.PlaceDetail.Coordinates != coords {
			t.Error("Coordinates were replaced")
		}
	})

	t.Run("empty name records no place and allocates nothing", func(t *testing.T) {
		ev := &Event{Type: EventBirth}
		ev.SetPlaceName("")

		if ev.PlaceDetail != nil {
			t.Errorf("PlaceDetail = %+v, want nil for an empty name", ev.PlaceDetail)
		}
		if got := ev.PlaceName(); got != "" {
			t.Errorf("PlaceName() = %q, want empty", got)
		}
	})

	t.Run("Attribute records the name", func(t *testing.T) {
		attr := &Attribute{Type: "OCCU"}
		attr.SetPlaceName("Salem, MA")

		if got := attr.PlaceName(); got != "Salem, MA" {
			t.Errorf("PlaceName() = %q, want %q", got, "Salem, MA")
		}
	})

	t.Run("nil receivers are safe", func(t *testing.T) {
		var ev *Event
		ev.SetPlaceName("Boston, MA")

		var attr *Attribute
		attr.SetPlaceName("Boston, MA")
	})
}
