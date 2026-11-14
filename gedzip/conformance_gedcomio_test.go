package gedzip

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// The four official GEDZIP archives of the FamilySearch/GEDCOM.io test files
// (#127), fetched by `make fetch-corpora` rather than vendored because the
// repository declares no licence. Each archive has an unpacked twin in the
// same lock: for gedcom70/maximal70.gdz, the files under gedcom70/maximal70/.
// The archive is read with the default limits and held to the twin byte for
// byte, then written back out and read again. They live here rather than
// beside the decoder's GEDCOM.io test because decoder cannot import gedzip.

// gedcomioWebRefs is how many web references each archive's dataset holds.
// The maximal files reference two http files, which have no entry.
var gedcomioWebRefs = map[string]int{
	"gedcom70/maximal70.gdz": 2,
	"gedcom70/minimal70.gdz": 0,
	"gedcom71/maximal71.gdz": 2,
	"gedcom71/minimal71.gdz": 0,
}

func TestGEDCOMIOArchives(t *testing.T) {
	dir, lock := corpora.GEDCOMIO(t)
	read := func(t *testing.T, rel string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))) // #nosec G304 -- a pinned lock entry under the fetched corpus dir
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		return data
	}

	var archives int
	for _, e := range lock.Entries {
		if !strings.HasSuffix(e.Path, ".gdz") {
			continue
		}
		archives++
		t.Run(e.Path, func(t *testing.T) {
			// The unpacked twin: every lock entry under the archive's own
			// directory, by entry name.
			twinDir := strings.TrimSuffix(e.Path, ".gdz") + "/"
			twin := make(map[string][]byte)
			for _, f := range lock.Entries {
				if name, ok := strings.CutPrefix(f.Path, twinDir); ok {
					twin[name] = read(t, f.Path)
				}
			}
			if _, ok := twin[datasetName]; !ok {
				t.Fatalf("the lock has no %s%s to compare with", twinDir, datasetName)
			}

			rc, err := OpenReader(filepath.Join(dir, filepath.FromSlash(e.Path)), nil)
			if err != nil {
				t.Fatalf("OpenReader() with default limits: %v", err)
			}
			defer func() { _ = rc.Close() }()
			r := &rc.Reader

			// Every entry is usable and is a file of the twin, and the
			// reverse.
			var names []string
			for _, entry := range r.Entries() {
				if entry.Unusable != nil {
					t.Errorf("entry %q is unusable: %v", entry.Name, entry.Unusable)
				}
				names = append(names, entry.Name)
			}
			sort.Strings(names)
			if want := sortedKeys(twin); !reflect.DeepEqual(names, want) {
				t.Errorf("entries = %q, want the unpacked twin's files %q", names, want)
			}
			if got := mustReadRef(t, r, datasetName); !bytes.Equal(got, twin[datasetName]) {
				t.Errorf("%s differs from the unpacked twin (%d bytes, want %d)", datasetName, len(got), len(twin[datasetName]))
			}

			for _, strict := range []bool{true, false} {
				opts := decoder.DefaultOptions()
				opts.StrictMode = strict
				res, err := r.DecodeWithDiagnostics(opts)
				if err != nil {
					t.Fatalf("DecodeWithDiagnostics(strict=%v): %v", strict, err)
				}
				for _, d := range res.Diagnostics {
					if d.Severity == decoder.SeverityError {
						t.Errorf("DecodeWithDiagnostics(strict=%v): %s", strict, d)
					}
				}
			}
			doc, err := r.Decode(nil)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			// Every local FILE and FILE.TRAN payload opens to the twin's
			// file, and every media file of the twin is referenced.
			var web int
			referenced := map[string]bool{datasetName: true}
			for _, ref := range collectFileRefs(doc) {
				kind, name := ClassifyFileRef(ref.ref)
				switch kind {
				case RefLocal:
					want, ok := twin[name]
					if !ok {
						t.Errorf("%s: %q names %q, which the unpacked twin lacks", ref.xref, ref.ref, name)
						continue
					}
					referenced[name] = true
					if got := mustReadRef(t, r, ref.ref); !bytes.Equal(got, want) {
						t.Errorf("%s: %q differs from the unpacked twin (%d bytes, want %d)", ref.xref, ref.ref, len(got), len(want))
					}
				case RefWeb:
					web++
					if !strings.HasPrefix(ref.ref, "http://") {
						t.Errorf("%s: web reference %q, want an http URL", ref.xref, ref.ref)
					}
				default:
					t.Errorf("%s: %q classifies as %s", ref.xref, ref.ref, kind)
				}
			}
			if want, ok := gedcomioWebRefs[e.Path]; !ok {
				t.Errorf("no web reference count in gedcomioWebRefs; the dataset has %d", web)
			} else if web != want {
				t.Errorf("%d web references, want %d", web, want)
			}
			if got, want := sortedKeys(referenced), sortedKeys(twin); !reflect.DeepEqual(got, want) {
				t.Errorf("local references name %q, want every file of the twin %q", got, want)
			}
			if problems := r.CheckRefs(doc); problems != nil {
				t.Errorf("CheckRefs() = %+v, want none", problems)
			}

			// Written back out with the archive as the media source, it reads
			// to the same document and the same media.
			again := mustReader(t, mustWrite(t, doc, r.FS(), nil), nil)
			rewritten, err := again.Decode(nil)
			if err != nil {
				t.Fatalf("Decode of the rewritten archive: %v", err)
			}
			assertSameDocument(t, rewritten, doc)
			if problems := again.CheckRefs(rewritten); problems != nil {
				t.Errorf("rewritten archive: CheckRefs() = %+v, want none", problems)
			}
			var rewrittenNames []string
			for _, entry := range again.Entries() {
				rewrittenNames = append(rewrittenNames, entry.Name)
				if entry.Name == datasetName {
					continue
				}
				if got := mustReadRef(t, again, entry.Name); !bytes.Equal(got, twin[entry.Name]) {
					t.Errorf("rewritten archive: %q differs from the unpacked twin (%d bytes, want %d)", entry.Name, len(got), len(twin[entry.Name]))
				}
			}
			sort.Strings(rewrittenNames)
			if !reflect.DeepEqual(rewrittenNames, names) {
				t.Errorf("rewritten archive: entries = %q, want %q", rewrittenNames, names)
			}
		})
	}
	if archives != len(gedcomioWebRefs) {
		t.Errorf("the lock has %d .gdz archives, want %d", archives, len(gedcomioWebRefs))
	}
}

// mustReadRef opens the entry a FILE payload names and reads it whole.
func mustReadRef(t *testing.T, r *Reader, ref string) []byte {
	t.Helper()
	rc, err := r.Open(ref)
	if err != nil {
		t.Fatalf("Open(%q): %v", ref, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read %q: %v", ref, err)
	}
	return data
}

// assertSameDocument fails t unless got encodes to the same dataset as want
// and holds the same typed records.
func assertSameDocument(t *testing.T, got, want *gedcom.Document) {
	t.Helper()
	if g, w := encoded(t, got), encoded(t, want); g != w {
		t.Errorf("the rewritten dataset encodes differently (%d bytes, want %d)", len(g), len(w))
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("the rewritten document differs from the one the archive decoded to")
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
