package decoder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
)

// ErrNotRecords is wrapped by the DecodeRecords error for input that is not
// records only: its first line is not level 0, or it holds a HEAD or TRLR
// line. The error's message gives the line.
var ErrNotRecords = errors.New("not a records-only fragment")

// DecodeRecords decodes a fragment of GEDCOM text holding only level-0
// records, with no HEAD and no TRLR, as if its lines were records of doc: the
// way to turn edited text back into records for Document.ReplaceRecord or
// Document.AddRecord. The records are not added to doc; doc is only read.
//
// The fragment is read under doc's GEDCOM version (Header.Version, or 5.5 when
// it has none) and identifier grammar (Header.DeclaredVersion), so a record
// decodes to the same Tags and Entity [Decode] gives the same lines in a whole
// file declaring that version. Its encoding is doc's (Header.DetectedEncoding)
// as a whole file's CHAR declaration would apply it, with UTF-8 winning: the
// fragment is read as UTF-8 when it is valid UTF-8 (the encoder writes UTF-8,
// so text printed from doc always is), and in doc's single-byte encoding, such
// as ANSEL, only when it is not. The fragment is read into memory first.
//
// Diagnostics are reported as [DecodeWithDiagnostics] reports them, with line
// numbers counted from the fragment's first line, and opts means what it
// means there, except that PreserveLayout is ignored. A record redefining an
// XRef earlier in the fragment is reported and kept; one redefining an XRef
// of doc is not this function's to judge, and AddRecord rejects it.
//
// The records are new to doc: their LineNumber, and that of every tag, is 0,
// as for a record built in code. Fragment line numbers would otherwise name
// lines of doc's source, and an encode preserving doc's layout
// (encoder.EncodeOptions.PreserveLayout) would borrow those lines' layout
// for the new records and take it from the records that own them.
//
// An empty fragment decodes to no records. An error wrapping ErrNotRecords
// is returned when the fragment's first line is not level 0 or it holds a
// HEAD or TRLR line, and an error when doc is nil. Other errors are those of
// [DecodeWithDiagnostics]: a lenient decode interrupted by an I/O failure
// returns the records read so far with the error.
func DecodeRecords(r io.Reader, doc *gedcom.Document, opts *DecodeOptions) ([]*gedcom.Record, Diagnostics, error) {
	if doc == nil {
		return nil, nil, errors.New("decode records: target document is nil")
	}
	res, err := decode(r, opts, true, doc, checkRecordsOnly)
	if res == nil {
		return nil, nil, err
	}
	records := res.Document.Records
	for _, record := range records {
		record.LineNumber = 0
		for _, tag := range record.Tags {
			tag.LineNumber = 0
		}
	}
	return records, res.Diagnostics, err
}

// fragmentReader returns the reader DecodeRecords and DecodeHeader read a
// fragment through, decoding it as UTF-8 when it is valid UTF-8 and in
// target's detected single-byte encoding otherwise (see DecodeRecords).
func fragmentReader(r io.Reader, target *gedcom.Document, opts charset.ReaderOptions) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decode fragment: %w", err)
	}
	var detected gedcom.Encoding
	if target.Header != nil {
		detected = target.Header.DetectedEncoding
	}
	if enc := singleByteEncoding(detected); enc != charset.EncodingUnknown && !utf8.Valid(data) {
		return charset.NewReaderWithEncoding(bytes.NewReader(data), enc), nil
	}
	// A headerless input is read as a whole file without a CHAR line is: a
	// byte order mark, UTF-16 recognized by its first line, or else UTF-8.
	reader, _ := charset.NewDetectingReader(bytes.NewReader(data), opts)
	return reader, nil
}

// singleByteEncoding returns the charset encoding of a single-byte
// Header.DetectedEncoding, or EncodingUnknown for any other value. ASCII is
// not one: it is read as UTF-8 is.
func singleByteEncoding(enc gedcom.Encoding) charset.Encoding {
	switch enc {
	case gedcom.EncodingANSEL:
		return charset.EncodingANSEL
	case gedcom.EncodingLATIN1:
		return charset.EncodingLATIN1
	case gedcom.EncodingWindows1250:
		return charset.EncodingWindows1250
	case gedcom.EncodingCP437:
		return charset.EncodingCP437
	case gedcom.EncodingCP850:
		return charset.EncodingCP850
	case gedcom.EncodingMacRoman:
		return charset.EncodingMacRoman
	default:
		return charset.EncodingUnknown
	}
}

// fragmentVersions returns the version a fragment of target is read as and the
// version whose identifier grammar it follows, as decode derives them from a
// whole file's header.
func fragmentVersions(target *gedcom.Document) (textVersion, xrefVersion gedcom.Version) {
	textVersion = gedcom.Version55
	if h := target.Header; h != nil {
		if h.Version != "" {
			textVersion = h.Version
		}
		xrefVersion = h.DeclaredVersion
	}
	return textVersion, xrefVersion
}

// checkRecordsOnly returns an error wrapping ErrNotRecords when lines are not
// those of level-0 records alone: the first is not level 0, or one is a HEAD
// or TRLR. A HEAD or TRLR carrying an XRef is not the header or trailer, and
// is kept as a record as decode keeps it.
func checkRecordsOnly(lines []*parser.Line) error {
	if len(lines) > 0 && lines[0].Level != 0 {
		return fmt.Errorf("decode records: line %d is level %d: %w", lines[0].LineNumber, lines[0].Level, ErrNotRecords)
	}
	for _, line := range lines {
		if line.Level == 0 && line.XRef == "" && (line.Tag == "HEAD" || line.Tag == "TRLR") {
			return fmt.Errorf("decode records: line %d is %s: %w", line.LineNumber, line.Tag, ErrNotRecords)
		}
	}
	return nil
}
