package encoder

import (
	"fmt"
	"io"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// inlineMediaSet promotes inline media links to OBJE records for GEDCOM 7.0
// output, which has no inline link (issue #733). A link written from a typed
// entity with Files and no MediaXRef is given a fresh record identifier as it
// is written, and the OBJE record holding its files is written after every
// other record. It is used through a pointer, with nil meaning "promote
// nothing": such a link is then written inline, in the 5.5.1 layout.
type inlineMediaSet struct {
	used    map[string]bool              // record identifiers already taken
	xrefs   map[*gedcom.MediaLink]string // identifier given to each promoted link
	records []*gedcom.Record             // the promoted records, in write order
}

// inlineMediaPromotion returns an empty set for doc when the output is GEDCOM
// 7.0 (see outputVersion and gateVersion), and nil otherwise.
func inlineMediaPromotion(doc *gedcom.Document, opts *EncodeOptions) *inlineMediaSet {
	if gateVersion(outputVersion(doc.Header, opts)) != gedcom.Version70 {
		return nil
	}
	set := &inlineMediaSet{used: make(map[string]bool), xrefs: make(map[*gedcom.MediaLink]string)}
	for _, record := range doc.Records {
		if record != nil && record.XRef != "" {
			set.used[record.XRef] = true
		}
	}
	return set
}

// promote returns the identifier of the OBJE record that holds link's files,
// creating that record on first sight of link, or "" for a nil set. The
// identifier is the first "@INLINE_OBJEn@" no record of the document uses.
func (s *inlineMediaSet) promote(link *gedcom.MediaLink) string {
	if s == nil {
		return ""
	}
	if xref, ok := s.xrefs[link]; ok {
		return xref
	}
	var xref string
	for n := len(s.records) + 1; ; n++ {
		xref = fmt.Sprintf("@INLINE_OBJE%d@", n)
		if !s.used[xref] {
			break
		}
	}
	s.used[xref] = true
	s.xrefs[link] = xref
	s.records = append(s.records, &gedcom.Record{
		XRef:   xref,
		Type:   gedcom.RecordTypeMedia,
		Entity: &gedcom.MediaObject{XRef: xref, Files: link.Files},
	})
	return xref
}

// writeRecords writes the records promoted so far. A nil set writes nothing.
func (s *inlineMediaSet) writeRecords(w io.Writer, opts *EncodeOptions, sharedNotes *sharedNoteSet) error {
	if s == nil {
		return nil
	}
	for _, record := range s.records {
		if err := writeRecord(w, record, opts, sharedNotes); err != nil {
			return err
		}
	}
	return nil
}
