package gedcom

import "testing"

// TestFirstOfAccessors covers the nil-safe accessors added by issue #714, which
// replace indexing Names[0] and looping over Events/Attributes per lookup.
func TestFirstOfAccessors(t *testing.T) {
	t.Run("nil receivers", func(t *testing.T) {
		var i *Individual
		if i.Name() != nil || i.FullName() != "" || i.BirthPlace() != "" || i.DeathPlace() != "" ||
			i.EventOf(EventBirth) != nil || i.AttributeOf(AttributeOccupation) != nil {
			t.Error("nil *Individual accessors returned non-zero values")
		}
		var f *Family
		if f.MarriageEvent() != nil {
			t.Error("nil *Family MarriageEvent() returned non-nil")
		}
	})

	t.Run("empty individual", func(t *testing.T) {
		i := &Individual{Names: []*PersonalName{nil}, Events: []*Event{nil}, Attributes: []*Attribute{nil}}
		if i.Name() != nil || i.FullName() != "" || i.BirthPlace() != "" || i.DeathPlace() != "" ||
			i.EventOf(EventBirth) != nil || i.AttributeOf(AttributeOccupation) != nil {
			t.Error("accessors on an individual with only nil entries returned non-zero values")
		}
	})

	t.Run("populated individual", func(t *testing.T) {
		first := &PersonalName{Full: "John /Doe/"}
		occu := &Attribute{Type: AttributeOccupation, Value: "Smith"}
		chr := &Event{Type: EventChristening}
		i := &Individual{
			Names: []*PersonalName{nil, first, {Full: "Johnny /Doe/"}},
			Events: []*Event{
				{Type: EventBirth, IsNegative: true, PlaceDetail: &PlaceDetail{Name: "Nowhere"}},
				{Type: EventBirth, PlaceDetail: &PlaceDetail{Name: "Boston"}},
				{Type: EventDeath},
				chr,
			},
			Attributes: []*Attribute{nil, {Type: AttributeEducation}, occu},
		}
		if got := i.Name(); got != first {
			t.Errorf("Name() = %v, want first non-nil name", got)
		}
		if got := i.FullName(); got != "John /Doe/" {
			t.Errorf("FullName() = %q, want %q", got, "John /Doe/")
		}
		if got := i.BirthPlace(); got != "Boston" {
			t.Errorf("BirthPlace() = %q, want %q (negative assertion skipped)", got, "Boston")
		}
		if got := i.DeathPlace(); got != "" {
			t.Errorf("DeathPlace() = %q, want empty for a death with no place", got)
		}
		if got := i.EventOf(EventChristening); got != chr {
			t.Errorf("EventOf(CHR) = %v, want the christening event", got)
		}
		if got := i.EventOf(EventBurial); got != nil {
			t.Errorf("EventOf(BURI) = %v, want nil", got)
		}
		if got := i.AttributeOf(AttributeOccupation); got != occu {
			t.Errorf("AttributeOf(OCCU) = %v, want the occupation", got)
		}
	})

	t.Run("family marriage", func(t *testing.T) {
		marr := &Event{Type: EventMarriage}
		f := &Family{Events: []*Event{nil, {Type: EventMarriage, IsNegative: true}, {Type: EventDivorce}, marr}}
		if got := f.MarriageEvent(); got != marr {
			t.Errorf("MarriageEvent() = %v, want the first occurred marriage", got)
		}
		if got := (&Family{}).MarriageEvent(); got != nil {
			t.Errorf("MarriageEvent() on empty family = %v, want nil", got)
		}
	})
}
