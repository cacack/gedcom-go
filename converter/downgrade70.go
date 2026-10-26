package converter

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// v7OnlyTags are the GEDCOM 7.0 structures with no GEDCOM 5.5/5.5.1 form that
// dropV7OnlyTags removes on a downgrade, in the order they are reported. NO
// (dropNegativeAssertions) and SNOTE (downgradeSharedNotes) are handled
// separately: NO is dropped by its own pass, and SNOTE has a 5.x form.
var v7OnlyTags = []string{"EXID", "TRAN", "PHRASE", "UID", "CREA"}

// dropV7OnlyTags removes every v7OnlyTags tag, at any level and together with
// its subordinate block, from each record's Tags on a 7.0 -> 5.5/5.5.1
// downgrade, and reports each as data loss. Before issue #570 these were
// reported as dropped but left in record.Tags, so the encoder wrote them into
// a 5.x file that does not define them.
//
// A matching tag nested inside a removed block goes with it and is covered by
// the outer tag's entry. It runs after dropNegativeAssertions, so tags under a
// dropped NO are not reported twice, and after transformEXIDToVendorTags, so an
// EXID mapped to _FSFTID is not reported as dropped.
//
// The typed entity's record-level fields for these tags are cleared too
// (clearV7OnlyEntityFields), so the two representations agree. That matters
// to the output: the encoder writes a record whose Tags are empty from its
// entity, which would put a dropped UID or CREA back into a record left with
// no other tags, or into a record built in code. A record without Tags is
// reported from its entity, since the entity is what the encoder writes.
func dropV7OnlyTags(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	found := make(map[string][]string)
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		hadTags := len(record.Tags) > 0
		removed := dropTagBlocks(record, func(tag *gedcom.Tag) bool {
			return slices.Contains(v7OnlyTags, tag.Tag)
		})
		cleared := clearV7OnlyEntityFields(record)
		if !hadTags {
			removed = cleared
		}
		for _, tag := range v7OnlyTags {
			if removed[tag] {
				found[tag] = append(found[tag], record.XRef)
			}
		}
	}

	reason := "Tag not supported in GEDCOM " + targetVersion.String()
	for _, tag := range v7OnlyTags {
		xrefs := found[tag]
		if len(xrefs) == 0 {
			continue
		}
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         tag + " tags",
			Reason:          reason,
			AffectedRecords: xrefs,
		})
		for _, xref := range xrefs {
			report.AddDropped(gedcom.ConversionNote{
				Path:     BuildNestedPath(getRecordTypeByXRef(doc, xref), xref, tag),
				Original: tag,
				Result:   "",
				Reason:   reason,
			})
		}
	}
}

// clearV7OnlyEntityFields clears the record-level fields of record's typed
// entity that the encoder writes as a v7OnlyTags tag (CreationDate as CREA,
// UIDs as UID, ExternalIDs as EXID, a shared note's Translations as TRAN) and
// returns the tags whose fields held data. Nested fields, such as an event's
// UIDs, are not reached; they are written only for a record without Tags.
func clearV7OnlyEntityFields(record *gedcom.Record) map[string]bool {
	crea, uids, exids, trans := v7OnlyEntityFields(record.Entity)
	cleared := make(map[string]bool)
	if crea != nil && *crea != nil {
		cleared["CREA"] = true
		*crea = nil
	}
	if uids != nil && len(*uids) > 0 {
		cleared["UID"] = true
		*uids = nil
	}
	if exids != nil && len(*exids) > 0 {
		cleared["EXID"] = true
		*exids = nil
	}
	if trans != nil && len(*trans) > 0 {
		cleared["TRAN"] = true
		*trans = nil
	}
	return cleared
}

// v7OnlyEntityFields returns pointers to the fields clearV7OnlyEntityFields
// clears on entity, nil for each one its type (or a nil entity) does not have.
func v7OnlyEntityFields(entity any) (crea **gedcom.ChangeDate, uids *[]string, exids *[]*gedcom.ExternalID, trans *[]*gedcom.SharedNoteTranslation) {
	switch e := entity.(type) {
	case *gedcom.Individual:
		if e != nil {
			return &e.CreationDate, &e.UIDs, &e.ExternalIDs, nil
		}
	case *gedcom.Family:
		if e != nil {
			return &e.CreationDate, &e.UIDs, &e.ExternalIDs, nil
		}
	case *gedcom.Source:
		if e != nil {
			return &e.CreationDate, &e.UIDs, &e.ExternalIDs, nil
		}
	case *gedcom.MediaObject:
		if e != nil {
			return &e.CreationDate, &e.UIDs, &e.ExternalIDs, nil
		}
	case *gedcom.Submitter:
		if e != nil {
			return nil, nil, &e.ExternalIDs, nil
		}
	case *gedcom.Repository:
		if e != nil {
			return nil, nil, &e.ExternalIDs, nil
		}
	case *gedcom.SharedNote:
		if e != nil {
			return nil, nil, &e.ExternalIDs, &e.Translations
		}
	}
	return nil, nil, nil, nil
}

// dropTagBlocks removes each tag drop matches, with its subordinate block,
// from record.Tags and returns the names it removed. It leaves record.Tags
// untouched (and allocates nothing) when nothing matches. A nil tag has no
// level to compare and is kept, as the caller's element; drop is never called
// with nil.
func dropTagBlocks(record *gedcom.Record, drop func(*gedcom.Tag) bool) map[string]bool {
	tags := record.Tags
	var removed map[string]bool
	var out []*gedcom.Tag
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil || !drop(tag) {
			if out != nil {
				out = append(out, tag)
			}
			continue
		}
		if out == nil {
			out = append(make([]*gedcom.Tag, 0, len(tags)), tags[:i]...)
			removed = make(map[string]bool)
		}
		removed[tag.Tag] = true
		j := i + 1
		for j < len(tags) && (tags[j] == nil || tags[j].Level > tag.Level) {
			if tags[j] == nil {
				out = append(out, nil)
			}
			j++
		}
		i = j - 1
	}
	if out != nil {
		record.Tags = out
	}
	return removed
}

// sharedNoteOnlyTags are the level-1 substructures of a 7.0 shared note record
// that a 5.x NOTE record does not define, in the order they are reported.
var sharedNoteOnlyTags = []string{"MIME", "LANG"}

// downgradeSharedNotes turns every GEDCOM 7.0 SNOTE record into a GEDCOM
// 5.5/5.5.1 NOTE record, and every SNOTE tag into a NOTE tag, on a downgrade:
// the reverse of upgradeNoteRecordsForV7. 5.x has no SNOTE, and its NOTE
// record and "NOTE @N1@" pointer carry the same shared note, so the note
// survives instead of being written as an SNOTE the target does not define
// (issue #570).
//
// Both representations are rewritten: the record's Type and Entity (a
// SharedNote becomes a Note with the same XRef, Text and Tags), and each
// SNOTE tag in record.Tags and doc.Header.Tags. The record's level-1 MIME and
// LANG, which a 5.x NOTE record does not define, are removed from its Tags
// and reported as data loss (from the entity's MIME and Language for a record
// without Tags); its TRAN and EXID went with dropV7OnlyTags, which runs
// first, so the Note shares the record's final tags.
func downgradeSharedNotes(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	records := 0
	lost := make(map[string][]string)
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeSharedNote {
			continue
		}
		note, _ := record.Entity.(*gedcom.SharedNote)
		removed := dropSharedNoteOnlyTags(record, note)
		for _, tag := range sharedNoteOnlyTags {
			if removed[tag] {
				lost[tag] = append(lost[tag], record.XRef)
				report.AddDropped(gedcom.ConversionNote{
					Path:     BuildNestedPath(string(record.Type), record.XRef, tag),
					Original: tag,
					Result:   "",
					Reason:   "GEDCOM " + targetVersion.String() + " NOTE records have no " + tag,
				})
			}
		}

		report.AddNormalized(gedcom.ConversionNote{
			Path:     BuildRecordPath(string(record.Type), record.XRef),
			Original: string(gedcom.RecordTypeSharedNote),
			Result:   string(gedcom.RecordTypeNote),
			Reason:   "GEDCOM 5.5/5.5.1 has no SNOTE (shared note) record; NOTE records carry shared notes",
		})
		record.Type = gedcom.RecordTypeNote
		if note != nil {
			record.Entity = &gedcom.Note{
				XRef: note.XRef,
				Text: note.Text,
				Tags: record.Tags,
			}
		}
		records++
	}
	for _, tag := range sharedNoteOnlyTags {
		if xrefs := lost[tag]; len(xrefs) > 0 {
			report.AddDataLoss(gedcom.DataLossItem{
				Feature:         "Shared note " + tag,
				Reason:          "GEDCOM " + targetVersion.String() + " NOTE records have no " + tag,
				AffectedRecords: xrefs,
			})
		}
	}

	pointers := 0
	if doc.Header != nil {
		pointers += retagSharedNoteTags(doc.Header.Tags)
	}
	for _, record := range doc.Records {
		if record != nil {
			pointers += retagSharedNoteTags(record.Tags)
		}
	}

	if records > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "SNOTE_RECORD_TO_NOTE",
			Description: "Converted SNOTE records to NOTE records",
			Count:       records,
		})
	}
	if pointers > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "SNOTE_POINTER_TO_NOTE",
			Description: "Converted SNOTE pointers into NOTE pointers",
			Count:       pointers,
		})
	}
}

// dropSharedNoteOnlyTags removes the level-1 sharedNoteOnlyTags blocks from a
// shared note record's Tags and returns the names removed. A record without
// Tags is written from its entity, so for one the names come from note's MIME
// and Language instead, which the Note that replaces it does not carry.
func dropSharedNoteOnlyTags(record *gedcom.Record, note *gedcom.SharedNote) map[string]bool {
	if len(record.Tags) == 0 {
		if note == nil {
			return nil
		}
		return map[string]bool{"MIME": note.MIME != "", "LANG": note.Language != ""}
	}
	return dropTagBlocks(record, func(tag *gedcom.Tag) bool {
		return tag.Level == 1 && slices.Contains(sharedNoteOnlyTags, tag.Tag)
	})
}

// retagSharedNoteTags renames each SNOTE tag to NOTE and returns how many it
// renamed.
func retagSharedNoteTags(tags []*gedcom.Tag) int {
	count := 0
	for _, tag := range tags {
		if tag != nil && tag.Tag == "SNOTE" {
			tag.Tag = "NOTE"
			count++
		}
	}
	return count
}
