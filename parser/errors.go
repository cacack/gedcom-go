package parser

import (
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v3/internal/clip"
)

// MaxContextBytes bounds the input a [ParseError] echoes back (#744). Its
// Context, and its Message, which can quote part of the line (an XRef, say),
// each hold at most MaxContextBytes bytes of the text, cut back to the start
// of a UTF-8 character and followed by "…" when longer. The full line is not
// retained, so a malformed line near [MaxLineBytes] yields an error string of
// a few hundred bytes rather than one the size of the line.
const MaxContextBytes = 256

// ParseError represents an error that occurred during parsing.
// It includes line number and context for better error reporting.
type ParseError struct {
	// Line is the line number where the error occurred (1-based)
	Line int

	// Message describes what went wrong, clipped to MaxContextBytes.
	Message string

	// Context provides the actual line content that caused the error, clipped
	// to MaxContextBytes.
	Context string

	// Err is the underlying error, if any
	Err error
}

func (e *ParseError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("line %d: %s (context: %q)", e.Line, e.Message, e.Context)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// newParseError creates a new ParseError with the given details.
func newParseError(line int, message, context string) error {
	return wrapParseError(line, message, context, nil)
}

// lineLocatedError is implemented by reader errors that know the physical line
// of the byte that failed, such as charset.InvalidUTF8Error.
type lineLocatedError interface {
	ErrorLine() int
}

// readErrorLine picks the line to report for a reader failure. The parser's own
// counter only reaches the last line the reader handed over, so it lags whenever
// a read is rejected part-way through a chunk it had already consumed; an error
// that carries its own physical line is authoritative and wins.
func readErrorLine(err error, fallback int) int {
	var located lineLocatedError
	if errors.As(err, &located) {
		if line := located.ErrorLine(); line > 0 {
			return line
		}
	}
	return fallback
}

// wrapParseError wraps an existing error with parse context. Every
// ParseError the parser builds goes through it, so every one is clipped to
// MaxContextBytes.
func wrapParseError(line int, message, context string, err error) *ParseError {
	return &ParseError{
		Line:    line,
		Message: clip.String(message, MaxContextBytes),
		Context: clip.String(context, MaxContextBytes),
		Err:     err,
	}
}
