package gedzip

import (
	"archive/zip"
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// refsGED has a reference of every kind, as FILE and as FILE.TRAN.
const refsGED = `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE media/a%20b.jpg
2 FORM image/jpeg
2 TRAN media/thumb.jpg
3 FORM image/jpeg
2 TRAN media/missing-thumb.jpg
3 FORM image/jpeg
1 FILE media/missing.jpg
2 FORM image/jpeg
0 @O2@ OBJE
1 FILE https://example.com/a.jpg
2 FORM image/jpeg
2 TRAN ftp://example.com/a.png
3 FORM image/png
2 TRAN file:///home/a.png
3 FORM image/png
2 TRAN ../a.png
3 FORM image/png
1 FILE file:///home/a.jpg
2 FORM image/jpeg
1 FILE C:\photos\a.jpg
2 FORM image/jpeg
1 FILE link
2 FORM image/jpeg
1 FILE media/missing.jpg
2 FORM image/jpeg
0 TRLR
`

func TestCollectFileRefs(t *testing.T) {
	r := mustReader(t, buildZip(t, file("gedcom.ged", refsGED)), nil)
	doc, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []fileRef{
		{xref: "@O1@", ref: "media/a%20b.jpg"},
		{xref: "@O1@", ref: "media/thumb.jpg", translation: true},
		{xref: "@O1@", ref: "media/missing-thumb.jpg", translation: true},
		{xref: "@O1@", ref: "media/missing.jpg"},
		{xref: "@O2@", ref: "https://example.com/a.jpg"},
		{xref: "@O2@", ref: "ftp://example.com/a.png", translation: true},
		{xref: "@O2@", ref: "file:///home/a.png", translation: true},
		{xref: "@O2@", ref: "../a.png", translation: true},
		{xref: "@O2@", ref: "file:///home/a.jpg"},
		{xref: "@O2@", ref: `C:\photos\a.jpg`},
		{xref: "@O2@", ref: "link"},
		{xref: "@O2@", ref: "media/missing.jpg"},
	}
	if got := collectFileRefs(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("collectFileRefs() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestCollectFileRefsTolerantOfSparseDocuments(t *testing.T) {
	if got := collectFileRefs(nil); got != nil {
		t.Errorf("collectFileRefs(nil) = %v", got)
	}
	doc := &gedcom.Document{Records: []*gedcom.Record{
		nil,
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}},
		{XRef: "@O1@", Type: gedcom.RecordTypeMedia, Entity: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{
			nil,
			{FileRef: "", Translations: []*gedcom.MediaTranslation{nil, {FileRef: "t.txt"}}},
		}}},
	}}
	want := []fileRef{{xref: "@O1@", ref: ""}, {xref: "@O1@", ref: "t.txt", translation: true}}
	if got := collectFileRefs(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("collectFileRefs() = %+v, want %+v", got, want)
	}
}

func TestReaderCheckRefs(t *testing.T) {
	r := mustReader(t, buildZip(t,
		file("gedcom.ged", refsGED),
		file("media/a b.jpg", "x"),
		file("media/thumb.jpg", "x"),
		file("unreferenced.jpg", "x"),
		zipEntry{name: "link", data: "x", mod: func(h *zip.FileHeader) { h.SetMode(fs.ModeSymlink | 0o777) }},
	), nil)
	doc, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		xref        string
		ref         string
		translation bool
		kind        RefKind
		err         error
	}{
		{"@O1@", "media/missing-thumb.jpg", true, RefLocal, ErrEntryNotFound},
		{"@O1@", "media/missing.jpg", false, RefLocal, ErrEntryNotFound},
		{"@O2@", "file:///home/a.png", true, RefFileURL, ErrNotLocal},
		{"@O2@", "../a.png", true, RefInvalid, ErrNotLocal},
		{"@O2@", "file:///home/a.jpg", false, RefFileURL, ErrNotLocal},
		{"@O2@", `C:\photos\a.jpg`, false, RefInvalid, ErrNotLocal},
		{"@O2@", "link", false, RefLocal, ErrUnusableEntry},
		{"@O2@", "media/missing.jpg", false, RefLocal, ErrEntryNotFound},
	}
	got := r.CheckRefs(doc)
	if len(got) != len(want) {
		t.Fatalf("CheckRefs() reported %d problems, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.XRef != w.xref || g.FileRef != w.ref || g.Translation != w.translation || g.Kind != w.kind {
			t.Errorf("problem %d = %+v, want %+v", i, g, w)
		}
		if !errors.Is(g.Err, w.err) {
			t.Errorf("problem %d error = %v, want %v", i, g.Err, w.err)
		}
		for _, other := range []error{ErrEntryNotFound, ErrNotLocal, ErrUnusableEntry} {
			if other != w.err && errors.Is(g.Err, other) {
				t.Errorf("problem %d error = %v, must not be %v", i, g.Err, other)
			}
		}
	}
}

func TestReaderCheckRefsNothingToReport(t *testing.T) {
	const ged = "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n1 FILE a.jpg\n2 FORM image/jpeg\n" +
		"1 FILE http://example.com/b.jpg\n2 FORM image/jpeg\n0 TRLR\n"
	r := mustReader(t, buildZip(t, file("gedcom.ged", ged), file("a.jpg", "x")), nil)
	doc, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(collectFileRefs(doc)) != 2 {
		t.Fatal("fixture references were not decoded")
	}
	if got := r.CheckRefs(doc); got != nil {
		t.Errorf("CheckRefs() = %+v, want nil", got)
	}
	if got := r.CheckRefs(nil); got != nil {
		t.Errorf("CheckRefs(nil) = %+v, want nil", got)
	}
	if got := r.CheckRefs(&gedcom.Document{}); got != nil {
		t.Errorf("CheckRefs(empty) = %+v, want nil", got)
	}
}
