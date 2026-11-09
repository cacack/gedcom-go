package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// source is a fake upstream: it serves body under /base/<path> and counts
// requests. A path missing from body is a 404.
type source struct {
	body     map[string]string
	requests atomic.Int32
}

func (s *source) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)
	content, ok := s.body[strings.TrimPrefix(r.URL.Path, "/base/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(content))
}

func entry(path, content string) corpora.Entry {
	sum := sha256.Sum256([]byte(content))
	return corpora.Entry{Path: path, Size: int64(len(content)), SHA256: hex.EncodeToString(sum[:])}
}

// files is the pinned content the tests fetch; the paths need escaping.
var files = map[string]string{
	"gedcom70/a.ged":            "0 HEAD\n0 TRLR\n",
	"gedcom70/media/Brontë.jpg": "jpeg",
	"gedcom71/b c.ged":          "",
}

func newFetcher(t *testing.T, served map[string]string) (fetcher, *source) {
	t.Helper()
	src := &source{body: served}
	srv := httptest.NewServer(src)
	t.Cleanup(srv.Close)
	lock := corpora.Lock{Base: srv.URL + "/base/"}
	for _, p := range []string{"gedcom70/a.ged", "gedcom70/media/Brontë.jpg", "gedcom71/b c.ged"} {
		lock.Entries = append(lock.Entries, entry(p, files[p]))
	}
	return fetcher{client: srv.Client(), lock: lock, dir: t.TempDir()}, src
}

func fetch(f fetcher, require bool) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = f.fetchAll(require, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestFetchDownloadsAndVerifies(t *testing.T) {
	f, src := newFetcher(t, files)
	code, stdout, stderr := fetch(f, true)
	if code != 0 || !strings.Contains(stdout, "3 downloaded") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for p, want := range files {
		got, err := os.ReadFile(filepath.Join(f.dir, filepath.FromSlash(p)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", p, got, err, want)
		}
	}
	assertNoTempFiles(t, f.dir)

	// A second run finds everything in place and makes no requests.
	src.requests.Store(0)
	if code, stdout, _ := fetch(f, true); code != 0 || !strings.Contains(stdout, "3 already present") || src.requests.Load() != 0 {
		t.Errorf("rerun: code=%d stdout=%q requests=%d", code, stdout, src.requests.Load())
	}
}

func TestFetchReplacesAStaleLocalFile(t *testing.T) {
	f, _ := newFetcher(t, files)
	stale := filepath.Join(f.dir, "gedcom70", "a.ged")
	if err := os.MkdirAll(filepath.Dir(stale), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := fetch(f, true); code != 0 || !strings.Contains(stdout, "3 downloaded") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if got, _ := os.ReadFile(stale); string(got) != files["gedcom70/a.ged"] {
		t.Errorf("stale file not replaced: %q", got)
	}
}

func TestFetchNetworkFailureWarns(t *testing.T) {
	served := map[string]string{"gedcom70/a.ged": files["gedcom70/a.ged"]} // the rest 404

	t.Run("without -require", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		f, _ := newFetcher(t, served)
		code, stdout, stderr := fetch(f, false)
		if code != 0 || !strings.Contains(stdout, "2 unreachable") || !strings.Contains(stderr, "⚠ 2 files could not be downloaded") ||
			!strings.Contains(stderr, "HTTP 404") {
			t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
	t.Run("as a GitHub Actions annotation", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		f, _ := newFetcher(t, served)
		if code, _, stderr := fetch(f, false); code != 0 || !strings.Contains(stderr, "::warning::2 files") {
			t.Errorf("code=%d stderr=%q", code, stderr)
		}
	})
	t.Run("with -require", func(t *testing.T) {
		f, _ := newFetcher(t, served)
		if code, _, stderr := fetch(f, true); code != 1 || !strings.Contains(stderr, "(-require)") {
			t.Errorf("code=%d stderr=%q", code, stderr)
		}
	})
	t.Run("server unreachable", func(t *testing.T) {
		f, _ := newFetcher(t, files)
		f.lock.Base = "http://127.0.0.1:1/base/" // nothing listens on port 1
		if code, stdout, _ := fetch(f, false); code != 0 || !strings.Contains(stdout, "3 unreachable") {
			t.Errorf("code=%d stdout=%q", code, stdout)
		}
	})
	t.Run("body cut short", func(t *testing.T) {
		f, _ := newFetcher(t, files)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("0 HEAD"))
		}))
		t.Cleanup(srv.Close)
		f.lock.Base = srv.URL + "/base/"
		f.lock.Entries = f.lock.Entries[:1]
		if code, stdout, stderr := fetch(f, false); code != 0 || !strings.Contains(stdout, "1 unreachable") {
			t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		assertNoTempFiles(t, f.dir)
	})
}

func TestFetchContentMismatchFails(t *testing.T) {
	for name, content := range map[string]string{
		"different bytes": "0 HEAD\n0 TRLX\n",
		"longer":          files["gedcom70/a.ged"] + "extra",
		"shorter":         "0 HEAD\n",
	} {
		t.Run(name, func(t *testing.T) {
			served := map[string]string{}
			for p, c := range files {
				served[p] = c
			}
			served["gedcom70/a.ged"] = content
			f, _ := newFetcher(t, served)
			// Mismatch is fatal even without -require.
			code, _, stderr := fetch(f, false)
			if code != 1 || !strings.Contains(stderr, "differs from the lock") {
				t.Errorf("code=%d stderr=%q", code, stderr)
			}
			if _, err := os.Stat(filepath.Join(f.dir, "gedcom70", "a.ged")); err == nil {
				t.Error("mismatched content was written into place")
			}
			assertNoTempFiles(t, f.dir)
		})
	}
}

func TestFetchLocalErrorFails(t *testing.T) {
	f, _ := newFetcher(t, files)
	// A file where the gedcom70 directory belongs stops both the check of the
	// existing copy and the download: a local failure, never a warning.
	if err := os.WriteFile(filepath.Join(f.dir, "gedcom70"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := fetch(f, false); code != 1 || !strings.Contains(stdout, "2 failed") {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestURLEscapesSegments(t *testing.T) {
	f := fetcher{lock: corpora.Lock{Base: "https://example.com/x/"}}
	got := f.url(corpora.Entry{Path: "gedcom70/media/Charlotte Brontë.jpg"})
	if want := "https://example.com/x/gedcom70/media/Charlotte%20Bront%C3%AB.jpg"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	base := srv.URL + "/"

	runCmd := func(args ...string) (int, string) {
		var out, errOut bytes.Buffer
		code := run(args, &out, &errOut)
		return code, out.String() + errOut.String()
	}

	if code, _ := runCmd("-nosuchflag"); code != 2 {
		t.Errorf("bad flag: code %d, want 2", code)
	}
	if code, _ := runCmd("extra"); code != 2 {
		t.Errorf("extra argument: code %d, want 2", code)
	}
	dir := t.TempDir()
	if code, out := runCmd("-base", base, "-dir", dir); code != 0 || !strings.Contains(out, "unreachable") {
		t.Errorf("unreachable without -require: code %d, output %q", code, out)
	}
	if code, out := runCmd("-base", base, "-dir", dir, "-require"); code != 1 {
		t.Errorf("unreachable with -require: code %d, output %q", code, out)
	}

	// Without -dir, the corpus goes under the repository root.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	want := filepath.Join(root, filepath.FromSlash(corpora.GEDCOMIODir))
	if code, out := runCmd("-base", base); code != 0 || !strings.Contains(out, want) {
		t.Errorf("default dir: code %d, output %q, want it to name %s", code, out, want)
	}
	t.Chdir(t.TempDir())
	if code, out := runCmd("-base", base); code != 1 || !strings.Contains(out, "go.mod") {
		t.Errorf("no repository root: code %d, output %q", code, out)
	}
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(d.Name(), ".fetch-") {
			t.Errorf("temporary file left behind: %s", path)
		}
		return nil
	})
}
