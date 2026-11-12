package validator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// gedcom.io's warnings70.ged and warnings71.ged (issue #718) hold structures
// that are legal but that the spec deprecates, so a validator may warn on
// them. Upstream describes them only as "things that are legal in FamilySearch
// GEDCOM 7.0 but may generate warnings"; the list below reads each one off the
// file against the deprecation notes of the current FamilySearch GEDCOM spec.
// The two files are identical apart from GEDC.VERS, so they share it.
//
// Both sides are recorded and self-cleaning. A designed warning counts as
// raised when any issue lands on one of its lines; one not raised must be in
// gedcomioWarningGaps, and a gap that starts being raised fails asking for its
// entry to be removed. Every issue we raise that no designed warning explains
// is listed in gedcomioWarningExtras, exactly.

type designedWarning struct {
	name  string
	tags  []string // the tags on those lines, checked so a slip in lines fails
	lines []int    // 1-based source lines that carry it
}

var gedcomioWarningFiles = []string{"gedcom70/warnings70.ged", "gedcom71/warnings71.ged"}

var gedcomioDesignedWarnings = []designedWarning{
	{"deprecated ADR1/ADR2/ADR3 (ADDRESS_STRUCTURE)", []string{"ADR1", "ADR2", "ADR3"}, []int{9, 10, 11, 48, 49, 50, 57, 58, 59}},
	{"deprecated HEAD.SOUR.DATA (HEADER)", []string{"DATA"}, []int{12}},
	{"deprecated EXID without EXID.TYPE (IDENTIFIER_STRUCTURE)", []string{"EXID"}, []int{17, 32, 43, 52}},
	{"deprecated ord-STAT values COMPLETED, EXCLUDED, PRE_1970, UNCLEARED, SUBMITTED, INFANT (enumset-ord-STAT)", []string{"STAT"}, []int{19, 23, 26, 29, 34, 37, 40}},
}

// gedcomioWarningGaps lists designed warnings we do not raise, by name.
var gedcomioWarningGaps = map[string]string{
	"deprecated ADR1/ADR2/ADR3 (ADDRESS_STRUCTURE)":                                                             "no deprecation checks (#742)",
	"deprecated HEAD.SOUR.DATA (HEADER)":                                                                        "no deprecation checks (#742)",
	"deprecated EXID without EXID.TYPE (IDENTIFIER_STRUCTURE)":                                                  "no deprecation checks (#742)",
	"deprecated ord-STAT values COMPLETED, EXCLUDED, PRE_1970, UNCLEARED, SUBMITTED, INFANT (enumset-ord-STAT)": "no enumeration status checks (#742)",
}

// gedcomioWarningExtras lists, per file, the issues we raise that no designed
// warning explains, as "CODE line N".
var gedcomioWarningExtras = map[string][]string{
	"gedcom70/warnings70.ged": {
		"EMPTY_FAMILY line 16", // a legal memberless FAM: data-quality heuristic, warning (#721)
		"MISSING_NAME line 31", // INDI.NAME is {0:M} in 7.0: data-quality heuristic, warning (#721)
	},
	"gedcom71/warnings71.ged": {
		"EMPTY_FAMILY line 16", // as above (#721)
		"MISSING_NAME line 31", // as above (#721)
	},
}

func TestGEDCOMIOWarningsFiles(t *testing.T) {
	dir, _ := corpora.GEDCOMIO(t)
	v := NewWithOptions(&ValidateOptions{
		TagRegistry:        NewTagRegistry(),
		ValidateCustomTags: true,
		Strictness:         StrictnessStrict,
	})
	for _, file := range gedcomioWarningFiles {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file))) // #nosec G304 -- a pinned lock entry under the fetched corpus dir
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			doc, err := decoder.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			issues := v.ValidateAll(doc)
			lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

			explained := map[int]bool{}
			for _, w := range gedcomioDesignedWarnings {
				raised := slices.ContainsFunc(issues, func(i Issue) bool { return slices.Contains(w.lines, i.LineNumber) })
				for _, l := range w.lines {
					explained[l] = true
					if fields := strings.Fields(lines[l-1]); len(fields) < 2 || !slices.Contains(w.tags, fields[1]) {
						t.Fatalf("line %d is %q, not one of %v; fix gedcomioDesignedWarnings", l, lines[l-1], w.tags)
					}
				}
				reason, gap := gedcomioWarningGaps[w.name]
				switch {
				case raised && gap:
					t.Errorf("now raised: %s; remove it from gedcomioWarningGaps (was: %s)", w.name, reason)
				case !raised && !gap:
					t.Errorf("not raised: %s, and it is not listed in gedcomioWarningGaps", w.name)
				}
			}

			var extras []string
			for _, i := range issues {
				if i.Severity == SeverityError {
					t.Errorf("%s line %d is an ERROR; the file is valid", i.Code, i.LineNumber)
				}
				if !explained[i.LineNumber] {
					extras = append(extras, fmt.Sprintf("%s line %d", i.Code, i.LineNumber))
				}
			}
			slices.Sort(extras)
			if want := gedcomioWarningExtras[file]; !slices.Equal(extras, want) {
				t.Errorf("issues no designed warning explains changed; update gedcomioWarningExtras[%q]\n got: %q\nwant: %q", file, extras, want)
			}
		})
	}
}
