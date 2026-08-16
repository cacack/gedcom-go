package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MaxNestingDepth is the number of nesting levels accepted, derived from the
// GEDCOM grammar rather than from any resource limit: the level field is at
// most two digits, so the deepest valid level is 99. Valid levels run from 0
// through MaxNestingDepth-1, and a line with a deeper level is rejected.
const MaxNestingDepth = 100

// Parser parses GEDCOM files into Line structures.
type Parser struct {
	lineNumber int
	lastLevel  int
}

// ParseOptions configures the behavior of ParseWithOptions.
type ParseOptions struct {
	// Lenient controls error handling behavior.
	// If true, the parser collects errors and continues parsing.
	// If false (default), the parser fails on the first error.
	Lenient bool

	// MaxErrors is the maximum number of errors to collect in lenient mode.
	// When reached, parsing continues but errors are no longer collected.
	// A value of 0 means unlimited errors will be collected.
	MaxErrors int
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		lineNumber: 0,
		lastLevel:  -1,
	}
}

// Reset resets the parser state for reuse.
func (p *Parser) Reset() {
	p.lineNumber = 0
	p.lastLevel = -1
}

// ParseLine parses a single GEDCOM line.
// GEDCOM line format: LEVEL [XREF] TAG [VALUE]
// Examples:
//
//	0 HEAD
//	0 @I1@ INDI
//	1 NAME John /Smith/
//	2 GIVN John
//
// An XRef containing a space (e.g. "0 @NoTe ref@ NOTE text") violates the
// GEDCOM grammar. Such a line is reported as an error, but the recovered Line
// is returned alongside it so lenient callers can keep the record instead of
// dropping it; see [parseSpacedXRef]. A returned Line is therefore not proof
// that the line was well formed — check the error too.
func (p *Parser) ParseLine(input string) (*Line, error) {
	p.lineNumber++

	// Trim line endings (CRLF, LF, CR)
	line := strings.TrimRight(input, "\r\n")

	// Empty or whitespace-only lines are invalid
	if strings.TrimSpace(line) == "" {
		return nil, newParseError(p.lineNumber, "empty line", input)
	}

	// Split into parts
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, newParseError(p.lineNumber, "line must have at least level and tag", line)
	}

	// Parse level (first part)
	level, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, wrapParseError(p.lineNumber, "invalid level number", line, err)
	}

	if level < 0 {
		return nil, newParseError(p.lineNumber, "level cannot be negative", line)
	}

	// Check nesting depth. The GEDCOM level field is at most two digits, so
	// level 99 (MaxNestingDepth-1) is the deepest the grammar allows.
	if level >= MaxNestingDepth {
		return nil, newParseError(p.lineNumber,
			fmt.Sprintf("level %d exceeds maximum nesting depth (valid levels are 0-%d)",
				level, MaxNestingDepth-1),
			line)
	}

	// Parse XRef and Tag
	var xref, tag string
	var valueStartIdx int

	// Check if second part is an XRef (starts with @ and ends with @)
	if strings.HasPrefix(parts[1], "@") && strings.HasSuffix(parts[1], "@") {
		xref = parts[1]
		if len(parts) < 3 {
			return nil, newParseError(p.lineNumber, "line with xref must have a tag", line)
		}
		tag = parts[2]
		valueStartIdx = 3
	} else if id, rest, ok := splitSpacedXRef(parts[1], line); ok {
		return p.parseSpacedXRef(level, line, id, rest)
	} else {
		tag = parts[1]
		valueStartIdx = 2
	}

	// Parse value (everything after the tag)
	var value string
	if valueStartIdx < len(parts) {
		// Find the position in the original line where the value starts
		// We need to preserve original spacing in the value
		tagPos := strings.Index(line, tag)
		if tagPos >= 0 {
			afterTag := tagPos + len(tag)
			if afterTag < len(line) {
				value = strings.TrimLeft(line[afterTag:], " ")
			}
		}
	}

	return &Line{
		Level:      level,
		Tag:        tag,
		Value:      value,
		XRef:       xref,
		LineNumber: p.lineNumber,
	}, nil
}

// splitSpacedXRef detects an XRef identifier containing a space, e.g.
// "0 @NoTe ref@ NOTE mixed case and space", which strings.Fields splits across
// two or more fields. It returns the identifier and the rest of the line.
//
// ok is false unless the field in the XRef position opens with "@", a later
// "@" closes the identifier at a field boundary, and the identifier really
// does contain a space. Every other shape keeps its existing parse (the field
// becomes the tag): requiring a space keeps well-formed identifiers such as
// "0 @I1@INDI" on that path, and requiring a field boundary keeps a pointer in
// the value ("1 @I1 NOTE see @P1@") from posing as the closing "@".
func splitSpacedXRef(xrefField, line string) (xref, rest string, ok bool) {
	if !strings.HasPrefix(xrefField, "@") {
		return "", "", false
	}

	// The level is numeric, so the first "@" in the line opens the XRef.
	open := strings.Index(line, "@")
	closeOffset := strings.Index(line[open+1:], "@")
	if closeOffset < 0 {
		return "", "", false
	}
	closeIdx := open + 1 + closeOffset

	xref = line[open : closeIdx+1]
	if !strings.Contains(xref, " ") {
		return "", "", false
	}

	rest = line[closeIdx+1:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", "", false
	}
	return xref, rest, true
}

// parseSpacedXRef parses a line whose XRef contains a space, as located by
// [splitSpacedXRef]. The GEDCOM grammar forbids spaces inside an XRef, so the
// line is always reported as an error: strict callers reject the file, lenient
// callers record an INVALID_XREF diagnostic.
//
// The recovered Line is returned alongside the error so lenient callers can
// keep the record. Dropping the line instead would lose the record and
// silently reparent its subordinate lines onto the preceding record.
//
// The identifier is kept verbatim, spaces included, so nothing is lost; note
// that re-encoding such a document reproduces a line this parser rejects in
// strict mode.
func (p *Parser) parseSpacedXRef(level int, line, xref, rest string) (*Line, error) {
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return nil, newParseError(p.lineNumber, "line with xref must have a tag", line)
	}

	// rest holds only whitespace before the tag, so this finds the tag itself.
	tag := fields[0]
	afterTag := strings.Index(rest, tag) + len(tag)
	value := strings.TrimLeft(rest[afterTag:], " ")

	recovered := &Line{
		Level:      level,
		Tag:        tag,
		Value:      value,
		XRef:       xref,
		LineNumber: p.lineNumber,
	}
	return recovered, newParseError(p.lineNumber, "xref contains a space: "+xref, line)
}

// Parse reads a GEDCOM file from a reader and returns all parsed lines.
// Supports all line ending styles: LF (Unix), CRLF (Windows), CR (old Macintosh).
//
// If r fails with an error exposing ErrorLine() int, that line is reported as
// ParseError.Line in preference to the parser's own counter.
func (p *Parser) Parse(r io.Reader) ([]*Line, error) {
	p.Reset()

	scanner := bufio.NewScanner(r)
	// Use custom split function that handles CR, LF, and CRLF line endings
	scanner.Split(scanGEDCOMLines)
	var lines []*Line

	for scanner.Scan() {
		text := scanner.Text()
		line, err := p.ParseLine(text)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, wrapParseError(readErrorLine(err, p.lineNumber), "error reading input", "", err)
	}

	return lines, nil
}

// ParseWithOptions reads a GEDCOM file with configurable error handling.
// In lenient mode, it collects parse errors and continues parsing.
// Returns:
//   - lines: parsed lines, including any line recovered from a reported error
//     (see [Parser.ParseLine]); may be partial in lenient mode
//   - parseErrors: syntax errors encountered (only populated in lenient mode);
//     an error here does not imply its line is missing from lines
//   - fatalErr: unrecoverable errors like I/O failures
//
// If r fails with an error exposing ErrorLine() int, that line is reported as
// ParseError.Line in preference to the parser's own counter.
func (p *Parser) ParseWithOptions(r io.Reader, opts *ParseOptions) (
	lines []*Line,
	parseErrors []*ParseError,
	fatalErr error,
) {
	p.Reset()

	// Handle nil options - default to strict mode
	if opts == nil {
		opts = &ParseOptions{}
	}
	// Normalize negative MaxErrors to unlimited (0)
	if opts.MaxErrors < 0 {
		opts.MaxErrors = 0
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(scanGEDCOMLines)

	for scanner.Scan() {
		text := scanner.Text()
		line, err := p.ParseLine(text)
		if err != nil {
			if !opts.Lenient {
				// Strict mode: fail on first error
				return nil, nil, err
			}

			// Lenient mode: collect the error and continue
			var parseErr *ParseError
			if pe, ok := err.(*ParseError); ok {
				parseErr = pe
			} else {
				// Wrap non-ParseError errors
				parseErr = &ParseError{
					Line:    p.lineNumber,
					Message: err.Error(),
					Context: text,
					Err:     err,
				}
			}

			// Only collect if under MaxErrors limit (0 = unlimited)
			if opts.MaxErrors == 0 || len(parseErrors) < opts.MaxErrors {
				parseErrors = append(parseErrors, parseErr)
			}
			// A recovered line (e.g. an XRef containing a space) is kept so the
			// record survives; otherwise the problematic line is skipped.
			if line != nil {
				lines = append(lines, line)
			}
			continue
		}
		lines = append(lines, line)
	}

	// Scanner errors are I/O errors - always fatal
	if err := scanner.Err(); err != nil {
		fatalErr = wrapParseError(readErrorLine(err, p.lineNumber), "error reading input", "", err)
		return lines, parseErrors, fatalErr
	}

	return lines, parseErrors, nil
}

// scanGEDCOMLines is a split function for bufio.Scanner that handles
// all GEDCOM line ending styles: LF, CRLF, and CR (old Macintosh).
// This is based on bufio.ScanLines but adds CR-only support.
func scanGEDCOMLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for CR or LF
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			// Found LF - this could be standalone or part of CRLF
			return i + 1, data[0:i], nil
		}
		if data[i] == '\r' {
			// Found CR - check if followed by LF (CRLF)
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					// CRLF - return line without either terminator
					return i + 2, data[0:i], nil
				}
				// CR alone - return line
				return i + 1, data[0:i], nil
			}
			// CR at end of data - need more data to determine if CRLF
			if !atEOF {
				return 0, nil, nil
			}
			// At EOF with CR - treat as line ending
			return i + 1, data[0:i], nil
		}
	}

	// If we're at EOF, return remaining data as final line
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}
