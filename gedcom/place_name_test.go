package gedcom

import "testing"

// TestPlaceNameAccessors covers the nil-safe read path added by issue #506.
// Reading a place used to be `ev.Place`, a scalar with a safe zero value;
// PlaceDetail is a pointer and now the sole place carrier, so reading it
// without an accessor turns a one-liner into a nil check at every call site.
func TestPlaceNameAccessors(t *testing.T) {
	tests := []struct {
		name   string
		detail *PlaceDetail
		want   string
	}{
		{
			name: "no place recorded",
			want: "",
		},
		{
			name:   "PlaceDetail with a name",
			detail: &PlaceDetail{Name: "Boston, Suffolk, Massachusetts, USA"},
			want:   "Boston, Suffolk, Massachusetts, USA",
		},
		{
			name:   "PlaceDetail with empty Name",
			detail: &PlaceDetail{Form: "City, State"},
			want:   "",
		},
		{
			name:   "PlaceDetail with coordinates but no Name",
			detail: &PlaceDetail{Coordinates: &Coordinates{Latitude: "N42.3601"}},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run("Event/"+tt.name, func(t *testing.T) {
			ev := &Event{Type: EventBirth, PlaceDetail: tt.detail}
			if got := ev.PlaceName(); got != tt.want {
				t.Errorf("Event.PlaceName() = %q, want %q", got, tt.want)
			}
		})

		t.Run("Attribute/"+tt.name, func(t *testing.T) {
			attr := &Attribute{Type: "OCCU", PlaceDetail: tt.detail}
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

// TestSetPlaceName covers the nil-safe write path, the counterpart of the read
// accessors above. PlaceDetail is a pointer and the sole place carrier, so a
// caller migrating off the removed Place scalar who assigns through the pointer
// directly panics on a freshly built Event, and one who skips the allocation
// leaves PlaceDetail nil and the encoder emits no PLAC line at all.
func TestSetPlaceName(t *testing.T) {
	t.Run("allocates PlaceDetail when absent", func(t *testing.T) {
		ev := &Event{Type: EventBirth}
		ev.SetPlaceName("Boston, MA")

		if ev.PlaceDetail == nil {
			t.Fatal("SetPlaceName did not allocate PlaceDetail")
		}
		if got := ev.PlaceName(); got != "Boston, MA" {
			t.Errorf("PlaceName() = %q, want %q", got, "Boston, MA")
		}
	})

	t.Run("preserves Form and Coordinates", func(t *testing.T) {
		coords := &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"}
		ev := &Event{
			Type:        EventBirth,
			PlaceDetail: &PlaceDetail{Name: "Old", Form: "City, State", Coordinates: coords},
		}
		ev.SetPlaceName("Boston, MA")

		if got := ev.PlaceDetail.Name; got != "Boston, MA" {
			t.Errorf("Name = %q, want %q", got, "Boston, MA")
		}
		if got := ev.PlaceDetail.Form; got != "City, State" {
			t.Errorf("Form = %q, want it preserved", got)
		}
		if ev.PlaceDetail.Coordinates != coords {
			t.Error("Coordinates were not preserved")
		}
	})

	t.Run("empty name on a placeless event allocates nothing", func(t *testing.T) {
		// A non-nil PlaceDetail with an empty Name encodes as a bare PLAC line,
		// so allocating one here would turn `ev.Place = ""` -- a no-op in v2 --
		// into a placeless PLAC in the output. Threading an optional place
		// through the setter must stay a no-op.
		ev := &Event{Type: EventBirth}
		ev.SetPlaceName("")

		if ev.PlaceDetail != nil {
			t.Errorf(`SetPlaceName("") allocated PlaceDetail %+v, want nil`, ev.PlaceDetail)
		}
	})

	t.Run("empty name on an existing carrier blanks the name", func(t *testing.T) {
		// Once a carrier exists it is kept: the caller has recorded a place, and
		// dropping the carrier here would discard Form and Coordinates with it.
		coords := &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"}
		ev := &Event{
			Type:        EventBirth,
			PlaceDetail: &PlaceDetail{Name: "Boston, MA", Coordinates: coords},
		}
		ev.SetPlaceName("")

		if ev.PlaceDetail == nil {
			t.Fatal(`SetPlaceName("") dropped an existing PlaceDetail`)
		}
		if got := ev.PlaceDetail.Name; got != "" {
			t.Errorf("Name = %q, want empty", got)
		}
		if ev.PlaceDetail.Coordinates != coords {
			t.Error("Coordinates were not preserved")
		}
	})

	t.Run("Attribute allocates and reads back", func(t *testing.T) {
		attr := &Attribute{Type: "OCCU", Value: "Farmer"}
		attr.SetPlaceName("Iowa")

		if attr.PlaceDetail == nil {
			t.Fatal("SetPlaceName did not allocate PlaceDetail")
		}
		if got := attr.PlaceName(); got != "Iowa" {
			t.Errorf("PlaceName() = %q, want %q", got, "Iowa")
		}
	})

	t.Run("Attribute empty name allocates nothing", func(t *testing.T) {
		attr := &Attribute{Type: "OCCU", Value: "Farmer"}
		attr.SetPlaceName("")

		if attr.PlaceDetail != nil {
			t.Errorf(`SetPlaceName("") allocated PlaceDetail %+v, want nil`, attr.PlaceDetail)
		}
	})

	t.Run("nil Event is safe", func(t *testing.T) {
		var ev *Event
		ev.SetPlaceName("Boston, MA") // must not panic
	})

	t.Run("nil Attribute is safe", func(t *testing.T) {
		var attr *Attribute
		attr.SetPlaceName("Iowa") // must not panic
	})
}
