package converter

import (
	"strconv"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

func TestNormalizeXRefsForV7(t *testing.T) {
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

			normalizeXRefsForV7(doc, report)

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
			mapping, _ := buildXRefMapping(doc)

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

func TestBuildXRefMappingV7(t *testing.T) {
	doc := createDocWithXRefs([]string{"@I 1@", "@I_1@", "@i_1@", "@I1@", "@i1@", "@a-b.c@", "@I 1@"})
	doc.XRefMap = nil // definitions must come from Records, not the map
	doc.Header.Submitter = "@sub mitter@"
	doc.Records[1].Tags = []*gedcom.Tag{{Level: 1, Tag: "NOTE", Value: "@n1@"}, {Level: 1, Tag: "NOTE", Value: "@i1@"}}

	got, dangling := buildXRefMapping(doc)
	want := map[string]string{
		"@I 1@":        "@I_1_2@",
		"@i_1@":        "@I_1_3@",
		"@i1@":         "@I1_2@",
		"@a-b.c@":      "@A_B_C@",
		"@sub mitter@": "@SUB_MITTER@",
		"@n1@":         "@N1@",
	}
	if len(got) != len(want) {
		t.Errorf("mapping = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mapping[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(dangling) != 2 || dangling[0] != "@sub mitter@" || dangling[1] != "@n1@" {
		t.Errorf("dangling = %q, want [@sub mitter@ @n1@]", dangling)
	}
}

// TestBuildXRefMappingDanglingNoFalseJoin pins that a dangling pointer joins a
// record only when it differs from the record's XRef by case alone; otherwise
// it gets a fresh identifier rather than resolving to an unrelated record.
func TestBuildXRefMappingDanglingNoFalseJoin(t *testing.T) {
	doc := createDocWithXRefs([]string{"@I_1@", "@I 2@", "@i1@", "@NoTe ref@"})
	doc.Records[0].Tags = []*gedcom.Tag{
		{Level: 1, Tag: "ASSO", Value: "@I-1@"},      // sanitizes onto valid @I_1@
		{Level: 1, Tag: "ASSO", Value: "@i_2@"},      // sanitizes onto renamed @I 2@
		{Level: 1, Tag: "ASSO", Value: "@i1_2@"},     // sanitizes onto suffixed @i1@
		{Level: 1, Tag: "NOTE", Value: "@note ref@"}, // spaced: not visited (#591)
		{Level: 1, Tag: "ASSO", Value: "@i_1@"},      // case-only match of @I_1@
	}
	doc.Header.Tags = []*gedcom.Tag{{Level: 1, Tag: "SUBN", Value: "@sn1@"}}

	got, dangling := buildXRefMapping(doc)
	want := map[string]string{
		"@I 2@":      "@I_2@",
		"@i1@":       "@I1@",
		"@NoTe ref@": "@NOTE_REF@",
		"@sn1@":      "@SN1@",
		"@I-1@":      "@I_1_2@",
		"@i_2@":      "@I_2_2@",
		"@i1_2@":     "@I1_2@",
		"@i_1@":      "@I_1@",
	}
	if len(got) != len(want) {
		t.Errorf("mapping = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mapping[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(dangling) != 5 {
		t.Errorf("dangling = %q, want 5 entries", dangling)
	}
}

// TestBuildXRefMappingAvoidsVoid pins that a rewritten XRef never becomes
// @VOID@, the GEDCOM 7.0 null pointer.
func TestBuildXRefMappingAvoidsVoid(t *testing.T) {
	doc := createDocWithXRefs([]string{"@void@", "@Void@"})
	got, _ := buildXRefMapping(doc)
	if got["@void@"] != "@VOID_2@" || got["@Void@"] != "@VOID_3@" {
		t.Errorf("mapping = %v, want @void@ -> @VOID_2@, @Void@ -> @VOID_3@", got)
	}
}

// TestBuildXRefMappingManyCollisions pins that suffix search stays linear
// when many XRefs sanitize to the same identifier.
func TestBuildXRefMappingManyCollisions(t *testing.T) {
	const n = 20000
	xrefs := make([]string, n)
	for i := range xrefs {
		xrefs[i] = "@I" + string(rune(0x4e00+i)) + "@" // all sanitize to @I_@
	}
	got, _ := buildXRefMapping(createDocWithXRefs(xrefs))
	seen := make(map[string]bool, n)
	for _, v := range got {
		if seen[v] {
			t.Fatalf("duplicate target %q", v)
		}
		seen[v] = true
	}
	if len(got) != n || got[xrefs[n-1]] != "@I__"+strconv.Itoa(n)+"@" {
		t.Errorf("len = %d, last = %q", len(got), got[xrefs[n-1]])
	}
}

func TestToV7XRefBody(t *testing.T) {
	for in, want := range map[string]string{"@@": "_", "@é x@": "__X", "@ok_1@": "OK_1"} {
		if got := toV7XRefBody(in); got != want {
			t.Errorf("toV7XRefBody(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeXRefsForV7ReportsDanglingPointer(t *testing.T) {
	doc := createDocWithXRefs([]string{"@TEST@"})
	doc.Header.Submitter = "@test@"
	report := &gedcom.ConversionReport{}

	normalizeXRefsForV7(doc, report)

	if doc.Header.Submitter != "@TEST@" {
		t.Errorf("Submitter = %q, want @TEST@", doc.Header.Submitter)
	}
	if len(report.Normalized) != 1 || report.Normalized[0].Original != "@test@" || report.Normalized[0].Result != "@TEST@" {
		t.Errorf("Normalized = %+v, want one @test@ -> @TEST@ note", report.Normalized)
	}
}

// TestBuildXRefMappingReservesValidDangling pins that a rewritten record never
// takes the name of a valid pointer no record defines, which would make that
// dangling pointer resolve to an unrelated record. A valid pointer differing
// only in case from a record still joins that record's rewrite.
func TestBuildXRefMappingReservesValidDangling(t *testing.T) {
	doc := createDocWithXRefs([]string{"@I-1@", "@a-b@", "@test@"})
	doc.Records[0].Tags = []*gedcom.Tag{
		{Level: 1, Tag: "ASSO", Value: "@I_1@"},   // valid, undefined
		{Level: 1, Tag: "ASSO", Value: "@I_1_2@"}, // valid, undefined: next candidate
		{Level: 1, Tag: "ASSO", Value: "@TEST@"},  // case-only match of @test@
	}
	doc.Header.Tags = []*gedcom.Tag{{Level: 1, Tag: "SUBN", Value: "@A_B@"}}

	got, dangling := buildXRefMapping(doc)
	want := map[string]string{
		"@I-1@":  "@I_1_3@",
		"@a-b@":  "@A_B_2@",
		"@test@": "@TEST@",
	}
	if len(got) != len(want) || len(dangling) != 0 {
		t.Errorf("mapping = %v, dangling = %q, want %v and none", got, dangling, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mapping[%q] = %q, want %q", k, got[k], v)
		}
	}
}

// TestBuildXRefMappingASCIICaseFold pins that the case-only join folds ASCII
// letters only: dotless i (U+0131) uppercases to I under Unicode rules but is
// not a case variant of it here.
func TestBuildXRefMappingASCIICaseFold(t *testing.T) {
	doc := createDocWithXRefs([]string{"@I1@", "@ı1@"})
	got, _ := buildXRefMapping(doc)
	if got["@ı1@"] != "@_1@" {
		t.Errorf("mapping = %v, want @ı1@ -> @_1@", got)
	}
	if asciiUpper("aıſz") != "AıſZ" {
		t.Errorf("asciiUpper folded non-ASCII letters")
	}
}
