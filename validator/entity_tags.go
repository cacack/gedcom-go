// entity_tags.go reports typed edits the encoder will not write
// (docs/decisions/0003-lossless-dual-storage.md).
//
// Record.Tags is authoritative on encode, so a typed field edited on a decoded
// record -- a renamed person, a new header SOUR -- is silently absent from the
// file. encoder.SyncTags is the step that carries such an edit into Tags; this
// rule reports the records whose typed fields and Tags disagree, before
// anything is written.

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// validateEntityTags reports, as CodeEntityTagsDiverged, the header and each
// record whose typed fields would encode differently from its raw Tags: those
// encoder.SyncTags would change. It runs SyncTags on a shallow copy of the
// header and records, which is enough because SyncTags replaces their Value
// and Tags rather than editing a Tag, so the document is never modified. A
// document SyncTags cannot process gets one Info issue under the same code,
// carrying the error, since the comparison could not be made.
func validateEntityTags(doc *gedcom.Document) []Issue {
	synced := &gedcom.Document{Records: make([]*gedcom.Record, len(doc.Records))}
	if doc.Header != nil {
		header := *doc.Header
		synced.Header = &header
	}
	for i, record := range doc.Records {
		if record != nil {
			copied := *record
			synced.Records[i] = &copied
		}
	}
	if err := encoder.SyncTags(synced); err != nil {
		return []Issue{{
			Severity: SeverityInfo,
			Code:     CodeEntityTagsDiverged,
			Message:  "typed fields could not be compared with Record.Tags; encoder.SyncTags fails: " + err.Error(),
			Details:  map[string]string{"error": err.Error()},
		}}
	}

	var issues []Issue
	if doc.Header != nil && !sameTags(doc.Header.Tags, synced.Header.Tags) {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     CodeEntityTagsDiverged,
			Message:  "header typed fields and Header.Tags disagree; Encode writes Header.Tags, and encoder.SyncTags would rewrite it from the typed fields",
			Details:  map[string]string{"record": "HEAD"},
		})
	}
	for i, record := range doc.Records {
		if record == nil {
			continue
		}
		got := synced.Records[i]
		if record.Value == got.Value && sameTags(record.Tags, got.Tags) {
			continue
		}
		issues = append(issues, Issue{
			Severity:   SeverityWarning,
			Code:       CodeEntityTagsDiverged,
			Message:    fmt.Sprintf("%s record %s: typed fields and Record.Tags disagree; Encode writes Record.Tags, and encoder.SyncTags would rewrite it from the typed fields", record.Type, record.XRef),
			RecordXRef: record.XRef,
			LineNumber: record.LineNumber,
			Details:    map[string]string{"record": string(record.Type)},
		})
	}
	return issues
}

// sameTags reports whether two tag lists write the same lines.
func sameTags(a, b []*gedcom.Tag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x == nil || y == nil {
			if x != y {
				return false
			}
			continue
		}
		if x.Level != y.Level || x.Tag != y.Tag || x.Value != y.Value || x.XRef != y.XRef {
			return false
		}
	}
	return true
}
