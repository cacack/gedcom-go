package gedcom

import "testing"

// Helper function to create a test document with individuals and families for family tests
func createFamilyTestDocument() *Document {
	// Create individuals
	i1 := &Individual{XRef: "@I1@"}
	i2 := &Individual{XRef: "@I2@"}
	i3 := &Individual{XRef: "@I3@"}
	i4 := &Individual{XRef: "@I4@"}
	i5 := &Individual{XRef: "@I5@"}
	i6 := &Individual{XRef: "@I6@"}
	i7 := &Individual{XRef: "@I7@"}
	i8 := &Individual{XRef: "@I8@"}
	i9 := &Individual{XRef: "@I9@"}
	i10 := &Individual{XRef: "@I10@"}

	// Create families
	f1 := &Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@", "@I4@"}}
	f2 := &Family{XRef: "@F2@", Husband: "@I5@", Children: []string{"@I6@"}} // husband only
	f3 := &Family{XRef: "@F3@", Wife: "@I7@", Children: []string{"@I8@"}}    // wife only
	f4 := &Family{XRef: "@F4@", Husband: "@I9@", Wife: "@I10@"}              // no children
	f5 := &Family{XRef: "@F5@"}                                              // empty family

	// Create records
	records := []*Record{
		{Type: RecordTypeIndividual, Entity: i1},
		{Type: RecordTypeIndividual, Entity: i2},
		{Type: RecordTypeIndividual, Entity: i3},
		{Type: RecordTypeIndividual, Entity: i4},
		{Type: RecordTypeIndividual, Entity: i5},
		{Type: RecordTypeIndividual, Entity: i6},
		{Type: RecordTypeIndividual, Entity: i7},
		{Type: RecordTypeIndividual, Entity: i8},
		{Type: RecordTypeIndividual, Entity: i9},
		{Type: RecordTypeIndividual, Entity: i10},
		{Type: RecordTypeFamily, Entity: f1},
		{Type: RecordTypeFamily, Entity: f2},
		{Type: RecordTypeFamily, Entity: f3},
		{Type: RecordTypeFamily, Entity: f4},
		{Type: RecordTypeFamily, Entity: f5},
	}

	// Build XRefMap
	xrefMap := make(map[string]*Record)
	for _, r := range records {
		switch v := r.Entity.(type) {
		case *Individual:
			xrefMap[v.XRef] = r
		case *Family:
			xrefMap[v.XRef] = r
		}
	}

	return &Document{
		Records: records,
		XRefMap: xrefMap,
	}
}

func TestFamily_HusbandIndividual(t *testing.T) {
	doc := createFamilyTestDocument()

	tests := []struct {
		name     string
		family   *Family
		doc      *Document
		wantXRef string
		wantNil  bool
	}{
		{
			name:     "normal family with husband",
			family:   &Family{Husband: "@I1@"},
			doc:      doc,
			wantXRef: "@I1@",
		},
		{
			name:    "family without husband (wife only)",
			family:  &Family{Wife: "@I7@"},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "nil document",
			family:  &Family{Husband: "@I1@"},
			doc:     nil,
			wantNil: true,
		},
		{
			name:    "invalid husband xref",
			family:  &Family{Husband: "@INVALID@"},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "empty husband xref",
			family:  &Family{Husband: ""},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "empty family",
			family:  &Family{},
			doc:     doc,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.family.HusbandIndividual(tt.doc)

			if tt.wantNil {
				if got != nil {
					t.Errorf("HusbandIndividual() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("HusbandIndividual() = nil, want individual with XRef %s", tt.wantXRef)
				return
			}

			if got.XRef != tt.wantXRef {
				t.Errorf("HusbandIndividual().XRef = %s, want %s", got.XRef, tt.wantXRef)
			}
		})
	}
}

func TestFamily_WifeIndividual(t *testing.T) {
	doc := createFamilyTestDocument()

	tests := []struct {
		name     string
		family   *Family
		doc      *Document
		wantXRef string
		wantNil  bool
	}{
		{
			name:     "normal family with wife",
			family:   &Family{Wife: "@I2@"},
			doc:      doc,
			wantXRef: "@I2@",
		},
		{
			name:    "family without wife (husband only)",
			family:  &Family{Husband: "@I5@"},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "nil document",
			family:  &Family{Wife: "@I2@"},
			doc:     nil,
			wantNil: true,
		},
		{
			name:    "invalid wife xref",
			family:  &Family{Wife: "@INVALID@"},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "empty wife xref",
			family:  &Family{Wife: ""},
			doc:     doc,
			wantNil: true,
		},
		{
			name:    "empty family",
			family:  &Family{},
			doc:     doc,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.family.WifeIndividual(tt.doc)

			if tt.wantNil {
				if got != nil {
					t.Errorf("WifeIndividual() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("WifeIndividual() = nil, want individual with XRef %s", tt.wantXRef)
				return
			}

			if got.XRef != tt.wantXRef {
				t.Errorf("WifeIndividual().XRef = %s, want %s", got.XRef, tt.wantXRef)
			}
		})
	}
}

func TestFamily_ChildrenIndividuals(t *testing.T) {
	doc := createFamilyTestDocument()

	tests := []struct {
		name      string
		family    *Family
		doc       *Document
		wantXRefs []string
	}{
		{
			name:      "normal family with children",
			family:    &Family{Children: []string{"@I3@", "@I4@"}},
			doc:       doc,
			wantXRefs: []string{"@I3@", "@I4@"},
		},
		{
			name:      "family with one child",
			family:    &Family{Children: []string{"@I6@"}},
			doc:       doc,
			wantXRefs: []string{"@I6@"},
		},
		{
			name:      "family with no children",
			family:    &Family{Husband: "@I9@", Wife: "@I10@"},
			doc:       doc,
			wantXRefs: []string{},
		},
		{
			name:      "nil document",
			family:    &Family{Children: []string{"@I3@", "@I4@"}},
			doc:       nil,
			wantXRefs: []string{},
		},
		{
			name:      "family with invalid xref (filters out)",
			family:    &Family{Children: []string{"@I3@", "@INVALID@", "@I4@"}},
			doc:       doc,
			wantXRefs: []string{"@I3@", "@I4@"},
		},
		{
			name:      "family with all invalid xrefs",
			family:    &Family{Children: []string{"@INVALID1@", "@INVALID2@"}},
			doc:       doc,
			wantXRefs: []string{},
		},
		{
			name:      "empty family",
			family:    &Family{},
			doc:       doc,
			wantXRefs: []string{},
		},
		{
			name:      "nil children slice",
			family:    &Family{Children: nil},
			doc:       doc,
			wantXRefs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.family.ChildrenIndividuals(tt.doc)

			if len(got) != len(tt.wantXRefs) {
				t.Errorf("ChildrenIndividuals() returned %d individuals, want %d", len(got), len(tt.wantXRefs))
				return
			}

			for i, ind := range got {
				if ind.XRef != tt.wantXRefs[i] {
					t.Errorf("ChildrenIndividuals()[%d].XRef = %s, want %s", i, ind.XRef, tt.wantXRefs[i])
				}
			}
		})
	}
}

func TestFamily_AllMembers(t *testing.T) {
	doc := createFamilyTestDocument()

	tests := []struct {
		name      string
		family    *Family
		doc       *Document
		wantXRefs []string
	}{
		{
			name:      "full family (husband, wife, children)",
			family:    &Family{Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@", "@I4@"}},
			doc:       doc,
			wantXRefs: []string{"@I1@", "@I2@", "@I3@", "@I4@"},
		},
		{
			name:      "husband only with child",
			family:    &Family{Husband: "@I5@", Children: []string{"@I6@"}},
			doc:       doc,
			wantXRefs: []string{"@I5@", "@I6@"},
		},
		{
			name:      "wife only with child",
			family:    &Family{Wife: "@I7@", Children: []string{"@I8@"}},
			doc:       doc,
			wantXRefs: []string{"@I7@", "@I8@"},
		},
		{
			name:      "married couple no children",
			family:    &Family{Husband: "@I9@", Wife: "@I10@"},
			doc:       doc,
			wantXRefs: []string{"@I9@", "@I10@"},
		},
		{
			name:      "empty family",
			family:    &Family{},
			doc:       doc,
			wantXRefs: []string{},
		},
		{
			name:      "nil document",
			family:    &Family{Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@"}},
			doc:       nil,
			wantXRefs: []string{},
		},
		{
			name:      "family with invalid husband xref (filters out)",
			family:    &Family{Husband: "@INVALID@", Wife: "@I2@", Children: []string{"@I3@"}},
			doc:       doc,
			wantXRefs: []string{"@I2@", "@I3@"},
		},
		{
			name:      "family with invalid wife xref (filters out)",
			family:    &Family{Husband: "@I1@", Wife: "@INVALID@", Children: []string{"@I3@"}},
			doc:       doc,
			wantXRefs: []string{"@I1@", "@I3@"},
		},
		{
			name:      "family with some invalid child xrefs (filters out)",
			family:    &Family{Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@", "@INVALID@"}},
			doc:       doc,
			wantXRefs: []string{"@I1@", "@I2@", "@I3@"},
		},
		{
			name:      "family with all invalid xrefs",
			family:    &Family{Husband: "@INVALID1@", Wife: "@INVALID2@", Children: []string{"@INVALID3@"}},
			doc:       doc,
			wantXRefs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.family.AllMembers(tt.doc)

			if len(got) != len(tt.wantXRefs) {
				t.Errorf("AllMembers() returned %d individuals, want %d", len(got), len(tt.wantXRefs))
				return
			}

			for i, ind := range got {
				if ind.XRef != tt.wantXRefs[i] {
					t.Errorf("AllMembers()[%d].XRef = %s, want %s", i, ind.XRef, tt.wantXRefs[i])
				}
			}
		})
	}
}

// TestFamily_OrderPreservation verifies that order is preserved correctly.
func TestFamily_OrderPreservation(t *testing.T) {
	doc := createFamilyTestDocument()

	t.Run("AllMembers returns husband, wife, children in order", func(t *testing.T) {
		family := &Family{
			Husband:  "@I1@",
			Wife:     "@I2@",
			Children: []string{"@I3@", "@I4@"},
		}

		got := family.AllMembers(doc)

		expectedOrder := []string{"@I1@", "@I2@", "@I3@", "@I4@"}
		if len(got) != len(expectedOrder) {
			t.Fatalf("AllMembers() returned %d members, want %d", len(got), len(expectedOrder))
		}

		for i, ind := range got {
			if ind.XRef != expectedOrder[i] {
				t.Errorf("AllMembers()[%d].XRef = %s, want %s", i, ind.XRef, expectedOrder[i])
			}
		}
	})

	t.Run("ChildrenIndividuals preserves GEDCOM file order", func(t *testing.T) {
		// Children should appear in the order they were defined in the GEDCOM
		family := &Family{
			Children: []string{"@I4@", "@I3@"}, // Reverse order
		}

		got := family.ChildrenIndividuals(doc)

		expectedOrder := []string{"@I4@", "@I3@"}
		if len(got) != len(expectedOrder) {
			t.Fatalf("ChildrenIndividuals() returned %d children, want %d", len(got), len(expectedOrder))
		}

		for i, ind := range got {
			if ind.XRef != expectedOrder[i] {
				t.Errorf("ChildrenIndividuals()[%d].XRef = %s, want %s", i, ind.XRef, expectedOrder[i])
			}
		}
	})
}
