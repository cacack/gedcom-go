package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

func TestNormalizeXRefsToUppercase(t *testing.T) {
	tests := []struct {
		name       string
		inputXRefs []string
		wantXRefs  []string
		wantCount  int
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
