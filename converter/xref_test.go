package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestNormalizeXRefsToUppercase(t *testing.T) {
	tests := []struct {
		name        string
		inputXRefs  []string
		wantXRefs   []string
		wantCount   int
	}{
		{
			name:       "lowercase to uppercase",
			inputXRefs: []string{"@i1@", "@f1@"},
			wantXRefs:  []string{"@I1@", "@F1@"},
			wantCount:  2,
		},
		{
			name:       "already uppercase",
			inputXRefs: []string{"@I1@", "@F1@"},
			wantXRefs:  []string{"@I1@", "@F1@"},
			wantCount:  0,
		},
		{
			name:       "mixed case",
			inputXRefs: []string{"@I1@", "@f2@", "@S3@"},
			wantXRefs:  []string{"@I1@", "@F2@", "@S3@"},
			wantCount:  1,
		},
		{
			name:       "empty records",
			inputXRefs: []string{},
			wantXRefs:  []string{},
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createDocWithXRefs(tt.inputXRefs)
			report := &gedcom.ConversionReport{}

			normalizeXRefsToUppercase(doc, report)

			for i, record := range doc.Records {
				if record.XRef != tt.wantXRefs[i] {
					t.Errorf("Record[%d].XRef = %v, want %v", i, record.XRef, tt.wantXRefs[i])
				}
			}

			// Check report
			if tt.wantCount > 0 {
				found := false
				for _, tr := range report.Transformations {
					if tr.Type == "XREF_UPPERCASE" {
						found = true
						if tr.Count != tt.wantCount {
							t.Errorf("Count = %d, want %d", tr.Count, tt.wantCount)
						}
					}
				}
				if !found {
					t.Error("Should have XREF_UPPERCASE transformation")
				}
			}
		})
	}
}

func TestBuildXRefMapping(t *testing.T) {
	tests := []struct {
		name       string
		inputXRefs []string
		wantCount  int
	}{
		{
			name:       "all lowercase",
			inputXRefs: []string{"@i1@", "@f1@", "@s1@"},
			wantCount:  3,
		},
		{
			name:       "all uppercase",
			inputXRefs: []string{"@I1@", "@F1@", "@S1@"},
			wantCount:  0,
		},
		{
			name:       "mixed",
			inputXRefs: []string{"@I1@", "@f1@"},
			wantCount:  1,
		},
		{
			name:       "empty XRef ignored",
			inputXRefs: []string{"", "@i1@"},
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createDocWithXRefs(tt.inputXRefs)
			mapping := buildXRefMapping(doc)

			if len(mapping) != tt.wantCount {
				t.Errorf("Mapping count = %d, want %d", len(mapping), tt.wantCount)
			}

			// Verify all mappings are to uppercase
			for old, new := range mapping {
				if new != "@"+old[1:len(old)-1]+"@" && new != old {
					// new should be uppercase version
					for i := 1; i < len(new)-1; i++ {
						c := new[i]
						if c >= 'a' && c <= 'z' {
							t.Errorf("Mapping %s -> %s should be uppercase", old, new)
						}
					}
				}
			}
		})
	}
}

func TestUpdateXRefDefinitions(t *testing.T) {
	t.Run("updates record and entity XRefs", func(t *testing.T) {
		ind := &gedcom.Individual{XRef: "@i1@"}
		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{XRef: "@i1@", Entity: ind},
			},
		}
		mapping := map[string]string{"@i1@": "@I1@"}

		updateXRefDefinitions(doc, mapping)

		if doc.Records[0].XRef != "@I1@" {
			t.Errorf("Record XRef = %v, want @I1@", doc.Records[0].XRef)
		}
		if ind.XRef != "@I1@" {
			t.Errorf("Individual XRef = %v, want @I1@", ind.XRef)
		}
	})

	t.Run("no entity is handled", func(t *testing.T) {
		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{XRef: "@i1@", Entity: nil},
			},
		}
		mapping := map[string]string{"@i1@": "@I1@"}

		updateXRefDefinitions(doc, mapping)

		if doc.Records[0].XRef != "@I1@" {
			t.Errorf("Record XRef = %v, want @I1@", doc.Records[0].XRef)
		}
	})
}

func TestUpdateEntityXRef(t *testing.T) {
	tests := []struct {
		name   string
		entity interface{}
		newXRef string
	}{
		{
			name:   "Individual",
			entity: &gedcom.Individual{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "Family",
			entity: &gedcom.Family{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "Source",
			entity: &gedcom.Source{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "Repository",
			entity: &gedcom.Repository{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "Note",
			entity: &gedcom.Note{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "MediaObject",
			entity: &gedcom.MediaObject{XRef: "@old@"},
			newXRef: "@NEW@",
		},
		{
			name:   "Submitter",
			entity: &gedcom.Submitter{XRef: "@old@"},
			newXRef: "@NEW@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateEntityXRef(tt.entity, tt.newXRef)

			// Verify XRef was updated
			switch e := tt.entity.(type) {
			case *gedcom.Individual:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.Family:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.Source:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.Repository:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.Note:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.MediaObject:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			case *gedcom.Submitter:
				if e.XRef != tt.newXRef {
					t.Errorf("XRef = %v, want %v", e.XRef, tt.newXRef)
				}
			}
		})
	}
}

func TestUpdateXRefReferences(t *testing.T) {
	t.Run("updates header submitter", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Submitter: "@subm1@",
			},
			Records: []*gedcom.Record{},
		}
		mapping := map[string]string{"@subm1@": "@SUBM1@"}

		updateXRefReferences(doc, mapping)

		if doc.Header.Submitter != "@SUBM1@" {
			t.Errorf("Submitter = %v, want @SUBM1@", doc.Header.Submitter)
		}
	})

	t.Run("updates header tag XRefs", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Tags: []*gedcom.Tag{
					{Tag: "SUBM", Value: "@subm1@"},
				},
			},
			Records: []*gedcom.Record{},
		}
		mapping := map[string]string{"@subm1@": "@SUBM1@"}

		updateXRefReferences(doc, mapping)

		if doc.Header.Tags[0].Value != "@SUBM1@" {
			t.Errorf("Tag value = %v, want @SUBM1@", doc.Header.Tags[0].Value)
		}
	})

	t.Run("updates record tag XRefs", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{
					Tags: []*gedcom.Tag{
						{Tag: "FAMC", Value: "@f1@"},
					},
				},
			},
		}
		mapping := map[string]string{"@f1@": "@F1@"}

		updateXRefReferences(doc, mapping)

		if doc.Records[0].Tags[0].Value != "@F1@" {
			t.Errorf("Tag value = %v, want @F1@", doc.Records[0].Tags[0].Value)
		}
	})
}

func TestUpdateXRefInTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      *gedcom.Tag
		mapping  map[string]string
		wantValue string
	}{
		{
			name:     "nil tag",
			tag:      nil,
			mapping:  map[string]string{"@i1@": "@I1@"},
			wantValue: "",
		},
		{
			name:     "XRef value updated",
			tag:      &gedcom.Tag{Value: "@i1@"},
			mapping:  map[string]string{"@i1@": "@I1@"},
			wantValue: "@I1@",
		},
		{
			name:     "non-XRef value unchanged",
			tag:      &gedcom.Tag{Value: "regular text"},
			mapping:  map[string]string{"@i1@": "@I1@"},
			wantValue: "regular text",
		},
		{
			name:     "XRef not in mapping unchanged",
			tag:      &gedcom.Tag{Value: "@i2@"},
			mapping:  map[string]string{"@i1@": "@I1@"},
			wantValue: "@i2@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateXRefInTag(tt.tag, tt.mapping)

			if tt.tag != nil && tt.tag.Value != tt.wantValue {
				t.Errorf("Value = %v, want %v", tt.tag.Value, tt.wantValue)
			}
		})
	}
}

func TestUpdateEntityReferences(t *testing.T) {
	mapping := map[string]string{
		"@i1@": "@I1@",
		"@f1@": "@F1@",
		"@s1@": "@S1@",
		"@m1@": "@M1@",
		"@n1@": "@N1@",
		"@r1@": "@R1@",
	}

	t.Run("updates Individual references", func(t *testing.T) {
		ind := &gedcom.Individual{
			ChildInFamilies:  []gedcom.FamilyLink{{FamilyXRef: "@f1@"}},
			SpouseInFamilies: []string{"@f1@"},
			Associations: []*gedcom.Association{
				{IndividualXRef: "@i1@", SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}}},
			},
			SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}},
			Notes:           []string{"@n1@"},
			Media:           []*gedcom.MediaLink{{MediaXRef: "@m1@"}},
			LDSOrdinances:   []*gedcom.LDSOrdinance{{FamilyXRef: "@f1@"}},
			Events: []*gedcom.Event{
				{SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}}},
			},
			Attributes: []*gedcom.Attribute{
				{SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}}},
			},
			Tags: []*gedcom.Tag{{Value: "@i1@"}},
		}

		updateEntityReferences(ind, mapping)

		if ind.ChildInFamilies[0].FamilyXRef != "@F1@" {
			t.Errorf("ChildInFamilies XRef = %v, want @F1@", ind.ChildInFamilies[0].FamilyXRef)
		}
		if ind.SpouseInFamilies[0] != "@F1@" {
			t.Errorf("SpouseInFamilies XRef = %v, want @F1@", ind.SpouseInFamilies[0])
		}
		if ind.Associations[0].IndividualXRef != "@I1@" {
			t.Errorf("Association XRef = %v, want @I1@", ind.Associations[0].IndividualXRef)
		}
	})

	t.Run("updates Family references", func(t *testing.T) {
		fam := &gedcom.Family{
			Husband:         "@i1@",
			Wife:            "@i1@",
			Children:        []string{"@i1@"},
			SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}},
			Notes:           []string{"@n1@"},
			Media:           []*gedcom.MediaLink{{MediaXRef: "@m1@"}},
			LDSOrdinances:   []*gedcom.LDSOrdinance{{FamilyXRef: "@f1@"}},
			Events:          []*gedcom.Event{{Notes: []string{"@n1@"}}},
			Tags:            []*gedcom.Tag{{Value: "@i1@"}},
		}

		updateEntityReferences(fam, mapping)

		if fam.Husband != "@I1@" {
			t.Errorf("Husband = %v, want @I1@", fam.Husband)
		}
		if fam.Wife != "@I1@" {
			t.Errorf("Wife = %v, want @I1@", fam.Wife)
		}
		if fam.Children[0] != "@I1@" {
			t.Errorf("Children[0] = %v, want @I1@", fam.Children[0])
		}
	})

	t.Run("updates Source references", func(t *testing.T) {
		src := &gedcom.Source{
			RepositoryRef: "@r1@",
			Notes:         []string{"@n1@"},
			Media:         []*gedcom.MediaLink{{MediaXRef: "@m1@"}},
			Tags:          []*gedcom.Tag{{Value: "@r1@"}},
		}

		updateEntityReferences(src, mapping)

		if src.RepositoryRef != "@R1@" {
			t.Errorf("RepositoryRef = %v, want @R1@", src.RepositoryRef)
		}
	})

	t.Run("updates Repository references", func(t *testing.T) {
		repo := &gedcom.Repository{
			Notes: []string{"@n1@"},
			Tags:  []*gedcom.Tag{{Value: "@n1@"}},
		}

		updateEntityReferences(repo, mapping)

		if repo.Notes[0] != "@N1@" {
			t.Errorf("Notes[0] = %v, want @N1@", repo.Notes[0])
		}
	})

	t.Run("updates Note references", func(t *testing.T) {
		note := &gedcom.Note{
			Tags: []*gedcom.Tag{{Value: "@s1@"}},
		}

		updateEntityReferences(note, mapping)

		if note.Tags[0].Value != "@S1@" {
			t.Errorf("Tags[0].Value = %v, want @S1@", note.Tags[0].Value)
		}
	})

	t.Run("updates MediaObject references", func(t *testing.T) {
		media := &gedcom.MediaObject{
			SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}},
			Notes:           []string{"@n1@"},
			Tags:            []*gedcom.Tag{{Value: "@s1@"}},
		}

		updateEntityReferences(media, mapping)

		if media.SourceCitations[0].SourceXRef != "@S1@" {
			t.Errorf("SourceCitations[0].SourceXRef = %v, want @S1@", media.SourceCitations[0].SourceXRef)
		}
	})

	t.Run("updates Submitter references", func(t *testing.T) {
		subm := &gedcom.Submitter{
			Notes: []string{"@n1@"},
			Tags:  []*gedcom.Tag{{Value: "@n1@"}},
		}

		updateEntityReferences(subm, mapping)

		if subm.Notes[0] != "@N1@" {
			t.Errorf("Notes[0] = %v, want @N1@", subm.Notes[0])
		}
	})
}

func TestUpdateEventReferences(t *testing.T) {
	t.Run("nil event", func(t *testing.T) {
		// Should not panic
		updateEventReferences(nil, map[string]string{})
	})

	t.Run("updates all event references", func(t *testing.T) {
		mapping := map[string]string{"@s1@": "@S1@", "@m1@": "@M1@", "@n1@": "@N1@"}
		event := &gedcom.Event{
			SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@s1@"}},
			Media:           []*gedcom.MediaLink{{MediaXRef: "@m1@"}},
			Notes:           []string{"@n1@"},
			Tags:            []*gedcom.Tag{{Value: "@s1@"}},
		}

		updateEventReferences(event, mapping)

		if event.SourceCitations[0].SourceXRef != "@S1@" {
			t.Errorf("SourceXRef = %v, want @S1@", event.SourceCitations[0].SourceXRef)
		}
		if event.Media[0].MediaXRef != "@M1@" {
			t.Errorf("MediaXRef = %v, want @M1@", event.Media[0].MediaXRef)
		}
		if event.Notes[0] != "@N1@" {
			t.Errorf("Notes[0] = %v, want @N1@", event.Notes[0])
		}
	})
}

func TestUpdateXRefMap(t *testing.T) {
	t.Run("updates map keys", func(t *testing.T) {
		record1 := &gedcom.Record{XRef: "@I1@"}
		record2 := &gedcom.Record{XRef: "@F1@"}
		doc := &gedcom.Document{
			XRefMap: map[string]*gedcom.Record{
				"@i1@": record1,
				"@F1@": record2,
			},
		}
		mapping := map[string]string{"@i1@": "@I1@"}

		updateXRefMap(doc, mapping)

		if _, ok := doc.XRefMap["@i1@"]; ok {
			t.Error("Old key @i1@ should be removed")
		}
		if doc.XRefMap["@I1@"] != record1 {
			t.Error("New key @I1@ should map to record1")
		}
		if doc.XRefMap["@F1@"] != record2 {
			t.Error("Unmapped key @F1@ should be preserved")
		}
	})
}

func TestIsXRef(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"@I1@", true},
		{"@F123@", true},
		{"@a@", true},
		{"@@", false},  // too short
		{"@", false},   // too short
		{"I1@", false}, // missing leading @
		{"@I1", false}, // missing trailing @
		{"I1", false},  // no @
		{"", false},    // empty
		{"text", false},
		{"@@ invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := isXRef(tt.value)
			if got != tt.want {
				t.Errorf("isXRef(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestUpdateHelperFunctions(t *testing.T) {
	mapping := map[string]string{"@old@": "@NEW@"}

	t.Run("updateSourceCitations", func(t *testing.T) {
		citations := []*gedcom.SourceCitation{{SourceXRef: "@old@"}}
		updateSourceCitations(citations, mapping)
		if citations[0].SourceXRef != "@NEW@" {
			t.Errorf("SourceXRef = %v, want @NEW@", citations[0].SourceXRef)
		}
	})

	t.Run("updateMediaLinks", func(t *testing.T) {
		links := []*gedcom.MediaLink{{MediaXRef: "@old@"}}
		updateMediaLinks(links, mapping)
		if links[0].MediaXRef != "@NEW@" {
			t.Errorf("MediaXRef = %v, want @NEW@", links[0].MediaXRef)
		}
	})

	t.Run("updateLDSOrdinances", func(t *testing.T) {
		ordinances := []*gedcom.LDSOrdinance{{FamilyXRef: "@old@"}}
		updateLDSOrdinances(ordinances, mapping)
		if ordinances[0].FamilyXRef != "@NEW@" {
			t.Errorf("FamilyXRef = %v, want @NEW@", ordinances[0].FamilyXRef)
		}
	})

	t.Run("updateStringSlice", func(t *testing.T) {
		slice := []string{"@old@", "unchanged", "@other@"}
		updateStringSlice(slice, mapping)
		if slice[0] != "@NEW@" {
			t.Errorf("slice[0] = %v, want @NEW@", slice[0])
		}
		if slice[1] != "unchanged" {
			t.Errorf("slice[1] = %v, want unchanged", slice[1])
		}
		if slice[2] != "@other@" {
			t.Errorf("slice[2] = %v, want @other@", slice[2])
		}
	})
}

// Helper function to create a document with given XRefs
func createDocWithXRefs(xrefs []string) *gedcom.Document {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{},
		Records: make([]*gedcom.Record, len(xrefs)),
		XRefMap: make(map[string]*gedcom.Record),
	}
	for i, xref := range xrefs {
		doc.Records[i] = &gedcom.Record{XRef: xref}
		if xref != "" {
			doc.XRefMap[xref] = doc.Records[i]
		}
	}
	return doc
}
