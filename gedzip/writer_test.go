package gedzip

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// writeGED references local files (one with a space, one not ASCII, one
// twice, one under two spellings, text and binary) and web files, as FILE and
// as FILE.TRAN, across several OBJE records.
const writeGED = `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE media/a%20b.jpg
2 FORM image/jpeg
2 TRAN media/thumb.png
3 FORM image/png
2 TRAN media/transcript.TXT
3 FORM text/plain
1 FILE media/CharlotteBront%C3%AB.jpg
2 FORM image/jpeg
0 @O2@ OBJE
1 FILE https://example.com/a.jpg
2 FORM image/jpeg
2 TRAN ftp://example.com/a.png
3 FORM image/png
1 FILE media/a%20b.jpg
2 FORM image/jpeg
1 FILE media/a b.jpg
2 FORM image/jpeg
0 @O3@ OBJE
1 FILE notes.html
2 FORM text/html
2 TRAN MANIFEST.MF
3 FORM text/plain
1 FILE META-INF/data.bin
2 FORM application/octet-stream
0 @I1@ INDI
1 OBJE @O1@
0 TRLR
`

// writeMediaFS holds every local file writeGED references, and one it does not.
func writeMediaFS() fstest.MapFS {
	return fstest.MapFS{
		"media/a b.jpg":             {Data: []byte("space")},
		"media/thumb.png":           {Data: []byte("thumb")},
		"media/transcript.TXT":      {Data: []byte(strings.Repeat("transcript ", 100))},
		"media/CharlotteBrontë.jpg": {Data: []byte("bronte")},
		"notes.html":                {Data: []byte("<html></html>")},
		"MANIFEST.MF":               {Data: []byte("manifest")},
		"META-INF/data.bin":         {Data: []byte("data")},
		"unreferenced.jpg":          {Data: []byte("left out")},
	}
}

// wantWriteEntries is the archive Write makes of writeGED, in order.
var wantWriteEntries = []struct {
	name   string
	method uint16
}{
	{"gedcom.ged", zip.Deflate},
	{"MANIFEST.MF", zip.Store},
	{"META-INF/data.bin", zip.Store},
	{"media/CharlotteBrontë.jpg", zip.Store},
	{"media/a b.jpg", zip.Store},
	{"media/thumb.png", zip.Store},
	{"media/transcript.TXT", zip.Deflate},
	{"notes.html", zip.Deflate},
}

// decodeGED decodes a dataset held in a string.
func decodeGED(tb testing.TB, ged string) *gedcom.Document {
	tb.Helper()
	doc, err := decoder.Decode(strings.NewReader(ged))
	if err != nil {
		tb.Fatalf("decode: %v", err)
	}
	return doc
}

// mustWrite returns the archive Write produces.
func mustWrite(tb testing.TB, doc *gedcom.Document, media fs.FS, opts *WriteOptions) []byte {
	tb.Helper()
	var buf bytes.Buffer
	if err := Write(&buf, doc, media, opts); err != nil {
		tb.Fatalf("Write: %v", err)
	}
	return buf.Bytes()
}

// encoded is doc as the encoder writes it by default.
func encoded(tb testing.TB, doc *gedcom.Document) string {
	tb.Helper()
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		tb.Fatalf("encode: %v", err)
	}
	return buf.String()
}

// localEntryNames returns the entry names, sorted and distinct, that the local
// FILE and FILE.TRAN payloads of doc resolve to.
func localEntryNames(doc *gedcom.Document) []string {
	set := make(map[string]bool)
	for _, ref := range collectFileRefs(doc) {
		if kind, name := ClassifyFileRef(ref.ref); kind == RefLocal {
			set[name] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// mediaNames returns the names of the archive's entries other than the
// dataset, in archive order.
func mediaNames(r *Reader) []string {
	names := []string{}
	for _, e := range r.Entries() {
		if e.Name != datasetName {
			names = append(names, e.Name)
		}
	}
	return names
}

func TestWriteRoundTrip(t *testing.T) {
	doc := decodeGED(t, writeGED)
	before := encoded(t, doc)
	media := writeMediaFS()
	archive := mustWrite(t, doc, media, nil)

	if after := encoded(t, doc); after != before {
		t.Errorf("Write modified the document:\n%s\nwas\n%s", after, before)
	}

	// The container: order, names, methods, flags, timestamps.
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != len(wantWriteEntries) {
		t.Fatalf("archive has %d entries, want %d", len(zr.File), len(wantWriteEntries))
	}
	for i, f := range zr.File {
		want := wantWriteEntries[i]
		if f.Name != want.name || f.Method != want.method {
			t.Errorf("entry %d = %q method %d, want %q method %d", i, f.Name, f.Method, want.name, want.method)
		}
		const flagUTF8 = 0x800
		if ascii := !strings.ContainsFunc(f.Name, func(r rune) bool { return r > 0x7f }); ascii == (f.Flags&flagUTF8 != 0) {
			t.Errorf("entry %q: flags %#x, UTF-8 flag must be set exactly for a non-ASCII name", f.Name, f.Flags)
		}
		if f.NonUTF8 || f.Flags&flagEncrypted != 0 || !f.Mode().IsRegular() {
			t.Errorf("entry %q: NonUTF8=%v flags=%#x mode=%v", f.Name, f.NonUTF8, f.Flags, f.Mode())
		}
		if !f.Modified.Equal(entryTime) {
			t.Errorf("entry %q modified %v, want %v", f.Name, f.Modified, entryTime)
		}
		// Made on Unix (host 3): Info-ZIP's unzip garbles a UTF-8 name in an
		// entry made on FAT, archive/zip's default.
		if f.CreatorVersion>>8 != 3 || f.Mode() != entryMode {
			t.Errorf("entry %q: creator %#x mode %v, want Unix and %v", f.Name, f.CreatorVersion, f.Mode(), entryMode)
		}
	}

	// Read back through the package.
	r := mustReader(t, archive, nil)
	for _, e := range r.Entries() {
		if e.Unusable != nil {
			t.Errorf("entry %q is unusable: %v", e.Name, e.Unusable)
		}
	}
	got, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if after := encoded(t, got); after != before {
		t.Errorf("dataset read back as\n%s\nwant\n%s", after, before)
	}
	if problems := r.CheckRefs(got); problems != nil {
		t.Errorf("CheckRefs() = %+v, want none", problems)
	}
	for ref, name := range map[string]string{
		"media/a%20b.jpg":                  "media/a b.jpg",
		"media/a b.jpg":                    "media/a b.jpg",
		"media/CharlotteBront%C3%AB.jpg":   "media/CharlotteBrontë.jpg",
		"media/thumb.png":                  "media/thumb.png",
		"media/transcript.TXT":             "media/transcript.TXT",
		"notes.html":                       "notes.html",
		"MANIFEST.MF":                      "MANIFEST.MF",
		"META-INF/data.bin":                "META-INF/data.bin",
		"media/CharlotteBront%c3%ab.jpg":   "media/CharlotteBrontë.jpg",
		"media/%61%20b.jpg":                "media/a b.jpg",
		"media/CharlotteBront%C3%AB%2Ejpg": "media/CharlotteBrontë.jpg",
	} {
		data, err := readRef(r, ref)
		if err != nil {
			t.Errorf("Open(%q): %v", ref, err)
			continue
		}
		if want := string(media[name].Data); data != want {
			t.Errorf("Open(%q) = %q, want %q", ref, data, want)
		}
	}
	if _, err := readRef(r, "unreferenced.jpg"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("unreferenced media was written: Open error %v", err)
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	first := mustWrite(t, decodeGED(t, writeGED), writeMediaFS(), nil)
	second := mustWrite(t, decodeGED(t, writeGED), writeMediaFS(), &WriteOptions{})
	if !bytes.Equal(first, second) {
		t.Error("two writes of the same document and media differ")
	}
}

// TestWriteFromReaderFS is read, then write: the media source is another
// archive.
func TestWriteFromReaderFS(t *testing.T) {
	first := mustWrite(t, decodeGED(t, writeGED), writeMediaFS(), nil)
	r := mustReader(t, first, nil)
	doc, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}
	second := mustWrite(t, doc, r.FS(), nil)
	if !bytes.Equal(first, second) {
		t.Error("an archive rewritten from its own Reader.FS differs")
	}
}

// TestWriteEntriesMatchTheWrittenDataset pins the property the archive
// exists for: its media entries are exactly the local files the gedcom.ged
// it holds names. The typed media objects of the document given to Write
// are not that list when a record is written from its Tags, or when the
// encoder moves an inline media link into a new OBJE record.
func TestWriteEntriesMatchTheWrittenDataset(t *testing.T) {
	stale := decodeGED(t, writeGED)
	// Edited without encoder.SyncTags: the record is still written from its
	// Tags, so the dataset still names media/a b.jpg.
	stale.MediaObjects()[0].Files[0].FileRef = "media/edited.jpg"

	inline := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{{
			XRef: "@I1@",
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", Media: []*gedcom.MediaLink{{
				Files: []*gedcom.MediaFile{{
					FileRef:      "media/a%20b.jpg",
					Form:         "image/jpeg",
					Translations: []*gedcom.MediaTranslation{{FileRef: "media/thumb.png", Form: "image/png"}},
				}},
			}}},
		}},
	}

	tests := []struct {
		name string
		doc  *gedcom.Document
		want []string
	}{
		{"decoded", decodeGED(t, writeGED), []string{
			"MANIFEST.MF", "META-INF/data.bin", "media/CharlotteBrontë.jpg", "media/a b.jpg",
			"media/thumb.png", "media/transcript.TXT", "notes.html",
		}},
		{"typed edit without SyncTags", stale, []string{
			"MANIFEST.MF", "META-INF/data.bin", "media/CharlotteBrontë.jpg", "media/a b.jpg",
			"media/thumb.png", "media/transcript.TXT", "notes.html",
		}},
		{"inline media link", inline, []string{"media/a b.jpg", "media/thumb.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := mustReader(t, mustWrite(t, tt.doc, writeMediaFS(), nil), nil)
			written, err := r.Decode(nil)
			if err != nil {
				t.Fatal(err)
			}
			got := mediaNames(r)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("media entries = %q, want %q", got, tt.want)
			}
			if named := localEntryNames(written); !reflect.DeepEqual(got, named) {
				t.Errorf("media entries = %q, but the written dataset names %q", got, named)
			}
			if problems := r.CheckRefs(written); problems != nil {
				t.Errorf("CheckRefs() = %+v, want none", problems)
			}
		})
	}
}

func TestWriteWithoutLocalReferences(t *testing.T) {
	const webOnly = "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n1 FILE https://example.com/a.jpg\n2 FORM image/jpeg\n" +
		"2 TRAN HTTP://example.com/a.png\n3 FORM image/png\n0 TRLR\n"
	for name, ged := range map[string]string{"web only": webOnly, "no media": minimalGED} {
		t.Run(name, func(t *testing.T) {
			r := mustReader(t, mustWrite(t, decodeGED(t, ged), nil, nil), nil)
			if entries := r.Entries(); len(entries) != 1 || entries[0].Name != datasetName {
				t.Errorf("Entries() = %+v, want only %s", entries, datasetName)
			}
			doc, err := r.Decode(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := encoded(t, doc); got != ged {
				t.Errorf("dataset = %q, want %q", got, ged)
			}
		})
	}
}

func TestWriteVersions(t *testing.T) {
	doc := func(version gedcom.Version) *gedcom.Document {
		return &gedcom.Document{Header: &gedcom.Header{Version: version}}
	}
	target := func(version gedcom.Version) *WriteOptions {
		return &WriteOptions{Encode: &encoder.EncodeOptions{TargetVersion: version}}
	}
	tests := []struct {
		name string
		doc  *gedcom.Document
		opts *WriteOptions
		ok   bool
	}{
		{"7.0", doc("7.0"), nil, true},
		{"7.0 patch", doc("7.0.18"), nil, true},
		{"7.1", doc("7.1"), nil, true},
		{"7.0 with encode options", doc("7.0"), &WriteOptions{Encode: &encoder.EncodeOptions{LineEnding: "\r\n"}}, true},
		{"5.5.1 retargeted to 7.0", doc("5.5.1"), target("7.0"), true},
		{"5.5.1", doc("5.5.1"), nil, false},
		{"5.5", doc("5.5"), &WriteOptions{}, false},
		{"7.0 retargeted to 5.5.1", doc("7.0"), target("5.5.1"), false},
		{"unknown", doc("8.0"), nil, false},
		{"malformed", doc("7.0.x"), nil, false},
		{"surrounding space", doc(" 7.0"), nil, false},
		{"no version", doc(""), nil, false},
		{"no header", &gedcom.Document{}, nil, false},
		{"no header, target 7.0", &gedcom.Document{}, target("7.0"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Write(&buf, tt.doc, nil, tt.opts)
			if !tt.ok {
				if !errors.Is(err, ErrNotGEDCOM7) {
					t.Errorf("Write error = %v, want ErrNotGEDCOM7", err)
				}
				if err != nil && !strings.Contains(err.Error(), "converter") {
					t.Errorf("error %q does not point at the converter package", err)
				}
				if buf.Len() != 0 {
					t.Errorf("%d bytes written on refusal", buf.Len())
				}
				return
			}
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			// What was accepted reads back as GEDCOM 7.
			got, err := mustReader(t, buf.Bytes(), nil).Decode(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Header.Version != gedcom.Version70 {
				t.Errorf("written dataset is version %q", got.Header.Version)
			}
		})
	}
}

func TestWriteRefusals(t *testing.T) {
	obje := func(lines ...string) *gedcom.Document {
		return decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n"+strings.Join(lines, "\n")+"\n0 TRLR\n")
	}
	media := writeMediaFS()
	tests := []struct {
		name     string
		doc      *gedcom.Document
		media    fs.FS
		want     error
		contains string
	}{
		{"nil document", nil, media, encoder.ErrNilDocument, ""},
		{"file URL", obje("1 FILE file:///home/a.jpg"), media, ErrNotLocal, `"file:///home/a.jpg" (file-url)`},
		{"file URL translation", obje("1 FILE notes.html", "2 TRAN FILE:/a.jpg"), media, ErrNotLocal, "@O1@"},
		{"parent segment", obje("1 FILE ../a.jpg"), media, ErrNotLocal, `"../a.jpg" (invalid)`},
		{"absolute", obje("1 FILE /media/a.jpg"), media, ErrNotLocal, ""},
		{"backslash", obje(`1 FILE C:\photos\a.jpg`), media, ErrNotLocal, ""},
		{"empty payload", obje("1 FILE"), media, ErrNotLocal, ""},
		{"reserved name", obje("1 FILE gedcom.ged"), fstest.MapFS{"gedcom.ged": {Data: []byte("x")}}, ErrReservedName, `"gedcom.ged"`},
		{"reserved name, escaped", obje("1 FILE notes.html", "2 TRAN gedcom%2Eged"), media, ErrReservedName, `"gedcom%2Eged"`},
		{"missing", obje("1 FILE media/none.jpg"), media, ErrMissingMedia, `"media/none.jpg"`},
		{"missing translation", obje("1 FILE notes.html", "2 TRAN none.vtt"), media, ErrMissingMedia, `"none.vtt"`},
		{"directory", obje("1 FILE media"), media, ErrMissingMedia, "not a regular file"},
		{"nil media", obje("1 FILE notes.html"), nil, ErrMissingMedia, "no media source"},
		{"name is case-sensitive", obje("1 FILE Notes.html"), media, ErrMissingMedia, ""},
		{"encoder refusal", &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version70},
			Records: []*gedcom.Record{{XRef: "@I1@", Type: "INDI", Tags: []*gedcom.Tag{{Level: 1, Tag: "NOTE", Value: "bad\xff"}}}},
		}, media, encoder.ErrInvalidUTF8, datasetName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Write(&buf, tt.doc, tt.media, nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Write error = %v, want %v", err, tt.want)
			}
			if !strings.Contains(err.Error(), tt.contains) {
				t.Errorf("error %q does not contain %q", err, tt.contains)
			}
			if buf.Len() != 0 {
				t.Errorf("%d bytes written on refusal", buf.Len())
			}
		})
	}
}

// TestWriteReportsEveryRefusedReference checks that one error carries every
// problem, each distinct payload once.
func TestWriteReportsEveryRefusedReference(t *testing.T) {
	doc := decodeGED(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE media/one.jpg
2 TRAN media/two.jpg
1 FILE notes.html
1 FILE media/one.jpg
0 @O2@ OBJE
1 FILE three.jpg
1 FILE file:///four.jpg
1 FILE gedcom.ged
1 FILE https://example.com/five.jpg
0 TRLR
`)
	var buf bytes.Buffer
	err := Write(&buf, doc, writeMediaFS(), nil)
	for _, want := range []error{ErrMissingMedia, ErrNotLocal, ErrReservedName} {
		if !errors.Is(err, want) {
			t.Errorf("Write error = %v, want it to wrap %v", err, want)
		}
	}
	if buf.Len() != 0 {
		t.Errorf("%d bytes written on refusal", buf.Len())
	}
	if err == nil {
		return
	}
	msg := err.Error()
	for payload, count := range map[string]int{
		`"media/one.jpg"`: 1, `"media/two.jpg"`: 1, `"three.jpg"`: 1, `"file:///four.jpg"`: 1, `"gedcom.ged"`: 1,
		"notes.html": 0, "five.jpg": 0,
	} {
		if got := strings.Count(msg, payload); got != count {
			t.Errorf("error mentions %s %d times, want %d:\n%s", payload, got, count, msg)
		}
	}
	if got := strings.Count(msg, ErrMissingMedia.Error()); got != 3 {
		t.Errorf("error reports %d missing files, want 3:\n%s", got, msg)
	}
}

// TestReadBackRejectsAnUnreadableDataset covers the guard behind the
// read-back: bytes the decoder cannot read name no references to trust.
func TestReadBackRejectsAnUnreadableDataset(t *testing.T) {
	_, err := readBack([]byte("not a dataset"))
	if err == nil || !strings.Contains(err.Error(), "read back") {
		t.Errorf("readBack error = %v, want a read-back failure", err)
	}
}

// TestWriteVersionIsTheOneWritten pins the GEDCOM 7 check to the bytes as
// well as to the prediction: the encoder writes a decoded header from its
// Tags but chooses its rules by Header.Version, so a document on which the
// two disagree is not a GEDCOM 7 dataset either way round.
func TestWriteVersionIsTheOneWritten(t *testing.T) {
	const body = "0 @I1@ INDI\n0 TRLR\n"

	old := decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n"+body)
	old.Header.Version = gedcom.Version70
	current := decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n"+body)
	current.Header.Version = gedcom.Version551

	for name, doc := range map[string]*gedcom.Document{
		"5.5.1 tags, Header.Version 7.0": old,
		"7.0 tags, Header.Version 5.5.1": current,
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Write(&buf, doc, nil, nil); !errors.Is(err, ErrNotGEDCOM7) {
				t.Errorf("Write error = %v, want ErrNotGEDCOM7", err)
			}
			if buf.Len() != 0 {
				t.Errorf("%d bytes written on refusal", buf.Len())
			}
		})
	}
}

// TestWriteCoversInlineMedia writes a decoded 5.5.1 document as 7.0. The
// encoder writes its records from their Tags, so the inline OBJE structure
// stays in the dataset, and its FILE payloads are references like any other.
func TestWriteCoversInlineMedia(t *testing.T) {
	inline := func(lines ...string) *gedcom.Document {
		return decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n"+
			"0 @I1@ INDI\n1 BIRT\n"+strings.Join(lines, "\n")+"\n1 OBJE @O1@\n"+
			"0 @O1@ OBJE\n1 FILE notes.html\n2 FORM html\n0 TRLR\n")
	}
	opts := &WriteOptions{Encode: &encoder.EncodeOptions{TargetVersion: gedcom.Version70}}

	doc := inline("2 OBJE", "3 FILE media/a%20b.jpg", "4 FORM jpeg", "1 OBJE", "2 FILE media/thumb.png", "3 FORM png")
	r := mustReader(t, mustWrite(t, doc, writeMediaFS(), opts), nil)
	want := []string{"media/a b.jpg", "media/thumb.png", "notes.html"}
	if got := mediaNames(r); !slices.Equal(got, want) {
		t.Errorf("entries = %q, want %q", got, want)
	}
	written, err := r.Decode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if problems := r.CheckRefs(written); problems != nil {
		t.Errorf("CheckRefs = %+v, want none", problems)
	}

	refusals := []struct {
		name string
		doc  *gedcom.Document
		want error
	}{
		{"missing", inline("1 OBJE", "2 FILE media/absent.jpg"), ErrMissingMedia},
		{"file URL", inline("1 OBJE", "2 FILE file:///tmp/a.jpg"), ErrNotLocal},
		{"dataset name", inline("1 OBJE", "2 FILE gedcom.ged"), ErrReservedName},
	}
	for _, tt := range refusals {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Write(&buf, tt.doc, writeMediaFS(), opts)
			if !errors.Is(err, tt.want) {
				t.Errorf("Write error = %v, want %v", err, tt.want)
			}
			if err != nil && !strings.Contains(err.Error(), "@I1@") {
				t.Errorf("error %q does not name the record", err)
			}
			if buf.Len() != 0 {
				t.Errorf("%d bytes written on refusal", buf.Len())
			}
		})
	}
}

// faultyFS is a media source whose one file, "a.jpg", passes the checks and
// then fails as it is opened, read or closed.
type faultyFS struct {
	open, read, close error
}

func (f faultyFS) Stat(name string) (fs.FileInfo, error) {
	return fileInfo{name: name, size: 4}, nil
}

func (f faultyFS) Open(name string) (fs.File, error) {
	if f.open != nil {
		return nil, f.open
	}
	var r io.Reader = strings.NewReader("data")
	if f.read != nil {
		r = iotest.ErrReader(f.read)
	}
	return &faultyFile{Reader: r, info: fileInfo{name: name, size: 4}, close: f.close}, nil
}

type faultyFile struct {
	io.Reader
	info  fileInfo
	close error
}

func (f *faultyFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *faultyFile) Close() error               { return f.close }

// failAfter is a writer that fails once n bytes have been written.
type failAfter struct {
	n   int
	err error
}

func (w *failAfter) Write(p []byte) (int, error) {
	if len(p) > w.n {
		n := w.n
		w.n = 0
		return n, w.err
	}
	w.n -= len(p)
	return len(p), nil
}

func TestWriteFailuresAfterValidation(t *testing.T) {
	doc := decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n1 FILE a.jpg\n0 TRLR\n")
	boom := errors.New("boom")

	for name, media := range map[string]faultyFS{
		"open":  {open: boom},
		"read":  {read: boom},
		"close": {close: boom},
	} {
		t.Run("media "+name, func(t *testing.T) {
			err := Write(io.Discard, doc, media, nil)
			if !errors.Is(err, boom) || !strings.Contains(err.Error(), `"a.jpg"`) {
				t.Errorf("Write error = %v, want boom naming the file", err)
			}
		})
	}
	t.Run("media read and close", func(t *testing.T) {
		readErr := errors.New("read failed")
		if err := Write(io.Discard, doc, faultyFS{read: readErr, close: boom}, nil); !errors.Is(err, readErr) {
			t.Errorf("Write error = %v, want the read error", err)
		}
	})

	// A writer failing at every possible offset: the error always surfaces.
	size := len(mustWrite(t, doc, faultyFS{}, nil))
	for n := 0; n < size; n++ {
		if err := Write(&failAfter{n: n, err: boom}, doc, faultyFS{}, nil); !errors.Is(err, boom) {
			t.Fatalf("writer failing after %d of %d bytes: Write error = %v, want boom", n, size, err)
		}
	}
}

// TestWriteStreamsMedia checks that a media file is copied through, not read
// whole: a file far larger than any buffer is stored intact.
func TestWriteStreamsMedia(t *testing.T) {
	big := bytes.Repeat([]byte("0123456789abcdef"), 1<<16) // 1 MiB
	doc := decodeGED(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n1 FILE big.bin\n2 TRAN big.txt\n0 TRLR\n")
	archive := mustWrite(t, doc, fstest.MapFS{"big.bin": {Data: big}, "big.txt": {Data: big}}, nil)
	r := mustReader(t, archive, nil)
	for _, name := range []string{"big.bin", "big.txt"} {
		got, err := readRef(r, name)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(big) {
			t.Errorf("%s did not survive the round trip", name)
		}
	}
	if len(archive) < len(big) || len(archive) > len(big)+len(big)/4 {
		t.Errorf("archive is %d bytes: want big.bin stored (%d) and big.txt deflated", len(archive), len(big))
	}
}
