package decoder

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
	"github.com/cacack/gedcom-go/v3/version"
)

// DecodeResult contains the result of decoding a GEDCOM file with diagnostics.
// In lenient mode, Document may contain partial data even when diagnostics are present.
type DecodeResult struct {
	// Document is the parsed GEDCOM document.
	// In lenient mode, this may be a partial document if some lines failed to parse.
	Document *gedcom.Document

	// Diagnostics contains all issues encountered during parsing.
	// Empty if parsing was successful or StrictMode was enabled.
	Diagnostics Diagnostics
}

// Decode parses a GEDCOM file from an io.Reader and returns a Document.
// This is a convenience function that uses default options, so it decodes in
// lenient mode: malformed lines are recovered from or skipped rather than
// failing the decode. Use [DecodeWithOptions] with StrictMode set to fail on
// the first syntax error, or [DecodeWithDiagnostics] to see what was recovered.
func Decode(r io.Reader) (*gedcom.Document, error) {
	return DecodeWithOptions(r, DefaultOptions())
}

// DecodeWithOptions parses a GEDCOM file with custom options.
//
// It behaves exactly like [DecodeWithDiagnostics] and returns that result's
// Document; the only difference is that diagnostics are discarded. In
// particular, [DecodeOptions.StrictMode] means the same thing here: when false
// (the default), a document is returned with a nil error even if malformed
// lines were recovered from or skipped. A non-nil error is returned alongside
// a non-nil document when an I/O failure interrupts a lenient parse after some
// lines were read, or when no valid lines could be parsed at all.
func DecodeWithOptions(r io.Reader, opts *DecodeOptions) (*gedcom.Document, error) {
	res, err := decode(r, opts, false)
	if res == nil {
		return nil, err
	}
	return res.Document, err
}

// DecodeWithDiagnostics parses a GEDCOM file and returns both the document and any diagnostics.
// This function enables lenient parsing mode when StrictMode is false (the default).
//
// In lenient mode:
//   - Parse errors are collected as diagnostics rather than stopping parsing
//   - A partial document is returned if some valid data was parsed
//   - If a fatal I/O failure interrupts parsing after some lines were read, a
//     non-nil *DecodeResult is returned together with the non-nil error so
//     callers can still recover the partial document. Callers should inspect
//     both the returned *DecodeResult and error rather than discarding the
//     result on err != nil
//   - If no valid lines could be parsed at all, an empty document is returned
//     together with an error wrapping the first *parser.ParseError
//   - An error is returned with a nil *DecodeResult only when an I/O failure
//     occurs before any line was read
//
// In strict mode (StrictMode=true):
//   - Parsing fails on the first error that stands (see [DecodeOptions.StrictMode])
//   - Diagnostics will be empty on success
func DecodeWithDiagnostics(r io.Reader, opts *DecodeOptions) (*DecodeResult, error) {
	return decode(r, opts, true)
}

// decode is the single decode pipeline behind every entry point. collect
// controls only whether entity-level diagnostics are gathered; the document
// produced is identical either way, so callers that discard diagnostics skip
// the cost of building them.
//
//nolint:gocyclo // Lenient mode handling requires additional branches
func decode(r io.Reader, opts *DecodeOptions, collect bool) (*DecodeResult, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Check context cancellation before starting
	if opts.Context != nil {
		select {
		case <-opts.Context.Done():
			return nil, opts.Context.Err()
		default:
		}
	}

	// Wrap reader with UTF-8 validation. Lenient mode replaces each invalid
	// UTF-8 byte with U+FFFD and reports it, rather than losing everything
	// after it (#710). The callback must be set even when diagnostics are
	// not collected: a nil callback fails on the first invalid byte.
	var encodingDiags invalidUTF8Diagnostics
	var readerOpts charset.ReaderOptions
	if !opts.StrictMode {
		readerOpts.OnInvalidUTF8 = func(e *charset.ErrInvalidUTF8) {
			if collect {
				encodingDiags.add(e)
			}
		}
	}
	validatedReader := charset.NewReaderWithOptions(r, readerOpts)

	// Wrap with progress tracking if callback provided
	finalReader := validatedReader
	if opts.OnProgress != nil {
		finalReader = &progressReader{
			reader:    validatedReader,
			totalSize: opts.TotalSize,
			callback:  opts.OnProgress,
		}
	}

	// Parse with appropriate mode
	p := parser.NewParser()
	var lines []*parser.Line
	var diagnostics Diagnostics
	// fatalErr is set when an I/O failure interrupts a lenient-mode parse but
	// partial lines are still available. We fall through to the normal build
	// path (so the partial document gets level-jump normalization and entity
	// population) and surface fatalErr alongside the result at the end.
	var fatalErr error
	// firstParseErr is the first line-level error a lenient parse recovered
	// from, kept so a parse that recovers nothing can still report it.
	var firstParseErr *parser.ParseError

	// Both modes parse leniently: whether an XRef containing a space is an
	// error depends on the GEDCOM version, which is only known once the
	// header has been parsed (#579). Strict mode then fails on the first
	// error that stands. It stops at the first error of any other kind,
	// which stands whatever the version, so a garbage input is not read and
	// buffered in full before being rejected.
	parseOpts := &parser.ParseOptions{
		Lenient:   true,
		MaxErrors: 0, // Collect all errors
	}
	if opts.StrictMode {
		parseOpts.TolerateOnly = parser.ErrXRefContainsSpace
	}
	parsedLines, parseErrors, fe := p.ParseWithOptions(finalReader, parseOpts)

	// Detect GEDCOM version
	detectedVersion := version.DetectVersion(parsedLines)
	// The identifier grammar follows only the version the header declares:
	// a guessed version (no header, or an unrecognized VERS such as 7.1)
	// must not relax it to the 5.5 grammar.
	xrefVersion := version.DeclaredVersion(parsedLines)
	parseErrors = dropPermittedXRefErrors(parsedLines, parseErrors, xrefVersion)

	if opts.StrictMode {
		if len(parseErrors) > 0 {
			return nil, parseErrors[0]
		}
		if fe != nil {
			return nil, fe
		}
		lines = parsedLines
	} else {
		// Lenient mode: convert parse errors to diagnostics and continue.
		// Encoding repairs come first: they happen below the parser.
		diagnostics = encodingDiags.result()
		diagnostics = append(diagnostics, convertParseErrors(parseErrors)...)
		if len(parseErrors) > 0 {
			firstParseErr = parseErrors[0]
		}

		if fe != nil {
			if len(parsedLines) == 0 {
				// No partial data to recover; surface the I/O error directly.
				return nil, fe
			}
			// Partial data — fall through to the normal build path and return
			// fatalErr alongside the result.
			fatalErr = fe
		}

		lines = parsedLines
	}

	// Check context after parsing
	if opts.Context != nil {
		select {
		case <-opts.Context.Done():
			return nil, opts.Context.Err()
		default:
		}
	}

	// Check if we have any data to work with
	if len(lines) == 0 {
		// No valid lines parsed - return empty document with diagnostics
		doc := &gedcom.Document{
			XRefMap: make(map[string]*gedcom.Record),
			Header:  &gedcom.Header{Version: detectedVersion},
			Trailer: &gedcom.Trailer{},
		}
		result := &DecodeResult{
			Document:    doc,
			Diagnostics: diagnostics,
		}

		// If lines were rejected, report the first one so callers keep the
		// structured *parser.ParseError (ADR 0007).
		if firstParseErr != nil {
			return result, fmt.Errorf("no valid GEDCOM lines could be parsed: %w", firstParseErr)
		}

		// Empty input is valid
		return result, nil
	}

	// Create a collector for entity-level diagnostics. Only a lenient decode
	// that reports them keeps them, but every decode needs the version it
	// carries.
	collector := &diagnosticCollector{
		version:     xrefVersion,
		textVersion: detectedVersion,
		discard:     opts.StrictMode || !collect,
	}

	// Lenient mode: recover from malformed indentation in real-world exports by
	// clamping over-jumped levels to prevLevel+1 and emitting CodeBadLevelJump.
	// A dropped level-0 line takes its record's subordinate lines with it, so
	// they are not attached to the preceding record (#723).
	if !opts.StrictMode {
		lines = skipOrphanedRecords(lines, parseErrors, collector)
		normalizeLevelJumps(lines, collector)
	}

	// Build document from lines
	doc := buildDocument(lines, detectedVersion, collector)
	doc.Header.DeclaredVersion = xrefVersion

	// Convert raw tags to proper entity types
	populateEntities(doc, collector)

	// Merge entity-level diagnostics with parser diagnostics
	diagnostics = append(diagnostics, collector.diagnostics...)

	return &DecodeResult{
		Document:    doc,
		Diagnostics: diagnostics,
	}, fatalErr
}

// dropPermittedXRefErrors removes from errs each parser.ErrXRefContainsSpace
// error whose identifier is valid in version v, as decided by
// gedcom.IsPointerXRefForVersion: GEDCOM 5.5 and 5.5.1 allow a space inside an
// identifier, 7.0 does not (#579). The parser cannot make that call, because it
// does not know the version; it reports every spaced identifier and keeps the
// recovered line, so dropping the error is all that accepting one takes.
func dropPermittedXRefErrors(lines []*parser.Line, errs []*parser.ParseError, v gedcom.Version) []*parser.ParseError {
	var kept []*parser.ParseError
	for _, pe := range errs {
		if errors.Is(pe, parser.ErrXRefContainsSpace) {
			// Lines are in input order, so the recovered line is found by
			// its line number.
			i, found := sort.Find(len(lines), func(i int) int { return pe.Line - lines[i].LineNumber })
			if found && gedcom.IsPointerXRefForVersion(lines[i].XRef, v) {
				continue
			}
		}
		kept = append(kept, pe)
	}
	return kept
}

// convertParseErrors converts parser.ParseError instances to Diagnostics.
func convertParseErrors(parseErrors []*parser.ParseError) Diagnostics {
	if len(parseErrors) == 0 {
		return nil
	}

	diagnostics := make(Diagnostics, 0, len(parseErrors))
	for _, pe := range parseErrors {
		code := classifyParseError(pe.Message)
		diagnostics = append(diagnostics, NewParseError(pe.Line, code, pe.Message, pe.Context))
	}
	return diagnostics
}

// MaxInvalidEncodingDiagnostics is the number of per-byte CodeInvalidEncoding
// diagnostics a lenient decode reports before it stops reporting bytes one by
// one. Every invalid byte is still replaced; the bytes past the cap are counted
// in one closing CodeInvalidEncoding diagnostic, so input that is mostly not
// UTF-8 (a binary file, or a long legacy-encoded export) cannot make the
// diagnostics grow with the file.
const MaxInvalidEncodingDiagnostics = 100

// invalidUTF8Diagnostics accumulates the bytes a lenient decode replaced,
// keeping at most MaxInvalidEncodingDiagnostics of them as diagnostics.
type invalidUTF8Diagnostics struct {
	diags         Diagnostics
	suppressed    int
	firstOverLine int
}

func (c *invalidUTF8Diagnostics) add(e *charset.ErrInvalidUTF8) {
	if len(c.diags) < MaxInvalidEncodingDiagnostics {
		c.diags = append(c.diags, invalidUTF8Diagnostic(e))
		return
	}
	if c.suppressed == 0 {
		c.firstOverLine = e.Line
	}
	c.suppressed++
}

// result returns the per-byte diagnostics followed, if the cap was reached, by
// one diagnostic counting the bytes not reported individually, at the line of
// the first of them.
func (c *invalidUTF8Diagnostics) result() Diagnostics {
	if c.suppressed == 0 {
		return c.diags
	}
	return append(c.diags, NewParseError(c.firstOverLine, CodeInvalidEncoding,
		fmt.Sprintf("%d further invalid UTF-8 bytes, from this line on, replaced with U+FFFD and not reported individually", c.suppressed), ""))
}

// invalidUTF8Diagnostic reports an invalid UTF-8 byte that lenient mode
// replaced with U+FFFD. It is an error, not a warning: the character the byte
// stood for is lost, though the line it is on is kept.
func invalidUTF8Diagnostic(e *charset.ErrInvalidUTF8) Diagnostic {
	return NewParseError(e.Line, CodeInvalidEncoding,
		fmt.Sprintf("invalid UTF-8 byte 0x%02X at column %d replaced with U+FFFD", e.Byte, e.Column), "")
}

// classifyParseError maps a parse error message to a diagnostic code.
//
// CodeBadLevelJump is never returned here: it means the line was clamped and
// kept, and it is only ever emitted by normalizeLevelJumps. An over-limit
// level therefore classifies as CodeInvalidLevel, alongside the other level
// rejections. The remaining codes say only what was wrong with the line, not
// whether it survived.
func classifyParseError(message string) string {
	msg := strings.ToLower(message)

	switch {
	case strings.Contains(msg, "empty line"):
		return CodeEmptyLine
	case strings.Contains(msg, "invalid level") ||
		strings.Contains(msg, "level cannot be negative") ||
		strings.Contains(msg, "nesting depth"):
		return CodeInvalidLevel
	case strings.Contains(msg, "xref"):
		return CodeInvalidXRef
	default:
		return CodeSyntaxError
	}
}

// buildDocument constructs a Document from parsed lines.
// If collector is nil, no diagnostics are collected.
func buildDocument(lines []*parser.Line, ver gedcom.Version, collector *diagnosticCollector) *gedcom.Document {
	doc := &gedcom.Document{
		XRefMap: make(map[string]*gedcom.Record),
		Header:  &gedcom.Header{Version: ver},
		Trailer: &gedcom.Trailer{},
	}

	if len(lines) == 0 {
		return doc
	}

	// Build header
	buildHeader(doc, lines, ver, collector)

	// Build records and XRefMap
	buildRecords(doc, lines, collector)

	return doc
}

// buildHeader extracts header information from lines, and reports unknown
// header tags to collector (see reportUnknownHeaderTags).
//
//nolint:gocyclo // Header parsing requires handling many tag types
func buildHeader(doc *gedcom.Document, lines []*parser.Line, ver gedcom.Version, collector *diagnosticCollector) {
	inHead := false
	inSour := false

	for i, line := range lines {
		// The GEDCOM grammar gives HEAD no cross-reference identifier, so
		// `0 @X1@ HEAD` is not a header: treating it as one would let a second,
		// bogus header overwrite the real header's typed fields. buildRecords
		// keeps it as an ordinary record and reports it.
		if line.Level == 0 && line.Tag == "HEAD" && line.XRef == "" {
			inHead = true
			continue
		}

		if line.Level == 0 {
			inHead = false
			inSour = false
		}

		if !inHead {
			continue
		}

		// Preserve every header sub-tag in raw form (lossless dual storage,
		// mirroring how buildRecords populates Record.Tags). Typed fields are
		// still extracted below; this guarantees custom/unmapped header tags
		// (e.g. _RTLSAVE, _PROJECT_GUID, header NOTEs) are not dropped.
		doc.Header.Tags = append(doc.Header.Tags, &gedcom.Tag{
			Level:      line.Level,
			Tag:        line.Tag,
			Value:      line.Value,
			XRef:       line.XRef,
			LineNumber: line.LineNumber,
		})

		// Track when we're inside SOUR structure
		if line.Level == 1 && line.Tag == "SOUR" {
			inSour = true
			doc.Header.SourceSystem = collector.text(line.Value)
			continue
		}

		// Parse SCHMA structure (GEDCOM 7.0 only)
		if line.Level == 1 && line.Tag == "SCHMA" {
			inSour = false
			// Only parse SCHMA for GEDCOM 7.0; leave Schema nil for 5.5/5.5.1
			if ver == gedcom.Version70 {
				doc.Schema = &gedcom.SchemaDefinition{
					TagMappings: make(map[string]string),
				}
				parseSchemaTag(doc, lines, i)
			}
			continue
		}

		// Exit SOUR when we see another level 1 tag
		if line.Level == 1 {
			inSour = false
		}

		// Extract header fields. A header tag handled here must also be listed
		// in headerSubstructures, or it is reported as UNKNOWN_TAG.
		switch line.Tag {
		case "CHAR":
			// Level 1 only: HEAD.CHAR is the one charset declaration. It is
			// the level charset detection reads and the level the encoder
			// rewrites to UTF-8, so a CHAR nested deeper is neither honoured
			// nor typed here; it stays in Header.Tags and is written back
			// verbatim. See #464.
			//
			// tagToken, not line.Value: the encoding name is a token compared
			// by exact string in validator/encoding.go, so a producer's
			// "1 CHAR  UTF-8" would otherwise be reported invalid for a file
			// charset detection reads perfectly well. See #426.
			if line.Level == 1 {
				doc.Header.Encoding = gedcom.Encoding(tagToken(line.Value))
			}
		case "DATE":
			// Level 1 only: HEAD.SOUR.DATA.DATE is the source data's
			// publication date, not the file's transmission date.
			if line.Level == 1 {
				doc.Header.Date = collector.text(line.Value)
			}
		case "LANG":
			doc.Header.Language = collector.text(line.Value)
		case "COPR":
			doc.Header.Copyright = collector.text(line.Value)
		case "SUBM":
			// Level 1 only: the header's own pointer to the submitter record.
			if line.Level == 1 {
				doc.Header.Submitter = line.Value
			}
		case "_TREE":
			// Ancestry.com tree identifier (subordinate of SOUR)
			if inSour && line.Level == 2 {
				doc.Header.AncestryTreeID = collector.text(line.Value)
			}
		}
	}

	parseHeaderStructures(doc.Header, collector)

	// Ensure header has a version
	if doc.Header.Version == "" {
		doc.Header.Version = ver
	}

	// Detect vendor from HEAD.SOUR, SOUR.NAME and SOUR.CORP
	doc.Vendor = gedcom.DetectVendorFromHeader(doc.Header)

	reportUnknownHeaderTags(doc.Header.Tags, collector)
}

// parseHeaderStructures fills the Header's typed views of the HEAD.SOUR
// subtree (VERS, NAME, CORP), HEAD.DEST, HEAD.DATE.TIME and HEAD.PLAC.FORM from
// h.Tags, whose levels start at 1. Each is read only at its own position, so a
// VERS under GEDC or a DATE.TIME under SOUR.DATA is not mistaken for one.
// Unknown subordinates are left to reportUnknownHeaderTags, so the CORP address
// is parsed with a collector that discards diagnostics, to avoid reporting them
// twice; collector supplies the version text values are unescaped for.
func parseHeaderStructures(h *gedcom.Header, collector *diagnosticCollector) {
	quiet := collector.quiet()
	parent := ""
	for i, tag := range h.Tags {
		if tag.Level == 1 {
			parent = tag.Tag
			if tag.Tag == "DEST" {
				h.Destination = quiet.text(tag.Value)
			}
			continue
		}
		if tag.Level != 2 {
			continue
		}
		switch parent + "." + tag.Tag {
		case "SOUR.VERS":
			h.SourceVersion = quiet.text(tag.Value)
		case "SOUR.NAME":
			h.SourceName = quiet.text(tag.Value)
		case "SOUR.CORP":
			h.SourceCorporation = parseHeaderCorporation(h.Tags, i, quiet)
		case "DATE.TIME":
			h.Time = quiet.text(tag.Value)
		case "PLAC.FORM":
			h.PlaceForm = quiet.text(tag.Value)
		}
	}
}

// parseHeaderCorporation builds the CORP structure at tags[corpIdx] from its
// subordinates: an ADDR and the PHON, EMAIL, FAX and WWW siblings of it.
func parseHeaderCorporation(tags []*gedcom.Tag, corpIdx int, quiet *diagnosticCollector) *gedcom.Corporation {
	corp := &gedcom.Corporation{Name: quiet.text(tags[corpIdx].Value)}
	level := tags[corpIdx].Level
	for i := corpIdx + 1; i < len(tags) && tags[i].Level > level; i++ {
		tag := tags[i]
		if tag.Level != level+1 {
			continue
		}
		switch tag.Tag {
		case "ADDR":
			corp.Address = parseEventAddress(tags, i, tag.Level, quiet)
		case "PHON":
			corp.Phone = append(corp.Phone, quiet.text(tag.Value))
		case "EMAIL":
			corp.Email = append(corp.Email, quiet.text(tag.Value))
		case "FAX":
			corp.Fax = append(corp.Fax, quiet.text(tag.Value))
		case "WWW":
			corp.Website = append(corp.Website, quiet.text(tag.Value))
		}
	}
	return corp
}

// headerSubstructures lists, for each header context, the tags GEDCOM 5.5,
// 5.5.1 and 7.0 define beneath it -- the union of the three, as the record
// parsers accept any version's tags. A context is the dotted path of tags from
// HEAD. Only the contexts listed here are checked: the subordinates of a
// context absent from the map (a leaf such as HEAD.DEST, or a deep one such as
// a 7.0 HEAD.NOTE's SOUR citation) are not reported.
//
// Every key's last tag is listed under its parent, so a path through an
// unknown tag can never reach a checked context.
var headerSubstructures = map[string]map[string]bool{
	"HEAD": {
		"SOUR": true, "DEST": true, "DATE": true, "SUBM": true, "SUBN": true,
		"FILE": true, "COPR": true, "GEDC": true, "CHAR": true, "LANG": true,
		"PLAC": true, "NOTE": true, "SNOTE": true, "SCHMA": true,
	},
	"HEAD.SOUR":           {"VERS": true, "NAME": true, "CORP": true, "DATA": true},
	"HEAD.SOUR.CORP":      {"ADDR": true, "PHON": true, "EMAIL": true, "FAX": true, "WWW": true},
	"HEAD.SOUR.CORP.ADDR": {"ADR1": true, "ADR2": true, "ADR3": true, "CITY": true, "STAE": true, "POST": true, "CTRY": true},
	"HEAD.SOUR.DATA":      {"DATE": true, "COPR": true},
	"HEAD.SOUR.DATA.DATE": {"TIME": true},
	"HEAD.SOUR.DATA.COPR": {},
	"HEAD.DATE":           {"TIME": true},
	"HEAD.GEDC":           {"VERS": true, "FORM": true},
	"HEAD.CHAR":           {"VERS": true},
	"HEAD.PLAC":           {"FORM": true},
	"HEAD.NOTE":           {"MIME": true, "LANG": true, "TRAN": true, "SOUR": true},
	"HEAD.SCHMA":          {"TAG": true},
}

// reportUnknownHeaderTags reports each header tag that headerSubstructures
// does not list under its context, as the record parsers do for theirs
// (issue #449). Underscore-prefixed tags are extensions and are not reported
// (ADR 0003). CONT and CONC are accepted in every checked context: they
// continue a line's payload rather than add a substructure, and GEDCOM 7.0
// allows CONT under any line. Only the unknown tag itself is reported, not its
// subordinates. tags is Header.Tags, so its levels start at 1.
func reportUnknownHeaderTags(tags []*gedcom.Tag, collector *diagnosticCollector) {
	if !collector.collecting() {
		return
	}

	// path[i] is the tag at level i on the way to the current line.
	path := []string{"HEAD"}
	for _, tag := range tags {
		if tag.Level < 1 || tag.Level > len(path) {
			continue
		}
		path = append(path[:tag.Level], tag.Tag)

		known, checked := headerSubstructures[strings.Join(path[:tag.Level], ".")]
		if !checked || known[tag.Tag] || tag.Tag == "CONT" || tag.Tag == "CONC" ||
			strings.HasPrefix(tag.Tag, "_") {
			continue
		}
		collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
	}
}

// parseSchemaTag parses TAG subordinates within a SCHMA structure.
// TAG value format: "[tagname] [uri]" (space-separated)
func parseSchemaTag(doc *gedcom.Document, lines []*parser.Line, schmaIndex int) {
	schmaLevel := lines[schmaIndex].Level

	for j := schmaIndex + 1; j < len(lines); j++ {
		subLine := lines[j]

		// Stop when we reach the same or lower level (exiting SCHMA)
		if subLine.Level <= schmaLevel {
			break
		}

		// Parse TAG at level schmaLevel+1
		if subLine.Level == schmaLevel+1 && subLine.Tag == "TAG" {
			// Split on first space, then trim to handle multiple spaces/tabs
			parts := strings.SplitN(subLine.Value, " ", 2)
			if len(parts) == 2 {
				tagName := strings.TrimSpace(parts[0])
				uri := strings.TrimSpace(parts[1])
				doc.Schema.TagMappings[tagName] = uri
			}
			// Malformed TAG values (missing URI) are silently skipped
		}
	}
}

// buildRecords extracts records from lines and builds the XRefMap.
// If collector is nil, no diagnostics are collected.
func buildRecords(doc *gedcom.Document, lines []*parser.Line, collector *diagnosticCollector) {
	var currentRecord *gedcom.Record
	var currentTags []*gedcom.Tag

	for _, line := range lines {
		// Level 0 lines are records or structural tags
		if line.Level == 0 {
			// Save previous record if exists
			if currentRecord != nil {
				currentRecord.Tags = currentTags
				doc.Records = append(doc.Records, currentRecord)
				currentTags = nil
			}

			// Skip HEAD and TRLR — but only the real ones. Neither takes a
			// cross-reference identifier in the GEDCOM grammar, so a line such
			// as `0 @X1@ HEAD` is not the document's header or trailer. Skipping
			// it would drop the line and silently reparent its subordinates;
			// keep it as an ordinary record (Type "HEAD"/"TRLR", XRef "@X1@")
			// and report the misplaced identifier instead.
			if line.Tag == "HEAD" || line.Tag == "TRLR" {
				if line.XRef == "" {
					currentRecord = nil
					continue
				}
				collector.addXRefOnStructuralLine(line)
			}

			// Start new record
			currentRecord = newRecord(line)

			checkRecordType(line, collector)
			indexRecord(doc, currentRecord, line, collector)

			continue
		}

		// Add tags to current record
		if currentRecord != nil {
			currentTags = append(currentTags, newTag(line))
		}
	}

	// Save last record. DecodeRecord assembles a single record the same way
	// (newRecord, newTag, nil Tags when there are no subordinates); keep the
	// two in step.
	if currentRecord != nil {
		currentRecord.Tags = currentTags
		doc.Records = append(doc.Records, currentRecord)
	}
}

// newRecord returns the record a level-0 line starts, with no Tags yet.
func newRecord(line *parser.Line) *gedcom.Record {
	return &gedcom.Record{
		XRef:       line.XRef,
		Type:       gedcom.RecordType(line.Tag),
		Value:      line.Value,
		LineNumber: line.LineNumber,
	}
}

// newTag returns the raw tag a subordinate line is kept as.
//
// XRef is carried through even at level >= 1. Well-formed GEDCOM never has an
// XRef there, so this is empty for valid input; it is non-empty only when
// splitSpacedXRef recovered a malformed identifier (e.g. "1 @I 1@ NOTE").
// Lossless representation requires keeping it rather than dropping it on the
// floor.
func newTag(line *parser.Line) *gedcom.Tag {
	return &gedcom.Tag{
		Level:      line.Level,
		Tag:        line.Tag,
		Value:      line.Value,
		XRef:       line.XRef,
		LineNumber: line.LineNumber,
	}
}

// checkRecordType reports a level-0 line whose tag is not a record type the
// decoder knows. Such a line is still kept as a record of that type, with no
// entity.
func checkRecordType(line *parser.Line, collector *diagnosticCollector) {
	if isUnknownRecordTag(line.Tag) {
		collector.addUnknownRecordType(line)
	}
}

// isUnknownRecordTag reports whether a level-0 tag is outside the standard
// record set and not otherwise accounted for. Extension records ("_CUSTOM")
// are allowed. HEAD and TRLR only start a record when they carry an XRef,
// which buildRecords reports as INVALID_XREF. A tag starting with "@" is a
// level-0 XRef with no tag, which the parser already reported as INVALID_XREF.
func isUnknownRecordTag(tag string) bool {
	if tag == "HEAD" || tag == "TRLR" ||
		strings.HasPrefix(tag, "_") || strings.HasPrefix(tag, "@") {
		return false
	}
	return !standardRecordTypes[gedcom.RecordType(tag)]
}

// indexRecord adds a record just started from a level-0 line to doc.XRefMap,
// reporting a redefined XRef to collector.
func indexRecord(doc *gedcom.Document, record *gedcom.Record, line *parser.Line, collector *diagnosticCollector) {
	// Index in XRefMap if it has an XRef. The first definition of an XRef
	// wins: a later record reusing it stays in Records (nothing is dropped)
	// but pointers keep resolving to the first, and the redefinition is
	// reported.
	if line.XRef != "" {
		if first, exists := doc.XRefMap[line.XRef]; exists {
			collector.addDuplicateXRef(line, first.LineNumber)
		} else {
			doc.XRefMap[line.XRef] = record
		}
	}
}

// standardRecordTypes is the set of level-0 record tags the GEDCOM 5.5, 5.5.1
// and 7.0 grammars define, other than HEAD and TRLR.
var standardRecordTypes = map[gedcom.RecordType]bool{
	gedcom.RecordTypeIndividual: true,
	gedcom.RecordTypeFamily:     true,
	gedcom.RecordTypeSource:     true,
	gedcom.RecordTypeRepository: true,
	gedcom.RecordTypeNote:       true,
	gedcom.RecordTypeMedia:      true,
	gedcom.RecordTypeSubmitter:  true,
	gedcom.RecordTypeSubmission: true,
	gedcom.RecordTypeSharedNote: true,
}

// lineContext reconstructs a line from its parsed fields, the same shape the
// parser reports raw lines in, for a diagnostic's Context.
func lineContext(line *parser.Line) string {
	parts := []string{strconv.Itoa(line.Level)}
	if line.XRef != "" {
		parts = append(parts, line.XRef)
	}
	parts = append(parts, line.Tag)
	if line.Value != "" {
		parts = append(parts, line.Value)
	}
	return strings.Join(parts, " ")
}

// addDuplicateXRef records a level-0 line that defines an XRef an earlier
// record already defined (firstLine). SeverityError: every pointer to the
// XRef is ambiguous, and the decoder resolves it to the first definition.
func (c *diagnosticCollector) addDuplicateXRef(line *parser.Line, firstLine int) {
	if !c.collecting() {
		return
	}
	c.add(NewDiagnostic(
		line.LineNumber,
		SeverityError,
		CodeDuplicateXRef,
		fmt.Sprintf("duplicate xref %s: already defined at line %d; pointers resolve to the first definition", line.XRef, firstLine),
		lineContext(line),
	))
}

// addUnknownRecordType records a level-0 line whose tag is neither a standard
// record type nor an extension ("_" prefix), as CodeUnknownTag with
// SeverityWarning, the code and severity unknown subordinate tags use. A tag
// that is a standard record type in another case ("indi") is named as such:
// tags are case-sensitive, so the record still gets no entity.
func (c *diagnosticCollector) addUnknownRecordType(line *parser.Line) {
	if !c.collecting() {
		return
	}
	msg := fmt.Sprintf("unknown record type: %s", line.Tag)
	if upper := strings.ToUpper(line.Tag); standardRecordTypes[gedcom.RecordType(upper)] {
		msg += fmt.Sprintf(" (tags are case-sensitive; did you mean %s?)", upper)
	}
	c.add(NewDiagnostic(
		line.LineNumber,
		SeverityWarning,
		CodeUnknownTag,
		msg,
		lineContext(line),
	))
}

// addXRefOnStructuralLine records a level-0 HEAD or TRLR line that carries a
// cross-reference identifier the GEDCOM grammar does not allow there.
//
// The code is CodeInvalidXRef — an XRef where the grammar forbids one, the same
// classification the malformed-XRef shapes in the parser use. Severity is
// SeverityError, not SeverityWarning: the line is not a valid header or trailer,
// so the file is wrong even though the decoder keeps the data as a record.
func (c *diagnosticCollector) addXRefOnStructuralLine(line *parser.Line) {
	if !c.collecting() {
		return
	}
	c.add(NewDiagnostic(
		line.LineNumber,
		SeverityError,
		CodeInvalidXRef,
		fmt.Sprintf("xref not allowed on %s: %s", line.Tag, line.XRef),
		lineContext(line),
	))
}
