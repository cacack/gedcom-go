package decoder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Corpus tag coverage: which tags in real vendor files does the decoder not
// recognize?
//
// CONSTITUTION.md names the test corpus as the first form of evidence for
// scoping work, on the grounds that a construct appearing in real exports and
// mishandled by the library is a defect against Differentiator 3 (Real-World
// Compatibility) whether or not anyone has complained. This makes that evidence
// measurable instead of anecdotal: it answers "what don't we understand?" from
// the 20+ vendor exports in testdata/, and fails when the answer changes.
//
// Underscore-prefixed tags are excluded. They are vendor extensions, and ADR
// 0003 deliberately routes them through Record.Tags rather than the typed
// model, so UNKNOWN_TAG is the correct classification for them, not a defect.
//
// A tag here is not automatically a bug. Three kinds appear:
//   - standard tags unhandled in some context -- genuine gaps
//   - vendor tags that omit the underscore convention -- nothing to fix in the
//     decoder, but worth knowing they are indistinguishable from standard ones
//   - deliberately invalid tags in malformed fixtures -- expected
//
// The note on each entry says which.

// unhandledTags maps a tag to why it is currently unrecognized. Self-cleaning:
// a tag that stops being flagged fails the test asking to be removed, and a
// newly flagged tag fails it as a regression.
var unhandledTags = map[string]string{
	// Standard tags. These are the actionable entries.
	"RIN":   "standard 5.5.1 record ID; no typed access (#441) and unrecognized in every context",
	"FORM":  "standard; unrecognized in some contexts",
	"NUMB":  "standard; unrecognized in some contexts",
	"CONC":  "continuation mechanic, unrecognized in some contexts",
	"CONT":  "continuation mechanic, unrecognized in some contexts",
	"NAME":  "standard; unrecognized in some contexts",
	"NOTE":  "standard; unrecognized in some contexts",
	"SOUR":  "standard; unrecognized in some contexts",
	"ALIA":  "standard; #375 fixed level 1 on INDI, still unrecognized at level 2",
	"CAUS":  "standard EVENT_DETAIL subtag; see #402",
	"FAMC":  "standard; unrecognized in some contexts",
	"TITL":  "standard; unrecognized in some contexts",
	"CENS":  "standard event; unrecognized in some contexts",
	"ADDR":  "standard; unrecognized in some contexts",
	"AGNC":  "standard EVENT_DETAIL subtag; see #402",
	"BLOB":  "standard 5.5 multimedia blob",
	"OBJE":  "standard; unrecognized in some contexts",
	"RFN":   "standard 5.5.1 record file number",
	"DATE":  "standard; unrecognized in some contexts",
	"EMAIL": "standard 5.5.1/7.0 contact tag",
	"WWW":   "standard 5.5.1/7.0 contact tag",
	"UID":   "standard 7.0 tag",

	// GEDCOM 7.0 tags, all from maximal70.ged. Version Support is Principle 4
	// and Differentiator 2, so these are the sharpest entries in the list.
	"SNOTE": "GEDCOM 7.0 shared note; unrecognized in some contexts",
	"CREA":  "GEDCOM 7.0 creation timestamp",
	"ASSO":  "GEDCOM 7.0 association; unrecognized in some contexts",
	"REFN":  "GEDCOM 7.0 user reference number",
	"RELI":  "standard religion tag; unrecognized in some contexts",
	"RESI":  "standard residence; unrecognized in some contexts",

	// Vendor tags that omit the underscore convention. Nothing for the decoder
	// to fix -- recorded so they are not mistaken for standard-tag gaps.
	"FSID": "FamilySearch ID, no underscore prefix (pres2020.ged)",
	"HIST": "EasyTree vendor tag, no underscore prefix",
	"HEAL": "EasyTree vendor tag, no underscore prefix",
	"COMM": "vendor tag, no underscore prefix (royal92.ged)",
	"BUST": "vendor tag, no underscore prefix (structural-torture.ged)",

	// Deliberately invalid input.
	"INVALID": "malformed fixture: testdata/malformed/invalid-level.ged",
}

func corpusFixtures(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk("../testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.EqualFold(filepath.Ext(path), ".ged") {
			return nil
		}
		if filepath.Base(path) == "longsword.ged" {
			return nil // 46 MiB; see TestScaleFixture
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata: %v", err)
	}
	return files
}

func TestCorpusTagCoverage(t *testing.T) {
	found := map[string]int{}
	where := map[string]map[string]bool{}

	for _, path := range corpusFixtures(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		res, err := DecodeWithDiagnostics(bytes.NewReader(data), nil)
		if err != nil || res == nil {
			continue // undecodable fixtures are TestByteRoundTrip's business
		}
		for _, d := range res.Diagnostics {
			if d.Code != CodeUnknownTag {
				continue
			}
			tag := strings.TrimPrefix(d.Message, "unknown tag: ")
			if tag == d.Message || tag == "" || strings.HasPrefix(tag, "_") {
				continue // custom tags are Tags-only by design (ADR 0003)
			}
			found[tag]++
			if where[tag] == nil {
				where[tag] = map[string]bool{}
			}
			where[tag][filepath.Base(path)] = true
		}
	}

	for tag, n := range found {
		if _, known := unhandledTags[tag]; !known {
			var fx []string
			for f := range where[tag] {
				fx = append(fx, f)
			}
			sort.Strings(fx)
			t.Errorf("tag %q is unrecognized in %d places and is not in unhandledTags: %s",
				tag, n, strings.Join(fx, ", "))
		}
	}

	var fixed []string
	for tag, note := range unhandledTags {
		if found[tag] == 0 {
			fixed = append(fixed, fmt.Sprintf("%s (was: %s)", tag, note))
		}
	}
	if len(fixed) > 0 {
		sort.Strings(fixed)
		t.Errorf("these tags are now recognized; remove them from unhandledTags:\n  %s",
			strings.Join(fixed, "\n  "))
	}

	t.Logf("corpus tag coverage: %d non-underscore tags unrecognized across %d fixtures",
		len(found), len(corpusFixtures(t)))
}
