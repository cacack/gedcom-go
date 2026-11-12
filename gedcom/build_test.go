package gedcom

import (
	"errors"
	"strings"
	"testing"
)

func TestNewDocument(t *testing.T) {
	tests := []struct {
		version      Version
		wantDeclared Version
	}{
		{Version55, Version55},
		{Version551, Version551},
		{Version70, Version70},
		{"7.1", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			doc := NewDocument(tt.version)
			if doc.Header == nil {
				t.Fatal("Header is nil")
			}
			if doc.Header.Version != tt.version {
				t.Errorf("Header.Version = %q, want %q", doc.Header.Version, tt.version)
			}
			if doc.Header.DeclaredVersion != tt.wantDeclared {
				t.Errorf("Header.DeclaredVersion = %q, want %q", doc.Header.DeclaredVersion, tt.wantDeclared)
			}
			if doc.XRefMap == nil || len(doc.XRefMap) != 0 {
				t.Errorf("XRefMap = %v, want empty non-nil map", doc.XRefMap)
			}
			if doc.Records == nil || len(doc.Records) != 0 {
				t.Errorf("Records = %v, want empty non-nil slice", doc.Records)
			}
			if doc.Trailer == nil {
				t.Error("Trailer is nil")
			}
		})
	}
}

// TestNewDocument_DeclaredVersionSelectsGrammar checks that the document
// NewDocument returns applies the declared version's pointer grammar, which a
// Header literal setting only Version does not.
func TestNewDocument_DeclaredVersionSelectsGrammar(t *testing.T) {
	if !NewDocument(Version551).IsPointerXRef("@N 1@") {
		t.Error("5.5.1 document rejects spaced pointer @N 1@")
	}
	if NewDocument(Version70).IsPointerXRef("@N 1@") {
		t.Error("7.0 document accepts spaced pointer @N 1@")
	}
}

func TestDocument_AddRecord(t *testing.T) {
	doc := NewDocument(Version70)
	ind := &Individual{XRef: "@I1@", Sex: "M"}
	rec := &Record{XRef: "@I1@", Type: RecordTypeIndividual, Entity: ind}
	if err := doc.AddRecord(rec); err != nil {
		t.Fatalf("AddRecord: %v", err)
	}
	if got := doc.GetIndividual("@I1@"); got != ind {
		t.Errorf("GetIndividual(@I1@) = %v, want the added individual", got)
	}
	if len(doc.Records) != 1 || doc.Records[0] != rec {
		t.Errorf("Records = %v, want [added record]", doc.Records)
	}
}

func TestDocument_AddRecord_FamilyLinksResolve(t *testing.T) {
	doc := NewDocument(Version70)
	father := &Individual{XRef: "@I1@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}}}
	child := &Individual{XRef: "@I2@", ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}}}
	fam := &Family{XRef: "@F1@", Husband: "@I1@", Children: []string{"@I2@"}}
	for _, r := range []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: father},
		{XRef: "@I2@", Type: RecordTypeIndividual, Entity: child},
		{XRef: "@F1@", Type: RecordTypeFamily, Entity: fam},
	} {
		if err := doc.AddRecord(r); err != nil {
			t.Fatalf("AddRecord(%s): %v", r.XRef, err)
		}
	}
	parents := child.Parents(doc)
	if len(parents) != 1 || parents[0] != father {
		t.Errorf("child.Parents = %v, want [father]", parents)
	}
}

func TestDocument_AddRecord_DuplicateXRef(t *testing.T) {
	doc := NewDocument(Version70)
	first := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	if err := doc.AddRecord(first); err != nil {
		t.Fatalf("AddRecord(first): %v", err)
	}
	err := doc.AddRecord(&Record{XRef: "@I1@", Type: RecordTypeFamily})
	if !errors.Is(err, ErrDuplicateXRef) {
		t.Fatalf("AddRecord(duplicate) error = %v, want ErrDuplicateXRef", err)
	}
	if !strings.Contains(err.Error(), "@I1@") {
		t.Errorf("error %q does not name the XRef", err)
	}
	if len(doc.Records) != 1 || doc.Records[0] != first || doc.XRefMap["@I1@"] != first {
		t.Errorf("document changed by rejected add: Records=%v XRefMap=%v", doc.Records, doc.XRefMap)
	}
}

func TestDocument_AddRecord_EmptyXRef(t *testing.T) {
	doc := NewDocument(Version70)
	a := &Record{Type: RecordTypeSharedNote}
	b := &Record{Type: RecordTypeSharedNote}
	for _, r := range []*Record{a, b} {
		if err := doc.AddRecord(r); err != nil {
			t.Fatalf("AddRecord: %v", err)
		}
	}
	if len(doc.Records) != 2 || doc.Records[0] != a || doc.Records[1] != b {
		t.Errorf("Records = %v, want [a b]", doc.Records)
	}
	if _, ok := doc.XRefMap[""]; ok {
		t.Error("empty XRef was indexed")
	}
}

func TestDocument_AddRecord_NilArguments(t *testing.T) {
	var nilDoc *Document
	if err := nilDoc.AddRecord(&Record{XRef: "@I1@"}); !errors.Is(err, ErrNilDocument) {
		t.Errorf("nil document: error = %v, want ErrNilDocument", err)
	}
	doc := NewDocument(Version70)
	if err := doc.AddRecord(nil); !errors.Is(err, ErrNilRecord) {
		t.Errorf("nil record: error = %v, want ErrNilRecord", err)
	}
	if len(doc.Records) != 0 {
		t.Errorf("Records = %v after rejected nil add, want empty", doc.Records)
	}
}

// TestDocument_AddRecord_NilXRefMap checks that a Document literal with
// records but no XRefMap is indexed first, so a duplicate of an existing
// record is still caught.
func TestDocument_AddRecord_NilXRefMap(t *testing.T) {
	existing := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	doc := &Document{Records: []*Record{existing}}
	if err := doc.AddRecord(&Record{XRef: "@I1@"}); !errors.Is(err, ErrDuplicateXRef) {
		t.Errorf("duplicate of literal record: error = %v, want ErrDuplicateXRef", err)
	}
	added := &Record{XRef: "@I2@", Type: RecordTypeIndividual}
	if err := doc.AddRecord(added); err != nil {
		t.Fatalf("AddRecord(@I2@): %v", err)
	}
	if doc.GetRecord("@I1@") != existing || doc.GetRecord("@I2@") != added {
		t.Errorf("XRefMap = %v, want @I1@ and @I2@ indexed", doc.XRefMap)
	}
}

func TestDocument_ReindexXRefs(t *testing.T) {
	first := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	dup := &Record{XRef: "@I1@", Type: RecordTypeFamily}
	other := &Record{XRef: "@F1@", Type: RecordTypeFamily}
	noXRef := &Record{Type: RecordTypeSharedNote}
	stale := &Record{XRef: "@GONE@"}
	doc := &Document{
		Records: []*Record{first, nil, dup, noXRef, other},
		XRefMap: map[string]*Record{"@GONE@": stale},
	}
	doc.ReindexXRefs()

	want := map[string]*Record{"@I1@": first, "@F1@": other}
	if len(doc.XRefMap) != len(want) {
		t.Errorf("XRefMap = %v, want %v", doc.XRefMap, want)
	}
	for xref, rec := range want {
		if doc.XRefMap[xref] != rec {
			t.Errorf("XRefMap[%s] = %v, want %v", xref, doc.XRefMap[xref], rec)
		}
	}
	if _, ok := doc.XRefMap["@GONE@"]; ok {
		t.Error("stale entry @GONE@ survived reindex")
	}
}

func TestDocument_ReindexXRefs_NilDocument(t *testing.T) {
	var doc *Document
	doc.ReindexXRefs() // must not panic
}
