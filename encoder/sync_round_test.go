package encoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSyncRoundUnmodifiedConverges pins the invariant SyncTags' loop relies
// on: for an unedited document, one round returns every line as the same
// *gedcom.Tag (sameLines compares pointers), so it reports nothing changed and
// the loop stops after the first round rather than at maxSyncRounds. A merge
// that copied an unchanged line would make every record look changed and cost
// maxSyncRounds read-backs per call, with no failing output to show it.
func TestSyncRoundUnmodifiedConverges(t *testing.T) {
	paths, _ := filepath.Glob("../testdata/*/*.ged")
	checked := 0
	for _, path := range paths {
		if strings.Contains(path, "malformed") || filepath.Base(path) == "longsword.ged" {
			continue
		}
		data, err := os.ReadFile(path) // #nosec G304 -- a fixture path from the repo's own testdata
		if err != nil {
			t.Fatal(err)
		}
		doc, err := decoder.Decode(strings.NewReader(string(data)))
		if err != nil {
			continue
		}
		opts := DefaultOptions().forOutput(doc.Header)
		header := doc.Header
		syncHead := header != nil && len(header.Tags) > 0
		if syncHead {
			copied := *header
			header = &copied
		}
		var records []*gedcom.Record
		for _, record := range doc.Records {
			if syncable(record) {
				copied := *record
				records = append(records, &copied)
			}
		}
		headChanged, changed, err := syncRound(header, syncHead, records, opts, sharedNotePointers(doc, opts))
		if err != nil {
			continue // a fixture SyncTags cannot baseline; covered by TestSyncTagsUnmodifiedIsNoOp
		}
		checked++
		if headChanged {
			t.Errorf("%s: an unedited header reported changed after one round", filepath.Base(path))
		}
		if len(changed) > 0 {
			t.Errorf("%s: %d unedited records reported changed after one round (first %s %s)",
				filepath.Base(path), len(changed), changed[0].Type, changed[0].XRef)
		}
	}
	if checked == 0 {
		t.Fatal("no fixture was checked")
	}
}
