package gedcom

import "testing"

func TestIndividual_BirthEvent(t *testing.T) {
	tests := []struct {
		name   string
		events []*Event
		want   *Event
	}{
		{
			name: "has birth event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventBirth, Date: "1 JAN 1850"},
		},
		{
			name: "multiple birth events returns first",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventBirth, Date: "2 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventBirth, Date: "1 JAN 1850"},
		},
		{
			name: "no birth event",
			events: []*Event{
				{Type: EventDeath, Date: "1 JAN 1920"},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.BirthEvent()

			if tt.want == nil {
				if got != nil {
					t.Errorf("BirthEvent() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("BirthEvent() = nil, want %v", tt.want)
				return
			}

			if got.Type != tt.want.Type || got.Date != tt.want.Date {
				t.Errorf("BirthEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_DeathEvent(t *testing.T) {
	tests := []struct {
		name   string
		events []*Event
		want   *Event
	}{
		{
			name: "has death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventDeath, Date: "1 JAN 1920"},
		},
		{
			name: "multiple death events returns first",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
				{Type: EventDeath, Date: "2 JAN 1920"},
			},
			want: &Event{Type: EventDeath, Date: "1 JAN 1920"},
		},
		{
			name: "no death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.DeathEvent()

			if tt.want == nil {
				if got != nil {
					t.Errorf("DeathEvent() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("DeathEvent() = nil, want %v", tt.want)
				return
			}

			if got.Type != tt.want.Type || got.Date != tt.want.Date {
				t.Errorf("DeathEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_BirthDate(t *testing.T) {
	birthDate := mustParseDate("1 JAN 1850")
	deathDate := mustParseDate("1 JAN 1920")

	tests := []struct {
		name   string
		events []*Event
		want   *Date
	}{
		{
			name: "has birth date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: birthDate,
		},
		{
			name: "birth event without parsed date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: nil},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: nil,
		},
		{
			name: "no birth event",
			events: []*Event{
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.BirthDate()

			if tt.want == nil {
				if got != nil {
					t.Errorf("BirthDate() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("BirthDate() = nil, want %v", tt.want)
				return
			}

			if got.Original != tt.want.Original {
				t.Errorf("BirthDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_DeathDate(t *testing.T) {
	birthDate := mustParseDate("1 JAN 1850")
	deathDate := mustParseDate("1 JAN 1920")

	tests := []struct {
		name   string
		events []*Event
		want   *Date
	}{
		{
			name: "has death date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: deathDate,
		},
		{
			name: "death event without parsed date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: nil},
			},
			want: nil,
		},
		{
			name: "no death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.DeathDate()

			if tt.want == nil {
				if got != nil {
					t.Errorf("DeathDate() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("DeathDate() = nil, want %v", tt.want)
				return
			}

			if got.Original != tt.want.Original {
				t.Errorf("DeathDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIndividual_FamilySearchURL tests the FamilySearchURL helper method.
// This returns the FamilySearch.org URL for the individual's record.
// Ref: Issue #80
func TestIndividual_FamilySearchURL(t *testing.T) {
	tests := []struct {
		name           string
		familySearchID string
		want           string
	}{
		{
			name:           "typical ID",
			familySearchID: "KWCJ-QN7",
			want:           "https://www.familysearch.org/tree/person/details/KWCJ-QN7",
		},
		{
			name:           "another ID",
			familySearchID: "ABCD-123",
			want:           "https://www.familysearch.org/tree/person/details/ABCD-123",
		},
		{
			name:           "empty ID returns empty string",
			familySearchID: "",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{FamilySearchID: tt.familySearchID}
			got := i.FamilySearchURL()

			if got != tt.want {
				t.Errorf("FamilySearchURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper function to create a test document with individuals and families for relationship tests
func createRelationshipTestDocument(individuals []*Individual, families []*Family) *Document {
	doc := &Document{
		XRefMap: make(map[string]*Record),
	}
	for _, ind := range individuals {
		r := &Record{Type: RecordTypeIndividual, Entity: ind}
		doc.Records = append(doc.Records, r)
		doc.XRefMap[ind.XRef] = r
	}
	for _, fam := range families {
		r := &Record{Type: RecordTypeFamily, Entity: fam}
		doc.Records = append(doc.Records, r)
		doc.XRefMap[fam.XRef] = r
	}
	return doc
}

// TestIndividual_Parents tests the Parents relationship traversal method.
func TestIndividual_Parents(t *testing.T) {
	// Create test individuals
	father := &Individual{XRef: "@I1@", Names: []*PersonalName{{Full: "John /Doe/"}}}
	mother := &Individual{XRef: "@I2@", Names: []*PersonalName{{Full: "Jane /Doe/"}}}
	child := &Individual{
		XRef:            "@I3@",
		Names:           []*PersonalName{{Full: "Billy /Doe/"}},
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}},
	}
	family := &Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@"}}

	// Individual with multiple parental families (adoption scenario)
	adoptedChild := &Individual{
		XRef:  "@I10@",
		Names: []*PersonalName{{Full: "Adopted /Child/"}},
		ChildInFamilies: []FamilyLink{
			{FamilyXRef: "@F1@", Pedigree: "birth"},
			{FamilyXRef: "@F2@", Pedigree: "adopted"},
		},
	}
	adoptiveFather := &Individual{XRef: "@I11@", Names: []*PersonalName{{Full: "Adoptive /Father/"}}}
	adoptiveMother := &Individual{XRef: "@I12@", Names: []*PersonalName{{Full: "Adoptive /Mother/"}}}
	adoptiveFamily := &Family{XRef: "@F2@", Husband: "@I11@", Wife: "@I12@", Children: []string{"@I10@"}}

	// Individual with no parents
	orphan := &Individual{XRef: "@I4@", Names: []*PersonalName{{Full: "Orphan /Child/"}}}

	// Individual with invalid family xref
	invalidFamChild := &Individual{
		XRef:            "@I5@",
		Names:           []*PersonalName{{Full: "Invalid /FamRef/"}},
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@INVALID@"}},
	}

	// Family with only father (no mother)
	singleFather := &Individual{XRef: "@I6@", Names: []*PersonalName{{Full: "Single /Father/"}}}
	childOfSingleFather := &Individual{
		XRef:            "@I7@",
		Names:           []*PersonalName{{Full: "Child /OfSingleFather/"}},
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@F3@"}},
	}
	singleFatherFamily := &Family{XRef: "@F3@", Husband: "@I6@", Children: []string{"@I7@"}}

	// Family with only mother (no father)
	singleMother := &Individual{XRef: "@I8@", Names: []*PersonalName{{Full: "Single /Mother/"}}}
	childOfSingleMother := &Individual{
		XRef:            "@I9@",
		Names:           []*PersonalName{{Full: "Child /OfSingleMother/"}},
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@F4@"}},
	}
	singleMotherFamily := &Family{XRef: "@F4@", Wife: "@I8@", Children: []string{"@I9@"}}

	tests := []struct {
		name        string
		individual  *Individual
		doc         *Document
		wantXRefs   []string
		wantCount   int
		description string
	}{
		{
			name:       "child with both parents",
			individual: child,
			doc: createRelationshipTestDocument(
				[]*Individual{father, mother, child},
				[]*Family{family},
			),
			wantXRefs:   []string{"@I1@", "@I2@"},
			wantCount:   2,
			description: "Returns both father and mother",
		},
		{
			name:       "multiple parental families (adoption)",
			individual: adoptedChild,
			doc: createRelationshipTestDocument(
				[]*Individual{father, mother, adoptedChild, adoptiveFather, adoptiveMother},
				[]*Family{family, adoptiveFamily},
			),
			wantXRefs:   []string{"@I1@", "@I2@", "@I11@", "@I12@"},
			wantCount:   4,
			description: "Returns all parents from all parental families",
		},
		{
			name:       "child with only father",
			individual: childOfSingleFather,
			doc: createRelationshipTestDocument(
				[]*Individual{singleFather, childOfSingleFather},
				[]*Family{singleFatherFamily},
			),
			wantXRefs:   []string{"@I6@"},
			wantCount:   1,
			description: "Returns only father when no mother exists",
		},
		{
			name:       "child with only mother",
			individual: childOfSingleMother,
			doc: createRelationshipTestDocument(
				[]*Individual{singleMother, childOfSingleMother},
				[]*Family{singleMotherFamily},
			),
			wantXRefs:   []string{"@I8@"},
			wantCount:   1,
			description: "Returns only mother when no father exists",
		},
		{
			name:        "no parental families",
			individual:  orphan,
			doc:         createRelationshipTestDocument([]*Individual{orphan}, nil),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when individual has no FAMC links",
		},
		{
			name:        "nil document",
			individual:  child,
			doc:         nil,
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns nil when document is nil",
		},
		{
			name:       "invalid family xref",
			individual: invalidFamChild,
			doc: createRelationshipTestDocument(
				[]*Individual{invalidFamChild},
				[]*Family{family},
			),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid family xrefs gracefully",
		},
		{
			name: "invalid parent xref in family",
			individual: &Individual{
				XRef:            "@I100@",
				ChildInFamilies: []FamilyLink{{FamilyXRef: "@F100@"}},
			},
			doc: createRelationshipTestDocument(
				[]*Individual{{XRef: "@I100@"}},
				[]*Family{{XRef: "@F100@", Husband: "@INVALID@", Wife: "@ALSO_INVALID@"}},
			),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid individual xrefs in family gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.individual.Parents(tt.doc)

			if len(got) != tt.wantCount {
				t.Errorf("Parents() returned %d individuals, want %d (%s)", len(got), tt.wantCount, tt.description)
				return
			}

			if tt.wantXRefs != nil {
				for i, xref := range tt.wantXRefs {
					if got[i].XRef != xref {
						t.Errorf("Parents()[%d].XRef = %q, want %q", i, got[i].XRef, xref)
					}
				}
			}
		})
	}
}

// TestIndividual_Spouses tests the Spouses relationship traversal method.
func TestIndividual_Spouses(t *testing.T) {
	// Basic married couple
	husband := &Individual{
		XRef:             "@I1@",
		Names:            []*PersonalName{{Full: "John /Doe/"}},
		SpouseInFamilies: []string{"@F1@"},
	}
	wife := &Individual{
		XRef:             "@I2@",
		Names:            []*PersonalName{{Full: "Jane /Doe/"}},
		SpouseInFamilies: []string{"@F1@"},
	}
	family := &Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@"}

	// Person with multiple spouses (remarriage)
	remarriedHusband := &Individual{
		XRef:             "@I10@",
		Names:            []*PersonalName{{Full: "Robert /Andrews/"}},
		SpouseInFamilies: []string{"@F10@", "@F11@"},
	}
	firstWife := &Individual{
		XRef:             "@I11@",
		Names:            []*PersonalName{{Full: "First /Wife/"}},
		SpouseInFamilies: []string{"@F10@"},
	}
	secondWife := &Individual{
		XRef:             "@I12@",
		Names:            []*PersonalName{{Full: "Second /Wife/"}},
		SpouseInFamilies: []string{"@F11@"},
	}
	firstMarriage := &Family{XRef: "@F10@", Husband: "@I10@", Wife: "@I11@"}
	secondMarriage := &Family{XRef: "@F11@", Husband: "@I10@", Wife: "@I12@"}

	// Single person (no spouse)
	single := &Individual{
		XRef:  "@I3@",
		Names: []*PersonalName{{Full: "Single /Person/"}},
	}

	// Family with no wife (husband only)
	husbandOnly := &Individual{
		XRef:             "@I4@",
		Names:            []*PersonalName{{Full: "Husband /Only/"}},
		SpouseInFamilies: []string{"@F2@"},
	}
	familyNoWife := &Family{XRef: "@F2@", Husband: "@I4@", Children: []string{"@I5@"}}

	// Family with no husband (wife only)
	wifeOnly := &Individual{
		XRef:             "@I6@",
		Names:            []*PersonalName{{Full: "Wife /Only/"}},
		SpouseInFamilies: []string{"@F3@"},
	}
	familyNoHusband := &Family{XRef: "@F3@", Wife: "@I6@", Children: []string{"@I7@"}}

	tests := []struct {
		name        string
		individual  *Individual
		doc         *Document
		wantXRefs   []string
		wantCount   int
		description string
	}{
		{
			name:        "husband looking up wife",
			individual:  husband,
			doc:         createRelationshipTestDocument([]*Individual{husband, wife}, []*Family{family}),
			wantXRefs:   []string{"@I2@"},
			wantCount:   1,
			description: "Husband returns wife as spouse",
		},
		{
			name:        "wife looking up husband",
			individual:  wife,
			doc:         createRelationshipTestDocument([]*Individual{husband, wife}, []*Family{family}),
			wantXRefs:   []string{"@I1@"},
			wantCount:   1,
			description: "Wife returns husband as spouse",
		},
		{
			name:       "multiple spouses (remarriage)",
			individual: remarriedHusband,
			doc: createRelationshipTestDocument(
				[]*Individual{remarriedHusband, firstWife, secondWife},
				[]*Family{firstMarriage, secondMarriage},
			),
			wantXRefs:   []string{"@I11@", "@I12@"},
			wantCount:   2,
			description: "Returns all spouses from multiple marriages in order",
		},
		{
			name:        "single person",
			individual:  single,
			doc:         createRelationshipTestDocument([]*Individual{single}, nil),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice for unmarried person",
		},
		{
			name:        "nil document",
			individual:  husband,
			doc:         nil,
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns nil when document is nil",
		},
		{
			name:        "family with no wife",
			individual:  husbandOnly,
			doc:         createRelationshipTestDocument([]*Individual{husbandOnly}, []*Family{familyNoWife}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when family has no wife",
		},
		{
			name:        "family with no husband",
			individual:  wifeOnly,
			doc:         createRelationshipTestDocument([]*Individual{wifeOnly}, []*Family{familyNoHusband}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when family has no husband",
		},
		{
			name: "invalid family xref",
			individual: &Individual{
				XRef:             "@I100@",
				SpouseInFamilies: []string{"@INVALID@"},
			},
			doc:         createRelationshipTestDocument([]*Individual{{XRef: "@I100@"}}, nil),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid family xrefs gracefully",
		},
		{
			name: "invalid spouse xref in family",
			individual: &Individual{
				XRef:             "@I100@",
				SpouseInFamilies: []string{"@F100@"},
			},
			doc: createRelationshipTestDocument(
				[]*Individual{{XRef: "@I100@"}},
				[]*Family{{XRef: "@F100@", Husband: "@I100@", Wife: "@INVALID@"}},
			),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid spouse xrefs gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.individual.Spouses(tt.doc)

			if len(got) != tt.wantCount {
				t.Errorf("Spouses() returned %d individuals, want %d (%s)", len(got), tt.wantCount, tt.description)
				return
			}

			if tt.wantXRefs != nil {
				for i, xref := range tt.wantXRefs {
					if got[i].XRef != xref {
						t.Errorf("Spouses()[%d].XRef = %q, want %q", i, got[i].XRef, xref)
					}
				}
			}
		})
	}
}

// TestIndividual_Children tests the Children relationship traversal method.
func TestIndividual_Children(t *testing.T) {
	// Parent with children
	parent := &Individual{
		XRef:             "@I1@",
		Names:            []*PersonalName{{Full: "John /Doe/"}},
		SpouseInFamilies: []string{"@F1@"},
	}
	child1 := &Individual{XRef: "@I2@", Names: []*PersonalName{{Full: "Child /One/"}}}
	child2 := &Individual{XRef: "@I3@", Names: []*PersonalName{{Full: "Child /Two/"}}}
	child3 := &Individual{XRef: "@I4@", Names: []*PersonalName{{Full: "Child /Three/"}}}
	family := &Family{XRef: "@F1@", Husband: "@I1@", Children: []string{"@I2@", "@I3@", "@I4@"}}

	// Parent with children from multiple families
	remarriedParent := &Individual{
		XRef:             "@I10@",
		Names:            []*PersonalName{{Full: "Remarried /Parent/"}},
		SpouseInFamilies: []string{"@F10@", "@F11@"},
	}
	childFirstMarriage := &Individual{XRef: "@I11@", Names: []*PersonalName{{Full: "First /Marriage/"}}}
	childSecondMarriage := &Individual{XRef: "@I12@", Names: []*PersonalName{{Full: "Second /Marriage/"}}}
	firstFamily := &Family{XRef: "@F10@", Husband: "@I10@", Children: []string{"@I11@"}}
	secondFamily := &Family{XRef: "@F11@", Husband: "@I10@", Children: []string{"@I12@"}}

	// Person with no children
	childless := &Individual{
		XRef:             "@I5@",
		Names:            []*PersonalName{{Full: "Childless /Person/"}},
		SpouseInFamilies: []string{"@F2@"},
	}
	childlessFamily := &Family{XRef: "@F2@", Husband: "@I5@"}

	// Person who is not a spouse (not in any family as spouse)
	notASpouse := &Individual{XRef: "@I6@", Names: []*PersonalName{{Full: "Not /ASpouse/"}}}

	tests := []struct {
		name        string
		individual  *Individual
		doc         *Document
		wantXRefs   []string
		wantCount   int
		description string
	}{
		{
			name:        "parent with multiple children",
			individual:  parent,
			doc:         createRelationshipTestDocument([]*Individual{parent, child1, child2, child3}, []*Family{family}),
			wantXRefs:   []string{"@I2@", "@I3@", "@I4@"},
			wantCount:   3,
			description: "Returns all children in order",
		},
		{
			name:       "children from multiple families",
			individual: remarriedParent,
			doc: createRelationshipTestDocument(
				[]*Individual{remarriedParent, childFirstMarriage, childSecondMarriage},
				[]*Family{firstFamily, secondFamily},
			),
			wantXRefs:   []string{"@I11@", "@I12@"},
			wantCount:   2,
			description: "Returns children from all families in order",
		},
		{
			name:        "person with no children",
			individual:  childless,
			doc:         createRelationshipTestDocument([]*Individual{childless}, []*Family{childlessFamily}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when no children",
		},
		{
			name:        "person not a spouse",
			individual:  notASpouse,
			doc:         createRelationshipTestDocument([]*Individual{notASpouse}, nil),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when not a spouse in any family",
		},
		{
			name:        "nil document",
			individual:  parent,
			doc:         nil,
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns nil when document is nil",
		},
		{
			name: "invalid family xref",
			individual: &Individual{
				XRef:             "@I100@",
				SpouseInFamilies: []string{"@INVALID@"},
			},
			doc:         createRelationshipTestDocument([]*Individual{{XRef: "@I100@"}}, nil),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid family xrefs gracefully",
		},
		{
			name: "invalid child xref in family",
			individual: &Individual{
				XRef:             "@I100@",
				SpouseInFamilies: []string{"@F100@"},
			},
			doc: createRelationshipTestDocument(
				[]*Individual{{XRef: "@I100@"}},
				[]*Family{{XRef: "@F100@", Husband: "@I100@", Children: []string{"@INVALID@"}}},
			),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid child xrefs gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.individual.Children(tt.doc)

			if len(got) != tt.wantCount {
				t.Errorf("Children() returned %d individuals, want %d (%s)", len(got), tt.wantCount, tt.description)
				return
			}

			if tt.wantXRefs != nil {
				for i, xref := range tt.wantXRefs {
					if got[i].XRef != xref {
						t.Errorf("Children()[%d].XRef = %q, want %q", i, got[i].XRef, xref)
					}
				}
			}
		})
	}
}

// TestIndividual_ParentalFamilies tests the ParentalFamilies relationship traversal method.
func TestIndividual_ParentalFamilies(t *testing.T) {
	// Individual with one parental family
	singleFamily := &Individual{
		XRef:            "@I1@",
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}},
	}
	family1 := &Family{XRef: "@F1@", Husband: "@I2@", Wife: "@I3@"}

	// Individual with multiple parental families (adoption scenario)
	multipleFamily := &Individual{
		XRef: "@I10@",
		ChildInFamilies: []FamilyLink{
			{FamilyXRef: "@F1@", Pedigree: "birth"},
			{FamilyXRef: "@F2@", Pedigree: "adopted"},
		},
	}
	family2 := &Family{XRef: "@F2@", Husband: "@I4@", Wife: "@I5@"}

	// Individual with no parental families
	noFamily := &Individual{XRef: "@I20@"}

	// Individual with invalid family xref
	invalidFamily := &Individual{
		XRef:            "@I30@",
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@INVALID@"}},
	}

	tests := []struct {
		name        string
		individual  *Individual
		doc         *Document
		wantXRefs   []string
		wantCount   int
		description string
	}{
		{
			name:        "single parental family",
			individual:  singleFamily,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   []string{"@F1@"},
			wantCount:   1,
			description: "Returns single parental family",
		},
		{
			name:        "multiple parental families",
			individual:  multipleFamily,
			doc:         createRelationshipTestDocument(nil, []*Family{family1, family2}),
			wantXRefs:   []string{"@F1@", "@F2@"},
			wantCount:   2,
			description: "Returns all parental families in order",
		},
		{
			name:        "no parental families",
			individual:  noFamily,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when no parental families",
		},
		{
			name:        "nil document",
			individual:  singleFamily,
			doc:         nil,
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns nil when document is nil",
		},
		{
			name:        "invalid family xref",
			individual:  invalidFamily,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid family xrefs gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.individual.ParentalFamilies(tt.doc)

			if len(got) != tt.wantCount {
				t.Errorf("ParentalFamilies() returned %d families, want %d (%s)", len(got), tt.wantCount, tt.description)
				return
			}

			if tt.wantXRefs != nil {
				for i, xref := range tt.wantXRefs {
					if got[i].XRef != xref {
						t.Errorf("ParentalFamilies()[%d].XRef = %q, want %q", i, got[i].XRef, xref)
					}
				}
			}
		})
	}
}

// TestIndividual_SpouseFamilies tests the SpouseFamilies relationship traversal method.
func TestIndividual_SpouseFamilies(t *testing.T) {
	// Individual with one spouse family
	singleSpouse := &Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@"},
	}
	family1 := &Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@"}

	// Individual with multiple spouse families (remarriage)
	multipleSpouse := &Individual{
		XRef:             "@I10@",
		SpouseInFamilies: []string{"@F1@", "@F2@"},
	}
	family2 := &Family{XRef: "@F2@", Husband: "@I10@", Wife: "@I3@"}

	// Individual with no spouse families
	noSpouse := &Individual{XRef: "@I20@"}

	// Individual with invalid family xref
	invalidFamily := &Individual{
		XRef:             "@I30@",
		SpouseInFamilies: []string{"@INVALID@"},
	}

	tests := []struct {
		name        string
		individual  *Individual
		doc         *Document
		wantXRefs   []string
		wantCount   int
		description string
	}{
		{
			name:        "single spouse family",
			individual:  singleSpouse,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   []string{"@F1@"},
			wantCount:   1,
			description: "Returns single spouse family",
		},
		{
			name:        "multiple spouse families",
			individual:  multipleSpouse,
			doc:         createRelationshipTestDocument(nil, []*Family{family1, family2}),
			wantXRefs:   []string{"@F1@", "@F2@"},
			wantCount:   2,
			description: "Returns all spouse families in order",
		},
		{
			name:        "no spouse families",
			individual:  noSpouse,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns empty slice when no spouse families",
		},
		{
			name:        "nil document",
			individual:  singleSpouse,
			doc:         nil,
			wantXRefs:   nil,
			wantCount:   0,
			description: "Returns nil when document is nil",
		},
		{
			name:        "invalid family xref",
			individual:  invalidFamily,
			doc:         createRelationshipTestDocument(nil, []*Family{family1}),
			wantXRefs:   nil,
			wantCount:   0,
			description: "Skips invalid family xrefs gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.individual.SpouseFamilies(tt.doc)

			if len(got) != tt.wantCount {
				t.Errorf("SpouseFamilies() returned %d families, want %d (%s)", len(got), tt.wantCount, tt.description)
				return
			}

			if tt.wantXRefs != nil {
				for i, xref := range tt.wantXRefs {
					if got[i].XRef != xref {
						t.Errorf("SpouseFamilies()[%d].XRef = %q, want %q", i, got[i].XRef, xref)
					}
				}
			}
		})
	}
}
