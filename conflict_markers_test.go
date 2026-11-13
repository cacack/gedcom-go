package gedcomgo

import (
	"regexp"
	"strings"
	"testing"
)

// conflictMarker matches a line git writes when a merge or rebase conflict is
// left unresolved. Go files with one do not compile; Markdown files do not
// fail anything else, so a stray resolution can reach main unnoticed.
var conflictMarker = regexp.MustCompile(`^(<{7}|={7}|>{7})( |$)`)

func TestNoConflictMarkersInDocs(t *testing.T) {
	for _, path := range trackedDocs(t, ".") {
		for i, line := range strings.Split(readFile(t, path), "\n") {
			if conflictMarker.MatchString(line) {
				t.Errorf("%s:%d has an unresolved merge conflict marker: %q", path, i+1, line)
			}
		}
	}
}
