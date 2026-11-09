package corpora

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseLock(t *testing.T) {
	text := "# comment\r\n\r\nbase\thttps://example.com/x/\r\n" +
		sum + "\t3\ta/b.ged\r\n" + sum + "\t0\tc d/ë.jpg\n"
	lock, err := ParseLock(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	want := Lock{Base: "https://example.com/x/", Entries: []Entry{
		{Path: "a/b.ged", Size: 3, SHA256: sum},
		{Path: "c d/ë.jpg", Size: 0, SHA256: sum},
	}}
	if fmt.Sprint(lock) != fmt.Sprint(want) {
		t.Errorf("got %+v, want %+v", lock, want)
	}
}

func TestParseLockErrors(t *testing.T) {
	base := "base\thttps://example.com/\n"
	for name, text := range map[string]string{
		"base without slash": "base\thttps://example.com\n" + sum + "\t1\ta\n",
		"no base":            sum + "\t1\ta\n",
		"no entries":         base,
		"two fields":         base + sum + "\t1\n",
		"short hash":         base + "abc\t1\ta\n",
		"uppercase hash":     base + strings.ToUpper(sum) + "\t1\ta\n",
		"bad size":           base + sum + "\tx\ta\n",
		"negative size":      base + sum + "\t-1\ta\n",
		"empty path":         base + sum + "\t1\t\n",
		"absolute path":      base + sum + "\t1\t/a\n",
		"parent path":        base + sum + "\t1\t../a\n",
		"unclean path":       base + sum + "\t1\ta//b\n",
		"dot path":           base + sum + "\t1\t./a\n",
		"backslash path":     base + sum + "\t1\ta\\b\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseLock(strings.NewReader(text)); err == nil {
				t.Error("ParseLock succeeded")
			}
		})
	}
}

func TestParseLockLongLine(t *testing.T) {
	text := "base\thttps://example.com/\n" + strings.Repeat("x", 70*1024) + "\n"
	if _, err := ParseLock(strings.NewReader(text)); err == nil {
		t.Error("ParseLock succeeded on a line longer than the scanner buffer")
	}
}

func TestGEDCOMIOLockIsPinned(t *testing.T) {
	lock := GEDCOMIOLock()
	if !strings.HasPrefix(lock.Base, "https://raw.githubusercontent.com/FamilySearch/GEDCOM.io/") {
		t.Errorf("base %q is not the GEDCOM.io repository", lock.Base)
	}
	var gdz int
	for _, e := range lock.Entries {
		if !strings.HasPrefix(e.Path, "gedcom70/") && !strings.HasPrefix(e.Path, "gedcom71/") {
			t.Errorf("entry %q is outside gedcom70/ and gedcom71/", e.Path)
		}
		if strings.HasSuffix(e.Path, ".gdz") {
			gdz++
		}
	}
	if gdz != 4 {
		t.Errorf("%d .gdz entries, want 4", gdz)
	}
}

func TestRepoRoot(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("RepoRoot %s holds no go.mod", root)
	}

	t.Chdir(t.TempDir())
	if root, err := RepoRoot(); err == nil {
		t.Errorf("RepoRoot found %s above a temporary directory", root)
	}
}

func TestMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ok"), 2)
	writeFile(t, filepath.Join(dir, "short"), 1)
	if err := os.Mkdir(filepath.Join(dir, "dir"), 0o750); err != nil {
		t.Fatal(err)
	}
	lock := Lock{Entries: []Entry{
		{Path: "ok", Size: 2}, {Path: "short", Size: 2}, {Path: "dir", Size: 0}, {Path: "absent", Size: 0},
	}}
	var got []string
	for _, e := range Missing(dir, lock) {
		got = append(got, e.Path)
	}
	if strings.Join(got, ",") != "short,dir,absent" {
		t.Errorf("Missing = %v, want [short dir absent]", got)
	}
}

// fakeTB records the Skip or Fatalf that GEDCOMIO makes instead of ending the
// test, so both outcomes can be checked.
type fakeTB struct {
	testing.TB
	skipped, failed string
}

func (f *fakeTB) Helper()                        {}
func (f *fakeTB) Skip(args ...any)               { f.skipped = fmt.Sprint(args...) }
func (f *fakeTB) Fatalf(format string, a ...any) { f.failed = fmt.Sprintf(format, a...) }

func TestGEDCOMIO(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), 0)
	t.Chdir(root)

	t.Run("absent skips", func(t *testing.T) {
		t.Setenv(RequireEnv, "")
		var tb fakeTB
		if dir, _ := GEDCOMIO(&tb); dir != "" || !strings.Contains(tb.skipped, "make fetch-corpora") || tb.failed != "" {
			t.Errorf("dir=%q skipped=%q failed=%q", dir, tb.skipped, tb.failed)
		}
	})
	t.Run("absent fails when required", func(t *testing.T) {
		t.Setenv(RequireEnv, "1")
		var tb fakeTB
		if dir, _ := GEDCOMIO(&tb); dir != "" || !strings.Contains(tb.failed, RequireEnv) || tb.skipped != "" {
			t.Errorf("dir=%q skipped=%q failed=%q", dir, tb.skipped, tb.failed)
		}
	})
	t.Run("present returns the directory", func(t *testing.T) {
		want := filepath.Join(root, filepath.FromSlash(GEDCOMIODir))
		for _, e := range GEDCOMIOLock().Entries {
			writeFile(t, filepath.Join(want, filepath.FromSlash(e.Path)), e.Size)
		}
		var tb fakeTB
		dir, lock := GEDCOMIO(&tb)
		if dir != want || len(lock.Entries) == 0 || tb.skipped != "" || tb.failed != "" {
			t.Errorf("dir=%q entries=%d skipped=%q failed=%q", dir, len(lock.Entries), tb.skipped, tb.failed)
		}
	})
	t.Run("no repository root fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		var tb fakeTB
		if dir, _ := GEDCOMIO(&tb); dir != "" || tb.failed == "" {
			t.Errorf("dir=%q failed=%q", dir, tb.failed)
		}
	})
}

// writeFile creates path, and its parents, holding size zero bytes.
func writeFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
}
