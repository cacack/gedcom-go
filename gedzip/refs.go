package gedzip

import (
	"fmt"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// fileRef is one File Path payload of a document: a FILE line of an OBJE
// record, or a TRAN line beneath one.
type fileRef struct {
	xref        string // the OBJE record's XRef
	ref         string // the payload, as written
	translation bool   // true for FILE.TRAN
}

// collectFileRefs returns every FILE and FILE.TRAN payload of doc's OBJE
// records, in document order, each FILE followed by its translations. They are
// the only two structures whose payload is a File Path, so they are every
// reference a GEDZIP archive has to cover (#127). Payloads are returned as
// written, repeats included; an empty one is returned too, since it is a
// reference that names nothing.
func collectFileRefs(doc *gedcom.Document) []fileRef {
	var refs []fileRef
	for _, media := range doc.MediaObjects() {
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
	}
	return refs
}

// RefProblem is a FILE or FILE.TRAN payload that a GEDZIP archive cannot
// satisfy, as reported by [Reader.CheckRefs] (#127).
type RefProblem struct {
	// XRef is the XRef of the OBJE record holding the reference.
	XRef string

	// FileRef is the payload as written in the document.
	FileRef string

	// Translation is true when the payload is that of a FILE.TRAN line rather
	// than of a FILE line.
	Translation bool

	// Kind is what [ClassifyFileRef] makes of the payload: [RefLocal] when the
	// entry it names is missing or unusable, otherwise [RefFileURL] or
	// [RefInvalid].
	Kind RefKind

	// Err says what is wrong. It wraps [ErrEntryNotFound] for a missing entry,
	// [ErrUnusableEntry] for one that cannot be read, and [ErrNotLocal] for a
	// file: URL or an invalid payload.
	Err error
}

// CheckRefs reports the FILE and FILE.TRAN payloads of doc's OBJE records that
// the archive does not satisfy (#127): a local file with no entry or with an
// unusable one, a file: URL, which must not be used in a GEDZIP, and a payload
// that is not a valid File Path. A web file (ftp, http, https) has no entry
// and is never a problem. No entry is read.
//
// Problems are in document order, one per reference, so a payload used twice
// is reported twice. The result is nil when every reference is satisfied or
// doc is nil. doc is normally the document this archive decoded to, but any
// document can be checked against the archive.
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

// notLocalError is the error for a reference that is not a local file.
func notLocalError(ref string, kind RefKind) error {
	return fmt.Errorf("%w: %q (%s)", ErrNotLocal, ref, kind)
}
