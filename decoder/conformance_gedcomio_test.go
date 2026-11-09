package decoder

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// The FamilySearch/GEDCOM.io 7.0 and 7.1 test files (issue #718), fetched by
// `make fetch-corpora` rather than vendored because the repository declares
// no licence. Every .ged file, including each GEDZIP-style directory's
// gedcom.ged, is decoded in strict and lenient mode like the vendored suites
// in conformance_suites_test.go, with the same self-cleaning known-failure
// lists. Keys are paths under the lock's base (testfiles/ upstream).

// gedcomioStrictKnownFailures lists files whose strict decode returns an
// error. Empty: every file decodes cleanly.
var gedcomioStrictKnownFailures = map[string]string{}

// gedcomioLenientKnownFailures lists files whose lenient decode returns an
// error or an error-severity diagnostic. Empty: every file decodes cleanly.
var gedcomioLenientKnownFailures = map[string]string{}

// gedcomioVersionKnownFailures lists files not decoded as GEDCOM 7.0. Every
// file is 7.0 or 7.1, and a later 7.x minor is read as 7.0 (#741).
var gedcomioVersionKnownFailures = map[string]string{
	"gedcom71/minimal71.ged":        "VERS 7.1 is not recognised and no 7.0-only tag rescues the guess, so it decodes as 5.5 (#741)",
	"gedcom71/minimal71/gedcom.ged": "VERS 7.1 is not recognised and no 7.0-only tag rescues the guess, so it decodes as 5.5 (#741)",
}

func TestGEDCOMIOTestFilesDecode(t *testing.T) {
	dir, lock := corpora.GEDCOMIO(t)
	for _, e := range lock.Entries {
		switch {
		case strings.HasSuffix(e.Path, ".gdz"):
			t.Run(e.Path, func(t *testing.T) {
				t.Skip("GEDZIP archives are not read yet (#127)")
			})
		case strings.HasSuffix(e.Path, ".ged"):
			t.Run(e.Path, func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(e.Path))) // #nosec G304 -- a pinned lock entry under the fetched corpus dir
				if err != nil {
					t.Fatalf("ReadFile: %v", err)
				}
				for _, mode := range []struct {
					name, list string
					strict     bool
					known      map[string]string
				}{
					{"strict", "gedcomioStrictKnownFailures", true, gedcomioStrictKnownFailures},
					{"lenient", "gedcomioLenientKnownFailures", false, gedcomioLenientKnownFailures},
				} {
					t.Run(mode.name, func(t *testing.T) {
						failure := decodeOutcome(data, mode.strict)
						reason, known := mode.known[e.Path]
						switch {
						case failure == "" && known:
							t.Errorf("now decodes cleanly; remove %q from %s (was: %s)", e.Path, mode.list, reason)
						case failure != "" && !known:
							t.Errorf("decode failed and the file is not listed in %s: %s", mode.list, failure)
						}
					})
				}
				t.Run("version", func(t *testing.T) {
					doc, err := Decode(bytes.NewReader(data))
					if err != nil {
						t.Fatalf("Decode: %v", err)
					}
					got := doc.Header.Version
					reason, known := gedcomioVersionKnownFailures[e.Path]
					switch {
					case got == gedcom.Version70 && known:
						t.Errorf("now decodes as 7.0; remove %q from gedcomioVersionKnownFailures (was: %s)", e.Path, reason)
					case got != gedcom.Version70 && !known:
						t.Errorf("decoded as %q, want 7.0, and the file is not listed in gedcomioVersionKnownFailures", got)
					}
				})
			})
		}
	}
}
