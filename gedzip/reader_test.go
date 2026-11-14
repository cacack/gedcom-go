package gedzip

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// minimalGED is the smallest dataset the tests package.
const minimalGED = "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME John /Doe/\n0 TRLR\n"

// zipEntry describes one entry of a test archive.
type zipEntry struct {
	name string
	data string
	// mod, if set, edits the header before it is written.
	mod func(*zip.FileHeader)
	// raw writes data as the entry's stored bytes with zip.Writer.CreateRaw,
	// so the header (which mod may make lie) is written exactly as given and
	// the entry has no data descriptor. The header starts as an honest Store
	// entry.
	raw bool
}

// file is a deflated entry written through zip.Writer.CreateHeader, which
// gives it a data descriptor.
func file(name, data string) zipEntry { return zipEntry{name: name, data: data} }

// buildZip writes entries, in order, to an in-memory archive.
func buildZip(tb testing.TB, entries ...zipEntry) []byte {
	tb.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.raw {
			hdr.Method = zip.Store
			hdr.CRC32 = crc32.ChecksumIEEE([]byte(e.data))
			hdr.CompressedSize64 = uint64(len(e.data))
			hdr.UncompressedSize64 = uint64(len(e.data))
		}
		if e.mod != nil {
			e.mod(hdr)
		}
		create := zw.CreateHeader
		if e.raw {
			create = zw.CreateRaw
		}
		w, err := create(hdr)
		if err != nil {
			tb.Fatalf("create %q: %v", e.name, err)
		}
		if _, err := io.WriteString(w, e.data); err != nil {
			tb.Fatalf("write %q: %v", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		tb.Fatalf("close archive: %v", err)
	}
	return buf.Bytes()
}

// newReader opens an in-memory archive.
func newReader(archive []byte, opts *ReadOptions) (*Reader, error) {
	return NewReader(bytes.NewReader(archive), int64(len(archive)), opts)
}

// mustReader opens an in-memory archive that has to be accepted.
func mustReader(tb testing.TB, archive []byte, opts *ReadOptions) *Reader {
	tb.Helper()
	r, err := newReader(archive, opts)
	if err != nil {
		tb.Fatalf("NewReader() error = %v", err)
	}
	return r
}

// readRef opens a reference and reads it whole.
func readRef(r *Reader, ref string) (string, error) {
	rc, err := r.Open(ref)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	return string(data), err
}

// declare makes a raw entry claim an uncompressed size it does not have.
func declare(size uint64) func(*zip.FileHeader) {
	return func(h *zip.FileHeader) { h.UncompressedSize64 = size }
}

// deflated is a raw entry holding data deflated, with a correct header that
// mod may then falsify.
func deflated(tb testing.TB, name, data string, mod func(*zip.FileHeader)) zipEntry {
	tb.Helper()
	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := io.WriteString(fw, data); err != nil {
		tb.Fatal(err)
	}
	if err := fw.Close(); err != nil {
		tb.Fatal(err)
	}
	return zipEntry{name: name, data: buf.String(), raw: true, mod: func(h *zip.FileHeader) {
		h.Method = zip.Deflate
		h.CRC32 = crc32.ChecksumIEEE([]byte(data))
		h.UncompressedSize64 = uint64(len(data))
		if mod != nil {
			mod(h)
		}
	}}
}

func TestReaderMinimalArchive(t *testing.T) {
	r := mustReader(t, buildZip(t, file("gedcom.ged", minimalGED)), nil)

	want, err := decoder.Decode(strings.NewReader(minimalGED))
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Decode(nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Decode() differs from decoding the inner stream directly")
	}

	res, err := r.DecodeWithDiagnostics(nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if !reflect.DeepEqual(res.Document, want) {
		t.Errorf("DecodeWithDiagnostics() document differs from decoding the inner stream directly")
	}

	wantEntries := []Entry{{Name: "gedcom.ged", Size: int64(len(minimalGED))}}
	if got := r.Entries(); !reflect.DeepEqual(got, wantEntries) {
		t.Errorf("Entries() = %+v, want %+v", got, wantEntries)
	}
}

// The dataset goes through the ordinary decoder, so a real file decodes to the
// same document inside an archive as outside one, diagnostics included.
func TestReaderDecodeMatchesDirectDecode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "gedcom-7.0", "maximal70.ged"))
	if err != nil {
		t.Fatal(err)
	}
	r := mustReader(t, buildZip(t, file("gedcom.ged", string(data))), nil)

	want, err := decoder.DecodeWithDiagnostics(bytes.NewReader(data), nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.DecodeWithDiagnostics(nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DecodeWithDiagnostics() differs from decoding the inner stream directly")
	}
	if len(got.Document.MediaObjects()) == 0 {
		t.Fatal("fixture has no OBJE records")
	}
}

func TestReaderDecodeOptionsArePassedThrough(t *testing.T) {
	const malformed = "0 HEAD\n1 GEDC\n2 VERS 7.0\nnot a line\n0 TRLR\n"
	r := mustReader(t, buildZip(t, file("gedcom.ged", malformed)), nil)

	if _, err := r.Decode(nil); err != nil {
		t.Errorf("lenient Decode() error = %v", err)
	}
	strict := &decoder.DecodeOptions{StrictMode: true}
	if _, err := r.Decode(strict); err == nil {
		t.Error("strict Decode() error = nil, want a parse error")
	}
	if _, err := r.DecodeWithDiagnostics(strict); err == nil {
		t.Error("strict DecodeWithDiagnostics() error = nil, want a parse error")
	}
	res, err := r.DecodeWithDiagnostics(nil)
	if err != nil || len(res.Diagnostics) == 0 {
		t.Errorf("lenient DecodeWithDiagnostics() = %d diagnostics, error %v; want diagnostics and no error",
			len(res.Diagnostics), err)
	}
}

// mixedArchive holds usable media beside one entry of every unusable kind.
func mixedArchive(t *testing.T) []byte {
	t.Helper()
	return buildZip(t,
		file("media/", ""),
		file("media/a b.jpg", "space"),
		file("gedcom.ged", minimalGED),
		file("media/CharlotteBrontë.jpg", "bronte"),
		file("50%20off.jpg", "literal percent"),
		file("Photo.jpg", "upper"),
		file("photo.jpg", "lower"),
		zipEntry{name: "stored.bin", data: "stored", raw: true},
		// Unusable from here on.
		file("../evil", "x"),
		file("/abs", "x"),
		file(`back\slash`, "x"),
		file("a//b", "x"),
		file("./dot", "x"),
		file("bad\xff.jpg", "x"),
		file("ctl\x01.jpg", "x"),
		zipEntry{name: "link", data: "gedcom.ged", mod: func(h *zip.FileHeader) { h.SetMode(fs.ModeSymlink | 0o777) }},
		zipEntry{name: "pipe", data: "x", mod: func(h *zip.FileHeader) { h.SetMode(fs.ModeNamedPipe | 0o644) }},
		zipEntry{name: "secret.jpg", data: "x", mod: func(h *zip.FileHeader) { h.Flags |= flagEncrypted }},
		zipEntry{name: "bzip2.bin", data: "x", raw: true, mod: func(h *zip.FileHeader) { h.Method = 12 }},
		file("dup.jpg", "first"),
		file("dup.jpg", "second"),
	)
}

func TestReaderOpen(t *testing.T) {
	r := mustReader(t, mixedArchive(t), nil)

	tests := []struct {
		name    string
		ref     string
		want    string
		wantErr []error
	}{
		{name: "percent-encoded space", ref: "media/a%20b.jpg", want: "space"},
		{name: "literal space", ref: "media/a b.jpg", want: "space"},
		{name: "percent-encoded non-ASCII", ref: "media/CharlotteBront%C3%AB.jpg", want: "bronte"},
		{name: "literal %20 in the entry name, reached by %2520", ref: "50%2520off.jpg", want: "literal percent"},
		{name: "entry names are not percent-decoded", ref: "50%20off.jpg", wantErr: []error{ErrEntryNotFound}},
		{name: "upper case", ref: "Photo.jpg", want: "upper"},
		{name: "lower case", ref: "photo.jpg", want: "lower"},
		{name: "no case folding", ref: "PHOTO.JPG", wantErr: []error{ErrEntryNotFound}},
		{name: "stored, no data descriptor", ref: "stored.bin", want: "stored"},
		{name: "the dataset", ref: "gedcom.ged", want: minimalGED},
		{name: "missing", ref: "media/none.jpg", wantErr: []error{ErrEntryNotFound}},
		{name: "directory entry is not an entry", ref: "media", wantErr: []error{ErrEntryNotFound}},
		{name: "web", ref: "https://example.com/a.jpg", wantErr: []error{ErrNotLocal}},
		{name: "file URL", ref: "file:///a.jpg", wantErr: []error{ErrNotLocal}},
		{name: "parent segment", ref: "../evil", wantErr: []error{ErrNotLocal}},
		{name: "absolute", ref: "/abs", wantErr: []error{ErrNotLocal}},
		{name: "backslash", ref: `back\slash`, wantErr: []error{ErrNotLocal}},
		{name: "empty", ref: "", wantErr: []error{ErrNotLocal}},
		{name: "symlink", ref: "link", wantErr: []error{ErrUnusableEntry}},
		{name: "named pipe", ref: "pipe", wantErr: []error{ErrUnusableEntry}},
		{name: "encrypted", ref: "secret.jpg", wantErr: []error{ErrUnusableEntry, ErrEncrypted}},
		{name: "unsupported method", ref: "bzip2.bin", wantErr: []error{ErrUnusableEntry}},
		{name: "duplicate", ref: "dup.jpg", wantErr: []error{ErrUnusableEntry}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readRef(r, tt.ref)
			for _, want := range tt.wantErr {
				if !errors.Is(err, want) {
					t.Errorf("Open(%q) error = %v, want %v", tt.ref, err, want)
				}
			}
			if len(tt.wantErr) == 0 && err != nil {
				t.Fatalf("Open(%q) error = %v", tt.ref, err)
			}
			if got != tt.want {
				t.Errorf("Open(%q) = %q, want %q", tt.ref, got, tt.want)
			}
			if len(tt.wantErr) > 0 && errors.Is(err, ErrEncrypted) != (tt.ref == "secret.jpg") {
				t.Errorf("Open(%q) error = %v, ErrEncrypted only for the encrypted entry", tt.ref, err)
			}
		})
	}
}

func TestReaderEntries(t *testing.T) {
	r := mustReader(t, mixedArchive(t), nil)

	usable := map[string]bool{
		"media/a b.jpg": true, "gedcom.ged": true, "media/CharlotteBrontë.jpg": true,
		"50%20off.jpg": true, "Photo.jpg": true, "photo.jpg": true, "stored.bin": true,
	}
	entries := r.Entries()
	if len(entries) != 20 {
		t.Errorf("Entries() has %d entries, want 20 (the directory entry is not listed)", len(entries))
	}
	for _, e := range entries {
		if e.Name == "media/" {
			t.Errorf("Entries() lists the directory entry %q", e.Name)
		}
		switch {
		case usable[e.Name] && e.Unusable != nil:
			t.Errorf("entry %q is unusable: %v", e.Name, e.Unusable)
		case !usable[e.Name] && !errors.Is(e.Unusable, ErrUnusableEntry):
			t.Errorf("entry %q Unusable = %v, want ErrUnusableEntry", e.Name, e.Unusable)
		}
	}
	if entries[1].Name != "gedcom.ged" || entries[1].Size != int64(len(minimalGED)) {
		t.Errorf("Entries()[1] = %+v, want gedcom.ged with its size, in archive order", entries[1])
	}

	entries[0].Name = "changed"
	if r.Entries()[0].Name == "changed" {
		t.Error("Entries() returned the Reader's own slice")
	}
}

func TestReaderFS(t *testing.T) {
	r := mustReader(t, mixedArchive(t), nil)
	fsys := r.FS()

	media := []string{"media/a b.jpg", "media/CharlotteBrontë.jpg", "50%20off.jpg", "Photo.jpg", "photo.jpg", "stored.bin"}
	if err := fstest.TestFS(fsys, media...); err != nil {
		t.Fatal(err)
	}

	var walked []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			walked = append(walked, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(walked) != len(media) {
		t.Errorf("FS holds %q, want exactly the usable media %q", walked, media)
	}

	data, err := fs.ReadFile(fsys, "media/a b.jpg")
	if err != nil || string(data) != "space" {
		t.Errorf("ReadFile() = %q, %v", data, err)
	}

	for _, name := range []string{"gedcom.ged", "link", "secret.jpg", "dup.jpg", "bzip2.bin", "nope", "media/nope"} {
		if _, err := fsys.Open(name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("FS Open(%q) error = %v, want fs.ErrNotExist", name, err)
		}
	}
	for _, name := range []string{"../evil", "/abs", "a//b", "./dot", ""} {
		if _, err := fsys.Open(name); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("FS Open(%q) error = %v, want fs.ErrInvalid", name, err)
		}
	}

	dir, err := fsys.Open("media")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := dir.Stat(); err != nil || !info.IsDir() || info.Mode().Perm()&0o222 != 0 || info.Sys() != nil {
		t.Errorf("directory Stat() = %+v, %v; want a read-only directory", info, err)
	}
	if _, err := dir.Read(make([]byte, 1)); err == nil {
		t.Error("reading a directory succeeded")
	}
	if err := dir.Close(); err != nil {
		t.Error(err)
	}
}

// A file entry that is also the directory of another opens as a file, and so
// does the entry beneath it.
func TestReaderFSFileAndDirectoryOfOneName(t *testing.T) {
	for _, order := range [][]string{{"a", "a/b"}, {"a/b", "a"}} {
		r := mustReader(t, buildZip(t, file("gedcom.ged", minimalGED), file(order[0], order[0]), file(order[1], order[1])), nil)
		fsys := r.FS()
		for _, name := range order {
			if data, err := fs.ReadFile(fsys, name); err != nil || string(data) != name {
				t.Errorf("ReadFile(%q) = %q, %v", name, data, err)
			}
		}
		list, err := fs.ReadDir(fsys, ".")
		if err != nil || len(list) != 1 || list[0].Name() != "a" || list[0].IsDir() {
			t.Errorf("ReadDir(.) = %v, %v; want the single file a", list, err)
		}
	}
}

func TestReaderFSReadsUnderTheEntryLimit(t *testing.T) {
	archive := buildZip(t, file("gedcom.ged", minimalGED), file("a.jpg", "x"))
	f, err := mustReader(t, archive, nil).FS().Open("a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, ok := f.(*mediaFile).ReadCloser.(*limitedReadCloser); !ok {
		t.Errorf("FS file reads through %T, want *limitedReadCloser", f.(*mediaFile).ReadCloser)
	}
}

func TestNewReaderDataset(t *testing.T) {
	encrypted := func(h *zip.FileHeader) { h.Flags |= flagEncrypted }
	symlink := func(h *zip.FileHeader) { h.SetMode(fs.ModeSymlink | 0o777) }

	tests := []struct {
		name        string
		entries     []zipEntry
		wantErr     error
		notErr      error
		wantText    []string
		notWantText []string
	}{
		{name: "empty archive", wantErr: ErrNoDataset},
		{name: "media only", entries: []zipEntry{file("a.jpg", "x")}, wantErr: ErrNoDataset, notWantText: []string{"found"}},
		{
			name:        "misnamed dataset at the root",
			entries:     []zipEntry{file("Family.GED", minimalGED), file("sub/inner.ged", minimalGED), file("a.jpg", "x")},
			wantErr:     ErrNoDataset,
			wantText:    []string{`"Family.GED"`, `exactly "gedcom.ged"`},
			notWantText: []string{"inner.ged", "a.jpg"},
		},
		{name: "name is case-sensitive", entries: []zipEntry{file("GEDCOM.GED", minimalGED)}, wantErr: ErrNoDataset, wantText: []string{`"GEDCOM.GED"`}},
		{name: "dataset in a directory", entries: []zipEntry{file("data/gedcom.ged", minimalGED)}, wantErr: ErrNoDataset},
		{name: "directory named like the dataset", entries: []zipEntry{file("gedcom.ged/", "")}, wantErr: ErrNoDataset},
		{
			name: "many misnamed datasets",
			entries: []zipEntry{
				file("1.ged", ""), file("2.ged", ""), file("3.ged", ""), file("4.ged", ""),
				file("5.ged", ""), file("6.ged", ""), file("7.ged", ""),
			},
			wantErr:     ErrNoDataset,
			wantText:    []string{`"5.ged"`, "and 2 more"},
			notWantText: []string{`"6.ged"`},
		},
		{
			name:    "encrypted dataset",
			entries: []zipEntry{{name: "gedcom.ged", data: minimalGED, mod: encrypted}},
			wantErr: ErrEncrypted,
		},
		{
			name:    "duplicate dataset",
			entries: []zipEntry{file("gedcom.ged", minimalGED), file("gedcom.ged", minimalGED)},
			wantErr: ErrUnusableEntry,
			notErr:  ErrEncrypted,
		},
		{
			name:    "symlink dataset",
			entries: []zipEntry{{name: "gedcom.ged", data: "x", mod: symlink}},
			wantErr: ErrUnusableEntry,
			notErr:  ErrEncrypted,
		},
		{
			name:    "dataset with an unsupported method",
			entries: []zipEntry{{name: "gedcom.ged", data: "x", raw: true, mod: func(h *zip.FileHeader) { h.Method = 12 }}},
			wantErr: ErrUnusableEntry,
		},
		{
			name: "unusable media does not block the dataset",
			entries: []zipEntry{
				file("gedcom.ged", minimalGED), file("../evil", "x"),
				{name: "secret.jpg", data: "x", mod: encrypted}, file("dup", "1"), file("dup", "2"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newReader(buildZip(t, tt.entries...), nil)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("NewReader() error = %v", err)
				}
				if _, err := r.Decode(nil); err != nil {
					t.Errorf("Decode() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewReader() error = %v, want %v", err, tt.wantErr)
			}
			if tt.notErr != nil && errors.Is(err, tt.notErr) {
				t.Errorf("NewReader() error = %v, must not be %v", err, tt.notErr)
			}
			if r != nil {
				t.Error("NewReader() returned a Reader with an error")
			}
			for _, text := range tt.wantText {
				if !strings.Contains(err.Error(), text) {
					t.Errorf("error %q does not mention %s", err, text)
				}
			}
			for _, text := range tt.notWantText {
				if strings.Contains(err.Error(), text) {
					t.Errorf("error %q mentions %s", err, text)
				}
			}
		})
	}
}

func TestNewReaderRejectsWhatIsNotAnArchive(t *testing.T) {
	if _, err := newReader([]byte(minimalGED), nil); err == nil {
		t.Error("NewReader() of a GEDCOM file error = nil")
	}
	if _, err := newReader(nil, nil); err == nil {
		t.Error("NewReader() of nothing error = nil")
	}
	if _, err := NewReader(nil, 0, nil); err == nil {
		t.Error("NewReader(nil) error = nil")
	}
	archive := buildZip(t, file("gedcom.ged", minimalGED))
	if _, err := NewReader(bytes.NewReader(archive), -1, nil); err == nil {
		t.Error("NewReader() with a negative size error = nil")
	}
}

// With GODEBUG=zipinsecurepath=0 archive/zip reports insecure names itself,
// alongside a usable reader. The names are already unreachable here, so the
// archive still opens.
func TestNewReaderWithZipInsecurePathCheck(t *testing.T) {
	t.Setenv("GODEBUG", "zipinsecurepath=0")
	archive := buildZip(t, file("gedcom.ged", minimalGED), file("../evil", "x"))
	if _, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive))); !errors.Is(err, zip.ErrInsecurePath) {
		t.Skipf("archive/zip did not report the insecure name (error %v)", err)
	}
	r := mustReader(t, archive, nil)
	if _, err := r.Decode(nil); err != nil {
		t.Errorf("Decode() error = %v", err)
	}
	if e := r.Entries()[1]; !errors.Is(e.Unusable, ErrUnusableEntry) {
		t.Errorf("entry %q Unusable = %v, want ErrUnusableEntry", e.Name, e.Unusable)
	}
}

// Entries written through zip.Writer.CreateHeader carry their sizes in a data
// descriptor after the data; raw ones carry them in the local header.
func TestReaderDataDescriptorEntries(t *testing.T) {
	archive := buildZip(t,
		file("gedcom.ged", minimalGED),
		file("described.jpg", "described"),
		zipEntry{name: "plain.jpg", data: "plain", raw: true},
	)
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	const dataDescriptor = 0x8
	if zr.File[1].Flags&dataDescriptor == 0 || zr.File[2].Flags&dataDescriptor != 0 {
		t.Fatalf("fixture flags = %#x, %#x; want a data descriptor on the first only", zr.File[1].Flags, zr.File[2].Flags)
	}

	r := mustReader(t, archive, nil)
	if _, err := r.Decode(nil); err != nil {
		t.Errorf("Decode() error = %v", err)
	}
	for ref, want := range map[string]string{"described.jpg": "described", "plain.jpg": "plain"} {
		if got, err := readRef(r, ref); err != nil || got != want {
			t.Errorf("Open(%q) = %q, %v; want %q", ref, got, err, want)
		}
	}
}

// An entry whose data does not match the size it declares is an error either
// way: never a short read, and never more bytes than declared.
func TestReaderLyingDeclaredSize(t *testing.T) {
	const data = "0123456789"
	for _, tt := range []struct {
		name     string
		declared uint64
	}{
		{"understated", 4},
		{"overstated", 64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			archive := buildZip(t,
				deflated(t, "gedcom.ged", minimalGED, declare(tt.declared)),
				deflated(t, "deflated.jpg", data, declare(tt.declared)),
				zipEntry{name: "stored.jpg", data: data, raw: true, mod: declare(tt.declared)},
			)
			r := mustReader(t, archive, nil)
			for _, ref := range []string{"deflated.jpg", "stored.jpg"} {
				got, err := readRef(r, ref)
				if err == nil {
					t.Errorf("reading %q with an %s size succeeded with %q", ref, tt.name, got)
				}
				if uint64(len(got)) > tt.declared {
					t.Errorf("reading %q returned %d bytes, more than the %d declared", ref, len(got), tt.declared)
				}
			}
			if _, err := r.Decode(nil); err == nil {
				t.Errorf("Decode() of a dataset with an %s size succeeded", tt.name)
			}
			if _, err := r.DecodeWithDiagnostics(nil); err == nil {
				t.Errorf("DecodeWithDiagnostics() of a dataset with an %s size succeeded", tt.name)
			}
		})
	}
}

func TestReaderChecksumMismatch(t *testing.T) {
	archive := buildZip(t,
		file("gedcom.ged", minimalGED),
		zipEntry{name: "a.jpg", data: "data", raw: true, mod: func(h *zip.FileHeader) { h.CRC32++ }},
	)
	if _, err := readRef(mustReader(t, archive, nil), "a.jpg"); !errors.Is(err, zip.ErrChecksum) {
		t.Errorf("read error = %v, want zip.ErrChecksum", err)
	}
}

// A central directory entry pointing at something that is not a local file
// header fails when opened, not when read.
func TestReaderBrokenLocalHeader(t *testing.T) {
	archive := buildZip(t, file("gedcom.ged", minimalGED), file("a.jpg", "x"))
	archive[0] ^= 0xFF // the first local header's signature
	r := mustReader(t, archive, nil)
	if _, err := r.Decode(nil); !errors.Is(err, zip.ErrFormat) {
		t.Errorf("Decode() error = %v, want zip.ErrFormat", err)
	}
	if _, err := r.DecodeWithDiagnostics(nil); !errors.Is(err, zip.ErrFormat) {
		t.Errorf("DecodeWithDiagnostics() error = %v, want zip.ErrFormat", err)
	}
	if _, err := r.Open("gedcom.ged"); !errors.Is(err, zip.ErrFormat) {
		t.Errorf("Open() error = %v, want zip.ErrFormat", err)
	}
	if got, err := readRef(r, "a.jpg"); err != nil || got != "x" {
		t.Errorf("Open(a.jpg) = %q, %v; the other entry must still read", got, err)
	}

	archive = buildZip(t, file("gedcom.ged", minimalGED), file("a.jpg", "x"))
	i := bytes.LastIndex(archive, []byte("PK\x03\x04"))
	if i <= 0 {
		t.Fatal("fixture has no second local header")
	}
	archive[i] ^= 0xFF // the second local header's signature
	if _, err := mustReader(t, archive, nil).FS().Open("a.jpg"); !errors.Is(err, zip.ErrFormat) {
		t.Errorf("FS Open() error = %v, want zip.ErrFormat", err)
	}
}

func TestReadOptionsDefaults(t *testing.T) {
	want := limits{
		entries:  100_000,
		document: 256 << 20,
		entry:    1 << 30,
		total:    4 << 30,
		ratio:    200,
	}
	var nilOpts *ReadOptions
	if got := nilOpts.resolve(); got != want {
		t.Errorf("nil options resolve to %+v, want %+v", got, want)
	}
	if got := (&ReadOptions{}).resolve(); got != want {
		t.Errorf("zero options resolve to %+v, want %+v", got, want)
	}
	custom := &ReadOptions{MaxEntries: 1, MaxDocumentBytes: 2, MaxEntryBytes: -1, MaxTotalBytes: 4, MaxCompressionRatio: -5}
	if got, want := custom.resolve(), (limits{entries: 1, document: 2, entry: -1, total: 4, ratio: -5}); got != want {
		t.Errorf("custom options resolve to %+v, want %+v", got, want)
	}
	if CompressionRatioMinBytes != 1<<20 {
		t.Errorf("CompressionRatioMinBytes = %d, want 1 MiB", CompressionRatioMinBytes)
	}
}

func TestReaderLimits(t *testing.T) {
	const (
		mib = 1 << 20
		gib = 1 << 30
	)
	docLen := int64(len(minimalGED))
	zeros := strings.Repeat("\x00", 2*mib)
	noRatio := func(o ReadOptions) *ReadOptions { o.MaxCompressionRatio = -1; return &o }
	// lie is a stored entry of one byte claiming size bytes, compressed and
	// uncompressed alike.
	lie := func(name string, size uint64) zipEntry {
		return zipEntry{name: name, data: "x", raw: true, mod: func(h *zip.FileHeader) {
			h.CompressedSize64, h.UncompressedSize64 = size, size
		}}
	}
	ged := file("gedcom.ged", minimalGED)

	many := make([]zipEntry, 0, DefaultMaxEntries+1)
	many = append(many, zipEntry{name: "gedcom.ged", data: minimalGED, raw: true})
	for i := 0; i < DefaultMaxEntries; i++ {
		many = append(many, zipEntry{name: fmt.Sprintf("m/%d", i), raw: true})
	}
	manyArchive := buildZip(t, many...)
	threeArchive := buildZip(t, ged, file("dir/", ""), file("a.jpg", "x"))
	docArchive := buildZip(t, ged)
	bigDocArchive := buildZip(t, lie("gedcom.ged", 256*mib+1))
	mediaArchive := buildZip(t, ged, file("a.jpg", "0123456789"))
	bigMediaArchive := buildZip(t, ged, lie("a.bin", gib+1))
	totalArchive := buildZip(t, ged, file("a.jpg", "0123456789"), file("b.jpg", "0123456789"))
	bigTotalArchive := buildZip(t, ged, lie("a", gib), lie("b", gib), lie("c", gib), lie("d", gib))
	hugeArchive := buildZip(t, ged, lie("a", math.MaxUint64-1), lie("b", math.MaxUint64-1))
	bombArchive := buildZip(t, ged, file("zeros.bin", zeros))
	smallBombArchive := buildZip(t, ged, file("zeros.bin", zeros[:mib]))
	bombDocArchive := buildZip(t, file("gedcom.ged", minimalGED+strings.Repeat("0 @N@ NOTE x\n", 200_000)))
	// A stored entry claiming as many compressed bytes as uncompressed ones
	// looks like ratio 1, but the archive is far too short to hold them.
	ratioLieArchive := buildZip(t, ged, lie("a.bin", 2*mib))

	tests := []struct {
		name    string
		archive []byte
		opts    *ReadOptions
		want    string // the option the error names; "" for success
	}{
		{"entries: default, over", manyArchive, nil, "MaxEntries"},
		{"entries: unlimited", manyArchive, &ReadOptions{MaxEntries: -1}, ""},
		{"entries: custom, over, directories count", threeArchive, &ReadOptions{MaxEntries: 2}, "MaxEntries"},
		{"entries: custom, at the limit", threeArchive, &ReadOptions{MaxEntries: 3}, ""},

		{"document: default, over", bigDocArchive, noRatio(ReadOptions{}), "MaxDocumentBytes"},
		{"document: unlimited", bigDocArchive, noRatio(ReadOptions{MaxDocumentBytes: -1}), ""},
		{"document: custom, over", docArchive, &ReadOptions{MaxDocumentBytes: docLen - 1}, "MaxDocumentBytes"},
		{"document: custom, at the limit", docArchive, &ReadOptions{MaxDocumentBytes: docLen}, ""},
		{"document: not bound by the entry limit", docArchive, &ReadOptions{MaxEntryBytes: 1}, ""},

		{"entry: default, over", bigMediaArchive, noRatio(ReadOptions{}), "MaxEntryBytes"},
		{"entry: unlimited", bigMediaArchive, noRatio(ReadOptions{MaxEntryBytes: -1}), ""},
		{"entry: custom, over", mediaArchive, &ReadOptions{MaxEntryBytes: 9}, "MaxEntryBytes"},
		{"entry: custom, at the limit", mediaArchive, &ReadOptions{MaxEntryBytes: 10}, ""},
		{"entry: not bound by the document limit", mediaArchive, &ReadOptions{MaxDocumentBytes: docLen}, ""},

		{"total: default, over", bigTotalArchive, noRatio(ReadOptions{}), "MaxTotalBytes"},
		{"total: unlimited", bigTotalArchive, noRatio(ReadOptions{MaxTotalBytes: -1}), ""},
		{"total: custom, over", totalArchive, &ReadOptions{MaxTotalBytes: docLen + 19}, "MaxTotalBytes"},
		{"total: custom, at the limit", totalArchive, &ReadOptions{MaxTotalBytes: docLen + 20}, ""},
		{"total: sum past 64 bits, default", hugeArchive, noRatio(ReadOptions{MaxEntryBytes: -1}), "MaxTotalBytes"},
		{"total: sum past 64 bits, unlimited", hugeArchive, noRatio(ReadOptions{MaxEntryBytes: -1, MaxTotalBytes: -1}), ""},

		{"ratio: default, over", bombArchive, nil, "MaxCompressionRatio"},
		{"ratio: unlimited", bombArchive, &ReadOptions{MaxCompressionRatio: -1}, ""},
		{"ratio: custom, over", bombArchive, &ReadOptions{MaxCompressionRatio: 500}, "MaxCompressionRatio"},
		{"ratio: custom, under", bombArchive, &ReadOptions{MaxCompressionRatio: 5000}, ""},
		{"ratio: not applied at 1 MiB", smallBombArchive, nil, ""},
		{"ratio: applies to the dataset", bombDocArchive, nil, "MaxCompressionRatio"},
		{"ratio: compressed size beyond the archive", ratioLieArchive, nil, "MaxCompressionRatio"},
		{"ratio: compressed size beyond the archive, unlimited", ratioLieArchive, &ReadOptions{MaxCompressionRatio: -1}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newReader(tt.archive, tt.opts)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("NewReader() error = %v", err)
				}
				return
			}
			if !errors.Is(err, ErrLimitExceeded) {
				t.Fatalf("NewReader() error = %v, want ErrLimitExceeded", err)
			}
			if !strings.Contains(err.Error(), tt.want+" ") {
				t.Errorf("error %q does not name %s", err, tt.want)
			}
			if r != nil {
				t.Error("NewReader() returned a Reader with an error")
			}
		})
	}

	t.Run("accepted entries read whole", func(t *testing.T) {
		r := mustReader(t, bombArchive, &ReadOptions{MaxCompressionRatio: -1})
		if got, err := readRef(r, "zeros.bin"); err != nil || got != zeros {
			t.Errorf("read %d bytes, error %v; want %d bytes", len(got), err, len(zeros))
		}
		r = mustReader(t, mediaArchive, &ReadOptions{MaxEntryBytes: 10, MaxDocumentBytes: docLen})
		if got, err := readRef(r, "a.jpg"); err != nil || got != "0123456789" {
			t.Errorf("entry of exactly MaxEntryBytes read as %q, %v", got, err)
		}
		if _, err := r.Decode(nil); err != nil {
			t.Errorf("dataset of exactly MaxDocumentBytes: Decode() error = %v", err)
		}
	})

	t.Run("declared size is clamped", func(t *testing.T) {
		r := mustReader(t, hugeArchive, noRatio(ReadOptions{MaxEntryBytes: -1, MaxTotalBytes: -1}))
		if got := r.Entries()[1].Size; got != math.MaxInt64 {
			t.Errorf("Size = %d, want math.MaxInt64", got)
		}
	})
}

// Reads are limited by the option that applies to the entry, and not at all
// when that option is unlimited.
func TestReaderOpenAppliesTheReadLimit(t *testing.T) {
	archive := buildZip(t, file("gedcom.ged", minimalGED), file("a.jpg", "x"))
	tests := []struct {
		name  string
		opts  *ReadOptions
		ref   string
		limit int64 // -1 for no limiting reader
		field string
	}{
		{"media, default", nil, "a.jpg", DefaultMaxEntryBytes, "MaxEntryBytes"},
		{"media, custom", &ReadOptions{MaxEntryBytes: 7}, "a.jpg", 7, "MaxEntryBytes"},
		{"media, unlimited", &ReadOptions{MaxEntryBytes: -1}, "a.jpg", -1, ""},
		{"dataset, default", nil, "gedcom.ged", DefaultMaxDocumentBytes, "MaxDocumentBytes"},
		{"dataset, custom", &ReadOptions{MaxDocumentBytes: 99}, "gedcom.ged", 99, "MaxDocumentBytes"},
		{"dataset, unlimited", &ReadOptions{MaxDocumentBytes: -1}, "gedcom.ged", -1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := mustReader(t, archive, tt.opts).Open(tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			l, limited := rc.(*limitedReadCloser)
			if limited != (tt.limit >= 0) {
				t.Fatalf("Open() returned %T", rc)
			}
			if limited && (l.remaining != tt.limit || l.field != tt.field) {
				t.Errorf("limit = %d (%s), want %d (%s)", l.remaining, l.field, tt.limit, tt.field)
			}
		})
	}
}

func TestLimitedReadCloser(t *testing.T) {
	const data = "0123456789"
	tests := []struct {
		name    string
		limit   int64
		want    string
		wantErr bool
	}{
		{"under the limit", 11, data, false},
		{"exactly the limit", 10, data, false},
		{"one over", 9, data[:9], true},
		{"far over", 3, data[:3], true},
		{"zero limit", 0, "", true},
		{"largest limit", math.MaxInt64, data, false},
	}
	wrap := map[string]func(io.Reader) io.Reader{
		"whole reads":    func(r io.Reader) io.Reader { return r },
		"one-byte reads": iotest.OneByteReader,
		"data with EOF":  iotest.DataErrReader,
	}
	for _, tt := range tests {
		for mode, reader := range wrap {
			t.Run(tt.name+", "+mode, func(t *testing.T) {
				l := &limitedReadCloser{rc: io.NopCloser(reader(strings.NewReader(data))), remaining: tt.limit, name: "e", field: "MaxEntryBytes"}
				got, err := io.ReadAll(l)
				if string(got) != tt.want {
					t.Errorf("read %q, want %q", got, tt.want)
				}
				if errors.Is(err, ErrLimitExceeded) != tt.wantErr || (!tt.wantErr && err != nil) {
					t.Errorf("error = %v, want limit error: %v", err, tt.wantErr)
				}
				if tt.wantErr {
					if n, again := l.Read(make([]byte, 4)); n != 0 || !errors.Is(again, ErrLimitExceeded) {
						t.Errorf("read after the limit = %d, %v; want 0 and the same error", n, again)
					}
				}
				if err := l.Close(); err != nil {
					t.Error(err)
				}
			})
		}
	}

	t.Run("an empty entry at a zero limit", func(t *testing.T) {
		l := &limitedReadCloser{rc: io.NopCloser(strings.NewReader(""))}
		if got, err := io.ReadAll(l); err != nil || len(got) != 0 {
			t.Errorf("read %q, %v", got, err)
		}
	})
}

func TestOpenReader(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "family.gdz")
	if err := os.WriteFile(archive, buildZip(t, file("gedcom.ged", minimalGED), file("a.jpg", "x")), 0o600); err != nil {
		t.Fatal(err)
	}
	notArchive := filepath.Join(dir, "family.ged")
	if err := os.WriteFile(notArchive, []byte(minimalGED), 0o600); err != nil {
		t.Fatal(err)
	}

	rc, err := OpenReader(archive, nil)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	doc, err := rc.Decode(nil)
	if err != nil || len(doc.Individuals()) != 1 {
		t.Errorf("Decode() = %v, %v", doc, err)
	}
	if got, err := readRef(&rc.Reader, "a.jpg"); err != nil || got != "x" {
		t.Errorf("Open() = %q, %v", got, err)
	}
	if err := rc.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if _, err := rc.Decode(nil); err == nil {
		t.Error("Decode() after Close() succeeded")
	}

	if _, err := OpenReader(filepath.Join(dir, "missing.gdz"), nil); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("OpenReader() of a missing file error = %v, want fs.ErrNotExist", err)
	}
	if _, err := OpenReader(notArchive, nil); !errors.Is(err, zip.ErrFormat) {
		t.Errorf("OpenReader() of a GEDCOM file error = %v, want zip.ErrFormat", err)
	}
	if _, err := OpenReader(archive, &ReadOptions{MaxEntries: 1}); !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("OpenReader() over a limit error = %v, want ErrLimitExceeded", err)
	}
}

// A Reader that was not made by NewReader has no archive; it must say so
// rather than panic.
func TestZeroReader(t *testing.T) {
	var r Reader
	if _, err := r.Decode(nil); !errors.Is(err, ErrNoDataset) {
		t.Errorf("Decode() error = %v, want ErrNoDataset", err)
	}
	if _, err := r.DecodeWithDiagnostics(nil); !errors.Is(err, ErrNoDataset) {
		t.Errorf("DecodeWithDiagnostics() error = %v, want ErrNoDataset", err)
	}
	if _, err := r.Open("a.jpg"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("Open() error = %v, want ErrEntryNotFound", err)
	}
	if got := r.Entries(); len(got) != 0 {
		t.Errorf("Entries() = %v", got)
	}
	if list, err := fs.ReadDir(r.FS(), "."); err != nil || len(list) != 0 {
		t.Errorf("FS is not empty: %v, %v", list, err)
	}
	if got := r.CheckRefs(nil); got != nil {
		t.Errorf("CheckRefs(nil) = %v", got)
	}
}
