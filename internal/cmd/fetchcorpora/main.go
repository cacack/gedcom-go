// Command fetchcorpora downloads the test corpora that may be used but not
// redistributed (see internal/corpora) into a gitignored directory, verifying
// every file against the SHA-256 and size pinned in its lock.
//
// Files already present with the pinned hash are left alone. A network or
// HTTP failure prints a warning and exits 0, so the tests that need the files
// skip; -require makes it exit 1 instead. A size or hash mismatch always exits
// 1: the pinned content changed, which no retry fixes.
//
//	go run ./internal/cmd/fetchcorpora [-require] [-dir DIR]
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
