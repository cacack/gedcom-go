package converter

import "github.com/cacack/gedcom-go/v3/gedcom"

// upgradeNoteRecordsForV7 turns every GEDCOM 5.5/5.5.1 NOTE record into a
// GEDCOM 7.0 SNOTE record, and every NOTE pointer to one of them into an SNOTE
// pointer. 7.0 has no NOTE record, and its NOTE structure takes text, so a
// "NOTE @N1@" left as written is read as the note text "@N1@" and the shared
// note is lost (issue #567).
//
// Both representations are rewritten: the record's Type and Entity (a Note
// becomes a SharedNote with the same XRef, Text and ExternalIDs; the typed
// SourceCitations and ChangeDate, which Note never parsed, stay empty though
// the SOUR and CHAN lines remain in record.Tags and are written), and
// each pointer in record.Tags and doc.Header.Tags. Typed NoteXRefs are left
// as they are; the encoder writes one naming an SNOTE record as SNOTE.
// A NOTE pointer that names no NOTE record is left unchanged.
//
// It runs after normalizeXRefsForV7, so records and pointers already carry
// their final identifiers.
func upgradeNoteRecordsForV7(doc *gedcom.Document, report *gedcom.ConversionReport) {
	records := 0
	notes := make(map[string]bool)
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeNote {
			continue
		}
		report.AddNormalized(gedcom.ConversionNote{
			Path:     BuildRecordPath(string(record.Type), record.XRef),
			Original: string(gedcom.RecordTypeNote),
			Result:   string(gedcom.RecordTypeSharedNote),
			Reason:   "GEDCOM 7.0 replaces NOTE records with SNOTE (shared note) records",
		})
		record.Type = gedcom.RecordTypeSharedNote
		if note, ok := record.Entity.(*gedcom.Note); ok && note != nil {
			record.Entity = &gedcom.SharedNote{
				XRef:        note.XRef,
				Text:        note.Text,
				ExternalIDs: note.ExternalIDs,
				// The record's tags, as the decoder fills SharedNote.Tags:
				// note.Tags still holds the CONC/CONT lines folded away by
				// consolidateCONCAndCONT, which an entity-built encode of a
				// record left with no tags would write back. The slice is
				// shared with the record, as the decoder shares it.
				Tags: record.Tags,
			}
		}
		if record.XRef != "" {
			notes[record.XRef] = true
		}
		records++
	}
	if records == 0 {
		return
	}

	pointers := 0
	if doc.Header != nil {
		pointers += retagNotePointers(doc.Header.Tags, notes)
	}
	for _, record := range doc.Records {
		if record != nil {
			pointers += retagNotePointers(record.Tags, notes)
		}
	}

	report.AddTransformation(gedcom.Transformation{
		Type:        "NOTE_RECORD_TO_SNOTE",
		Description: "Converted NOTE records to SNOTE records",
		Count:       records,
	})
	if pointers > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "NOTE_POINTER_TO_SNOTE",
			Description: "Converted NOTE pointers to shared notes into SNOTE pointers",
			Count:       pointers,
		})
	}
}

// retagNotePointers renames each NOTE tag whose value names one of notes to
// SNOTE, and returns how many it renamed.
func retagNotePointers(tags []*gedcom.Tag, notes map[string]bool) int {
	count := 0
	for _, tag := range tags {
		if tag != nil && tag.Tag == "NOTE" && notes[tag.Value] {
			tag.Tag = "SNOTE"
			count++
		}
	}
	return count
}
