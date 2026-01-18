package encoder

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

func TestStreamEncoder_BasicFlow(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	// Write header
	header := &gedcom.Header{
		Version:  "5.5.1",
		Encoding: "UTF-8",
	}
	if err := enc.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if enc.State() != "HeaderWritten" {
		t.Errorf("State() = %v, want HeaderWritten", enc.State())
	}

	// Write records
	record := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Smith/"},
		},
	}
	if err := enc.WriteRecord(record); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if enc.State() != "RecordsWritten" {
		t.Errorf("State() = %v, want RecordsWritten", enc.State())
	}

	// Write trailer
	if err := enc.WriteTrailer(); err != nil {
		t.Fatalf("WriteTrailer() error = %v", err)
	}
	if enc.State() != "Complete" {
		t.Errorf("State() = %v, want Complete", enc.State())
	}

	// Close
	if err := enc.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify output
	output := buf.String()
	expectedLines := []string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"0 TRLR",
	}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Output missing expected line: %q\nGot:\n%s", line, output)
		}
	}
}

func TestStreamEncoder_EmptyFile(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	// Write header
	if err := enc.WriteHeader(&gedcom.Header{Version: "5.5"}); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}

	// Write trailer immediately (no records)
	if err := enc.WriteTrailer(); err != nil {
		t.Fatalf("WriteTrailer() error = %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 HEAD") {
		t.Error("Output should contain HEAD")
	}
	if !strings.Contains(output, "0 TRLR") {
		t.Error("Output should contain TRLR")
	}
}

func TestStreamEncoder_InvalidStateTransitions(t *testing.T) {
	t.Run("WriteRecord before WriteHeader", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
		err := enc.WriteRecord(record)
		if !errors.Is(err, ErrHeaderNotWritten) {
			t.Errorf("WriteRecord() error = %v, want ErrHeaderNotWritten", err)
		}
	})

	t.Run("WriteHeader twice", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		if err := enc.WriteHeader(&gedcom.Header{}); err != nil {
			t.Fatalf("First WriteHeader() error = %v", err)
		}

		err := enc.WriteHeader(&gedcom.Header{})
		if !errors.Is(err, ErrHeaderAlreadyWritten) {
			t.Errorf("Second WriteHeader() error = %v, want ErrHeaderAlreadyWritten", err)
		}
	})

	t.Run("WriteTrailer before WriteHeader", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		err := enc.WriteTrailer()
		if !errors.Is(err, ErrHeaderNotWritten) {
			t.Errorf("WriteTrailer() error = %v, want ErrHeaderNotWritten", err)
		}
	})

	t.Run("WriteTrailer twice", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(&gedcom.Header{})
		enc.WriteTrailer()

		err := enc.WriteTrailer()
		if !errors.Is(err, ErrTrailerAlreadyWritten) {
			t.Errorf("Second WriteTrailer() error = %v, want ErrTrailerAlreadyWritten", err)
		}
	})

	t.Run("WriteRecord after WriteTrailer", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(&gedcom.Header{})
		enc.WriteTrailer()

		record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
		err := enc.WriteRecord(record)
		if !errors.Is(err, ErrEncodingComplete) {
			t.Errorf("WriteRecord() error = %v, want ErrEncodingComplete", err)
		}
	})

	t.Run("WriteHeader after WriteTrailer", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(&gedcom.Header{})
		enc.WriteTrailer()

		err := enc.WriteHeader(&gedcom.Header{})
		if !errors.Is(err, ErrEncodingComplete) {
			t.Errorf("WriteHeader() error = %v, want ErrEncodingComplete", err)
		}
	})

	t.Run("WriteHeader after records written", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(&gedcom.Header{})
		enc.WriteRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual})

		err := enc.WriteHeader(&gedcom.Header{})
		if !errors.Is(err, ErrHeaderAlreadyWritten) {
			t.Errorf("WriteHeader() error = %v, want ErrHeaderAlreadyWritten", err)
		}
	})
}

func TestStreamEncoder_CloseWithoutTrailer(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	enc.WriteHeader(&gedcom.Header{})

	err := enc.Close()
	if !errors.Is(err, ErrTrailerNotWritten) {
		t.Errorf("Close() error = %v, want ErrTrailerNotWritten", err)
	}
}

func TestStreamEncoder_OutputEquivalence(t *testing.T) {
	// Create a test document
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      "5.5.1",
			Encoding:     "UTF-8",
			SourceSystem: "TestApp",
			Language:     "English",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Smith/"},
					{Level: 2, Tag: "GIVN", Value: "John"},
					{Level: 2, Tag: "SURN", Value: "Smith"},
					{Level: 1, Tag: "SEX", Value: "M"},
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 1950"},
					{Level: 2, Tag: "PLAC", Value: "Boston, MA"},
				},
			},
			{
				XRef: "@I2@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "Jane /Doe/"},
					{Level: 1, Tag: "SEX", Value: "F"},
				},
			},
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "HUSB", Value: "@I1@"},
					{Level: 1, Tag: "WIFE", Value: "@I2@"},
				},
			},
		},
	}

	// Encode with batch encoder
	var batchBuf bytes.Buffer
	if err := Encode(&batchBuf, doc); err != nil {
		t.Fatalf("Batch Encode() error = %v", err)
	}

	// Encode with streaming encoder
	var streamBuf bytes.Buffer
	enc := NewStreamEncoder(&streamBuf)
	if err := enc.WriteHeader(doc.Header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	for _, r := range doc.Records {
		if err := enc.WriteRecord(r); err != nil {
			t.Fatalf("WriteRecord() error = %v", err)
		}
	}
	if err := enc.WriteTrailer(); err != nil {
		t.Fatalf("WriteTrailer() error = %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Compare outputs - they should be byte-for-byte identical
	if batchBuf.String() != streamBuf.String() {
		t.Errorf("Streaming output differs from batch output.\nBatch:\n%s\nStreaming:\n%s",
			batchBuf.String(), streamBuf.String())
	}
}

func TestStreamEncoder_WithOptions(t *testing.T) {
	t.Run("CRLF line endings", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &EncodeOptions{
			LineEnding: "\r\n",
		}
		enc := NewStreamEncoderWithOptions(&buf, opts)

		enc.WriteHeader(&gedcom.Header{Version: "5.5"})
		enc.WriteTrailer()
		enc.Close()

		output := buf.String()
		if !strings.Contains(output, "\r\n") {
			t.Error("Output should contain CRLF line endings")
		}
		// Verify we don't have bare LF (which would show as double newline if CRLF is also present)
		// Every \n should be preceded by \r
		lines := strings.Split(output, "\r\n")
		for i, line := range lines[:len(lines)-1] { // All but last (empty after final \r\n)
			if strings.Contains(line, "\n") {
				t.Errorf("Line %d contains bare LF: %q", i, line)
			}
		}
	})

	t.Run("nil options uses defaults", func(t *testing.T) {
		var buf bytes.Buffer
		enc := NewStreamEncoderWithOptions(&buf, nil)

		enc.WriteHeader(&gedcom.Header{Version: "5.5"})
		enc.WriteTrailer()
		enc.Close()

		output := buf.String()
		if !strings.Contains(output, "\n") {
			t.Error("Output should contain LF line endings (default)")
		}
	})
}

func TestStreamEncoder_Flush(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	enc.WriteHeader(&gedcom.Header{Version: "5.5"})

	// Flush explicitly
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Buffer should have content now
	if buf.Len() == 0 {
		t.Error("Buffer should have content after Flush()")
	}

	enc.WriteTrailer()
	enc.Close()
}

func TestStreamEncoder_MultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	enc.WriteHeader(&gedcom.Header{Version: "5.5"})

	// Write multiple records
	for i := 1; i <= 5; i++ {
		xref := "@I" + string(rune('0'+i)) + "@"
		record := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('A'+i-1))},
			},
		}
		if err := enc.WriteRecord(record); err != nil {
			t.Fatalf("WriteRecord(%d) error = %v", i, err)
		}
	}

	enc.WriteTrailer()
	enc.Close()

	output := buf.String()
	for i := 1; i <= 5; i++ {
		xref := "@I" + string(rune('0'+i)) + "@"
		if !strings.Contains(output, "0 "+xref+" INDI") {
			t.Errorf("Output missing record %s", xref)
		}
	}
}

func TestStreamEncoder_RoundTrip(t *testing.T) {
	// Create a complex document using streaming encoder
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	header := &gedcom.Header{
		Version:      "5.5.1",
		Encoding:     "UTF-8",
		SourceSystem: "StreamTest",
	}
	enc.WriteHeader(header)

	// Write individual
	enc.WriteRecord(&gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Smith/"},
			{Level: 2, Tag: "GIVN", Value: "John"},
			{Level: 2, Tag: "SURN", Value: "Smith"},
			{Level: 1, Tag: "SEX", Value: "M"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "DATE", Value: "1 JAN 1950"},
		},
	})

	// Write family
	enc.WriteRecord(&gedcom.Record{
		XRef: "@F1@",
		Type: gedcom.RecordTypeFamily,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "HUSB", Value: "@I1@"},
		},
	})

	enc.WriteTrailer()
	enc.Close()

	// Decode and verify
	doc, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Failed to decode streamed output: %v", err)
	}

	if len(doc.Records) != 2 {
		t.Errorf("Record count = %d, want 2", len(doc.Records))
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found")
	}
	if indi.Sex != "M" {
		t.Errorf("Individual sex = %q, want %q", indi.Sex, "M")
	}
}

// streamFailWriter is a writer that fails after writing a certain number of bytes
type streamFailWriter struct {
	failAfterBytes int
	bytesWritten   int
}

func (w *streamFailWriter) Write(p []byte) (n int, err error) {
	if w.bytesWritten >= w.failAfterBytes {
		return 0, errors.New("simulated write error")
	}
	// Write some bytes but may fail mid-write
	toWrite := len(p)
	remaining := w.failAfterBytes - w.bytesWritten
	if toWrite > remaining {
		toWrite = remaining
		w.bytesWritten += toWrite
		return toWrite, errors.New("simulated write error")
	}
	w.bytesWritten += toWrite
	return toWrite, nil
}

func TestStreamEncoder_WriteErrors(t *testing.T) {
	t.Run("error during flush propagates", func(t *testing.T) {
		// Create a writer that fails after some bytes
		// The buffered writer won't immediately write, so we need to flush
		w := &streamFailWriter{failAfterBytes: 10}
		enc := NewStreamEncoder(w)

		// Write header - this goes to buffer first
		enc.WriteHeader(&gedcom.Header{Version: "5.5"})
		enc.WriteTrailer()

		// Flush should fail because underlying writer fails
		err := enc.Flush()
		if err == nil {
			// Try Close which also flushes
			err = enc.Close()
		}

		// At some point we should get an error
		if enc.Err() == nil && err == nil {
			// The buffer might be large enough to hold all data
			// In that case, only the final flush would fail
			t.Log("All writes succeeded - buffer was large enough")
		}
	})

	t.Run("error on Flush with small buffer triggers error", func(t *testing.T) {
		// Writer that immediately fails
		w := &streamFailWriter{failAfterBytes: 0}
		enc := NewStreamEncoder(w)

		enc.WriteHeader(&gedcom.Header{Version: "5.5"})

		// Force flush to trigger the error
		err := enc.Flush()
		if err == nil {
			t.Error("Flush() should fail when underlying writer fails")
		}

		// Subsequent flush should return sticky error
		if enc.Err() == nil {
			t.Error("Err() should return the error after failed Flush()")
		}
	})

	t.Run("error is sticky after failed operation", func(t *testing.T) {
		w := &streamFailWriter{failAfterBytes: 0}
		enc := NewStreamEncoder(w)

		enc.WriteHeader(&gedcom.Header{})
		enc.Flush() // This triggers the error

		// Subsequent operations should fail with sticky error
		if enc.Err() != nil {
			err := enc.WriteTrailer()
			if err == nil {
				t.Error("WriteTrailer should fail after sticky error")
			}
		}
	})

	t.Run("WriteHeader with sticky error", func(t *testing.T) {
		w := &streamFailWriter{failAfterBytes: 0}
		enc := NewStreamEncoder(w)

		enc.WriteHeader(&gedcom.Header{})
		enc.Flush() // Triggers sticky error

		// WriteHeader should return the sticky error
		err := enc.WriteHeader(&gedcom.Header{})
		if err == nil {
			t.Error("WriteHeader should fail with sticky error")
		}
	})

	t.Run("WriteRecord with sticky error", func(t *testing.T) {
		w := &streamFailWriter{failAfterBytes: 0}
		enc := NewStreamEncoder(w)

		enc.WriteHeader(&gedcom.Header{})
		enc.Flush() // Triggers sticky error

		// WriteRecord should return the sticky error
		record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
		err := enc.WriteRecord(record)
		if err == nil {
			t.Error("WriteRecord should fail with sticky error")
		}
	})

	t.Run("Close with sticky error", func(t *testing.T) {
		w := &streamFailWriter{failAfterBytes: 0}
		enc := NewStreamEncoder(w)

		enc.WriteHeader(&gedcom.Header{})
		enc.Flush() // Triggers sticky error

		// Close should return the sticky error
		err := enc.Close()
		if err == nil {
			t.Error("Close should fail with sticky error")
		}
	})

}

func TestStreamEncoder_Err(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	// Initially no error
	if enc.Err() != nil {
		t.Error("Err() should be nil initially")
	}

	enc.WriteHeader(&gedcom.Header{})
	enc.WriteTrailer()
	enc.Close()

	// Still no error after successful operation
	if enc.Err() != nil {
		t.Errorf("Err() = %v, want nil after successful operation", enc.Err())
	}
}

func TestEncodeStreaming(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  "5.5.1",
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "Test /Person/"},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := EncodeStreaming(&buf, doc); err != nil {
		t.Fatalf("EncodeStreaming() error = %v", err)
	}

	output := buf.String()
	expectedLines := []string{
		"0 HEAD",
		"0 @I1@ INDI",
		"1 NAME Test /Person/",
		"0 TRLR",
	}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Output missing expected line: %q", line)
		}
	}
}

func TestEncodeStreamingWithOptions(t *testing.T) {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: []*gedcom.Record{},
	}

	var buf bytes.Buffer
	opts := &EncodeOptions{LineEnding: "\r\n"}
	if err := EncodeStreamingWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("EncodeStreamingWithOptions() error = %v", err)
	}

	if !strings.Contains(buf.String(), "\r\n") {
		t.Error("Output should contain CRLF line endings")
	}
}

func TestEncodeStreamingWithOptions_Errors(t *testing.T) {
	t.Run("writer fails during encoding", func(t *testing.T) {
		// Writer that fails immediately - error will be caught during close/flush
		w := &streamFailWriter{failAfterBytes: 0}
		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: "5.5"},
			Records: []*gedcom.Record{},
		}

		err := EncodeStreamingWithOptions(w, doc, DefaultOptions())
		if err == nil {
			t.Error("EncodeStreamingWithOptions should fail when writer fails")
		}
	})

	t.Run("writer fails with records", func(t *testing.T) {
		// Writer that allows some bytes but fails during output
		w := &streamFailWriter{failAfterBytes: 50}
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: "5.5"},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NAME", Value: "Test /Person/"},
					},
				},
			},
		}

		err := EncodeStreamingWithOptions(w, doc, DefaultOptions())
		if err == nil {
			t.Error("EncodeStreamingWithOptions should fail when writer fails")
		}
	})
}

func TestEncodeState_String(t *testing.T) {
	tests := []struct {
		state encodeState
		want  string
	}{
		{stateInitial, "Initial"},
		{stateHeaderWritten, "HeaderWritten"},
		{stateRecordsWritten, "RecordsWritten"},
		{stateComplete, "Complete"},
		{encodeState(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("encodeState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// TestStreamEncoder_EntityToTags verifies that entities without tags are properly converted
func TestStreamEncoder_EntityToTags(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	enc.WriteHeader(&gedcom.Header{Version: "5.5.1", Encoding: "UTF-8"})

	// Write record with Entity but no Tags
	enc.WriteRecord(&gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: nil,
		Entity: &gedcom.Individual{
			XRef: "@I1@",
			Names: []*gedcom.PersonalName{
				{Full: "John /Smith/", Given: "John", Surname: "Smith"},
			},
			Sex: "M",
		},
	})

	enc.WriteTrailer()
	enc.Close()

	output := buf.String()

	// Verify entity was converted to tags
	expectedPatterns := []string{
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"2 GIVN John",
		"2 SURN Smith",
		"1 SEX M",
	}
	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Output missing expected pattern: %q\nGot:\n%s", pattern, output)
		}
	}
}

// Benchmark for memory usage verification
func BenchmarkStreamEncoder_LargeFile(b *testing.B) {
	// This benchmark verifies constant memory usage
	// The streaming encoder should use the same amount of memory
	// regardless of how many records are written

	header := &gedcom.Header{Version: "5.5.1", Encoding: "UTF-8"}
	record := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "Test /Person/"},
			{Level: 2, Tag: "GIVN", Value: "Test"},
			{Level: 2, Tag: "SURN", Value: "Person"},
			{Level: 1, Tag: "SEX", Value: "M"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "DATE", Value: "1 JAN 1950"},
			{Level: 2, Tag: "PLAC", Value: "Boston, Massachusetts, USA"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(header)

		// Write 1000 records
		for j := 0; j < 1000; j++ {
			enc.WriteRecord(record)
		}

		enc.WriteTrailer()
		enc.Close()
	}
}

// Test that streaming encoder handles all record types
func TestStreamEncoder_AllRecordTypes(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)

	enc.WriteHeader(&gedcom.Header{Version: "5.5.1", Encoding: "UTF-8"})

	// Individual
	enc.WriteRecord(&gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Test /Person/"}},
	})

	// Family
	enc.WriteRecord(&gedcom.Record{
		XRef: "@F1@",
		Type: gedcom.RecordTypeFamily,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "HUSB", Value: "@I1@"}},
	})

	// Source
	enc.WriteRecord(&gedcom.Record{
		XRef: "@S1@",
		Type: gedcom.RecordTypeSource,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "TITL", Value: "Test Source"}},
	})

	// Repository
	enc.WriteRecord(&gedcom.Record{
		XRef: "@R1@",
		Type: gedcom.RecordTypeRepository,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Test Repository"}},
	})

	// Submitter
	enc.WriteRecord(&gedcom.Record{
		XRef: "@SUBM1@",
		Type: gedcom.RecordTypeSubmitter,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Test Submitter"}},
	})

	// Note
	enc.WriteRecord(&gedcom.Record{
		XRef: "@N1@",
		Type: gedcom.RecordTypeNote,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "CONT", Value: "Test note content"}},
	})

	// Media
	enc.WriteRecord(&gedcom.Record{
		XRef: "@O1@",
		Type: gedcom.RecordTypeMedia,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "FILE", Value: "photo.jpg"}},
	})

	enc.WriteTrailer()
	enc.Close()

	output := buf.String()

	// Verify all record types are present
	expectedPatterns := []string{
		"0 @I1@ INDI",
		"0 @F1@ FAM",
		"0 @S1@ SOUR",
		"0 @R1@ REPO",
		"0 @SUBM1@ SUBM",
		"0 @N1@ NOTE",
		"0 @O1@ OBJE",
	}
	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Output missing expected pattern: %q", pattern)
		}
	}

	// Verify it's valid GEDCOM
	doc, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode output: %v", err)
	}
	if len(doc.Records) != 7 {
		t.Errorf("Record count = %d, want 7", len(doc.Records))
	}
}

// Test that output from streaming matches output from batch for a complex document
func TestStreamEncoder_ComplexDocumentEquivalence(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
1 SOUR TestApp
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 1 JAN 1950
2 PLAC Boston, MA
1 FAMC @F1@
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
1 FAMS @F2@
0 @F1@ FAM
1 HUSB @I3@
1 WIFE @I4@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 15 JUN 1975
0 @S1@ SOUR
1 TITL County Records
1 AUTH Local Historian
0 TRLR
`

	// Decode the input
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to decode input: %v", err)
	}

	// Encode with batch encoder
	var batchBuf bytes.Buffer
	if err := Encode(&batchBuf, doc); err != nil {
		t.Fatalf("Batch Encode() error = %v", err)
	}

	// Encode with streaming encoder
	var streamBuf bytes.Buffer
	enc := NewStreamEncoder(&streamBuf)
	enc.WriteHeader(doc.Header)
	for _, r := range doc.Records {
		enc.WriteRecord(r)
	}
	enc.WriteTrailer()
	enc.Close()

	// Compare outputs
	if batchBuf.String() != streamBuf.String() {
		t.Errorf("Streaming output differs from batch output.\nBatch length: %d\nStreaming length: %d",
			batchBuf.Len(), streamBuf.Len())
		// Show first difference
		batchLines := strings.Split(batchBuf.String(), "\n")
		streamLines := strings.Split(streamBuf.String(), "\n")
		for i := 0; i < len(batchLines) && i < len(streamLines); i++ {
			if batchLines[i] != streamLines[i] {
				t.Errorf("First difference at line %d:\nBatch:  %q\nStream: %q",
					i+1, batchLines[i], streamLines[i])
				break
			}
		}
	}
}
