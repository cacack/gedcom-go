package gedcom

import (
	"errors"
	"reflect"
	"testing"
)

// attachDoc returns a document declaring v with an individual @I1@, a
// repository @R1@ and an OBJE record @O1@ for photos\ada.jpg.
func attachDoc(t *testing.T, v Version) *Document {
	t.Helper()
	d := NewDocument(v)
	for _, r := range []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: &Individual{XRef: "@I1@"}},
		{XRef: "@R1@", Type: RecordTypeRepository, Entity: &Repository{XRef: "@R1@"}},
		{XRef: "@O1@", Type: RecordTypeMedia, Entity: &MediaObject{XRef: "@O1@", Files: []*MediaFile{{FileRef: `photos\ada.jpg`, Form: "image/jpeg"}}}},
	} {
		if err := d.AddRecord(r); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func mediaXRefs(t *testing.T, d *Document, xref string) []string {
	t.Helper()
	links, ok := d.GetRecord(xref).MediaLinks()
	if !ok {
		t.Fatalf("%s cannot carry media links", xref)
	}
	var xrefs []string
	for _, l := range links {
		xrefs = append(xrefs, l.MediaXRef)
	}
	return xrefs
}

// TestAttachMediaReusesRecordNamingPath pins that an OBJE record naming the
// path is linked, not duplicated, with "\" read as "/" on either side.
func TestAttachMediaReusesRecordNamingPath(t *testing.T) {
	for _, path := range []string{"photos/ada.jpg", `photos\ada.jpg`} {
		t.Run(path, func(t *testing.T) {
			d := attachDoc(t, Version70)
			got, err := d.AttachMedia("@I1@", path)
			if err != nil {
				t.Fatalf("AttachMedia() error = %v", err)
			}
			if want := (AttachMediaResult{MediaXRef: "@O1@", Status: AttachLinked}); got != want {
				t.Errorf("AttachMedia() = %+v, want %+v", got, want)
			}
			if len(d.Records) != 3 {
				t.Errorf("records = %d, want 3 (none added)", len(d.Records))
			}
			if xrefs := mediaXRefs(t, d, "@I1@"); !reflect.DeepEqual(xrefs, []string{"@O1@"}) {
				t.Errorf("links = %v, want [@O1@]", xrefs)
			}
		})
	}
}

// TestAttachMediaComparesPathsExactly pins that only "\" is normalized: a
// different case or a different directory is another file.
func TestAttachMediaComparesPathsExactly(t *testing.T) {
	for _, path := range []string{"Photos/ada.jpg", "ada.jpg", "photos/ada.jpeg"} {
		d := attachDoc(t, Version70)
		got, err := d.AttachMedia("@I1@", path)
		if err != nil {
			t.Fatalf("AttachMedia(%q) error = %v", path, err)
		}
		if got.Status != AttachCreated {
			t.Errorf("AttachMedia(%q) status = %v, want created", path, got.Status)
		}
	}
}

// TestAttachMediaCreatesRecordWithFreeXRef pins that a new OBJE record takes
// the allocator's XRef, skipping one a dangling pointer names, and is indexed.
func TestAttachMediaCreatesRecordWithFreeXRef(t *testing.T) {
	d := attachDoc(t, Version70)
	// @O2@ is named by a dangling pointer, so it is not free.
	d.GetIndividual("@I1@").Media = []*MediaLink{{MediaXRef: "@O2@"}}

	got, err := d.AttachMedia("@I1@", "scans/letter.png")
	if err != nil {
		t.Fatalf("AttachMedia() error = %v", err)
	}
	if want := (AttachMediaResult{MediaXRef: "@O3@", Status: AttachCreated}); got != want {
		t.Fatalf("AttachMedia() = %+v, want %+v", got, want)
	}
	media := d.GetMediaObject("@O3@")
	if media == nil {
		t.Fatal("new OBJE record not indexed under @O3@")
	}
	want := &MediaObject{XRef: "@O3@", Files: []*MediaFile{{FileRef: "scans/letter.png", Form: "image/png"}}}
	if !reflect.DeepEqual(media, want) {
		t.Errorf("new record = %+v, want %+v", media, want)
	}
	if d.Records[len(d.Records)-1].Type != RecordTypeMedia {
		t.Error("new record type is not OBJE")
	}
	if xrefs := mediaXRefs(t, d, "@I1@"); !reflect.DeepEqual(xrefs, []string{"@O2@", "@O3@"}) {
		t.Errorf("links = %v, want [@O2@ @O3@]", xrefs)
	}
}

// TestAttachMediaFormFollowsVersion pins the FORM a new record gets for each
// declared version, including the fallback for an unknown extension.
func TestAttachMediaFormFollowsVersion(t *testing.T) {
	tests := []struct {
		version              Version
		path                 string
		recordForm, fileForm string
	}{
		{Version70, "a.jpg", "", "image/jpeg"},
		{Version70, "a.JPEG", "", "image/jpeg"},
		{Version70, "a.tif", "", "image/tiff"},
		{Version70, "a.heic", "", "application/octet-stream"},
		{Version70, `dir.d\noext`, "", "application/octet-stream"},
		{"", "a.jpg", "", "image/jpeg"},
		{Version551, "a.jpg", "", "jpg"},
		{Version551, "a.JPEG", "", "jpg"},
		{Version551, "a.tiff", "", "tif"},
		{Version551, "a.PNG", "", "png"},
		{Version55, "a.jpg", "jpeg", ""},
		{Version55, "a.tif", "tiff", ""},
		{Version55, "a.wav", "wav", ""},
		{Version55, "a.pdf", "pdf", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.version)+" "+tt.path, func(t *testing.T) {
			d := attachDoc(t, tt.version)
			got, err := d.AttachMedia("@I1@", tt.path)
			if err != nil {
				t.Fatalf("AttachMedia() error = %v", err)
			}
			media := d.GetMediaObject(got.MediaXRef)
			if media.Form != tt.recordForm || media.Files[0].Form != tt.fileForm {
				t.Errorf("FORM = record %q, file %q; want record %q, file %q",
					media.Form, media.Files[0].Form, tt.recordForm, tt.fileForm)
			}
		})
	}
}

// TestAttachMediaTwiceLinksOnce pins that a second call with the same
// arguments changes nothing and reports already linked.
func TestAttachMediaTwiceLinksOnce(t *testing.T) {
	for _, path := range []string{"photos/ada.jpg", "new.jpg"} {
		t.Run(path, func(t *testing.T) {
			d := attachDoc(t, Version70)
			first, err := d.AttachMedia("@I1@", path)
			if err != nil {
				t.Fatalf("first AttachMedia() error = %v", err)
			}
			records := len(d.Records)
			second, err := d.AttachMedia("@I1@", path)
			if err != nil {
				t.Fatalf("second AttachMedia() error = %v", err)
			}
			if want := (AttachMediaResult{MediaXRef: first.MediaXRef, Status: AttachAlreadyLinked}); second != want {
				t.Errorf("second AttachMedia() = %+v, want %+v", second, want)
			}
			if len(d.Records) != records {
				t.Errorf("records = %d, want %d", len(d.Records), records)
			}
			if xrefs := mediaXRefs(t, d, "@I1@"); !reflect.DeepEqual(xrefs, []string{first.MediaXRef}) {
				t.Errorf("links = %v, want one link to %s", xrefs, first.MediaXRef)
			}
		})
	}
}

// TestAttachMediaPrefersLinkedRecord pins that when two OBJE records name the
// file, the one the record already links counts as already linked.
func TestAttachMediaPrefersLinkedRecord(t *testing.T) {
	d := attachDoc(t, Version70)
	dup := &Record{XRef: "@O9@", Type: RecordTypeMedia, Entity: &MediaObject{Files: []*MediaFile{{FileRef: "photos/ada.jpg"}}}}
	if err := d.AddRecord(dup); err != nil {
		t.Fatal(err)
	}
	d.GetIndividual("@I1@").Media = []*MediaLink{nil, {MediaXRef: "@O9@"}}
	got, err := d.AttachMedia("@I1@", "photos/ada.jpg")
	if err != nil {
		t.Fatalf("AttachMedia() error = %v", err)
	}
	if want := (AttachMediaResult{MediaXRef: "@O9@", Status: AttachAlreadyLinked}); got != want {
		t.Errorf("AttachMedia() = %+v, want %+v", got, want)
	}
}

// TestAttachMediaErrorsLeaveDocumentUnchanged pins the errors and that the
// document is not changed by a failed call.
func TestAttachMediaErrorsLeaveDocumentUnchanged(t *testing.T) {
	tests := []struct {
		name       string
		xref, path string
		want       error
	}{
		{"unknown xref", "@X9@", "new.jpg", ErrUnknownXRef},
		{"empty xref", "", "new.jpg", ErrUnknownXRef},
		{"record without media links", "@R1@", "new.jpg", ErrNoMediaLinks},
		{"OBJE record", "@O1@", "new.jpg", ErrNoMediaLinks},
		{"empty path", "@I1@", "", ErrEmptyMediaPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := attachDoc(t, Version70)
			before := d.Clone()
			_, err := d.AttachMedia(tt.xref, tt.path)
			if !errors.Is(err, tt.want) {
				t.Fatalf("AttachMedia() error = %v, want %v", err, tt.want)
			}
			if !reflect.DeepEqual(d, before) {
				t.Error("document changed by a failed AttachMedia")
			}
		})
	}

	var nilDoc *Document
	if _, err := nilDoc.AttachMedia("@I1@", "a.jpg"); !errors.Is(err, ErrNilDocument) {
		t.Errorf("nil document error = %v, want ErrNilDocument", err)
	}
}

// TestAttachMediaExtensionlessPath pins that a 5.5 or 5.5.1 document, where
// FORM is required, gets no OBJE record without one: creating a record for a
// path with no extension fails and leaves the document unchanged, while a
// record that already names the path is linked.
func TestAttachMediaExtensionlessPath(t *testing.T) {
	for _, v := range []Version{Version55, Version551} {
		for _, path := range []string{"noext", `dir.d\noext`, "trailing."} {
			t.Run(string(v)+" "+path, func(t *testing.T) {
				d := attachDoc(t, v)
				before := d.Clone()
				_, err := d.AttachMedia("@I1@", path)
				if !errors.Is(err, ErrNoMediaFormat) {
					t.Fatalf("AttachMedia() error = %v, want ErrNoMediaFormat", err)
				}
				if !reflect.DeepEqual(d, before) {
					t.Error("document changed by a failed AttachMedia")
				}
			})
		}
		t.Run(string(v)+" existing record", func(t *testing.T) {
			d := attachDoc(t, v)
			d.GetMediaObject("@O1@").Files[0].FileRef = "scans/letter"
			got, err := d.AttachMedia("@I1@", `scans\letter`)
			if err != nil {
				t.Fatalf("AttachMedia() error = %v", err)
			}
			if want := (AttachMediaResult{MediaXRef: "@O1@", Status: AttachLinked}); got != want {
				t.Errorf("AttachMedia() = %+v, want %+v", got, want)
			}
		})
	}
}

// TestAttachMediaIndexesLiteralDocument pins that a Document literal with no
// XRefMap is indexed rather than reported as having no such record.
func TestAttachMediaIndexesLiteralDocument(t *testing.T) {
	d := &Document{Records: []*Record{{XRef: "@F1@", Type: RecordTypeFamily, Entity: &Family{}}}}
	got, err := d.AttachMedia("@F1@", "a.gif")
	if err != nil {
		t.Fatalf("AttachMedia() error = %v", err)
	}
	if want := (AttachMediaResult{MediaXRef: "@O1@", Status: AttachCreated}); got != want {
		t.Errorf("AttachMedia() = %+v, want %+v", got, want)
	}
}

func TestAttachStatusString(t *testing.T) {
	for s, want := range map[AttachStatus]string{
		AttachCreated: "created", AttachLinked: "linked", AttachAlreadyLinked: "already linked", 0: "unknown",
	} {
		if got := s.String(); got != want {
			t.Errorf("AttachStatus(%d).String() = %q, want %q", int(s), got, want)
		}
	}
}
