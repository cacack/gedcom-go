package encoder

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// encodeState represents the current state of the streaming encoder.
type encodeState int

const (
	stateInitial        encodeState = iota // Initial state, waiting for WriteHeader
	stateHeaderWritten                     // Header has been written, can write records or trailer
	stateRecordsWritten                    // At least one record has been written, can write more records or trailer
	stateComplete                          // Trailer has been written, encoding is complete
)

// String returns a human-readable name for the encode state.
func (s encodeState) String() string {
	switch s {
	case stateInitial:
		return "Initial"
	case stateHeaderWritten:
		return "HeaderWritten"
	case stateRecordsWritten:
		return "RecordsWritten"
	case stateComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// StreamEncoder provides a streaming interface for writing GEDCOM documents.
// It allows writing records one at a time with constant memory usage,
// enabling generation of very large GEDCOM files without loading the entire
// document into memory first.
//
// The encoder enforces valid GEDCOM structure through a state machine.
// The canonical call sequence is:
//
//	WriteHeader → WriteRecord (zero or more times) → WriteTrailer → Close
//
// Each step is required (except WriteRecord, which may be skipped). Close
// flushes any buffered data and verifies that WriteTrailer was called; if
// the trailer is missing, Close returns [ErrTrailerNotWritten] and the
// output file will be structurally invalid.
//
// Flush is optional and only useful for intermediate checkpointing (e.g.,
// forcing partial output visible to a downstream consumer). Close calls
// Flush internally, so an explicit Flush before Close is redundant.
//
// IMPORTANT: capture Close's error. Using `defer enc.Close()` silently
// discards [ErrTrailerNotWritten] and any flush failure — both of which
// indicate the output file is incomplete or corrupt. Prefer:
//
//	func writeStreamed(w io.Writer, records []*gedcom.Record) (err error) {
//	    enc := encoder.NewStreamEncoder(w)
//	    defer func() {
//	        if cerr := enc.Close(); cerr != nil && err == nil {
//	            err = cerr
//	        }
//	    }()
//	    if err := enc.WriteHeader(header); err != nil {
//	        return err
//	    }
//	    for _, rec := range records {
//	        if err := enc.WriteRecord(rec); err != nil {
//	            return err
//	        }
//	    }
//	    return enc.WriteTrailer()
//	}
type StreamEncoder struct {
	writer  *bufio.Writer
	options *EncodeOptions
	state   encodeState
	err     error // sticky error for early exit
}

// Errors returned by StreamEncoder for invalid state transitions.
var (
	ErrHeaderNotWritten      = errors.New("header must be written before writing records")
	ErrHeaderAlreadyWritten  = errors.New("header has already been written")
	ErrTrailerNotWritten     = errors.New("trailer has not been written")
	ErrTrailerAlreadyWritten = errors.New("trailer has already been written")
	ErrEncodingComplete      = errors.New("encoding is complete, no further writes allowed")
)

// NewStreamEncoder creates a new StreamEncoder that writes to w.
// It uses default encoding options (LF line endings, default max line length).
func NewStreamEncoder(w io.Writer) *StreamEncoder {
	return NewStreamEncoderWithOptions(w, DefaultOptions())
}

// NewStreamEncoderWithOptions creates a new StreamEncoder with custom options.
// If opts is nil, default options are used.
func NewStreamEncoderWithOptions(w io.Writer, opts *EncodeOptions) *StreamEncoder {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &StreamEncoder{
		writer:  bufio.NewWriter(w),
		options: opts,
		state:   stateInitial,
	}
}

// WriteHeader writes the GEDCOM header. This must be the first method called
// on the encoder and can only be called once.
//
// Returns ErrHeaderAlreadyWritten if the header has already been written,
// or ErrEncodingComplete if the encoding is already complete.
func (e *StreamEncoder) WriteHeader(h *gedcom.Header) error {
	if e.err != nil {
		return e.err
	}

	switch e.state {
	case stateInitial:
		// Valid state, proceed
	case stateHeaderWritten, stateRecordsWritten:
		return ErrHeaderAlreadyWritten
	case stateComplete:
		return ErrEncodingComplete
	}

	if err := writeHeader(e.writer, h, e.options); err != nil {
		e.err = err
		return err
	}

	e.state = stateHeaderWritten
	return nil
}

// WriteRecord writes a single GEDCOM record. WriteHeader must have been called
// before calling this method. This method can be called multiple times to write
// multiple records.
//
// Returns ErrHeaderNotWritten if the header has not been written,
// or ErrEncodingComplete if the encoding is already complete.
func (e *StreamEncoder) WriteRecord(r *gedcom.Record) error {
	if e.err != nil {
		return e.err
	}

	switch e.state {
	case stateInitial:
		return ErrHeaderNotWritten
	case stateHeaderWritten, stateRecordsWritten:
		// Valid states, proceed
	case stateComplete:
		return ErrEncodingComplete
	}

	if err := writeRecord(e.writer, r, e.options); err != nil {
		e.err = err
		return err
	}

	e.state = stateRecordsWritten
	return nil
}

// WriteTrailer writes the GEDCOM trailer (0 TRLR) to complete the document.
// This must be called after WriteHeader and optionally after WriteRecord calls.
//
// Returns ErrHeaderNotWritten if the header has not been written,
// or ErrTrailerAlreadyWritten if the trailer has already been written.
func (e *StreamEncoder) WriteTrailer() error {
	if e.err != nil {
		return e.err
	}

	switch e.state {
	case stateInitial:
		return ErrHeaderNotWritten
	case stateHeaderWritten, stateRecordsWritten:
		// Valid states, proceed
	case stateComplete:
		return ErrTrailerAlreadyWritten
	}

	if err := writeTrailer(e.writer, e.options); err != nil {
		e.err = err
		return err
	}

	e.state = stateComplete
	return nil
}

// Flush flushes any buffered data to the underlying writer.
// This can be called at any time to ensure data is written.
func (e *StreamEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	if err := e.writer.Flush(); err != nil {
		e.err = err
		return err
	}
	return nil
}

// Close flushes any buffered data and marks the encoder as complete.
// If the trailer has not been written, it returns [ErrTrailerNotWritten]
// but still flushes any buffered data (yielding a structurally invalid
// file). Always check Close's return value rather than using a bare
// `defer enc.Close()`, which silently drops the error.
//
// After Close is called, no further writes are allowed.
func (e *StreamEncoder) Close() error {
	// Always flush, even if there's an error
	flushErr := e.writer.Flush()

	// If we already have a sticky error, return it
	if e.err != nil {
		return e.err
	}

	// If flush failed, record and return it
	if flushErr != nil {
		e.err = flushErr
		return flushErr
	}

	// Check if trailer was written
	if e.state != stateComplete {
		e.err = ErrTrailerNotWritten
		return ErrTrailerNotWritten
	}

	return nil
}

// State returns the current state of the encoder.
// This is primarily useful for testing and debugging.
func (e *StreamEncoder) State() string {
	return e.state.String()
}

// Err returns any error that occurred during encoding.
// Once an error occurs, the encoder stops accepting further writes.
func (e *StreamEncoder) Err() error {
	return e.err
}

// EncodeStreaming is a convenience function that streams a complete document.
// It's equivalent to calling WriteHeader, WriteRecord for each record, and WriteTrailer.
// This function exists mainly for API symmetry with the batch Encode function.
func EncodeStreaming(w io.Writer, doc *gedcom.Document) error {
	return EncodeStreamingWithOptions(w, doc, DefaultOptions())
}

// EncodeStreamingWithOptions is like EncodeStreaming but with custom options.
func EncodeStreamingWithOptions(w io.Writer, doc *gedcom.Document, opts *EncodeOptions) error {
	enc := NewStreamEncoderWithOptions(w, opts)

	if err := enc.WriteHeader(doc.Header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, record := range doc.Records {
		if err := enc.WriteRecord(record); err != nil {
			return fmt.Errorf("write record %d: %w", i, err)
		}
	}

	if err := enc.WriteTrailer(); err != nil {
		return fmt.Errorf("write trailer: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}
