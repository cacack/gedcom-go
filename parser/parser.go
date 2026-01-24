package parser

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// MaxNestingDepth is the maximum allowed nesting depth to prevent stack overflow.
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

	// Check nesting depth
	if level > MaxNestingDepth {
		return nil, newParseError(p.lineNumber, "maximum nesting depth exceeded", line)
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

// Parse reads a GEDCOM file from a reader and returns all parsed lines.
// Supports all line ending styles: LF (Unix), CRLF (Windows), CR (old Macintosh).
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
		return nil, wrapParseError(p.lineNumber, "error reading input", "", err)
	}

	return lines, nil
}

// ParseWithOptions reads a GEDCOM file with configurable error handling.
// In lenient mode, it collects parse errors and continues parsing.
// Returns:
//   - lines: successfully parsed lines (may be partial in lenient mode)
//   - parseErrors: syntax errors encountered (only populated in lenient mode)
//   - fatalErr: unrecoverable errors like I/O failures
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
			// Skip the problematic line and continue parsing
			continue
		}
		lines = append(lines, line)
	}

	// Scanner errors are I/O errors - always fatal
	if err := scanner.Err(); err != nil {
		fatalErr = wrapParseError(p.lineNumber, "error reading input", "", err)
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
