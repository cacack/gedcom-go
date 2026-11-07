package converter

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// dropSubmissionRecords removes what GEDCOM 7.0 dropped from the 5.x
// submission mechanism: every SUBN record and the HEAD.SUBN pointer to it.
// 7.0 has no SUBN structure, so carrying either into 7.0 output writes lines a
// strict 7.0 reader rejects. The records' contents (FAMF, TEMP, ANCE, DESC,
// ORDI and the rest) have nowhere to go in 7.0, so each removed record is
// reported as data loss and as a Dropped note, as is the header pointer.
//
// It runs first on the upgrade paths, so no later transform reports a change
// to text inside a record that is then removed, and the XRef normalization
// that follows never renames an identifier that is about to disappear.
func dropSubmissionRecords(doc *gedcom.Document, report *gedcom.ConversionReport) {
	reason := "Submission records (SUBN) are not part of GEDCOM 7.0"

	var dropped []string
	kept := doc.Records[:0:0]
	for _, record := range doc.Records {
		if record != nil && record.Type == gedcom.RecordTypeSubmission {
			dropped = append(dropped, record.XRef)
			if record.XRef != "" && doc.XRefMap != nil && doc.XRefMap[record.XRef] == record {
				delete(doc.XRefMap, record.XRef)
			}
			continue
		}
		kept = append(kept, record)
	}
	if len(dropped) > 0 {
		doc.Records = kept
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         "SUBN records",
			Reason:          reason,
			AffectedRecords: dropped,
		})
		for _, xref := range dropped {
			report.AddDropped(gedcom.ConversionNote{
				Path:     BuildRecordPath(string(gedcom.RecordTypeSubmission), xref),
				Original: string(gedcom.RecordTypeSubmission),
				Reason:   reason,
			})
		}
	}

	if doc.Header == nil {
		return
	}
	var pointers []string
	tags, removed := dropTagBlocksIn(doc.Header.Tags, func(tag *gedcom.Tag) bool {
		if tag.Level == 1 && tag.Tag == "SUBN" {
			pointers = append(pointers, tag.Value)
			return true
		}
		return false
	})
	if !removed["SUBN"] {
		return
	}
	doc.Header.Tags = tags
	for _, pointer := range pointers {
		report.AddDropped(gedcom.ConversionNote{
			Path:     BuildNestedPath("HEAD", "", "SUBN"),
			Original: pointer,
			Reason:   reason,
		})
	}
}
