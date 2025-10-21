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
func (p *Parser) Parse(r io.Reader) ([]*Line, error) {
	p.Reset()

	scanner := bufio.NewScanner(r)
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
