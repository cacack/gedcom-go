package decoder

import (
	"errors"
	"fmt"
	"io"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
)

// ErrNotHeader is wrapped by the DecodeHeader error for input that is not a
// header alone: its first line is not "0 HEAD", or it holds another level-0
// line, such as a record, a TRLR or a second HEAD. The error's message gives
// the line.
var ErrNotHeader = errors.New("not a header-only fragment")

// DecodeHeader decodes a fragment of GEDCOM text holding only a header, a
// "0 HEAD" line and the lines beneath it, as if it were doc's header: the
// way to turn an edited header back into one for Document.ReplaceHeader, as
// DecodeRecords does for records. The header is not put in doc; doc is only
// read.
//
// The fragment is read under doc's GEDCOM version, identifier grammar and
// encoding, as DecodeRecords reads one, not under the version or CHAR the
// fragment's own lines declare. The header's Version and DeclaredVersion are
// doc's, and so is its DetectedEncoding, which describes the bytes doc was
// decoded from. A GEDC.VERS or CHAR line is kept in Tags and written as
// usual; changing the version is a conversion (see the converter package).
//
// Diagnostics are reported as [DecodeWithDiagnostics] reports them, with line
// numbers counted from the fragment's first line, and opts means what it
// means there, except that PreserveLayout is ignored.
//
// The header's tags are new to doc: the LineNumber of each is 0, as for a
// header built in code, for the reason DecodeRecords gives for records.
//
// An error wrapping ErrNotHeader is returned when the fragment is empty, its
// first line is not "0 HEAD", or it holds another level-0 line, and an error
// when doc is nil. A lenient decode that rejects every line of the fragment
// returns an error wrapping the first line's *parser.ParseError instead of
// ErrNotHeader. Other errors are those of [DecodeWithDiagnostics]: a
// lenient decode interrupted by an I/O failure returns the header read so far
// with the error.
func DecodeHeader(r io.Reader, doc *gedcom.Document, opts *DecodeOptions) (*gedcom.Header, Diagnostics, error) {
	if doc == nil {
		return nil, nil, errors.New("decode header: target document is nil")
	}
	res, err := decode(r, opts, true, doc, checkHeaderOnly)
	if res == nil {
		return nil, nil, err
	}
	header := res.Document.Header
	header.DetectedEncoding = ""
	if doc.Header != nil {
		header.DetectedEncoding = doc.Header.DetectedEncoding
	}
	for _, tag := range header.Tags {
		tag.LineNumber = 0
	}
	return header, res.Diagnostics, err
}

// checkHeaderOnly returns an error wrapping ErrNotHeader when lines are not
// those of a header alone: there are none, the first is not a "0 HEAD" line
// without an XRef, or a later line is level 0.
func checkHeaderOnly(lines []*parser.Line) error {
	if len(lines) == 0 {
		return fmt.Errorf("decode header: no 0 HEAD line: %w", ErrNotHeader)
	}
	if first := lines[0]; first.Level != 0 || first.XRef != "" || first.Tag != "HEAD" {
		return fmt.Errorf("decode header: line %d is not 0 HEAD: %w", first.LineNumber, ErrNotHeader)
	}
	for _, line := range lines[1:] {
		if line.Level == 0 {
			return fmt.Errorf("decode header: line %d is %s: %w", line.LineNumber, line.Tag, ErrNotHeader)
		}
	}
	return nil
}
