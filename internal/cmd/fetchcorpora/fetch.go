package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// requestTimeout bounds each download, so a stalled connection becomes a
// network failure (a warning) rather than a hung CI step.
const requestTimeout = 60 * time.Second

// networkError is a failure to obtain a file from the source: a transport
// error or a non-200 response. It is the only failure tolerated without
// -require; a size or hash mismatch, or a local I/O error, never is.
type networkError struct {
	err error
}

func (e *networkError) Error() string { return e.err.Error() }
func (e *networkError) Unwrap() error { return e.err }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fetchcorpora", flag.ContinueOnError)
	fs.SetOutput(stderr)
	require := fs.Bool("require", false, "exit 1 when a file cannot be downloaded (default: warn and exit 0)")
	dir := fs.String("dir", "", "directory to fetch into (default: "+corpora.GEDCOMIODir+" at the repository root)")
	base := fs.String("base", "", "override the lock's base URL (for tests)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 2
	}

	lock := corpora.GEDCOMIOLock()
	if *base != "" {
		lock.Base = *base
	}
	if *dir == "" {
		root, err := corpora.RepoRoot()
		if err != nil {
			fmt.Fprintf(stderr, "✗ %v\n", err)
			return 1
		}
		*dir = filepath.Join(root, filepath.FromSlash(corpora.GEDCOMIODir))
	}

	f := fetcher{client: &http.Client{Timeout: requestTimeout}, lock: lock, dir: *dir}
	return f.fetchAll(*require, stdout, stderr)
}

type fetcher struct {
	client *http.Client
	lock   corpora.Lock
	dir    string
}

// fetchAll fetches every entry and returns the exit code.
func (f fetcher) fetchAll(require bool, stdout, stderr io.Writer) int {
	var present, fetched int
	var unreachable, failed []string
	offline := false
	for _, e := range f.lock.Entries {
		ok, err := f.verified(e)
		if err == nil && ok {
			present++
			continue
		}
		// After one network failure, try no further downloads: each could
		// stall for requestTimeout, and a stalled CI step must not outlast
		// the job.
		if err == nil && offline {
			unreachable = append(unreachable, e.Path+": not attempted after an earlier network failure")
			continue
		}
		if err == nil {
			err = f.download(e)
		}
		var netErr *networkError
		switch {
		case err == nil:
			fetched++
		case errors.As(err, &netErr):
			unreachable = append(unreachable, err.Error())
			offline = true
		default:
			failed = append(failed, err.Error())
		}
	}

	fmt.Fprintf(stdout, "%s: %d already present, %d downloaded, %d unreachable, %d failed (of %d)\n",
		f.dir, present, fetched, len(unreachable), len(failed), len(f.lock.Entries))
	for _, msg := range failed {
		fmt.Fprintf(stderr, "✗ %s\n", msg)
	}
	if len(failed) > 0 {
		return 1
	}
	if len(unreachable) == 0 {
		return 0
	}
	for _, msg := range unreachable {
		fmt.Fprintf(stderr, "  %s\n", msg)
	}
	if require {
		fmt.Fprintf(stderr, "✗ %d files could not be downloaded (-require)\n", len(unreachable))
		return 1
	}
	warn := fmt.Sprintf("%d files could not be downloaded; the tests that need them will skip", len(unreachable))
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Fprintf(stderr, "::warning::%s\n", warn)
	} else {
		fmt.Fprintf(stderr, "⚠ %s\n", warn)
	}
	return 0
}

// verified reports whether e is already on disk with the pinned size and hash.
// A local file that does not match is reported as absent, to be replaced; an
// unreadable one is an error.
func (f fetcher) verified(e corpora.Entry) (bool, error) {
	file, err := os.Open(f.local(e)) // #nosec G304 -- path is a lock entry under the corpus dir; ParseLock rejects non-local paths
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, file)
	if err != nil {
		return false, err
	}
	return n == e.Size && hex.EncodeToString(h.Sum(nil)) == e.SHA256, nil
}

// download fetches e into a temporary file beside its destination and renames
// it into place only once size and hash match, so an interrupted or wrong
// download never leaves a file that looks fetched.
func (f fetcher) download(e corpora.Entry) error {
	resp, err := f.client.Get(f.url(e)) // #nosec G107 -- the URL is the pinned lock base plus a lock entry
	if err != nil {
		return &networkError{fmt.Errorf("%s: %w", e.Path, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return &networkError{fmt.Errorf("%s: HTTP %s", e.Path, resp.Status)}
	}

	dst := f.local(e)
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".fetch-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once renamed

	h := sha256.New()
	// Read one byte past the pinned size, so an oversized body is detected
	// without being read in full.
	n, copyErr := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, e.Size+1))
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	switch {
	case copyErr != nil:
		// Almost always the connection dropping mid-body.
		return &networkError{fmt.Errorf("%s: %w", e.Path, copyErr)}
	case n != e.Size:
		return fmt.Errorf("%s: size differs from the lock (pinned content changed)", e.Path)
	case hex.EncodeToString(h.Sum(nil)) != e.SHA256:
		return fmt.Errorf("%s: SHA-256 differs from the lock (pinned content changed)", e.Path)
	}
	return os.Rename(tmp.Name(), dst)
}

// url resolves e against the lock's base, escaping each path segment.
func (f fetcher) url(e corpora.Entry) string {
	segments := strings.Split(e.Path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return f.lock.Base + strings.Join(segments, "/")
}

func (f fetcher) local(e corpora.Entry) string {
	return filepath.Join(f.dir, filepath.FromSlash(e.Path))
}
