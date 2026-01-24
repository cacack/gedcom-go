package decoder

import (
	"errors"
	"io"
	"strings"

	"github.com/cacack/gedcom-go/charset"
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
	"github.com/cacack/gedcom-go/version"
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
// This is a convenience function that uses default options.
func Decode(r io.Reader) (*gedcom.Document, error) {
	return DecodeWithOptions(r, DefaultOptions())
}

// DecodeWithOptions parses a GEDCOM file with custom options.
func DecodeWithOptions(r io.Reader, opts *DecodeOptions) (*gedcom.Document, error) {
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

	// Wrap reader with UTF-8 validation
	validatedReader := charset.NewReader(r)

	// Wrap with progress tracking if callback provided
	finalReader := validatedReader
	if opts.OnProgress != nil {
		finalReader = &progressReader{
			reader:    validatedReader,
			totalSize: opts.TotalSize,
			callback:  opts.OnProgress,
		}
	}

	// Parse all lines
	p := parser.NewParser()
	lines, err := p.Parse(finalReader)
	if err != nil {
		// Preserve charset errors in the error message
		return nil, err
	}

	// Check context after parsing
	if opts.Context != nil {
		select {
		case <-opts.Context.Done():
			return nil, opts.Context.Err()
		default:
		}
	}

	// Detect GEDCOM version
	detectedVersion, err := version.DetectVersion(lines)
	if err != nil {
		return nil, err
	}

	// Build document from lines
	doc := buildDocument(lines, detectedVersion)

	// Convert raw tags to proper entity types
	// Pass nil collector for existing API (no diagnostics collection)
	populateEntities(doc, nil)

	return doc, nil
}

// DecodeWithDiagnostics parses a GEDCOM file and returns both the document and any diagnostics.
// This function enables lenient parsing mode when StrictMode is false (the default).
//
// In lenient mode:
//   - Parse errors are collected as diagnostics rather than stopping parsing
//   - A partial document is returned if some valid data was parsed
//   - An error is returned only if no valid records could be parsed
//
// In strict mode (StrictMode=true):
//   - Parsing fails on the first error (current behavior)
//   - Diagnostics will be empty on success
//
//nolint:gocyclo // Lenient mode handling requires additional branches
func DecodeWithDiagnostics(r io.Reader, opts *DecodeOptions) (*DecodeResult, error) {
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

	// Wrap reader with UTF-8 validation
	validatedReader := charset.NewReader(r)

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

	if opts.StrictMode {
		// Strict mode: use existing Parse behavior
		parsedLines, err := p.Parse(finalReader)
		if err != nil {
			return nil, err
		}
		lines = parsedLines
	} else {
		// Lenient mode: collect errors and continue
		parseOpts := &parser.ParseOptions{
			Lenient:   true,
			MaxErrors: 0, // Collect all errors
		}
		parsedLines, parseErrors, fatalErr := p.ParseWithOptions(finalReader, parseOpts)

		// Convert parse errors to diagnostics
		diagnostics = convertParseErrors(parseErrors)

		// Fatal errors (I/O failures) are always returned
		if fatalErr != nil {
			// Still return partial results with diagnostics
			if len(parsedLines) > 0 {
				doc := buildDocumentWithVersion(parsedLines, opts)
				return &DecodeResult{
					Document:    doc,
					Diagnostics: diagnostics,
				}, fatalErr
			}
			return nil, fatalErr
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
			Header:  &gedcom.Header{},
			Trailer: &gedcom.Trailer{},
		}
		result := &DecodeResult{
			Document:    doc,
			Diagnostics: diagnostics,
		}

		// If we had diagnostics, return an error indicating parsing failed
		if len(diagnostics) > 0 {
			return result, errors.New("no valid GEDCOM lines could be parsed")
		}

		// Empty input is valid
		return result, nil
	}

	// Detect GEDCOM version
	detectedVersion, err := version.DetectVersion(lines)
	if err != nil {
		// Version detection failed - still try to build partial document
		detectedVersion = ""
	}

	// Build document from lines
	doc := buildDocument(lines, detectedVersion)

	// Create a collector for entity-level diagnostics if in lenient mode
	var collector *diagnosticCollector
	if !opts.StrictMode {
		collector = &diagnosticCollector{
			lenient: true,
		}
	}

	// Convert raw tags to proper entity types
	populateEntities(doc, collector)

	// Merge entity-level diagnostics with parser diagnostics
	if collector != nil {
		diagnostics = append(diagnostics, collector.diagnostics...)
	}

	return &DecodeResult{
		Document:    doc,
		Diagnostics: diagnostics,
	}, nil
}

// buildDocumentWithVersion builds a document and detects the version.
// This is a helper for DecodeWithDiagnostics when handling fatal errors.
func buildDocumentWithVersion(lines []*parser.Line, _ *DecodeOptions) *gedcom.Document {
	detectedVersion, err := version.DetectVersion(lines)
	if err != nil {
		detectedVersion = ""
	}
	doc := buildDocument(lines, detectedVersion)
	// Pass nil collector for partial builds (diagnostics not collected)
	populateEntities(doc, nil)
	return doc
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

// classifyParseError maps a parse error message to a diagnostic code.
func classifyParseError(message string) string {
	msg := strings.ToLower(message)

	switch {
	case strings.Contains(msg, "empty line"):
		return CodeEmptyLine
	case strings.Contains(msg, "invalid level") || strings.Contains(msg, "level cannot be negative"):
		return CodeInvalidLevel
	case strings.Contains(msg, "xref"):
		return CodeInvalidXRef
	case strings.Contains(msg, "nesting") || strings.Contains(msg, "jump"):
		return CodeBadLevelJump
	default:
		return CodeSyntaxError
	}
}

// buildDocument constructs a Document from parsed lines.
func buildDocument(lines []*parser.Line, ver gedcom.Version) *gedcom.Document {
	doc := &gedcom.Document{
		XRefMap: make(map[string]*gedcom.Record),
		Header:  &gedcom.Header{Version: ver},
		Trailer: &gedcom.Trailer{},
	}

	if len(lines) == 0 {
		return doc
	}

	// Build header
	buildHeader(doc, lines, ver)

	// Build records and XRefMap
	buildRecords(doc, lines)

	return doc
}

// buildHeader extracts header information from lines.
//
//nolint:gocyclo // Header parsing requires handling many tag types
func buildHeader(doc *gedcom.Document, lines []*parser.Line, ver gedcom.Version) {
	inHead := false
	inSour := false

	for i, line := range lines {
		if line.Level == 0 && line.Tag == "HEAD" {
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

		// Track when we're inside SOUR structure
		if line.Level == 1 && line.Tag == "SOUR" {
			inSour = true
			doc.Header.SourceSystem = line.Value
			continue
		}

		// Parse SCHMA structure (GEDCOM 7.0)
		if line.Level == 1 && line.Tag == "SCHMA" {
			inSour = false
			// Initialize schema with empty TagMappings
			doc.Schema = &gedcom.SchemaDefinition{
				TagMappings: make(map[string]string),
			}
			// Parse TAG subordinates
			parseSchemaTag(doc, lines, i)
			continue
		}

		// Exit SOUR when we see another level 1 tag
		if line.Level == 1 {
			inSour = false
		}

		// Extract header fields
		switch line.Tag {
		case "CHAR":
			doc.Header.Encoding = gedcom.Encoding(line.Value)
		case "LANG":
			doc.Header.Language = line.Value
		case "COPR":
			doc.Header.Copyright = line.Value
		case "_TREE":
			// Ancestry.com tree identifier (subordinate of SOUR)
			if inSour && line.Level == 2 {
				doc.Header.AncestryTreeID = line.Value
			}
		}
	}

	// Ensure header has a version
	if doc.Header.Version == "" {
		doc.Header.Version = ver
	}

	// Detect vendor from source system
	doc.Vendor = gedcom.DetectVendor(doc.Header.SourceSystem)
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
			parts := strings.SplitN(subLine.Value, " ", 2)
			if len(parts) == 2 {
				tagName := parts[0]
				uri := parts[1]
				doc.Schema.TagMappings[tagName] = uri
			}
			// Malformed TAG values (missing URI) are silently skipped
		}
	}
}

// buildRecords extracts records from lines and builds the XRefMap.
func buildRecords(doc *gedcom.Document, lines []*parser.Line) {
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

			// Skip HEAD and TRLR
			if line.Tag == "HEAD" || line.Tag == "TRLR" {
				currentRecord = nil
				continue
			}

			// Start new record
			currentRecord = &gedcom.Record{
				XRef:       line.XRef,
				Type:       gedcom.RecordType(line.Tag),
				Value:      line.Value,
				LineNumber: line.LineNumber,
			}

			// Index in XRefMap if it has an XRef
			if line.XRef != "" {
				doc.XRefMap[line.XRef] = currentRecord
			}

			continue
		}

		// Add tags to current record
		if currentRecord != nil {
			tag := &gedcom.Tag{
				Level:      line.Level,
				Tag:        line.Tag,
				Value:      line.Value,
				LineNumber: line.LineNumber,
			}
			currentTags = append(currentTags, tag)
		}
	}

	// Save last record
	if currentRecord != nil {
		currentRecord.Tags = currentTags
		doc.Records = append(doc.Records, currentRecord)
	}
}
