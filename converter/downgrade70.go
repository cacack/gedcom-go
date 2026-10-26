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
// Only record.Tags is changed, the representation a decoded record is written
// from; the typed entity is left as it is.
func dropV7OnlyTags(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	found := make(map[string][]string)
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		for _, tag := range dropTagBlocks(record, v7OnlyTags) {
			found[tag] = append(found[tag], record.XRef)
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

// dropTagBlocks removes each tag named in names, with its subordinate block,
// from record.Tags and returns the names it removed, each once, in the order
// of names. It leaves record.Tags untouched (and allocates nothing) when
// nothing matches. A nil tag has no level to compare and is kept, as the
// caller's element.
func dropTagBlocks(record *gedcom.Record, names []string) []string {
	tags := record.Tags
	removed := make(map[string]bool)
	var out []*gedcom.Tag
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil || !slices.Contains(names, tag.Tag) {
			if out != nil {
				out = append(out, tag)
			}
			continue
		}
		if out == nil {
			out = append(make([]*gedcom.Tag, 0, len(tags)), tags[:i]...)
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
	if out == nil {
		return nil
	}
	record.Tags = out

	var result []string
	for _, name := range names {
		if removed[name] {
			result = append(result, name)
		}
	}
	return result
}

// downgradeSharedNotes turns every GEDCOM 7.0 SNOTE record into a GEDCOM
// 5.5/5.5.1 NOTE record, and every SNOTE tag into a NOTE tag, on a downgrade:
// the reverse of upgradeNoteRecordsForV7. 5.x has no SNOTE, and its NOTE
// record and "NOTE @N1@" pointer carry the same shared note, so the note
// survives instead of being written as an SNOTE the target does not define
// (issue #570).
//
// Both representations are rewritten: the record's Type and Entity (a
// SharedNote becomes a Note with the same XRef, Text and Tags; its MIME,
// Language, Translations and ExternalIDs have no 5.x NOTE form), and each
// SNOTE tag in record.Tags and doc.Header.Tags. It runs after dropV7OnlyTags,
// so the Note shares the record's final tags.
func downgradeSharedNotes(doc *gedcom.Document, report *gedcom.ConversionReport) {
	records := 0
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeSharedNote {
			continue
		}
		report.AddNormalized(gedcom.ConversionNote{
			Path:     BuildRecordPath(string(record.Type), record.XRef),
			Original: string(gedcom.RecordTypeSharedNote),
			Result:   string(gedcom.RecordTypeNote),
			Reason:   "GEDCOM 5.5/5.5.1 has no SNOTE (shared note) record; NOTE records carry shared notes",
		})
		record.Type = gedcom.RecordTypeNote
		if note, ok := record.Entity.(*gedcom.SharedNote); ok && note != nil {
			record.Entity = &gedcom.Note{
				XRef: note.XRef,
				Text: note.Text,
				Tags: record.Tags,
			}
		}
		records++
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
