package gedzip

import (
	"fmt"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/atsign"
)

// fileRef is one File Path payload of a document: a FILE line of an OBJE
// structure, or a TRAN line beneath one.
type fileRef struct {
	xref        string // the XRef of the record holding the reference
	ref         string // the payload, as written
	translation bool   // true for FILE.TRAN
}

// collectFileRefs returns every FILE and FILE.TRAN payload of doc's OBJE
// structures, in document order, each FILE followed by its translations.
// They are the only two structures whose payload is a File Path, so they are
// every reference a GEDZIP archive has to cover (#127). Payloads are returned
// as written, repeats included; an empty one is returned too, since it is a
// reference that names nothing.
//
// An OBJE record is read from its typed MediaObject. Any other record is read
// from its Tags, for an inline OBJE structure with FILE lines beneath it:
// GEDCOM 7 has no such structure, but the encoder writes a record's Tags as
// they are, so a decoded 5.5.1 record written as 7.0 keeps one, and its
// payload is then a reference the dataset makes. A record without Tags holds
// no inline reference once written: the encoder moves the inline links of
// such a record into OBJE records of their own when it writes GEDCOM 7.
func collectFileRefs(doc *gedcom.Document) []fileRef {
	if doc == nil {
		return nil
	}
	var version gedcom.Version
	if doc.Header != nil {
		version = doc.Header.Version
	}
	var refs []fileRef
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		if media, ok := record.Entity.(*gedcom.MediaObject); ok && media != nil {
			refs = appendMediaObjectFileRefs(refs, media)
		} else if record.Type != gedcom.RecordTypeMedia {
			refs = appendInlineFileRefs(refs, record, version)
		}
	}
	return refs
}

// appendMediaObjectFileRefs appends the references of an OBJE record.
func appendMediaObjectFileRefs(refs []fileRef, media *gedcom.MediaObject) []fileRef {
	for _, file := range media.Files {
		if file == nil {
			continue
		}
		refs = append(refs, fileRef{xref: media.XRef, ref: file.FileRef})
		for _, tran := range file.Translations {
			if tran == nil {
				continue
			}
			refs = append(refs, fileRef{xref: media.XRef, ref: tran.FileRef, translation: true})
		}
	}
	return refs
}

// appendInlineFileRefs appends the references in the Tags of a record that is
// not an OBJE record: each FILE line directly beneath an inline OBJE
// structure, at any depth, and each TRAN line directly beneath such a FILE.
// A payload is unescaped as the decoder unescapes one for version.
func appendInlineFileRefs(refs []fileRef, record *gedcom.Record, version gedcom.Version) []fileRef {
	const none = -1
	objeLevel, fileLevel := none, none
	for _, tag := range record.Tags {
		if tag == nil {
			continue
		}
		if tag.Level <= fileLevel {
			fileLevel = none
		}
		if tag.Level <= objeLevel {
			objeLevel = none
		}
		switch {
		case tag.Tag == "OBJE":
			objeLevel, fileLevel = tag.Level, none
		case tag.Tag == "FILE" && objeLevel != none && tag.Level == objeLevel+1:
			fileLevel = tag.Level
			refs = append(refs, fileRef{xref: record.XRef, ref: atsign.Unescape(tag.Value, version)})
		case tag.Tag == "TRAN" && fileLevel != none && tag.Level == fileLevel+1:
			refs = append(refs, fileRef{xref: record.XRef, ref: atsign.Unescape(tag.Value, version), translation: true})
		}
	}
	return refs
}

// RefProblem is a FILE or FILE.TRAN payload that a GEDZIP archive cannot
// satisfy, as reported by [Reader.CheckRefs] (#127).
type RefProblem struct {
	// XRef is the XRef of the record holding the reference: an OBJE record,
	// or the record an inline OBJE structure sits in.
	XRef string

	// FileRef is the payload as written in the document.
	FileRef string

	// Translation is true when the payload is that of a FILE.TRAN line rather
	// than of a FILE line.
	Translation bool

	// Kind is what [ClassifyFileRef] makes of the payload: [RefLocal] when the
	// entry it names is missing, unusable or the dataset, otherwise
	// [RefFileURL] or [RefInvalid].
	Kind RefKind

	// Err says what is wrong. It wraps [ErrEntryNotFound] for a missing entry,
	// [ErrUnusableEntry] for one that cannot be read, [ErrReservedName] for a
	// reference to gedcom.ged, and [ErrNotLocal] for a file: URL or an invalid
	// payload.
	Err error
}

// CheckRefs reports the FILE and FILE.TRAN payloads of doc that the archive
// does not satisfy (#127): a local file with no entry or with an unusable
// one, a reference to gedcom.ged, which is the dataset and no media file, a
// file: URL, which must not be used in a GEDZIP, and a payload that is not a
// valid File Path. A web file (ftp, http, https) has no entry and is never a
// problem. No entry is read. These are the references [Write] refuses, so a
// document with no problem can be written back with [Reader.FS] as its media.
//
// The payloads checked are those of OBJE records and of inline OBJE
// structures in the Tags of other records. Problems are in document order,
// one per reference, so a payload used twice is reported twice. The result is
// nil when every reference is satisfied or doc is nil. doc is normally the
// document this archive decoded to, but any document can be checked against
// the archive.
func (r *Reader) CheckRefs(doc *gedcom.Document) []RefProblem {
	var problems []RefProblem
	for _, ref := range collectFileRefs(doc) {
		kind, name := ClassifyFileRef(ref.ref)
		var err error
		switch kind {
		case RefWeb:
			continue
		case RefLocal:
			i, ok := r.byName[name]
			switch {
			case name == datasetName:
				err = reservedNameError(ref.ref)
			case !ok:
				err = fmt.Errorf("%w: %q", ErrEntryNotFound, name)
			case r.entries[i].Unusable != nil:
				err = r.entries[i].Unusable
			default:
				continue
			}
		default:
			err = notLocalError(ref.ref, kind)
		}
		problems = append(problems, RefProblem{
			XRef:        ref.xref,
			FileRef:     ref.ref,
			Translation: ref.translation,
			Kind:        kind,
			Err:         err,
		})
	}
	return problems
}

// reservedNameError is the error for a local reference that names the
// dataset's own entry.
func reservedNameError(ref string) error {
	return fmt.Errorf("%w: %q is the dataset's own entry; rename the file", ErrReservedName, ref)
}

// notLocalError is the error for a reference that is not a local file.
func notLocalError(ref string, kind RefKind) error {
	return fmt.Errorf("%w: %q (%s)", ErrNotLocal, ref, kind)
}
