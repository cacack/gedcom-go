package encoder

import (
	"io"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// EncodeRecord writes one record -- its level-0 line and every line beneath
// it -- to w, with no HEAD and no TRLR. It writes the lines [EncodeWithOptions]
// writes for r within a whole document, through the same writer, so a caller
// can show, diff or log a single record without encoding a header only to
// discard it (issue #878). If opts is nil, default options are used.
//
// doc is the document r is written as part of, and may be nil. It decides the
// version-dependent behavior exactly as [EncodeWithOptions] does:
//
//   - The output version is opts.TargetVersion, or doc.Header.Version when no
//     target is set. It decides CONC line wrapping (none for GEDCOM 7.0) and
//     every other version-specific rule in the package documentation. With a
//     nil doc only TargetVersion can set it; with neither, the version is
//     unknown and lines wrap at MaxLineLength unless DisableLineWrap is set.
//   - For 7.0 output, a note pointer in a record written from its Entity is
//     written as SNOTE when it names an SNOTE record in doc.Records, and as
//     NOTE otherwise. With a nil doc every note pointer is NOTE, as from a
//     bare [StreamEncoder].
//
// So a record decoded from a file and passed with its document encodes to the
// same bytes Encode writes for it there, under the same options. r need not be
// one of doc's records; doc only supplies the context.
//
// One rule differs from Encode, because it would write a second record: an
// inline media link in a record written from its Entity is written inline, in
// the 5.5.1 layout, even for 7.0 output, rather than promoted to an
// "@INLINE_OBJEn@" OBJE record. Records written from [gedcom.Record].Tags are
// unaffected.
//
// A nil r writes nothing and returns nil, as [StreamEncoder.WriteRecord] does;
// see the package documentation for the full nil policy. A record whose type
// is a custom tag writes nothing when opts.DropUnknownTags is set. Errors are
// those of Encode for a record ([ErrLineBreak], [ErrMalformedLine],
// [ErrInvalidUTF8], a write error), and once the options' Context is done it
// writes nothing and returns that context's error, wrapped.
func EncodeRecord(w io.Writer, doc *gedcom.Document, r *gedcom.Record, opts *EncodeOptions) error {
	if r == nil {
		return nil
	}
	if opts == nil {
		opts = DefaultOptions()
	}

	var header *gedcom.Header
	if doc != nil {
		header = doc.Header
	}
	opts = opts.forOutput(header)

	var sharedNotes *sharedNoteSet
	if doc != nil {
		sharedNotes = sharedNotePointers(doc, opts)
	}

	if err := opts.cancelled(); err != nil {
		return err
	}
	return writeRecord(w, r, opts, sharedNotes)
}
