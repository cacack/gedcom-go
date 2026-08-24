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
// This answers "what do real files contain that we do not understand?".
// spec7_coverage_test.go answers the other half, "what does the standard define
// that we do not understand?", from the specification rather than the corpus.
// Neither subsumes the other: a construct can be common in exports and rare in
// the spec, or defined by the spec and absent from every file in testdata/.
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
//
// The EVENT_DETAIL cycle (issues #402, #447, #448) removed nine entries at
// once: 34 unrecognized tags -> 25, i.e. 34 - 9 = 25. The nine split cleanly by
// the code path that stopped flagging them, and each was measured against the
// corpus before the change rather than inferred from the issue text:
//
//	#402 parseAttribute, which read only NOTE/AGE/DATE/PLAC/SOUR, so every
//	other EVENT_DETAIL child of any of the 13 attribute tags was flagged:
//	  ADDR (4 occurrences), AGNC (4), CAUS (17), OBJE (4)   -- 4 tags
//
//	#447 parseEvent and the citation/LDS/CHAN/NAME/REPO paths, which had no
//	typed field for these:
//	  ASSO (4), RELI (2), SNOTE (10)                        -- 3 tags
//
//	#448 parseFamily, which never read these contexts at all, so the FAM-level
//	tag itself was flagged:
//	  CENS (6), RESI (2)                                    -- 2 tags
//
// 4 + 3 + 2 = 9. No tag appears in two groups: RELI was flagged only under
// events (DEAT, MARR) in this corpus, never under an attribute.
var unhandledTags = map[string]string{
	// Standard tags. These are the actionable entries.
	"RIN":   "standard 5.5.1 record ID; no typed access (#441) and unrecognized in every context",
	"FORM":  "standard; unrecognized in some contexts",
	"NUMB":  "standard; unrecognized in some contexts",
	"CONC":  "continuation mechanic, unrecognized in some contexts",
	"CONT":  "continuation mechanic, unrecognized in some contexts",
	"NAME":  "standard; unrecognized in some contexts",
	"NOTE":  "standard 5.5 MULTIMEDIA_LINK subtag; every remaining occurrence is under an inline OBJE (#470)",
	"SOUR":  "standard; unrecognized in some contexts",
	"ALIA":  "standard; #375 fixed level 1 on INDI, still unrecognized at level 2",
	"FAMC":  "standard; unrecognized in some contexts",
	"TITL":  "standard; unrecognized in some contexts",
	"BLOB":  "standard 5.5 multimedia blob",
	"RFN":   "standard 5.5.1 record file number",
	"DATE":  "standard; unrecognized in some contexts",
	"EMAIL": "standard 5.5.1/7.0 contact tag",
	"WWW":   "standard 5.5.1/7.0 contact tag",
	"UID":   "standard 7.0 tag",

	// GEDCOM 7.0 tags, all from maximal70.ged. Version Support is Principle 4
	// and Differentiator 2, so these are the sharpest entries in the list.
	"CREA": "GEDCOM 7.0 creation timestamp",
	"REFN": "GEDCOM 7.0 user reference number",

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
