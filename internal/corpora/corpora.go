// Package corpora describes the third-party test corpora that are fetched at
// test time instead of vendored, and gives tests access to them.
//
// The FamilySearch/GEDCOM.io test files declare no licence, so they may not be
// redistributed in this repository. gedcomio.lock pins them instead: a base
// URL at an immutable commit, and each file's SHA-256 and size.
// `make fetch-corpora` (internal/cmd/fetchcorpora) downloads them into
// .corpora/gedcomio/ at the repository root, a gitignored directory outside
// testdata/ so the corpus walkers there never see files that are only
// sometimes present.
//
// Tests call [GEDCOMIO], which skips the test when the files are absent, or
// fails it when GEDCOM_REQUIRE_FETCHED_CORPORA=1 is set (as the nightly
// workflow does).
package corpora

import (
	"bufio"
	"bytes"
	_ "embed" // for the lock file
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// RequireEnv names the environment variable that turns a missing fetched
// corpus from a skip into a failure when set to "1".
const RequireEnv = "GEDCOM_REQUIRE_FETCHED_CORPORA"

// GEDCOMIODir is the fetched GEDCOM.io corpus directory, relative to the
// repository root.
const GEDCOMIODir = ".corpora/gedcomio"

//go:embed gedcomio.lock
var gedcomioLock []byte

// Entry is one pinned file of a lock.
type Entry struct {
	// Path is the file's slash-separated path under the lock's base URL, and
	// under the local corpus directory.
	Path   string
	Size   int64
	SHA256 string // lowercase hex
}

// Lock is a parsed lock file.
type Lock struct {
	// Base is the URL every Entry.Path is resolved against; it ends in "/".
	Base    string
	Entries []Entry
}

// GEDCOMIOLock returns the pinned GEDCOM.io lock.
func GEDCOMIOLock() Lock {
	lock, err := ParseLock(bytes.NewReader(gedcomioLock))
	if err != nil {
		panic(fmt.Sprintf("corpora: embedded gedcomio.lock: %v", err)) // a bad lock is a build defect
	}
	return lock
}

// ParseLock reads a lock: blank lines and lines starting with "#" are
// ignored, one "base<TAB>URL" line names the base URL, and every other line is
// "SHA-256<TAB>size<TAB>path".
func ParseLock(r io.Reader) (Lock, error) {
	var lock Lock
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		switch {
		case len(fields) == 2 && fields[0] == "base":
			if !strings.HasSuffix(fields[1], "/") {
				return Lock{}, fmt.Errorf("line %d: base URL must end in /", n)
			}
			lock.Base = fields[1]
		case len(fields) == 3:
			e, err := parseEntry(fields)
			if err != nil {
				return Lock{}, fmt.Errorf("line %d: %w", n, err)
			}
			lock.Entries = append(lock.Entries, e)
		default:
			return Lock{}, fmt.Errorf("line %d: want \"base<TAB>URL\" or \"sha256<TAB>size<TAB>path\"", n)
		}
	}
	if err := sc.Err(); err != nil {
		return Lock{}, err
	}
	if lock.Base == "" {
		return Lock{}, errors.New("no base line")
	}
	if len(lock.Entries) == 0 {
		return Lock{}, errors.New("no entries")
	}
	return lock, nil
}

func parseEntry(fields []string) (Entry, error) {
	sum, sizeText, rel := fields[0], fields[1], fields[2]
	if len(sum) != 64 || strings.Trim(sum, "0123456789abcdef") != "" {
		return Entry{}, fmt.Errorf("%q is not a lowercase hex SHA-256", sum)
	}
	size, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil || size < 0 {
		return Entry{}, fmt.Errorf("%q is not a size", sizeText)
	}
	// IsLocal rejects absolute, parent-escaping and (on Windows) reserved
	// paths, so an entry can never write outside the corpus directory.
	if strings.Contains(rel, "\\") || path.Clean(rel) != rel || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return Entry{}, fmt.Errorf("%q is not a clean relative path", rel)
	}
	return Entry{Path: rel, Size: size, SHA256: sum}, nil
}

// RepoRoot returns the repository root: the nearest directory at or above the
// working directory that holds go.mod.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod at or above the working directory")
		}
		dir = parent
	}
}

// Missing returns the entries of lock that are absent from dir, or present
// with the wrong size. It does not hash: the fetcher verified each file when
// it wrote it.
func Missing(dir string, lock Lock) []Entry {
	var missing []Entry
	for _, e := range lock.Entries {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(e.Path)))
		if err != nil || !info.Mode().IsRegular() || info.Size() != e.Size {
			missing = append(missing, e)
		}
	}
	return missing
}

// GEDCOMIO returns the directory holding the fetched GEDCOM.io test files and
// their lock. When any file is missing it skips t naming `make fetch-corpora`,
// or fails t when GEDCOM_REQUIRE_FETCHED_CORPORA=1.
func GEDCOMIO(t testing.TB) (dir string, lock Lock) {
	t.Helper()
	lock = GEDCOMIOLock()
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("corpora: %v", err)
		return "", Lock{}
	}
	dir = filepath.Join(root, filepath.FromSlash(GEDCOMIODir))
	if missing := Missing(dir, lock); len(missing) > 0 {
		msg := fmt.Sprintf("%d of %d GEDCOM.io test files are not in %s (first: %s); run `make fetch-corpora`",
			len(missing), len(lock.Entries), GEDCOMIODir, missing[0].Path)
		if os.Getenv(RequireEnv) == "1" {
			t.Fatalf("%s (%s=1 makes this a failure)", msg, RequireEnv)
		} else {
			t.Skip(msg)
		}
		return "", Lock{}
	}
	return dir, lock
}
