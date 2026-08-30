package gedcom

import "testing"

// TestNumberOfChildrenAccessor covers the accessor and setter that replaced the
// removed NumberOfChildren field (issue #485). The field was a second store for
// a fact Attributes already held, and the encoder preferred the attribute, so
// writing the field on a decoded family was silently discarded.
func TestNumberOfChildrenAccessor(t *testing.T) {
	t.Run("nil Family is safe", func(t *testing.T) {
		var f *Family
		if got := f.NumberOfChildren(); got != "" {
			t.Errorf("NumberOfChildren() on nil = %q, want empty", got)
		}
		f.SetNumberOfChildren("3") // must not panic
	})

	t.Run("zero Family reports empty", func(t *testing.T) {
		if got := (&Family{}).NumberOfChildren(); got != "" {
			t.Errorf("NumberOfChildren() = %q, want empty", got)
		}
	})

	t.Run("reads the NCHI attribute", func(t *testing.T) {
		f := &Family{Attributes: []*Attribute{
			{Type: "RESI", Value: "Farmhouse"},
			{Type: "NCHI", Value: "4"},
		}}
		if got := f.NumberOfChildren(); got != "4" {
			t.Errorf("NumberOfChildren() = %q, want %q", got, "4")
		}
	})

	t.Run("setter appends when absent", func(t *testing.T) {
		f := &Family{}
		f.SetNumberOfChildren("2")
		if got := f.NumberOfChildren(); got != "2" {
			t.Errorf("NumberOfChildren() = %q, want %q", got, "2")
		}
		if len(f.Attributes) != 1 {
			t.Fatalf("len(Attributes) = %d, want 1", len(f.Attributes))
		}
	})

	t.Run("setter updates in place and keeps subordinates", func(t *testing.T) {
		f := &Family{Attributes: []*Attribute{
			{Type: "NCHI", Value: "2", SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}}},
		}}
		f.SetNumberOfChildren("5")

		if len(f.Attributes) != 1 {
			t.Fatalf("setter added a second NCHI entry: %d attributes", len(f.Attributes))
		}
		if got := f.NumberOfChildren(); got != "5" {
			t.Errorf("NumberOfChildren() = %q, want %q", got, "5")
		}
		if len(f.Attributes[0].SourceCitations) != 1 {
			t.Error("setter dropped the attribute's subordinates")
		}
	})

	t.Run("nil attribute entries are skipped", func(t *testing.T) {
		f := &Family{Attributes: []*Attribute{nil, {Type: "NCHI", Value: "1"}}}
		if got := f.NumberOfChildren(); got != "1" {
			t.Errorf("NumberOfChildren() = %q, want %q", got, "1")
		}
	})
}
