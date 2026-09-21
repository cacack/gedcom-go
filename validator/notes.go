// notes.go validates the note-pointer invariants that the typed entities document
// but cannot enforce on their own.
//
// MediaObject is the only entity carrying both NoteXRefs (NOTE pointers) and
// SharedNoteXRefs (SNOTE pointers). The two are documented to partition the record's
// note pointers: disjoint, and together complete. Nothing in the type system holds
// that invariant -- both fields are plain []string, so any caller can violate it.
//
// It is violated in practice rather than in theory. Before the partition landed, the
// decoder appended every SNOTE pointer to *both* slices, so a media object that
// crossed the version boundary in that shape -- persisted, serialized and rehydrated,
// or copied out of an older decode -- still holds each shared-note pointer twice.
// With the concatenation dedup gone, such a value silently yields a duplicated entry
// from AllNotes and a duplicated NOTE line on typed-path re-encode, with no compile
// signal either way. A current decode reaches the same state honestly, from one OBJE
// carrying both `NOTE @N1@` and `SNOTE @N1@`.
//
// Validating it here makes the violation visible without adding cost to decoding.
//
// What is checked is the overlap between the two slices, and nothing else. A pointer
// repeated *within* one slice duplicates a note just as visibly, but it is not an
// invariant violation: the file named that note twice, and reproducing it is lossless
// representation working as intended. Only media objects are checked, because only
// MediaObject splits the two tags -- every other entity folds NOTE and SNOTE into a
// single NoteXRefs, where the partition does not exist to be violated.

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// NoteValidator checks note-pointer invariants on typed entities.
type NoteValidator struct{}

// NewNoteValidator creates a new NoteValidator.
func NewNoteValidator() *NoteValidator {
	return &NoteValidator{}
}

// ValidateNotePointers checks every media object for pointers listed in both
// NoteXRefs and SharedNoteXRefs, which are documented to never overlap. Each
// distinct overlapping pointer produces one warning.
//
// Media objects are the only records checked, and cross-slice overlap the only
// shape reported -- a pointer repeated within a single slice is legitimate. See
// this file's header comment for why.
//
// On each issue, RecordXRef is the media object and Details["xref"] the overlapping
// note pointer -- two different records, not the same one twice.
func (n *NoteValidator) ValidateNotePointers(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var issues []Issue

	// MediaObjects skips records whose entity is nil or not a media object, so
	// every element here is non-nil (ADR 0007: never panic on hand-built input).
	for _, media := range doc.MediaObjects() {
		issues = append(issues, n.checkMediaNotePartition(media)...)
	}

	return issues
}

// checkMediaNotePartition reports each pointer that appears in both of the media
// object's note slices. Reported once per distinct pointer, so a slice that repeats
// an overlapping pointer does not multiply the finding.
func (n *NoteValidator) checkMediaNotePartition(media *gedcom.MediaObject) []Issue {
	if len(media.NoteXRefs) == 0 || len(media.SharedNoteXRefs) == 0 {
		return nil
	}

	noteSet := make(map[string]struct{}, len(media.NoteXRefs))
	for _, xref := range media.NoteXRefs {
		noteSet[xref] = struct{}{}
	}

	var issues []Issue
	reported := make(map[string]struct{})

	for _, xref := range media.SharedNoteXRefs {
		if _, overlaps := noteSet[xref]; !overlaps {
			continue
		}
		if _, seen := reported[xref]; seen {
			continue
		}
		reported[xref] = struct{}{}

		// No line number: the typed note fields are not attributed to a source
		// line, and this check compares whole slices rather than walking raw tags.
		issues = append(issues, NewIssue(
			SeverityWarning,
			CodeOverlappingNotePointers,
			fmt.Sprintf("note pointer %s appears in both NoteXRefs and SharedNoteXRefs, which partition a media object's note pointers", xref),
			media.XRef,
		).WithDetail("xref", xref))
	}

	return issues
}
