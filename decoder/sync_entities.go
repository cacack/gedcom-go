package decoder

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// SyncEntities rebuilds the typed Entity of every record in doc that has Tags,
// from those Tags, as Decode builds it: the record's Tags are the authoritative
// form, so after code rewrites them the typed view can be brought back in step
// without a re-encode. Text is read by Header.Version's "@" rule and pointers by
// Header.DeclaredVersion's identifier grammar, as Decode does.
//
// A record with no Tags is left alone: its Entity is all there is (a
// hand-built record). Edits made to a tagged record's Entity and not written
// back to its Tags are discarded; the encoder writes such a record from its
// Tags too. Diagnostics are not reported. A nil doc is a no-op, and a nil
// header reads as one with no version.
func SyncEntities(doc *gedcom.Document) {
	if doc == nil {
		return
	}
	collector := &diagnosticCollector{discard: true}
	if doc.Header != nil {
		collector.version = doc.Header.DeclaredVersion
		collector.textVersion = doc.Header.Version
	}
	for _, record := range doc.Records {
		if record == nil || len(record.Tags) == 0 {
			continue
		}
		if !slices.Contains(record.Tags, nil) {
			populateRecordEntity(record, collector)
			continue
		}
		// A hand-built record can hold nil tags, which Decode never yields
		// and the parsers do not expect: they are parsed without them, and
		// contribute nothing, as the encoder writes nothing for them.
		clean := &gedcom.Record{XRef: record.XRef, Type: record.Type, Tags: slices.DeleteFunc(slices.Clone(record.Tags), func(t *gedcom.Tag) bool { return t == nil })}
		populateRecordEntity(clean, collector)
		record.Entity = clean.Entity
	}
}
