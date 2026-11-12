package decoder

import (
	"errors"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
)

// DecodeRecord decodes one record read by a streaming parser ([parser.Records],
// [parser.RecordIterator], or a [parser.LazyParser] lookup) into a typed
// [gedcom.Record], so streaming callers get the same Entity, Tags and fields
// [Decode] gives that record without building a whole Document. It is the
// bridge from raw lines to validator.StreamingValidator.ValidateRecord and to
// typed indexed access.
//
// ver is the GEDCOM version the file's header declares: pass
// version.DeclaredVersion(head.Lines) for the HEAD record, which the streaming
// parsers yield first. It decides the identifier grammar and the "@@" text
// unescaping, as the header does for [Decode]. Pass "" when the header declares
// no version this library recognizes; the record is then read by the GEDCOM 5.5
// text rule, the version [Decode] falls back to, though [Decode] can also guess
// from tags elsewhere in the file and DecodeRecord sees only this record.
//
// opts is read for StrictMode and Context only; nil means [DefaultOptions].
// When StrictMode is false, over-jumped levels are clamped as [Decode] clamps
// them, on a copy: raw is never modified. When Context is done, its error is
// returned.
//
// A HEAD or TRLR record decodes to a Record of that Type holding its Tags and
// no Entity; [Decode] keeps the real header and trailer in Document.Header and
// Document.Trailer instead of Document.Records. Diagnostics are not reported:
// use [DecodeWithDiagnostics] when they are needed.
//
// An error is returned, with a nil Record, when raw is nil, holds no lines, or
// does not start with a level-0 line (a file whose first line is subordinate;
// [Decode] drops such lines).
func DecodeRecord(raw *parser.RawRecord, ver gedcom.Version, opts *DecodeOptions) (*gedcom.Record, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	if opts.Context != nil {
		if err := opts.Context.Err(); err != nil {
			return nil, err
		}
	}
	if raw == nil || len(raw.Lines) == 0 {
		return nil, errors.New("decode record: no lines")
	}
	if raw.Lines[0].Level != 0 {
		return nil, errors.New("decode record: first line is not level 0")
	}

	textVersion := ver
	if textVersion == "" {
		textVersion = gedcom.Version55
	}
	collector := &diagnosticCollector{version: ver, textVersion: textVersion, discard: true}

	lines := raw.Lines
	if !opts.StrictMode {
		lines = copyLines(lines)
		normalizeLevelJumps(lines, collector)
	}

	record := newRecord(lines[0])
	if len(lines) > 1 {
		// A record with no subordinate lines keeps nil Tags, as in Decode.
		record.Tags = make([]*gedcom.Tag, 0, len(lines)-1)
	}
	for _, line := range lines[1:] {
		record.Tags = append(record.Tags, newTag(line))
	}
	populateRecordEntity(record, collector)
	return record, nil
}

// copyLines returns copies of lines, so normalizeLevelJumps can clamp levels
// without modifying lines a caller owns.
func copyLines(lines []*parser.Line) []*parser.Line {
	values := make([]parser.Line, len(lines))
	copies := make([]*parser.Line, len(lines))
	for i, line := range lines {
		values[i] = *line
		copies[i] = &values[i]
	}
	return copies
}
