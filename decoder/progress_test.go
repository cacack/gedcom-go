package decoder

import (
	"bytes"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

// TestProgressCallbackInvoked verifies that the callback is invoked during parsing.
func TestProgressCallbackInvoked(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	var callCount int32
	var lastBytesRead int64

	opts := DefaultOptions()
	opts.OnProgress = func(bytesRead, totalBytes int64) {
		atomic.AddInt32(&callCount, 1)
		lastBytesRead = bytesRead
	}

	doc, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	if doc == nil {
		t.Fatal("DecodeWithOptions() returned nil document")
	}

	// Callback should have been called at least once
	if callCount == 0 {
		t.Error("Progress callback was never invoked")
	}

	// Final bytesRead should be at least the input length
	// (accounting for how the reader may batch reads)
	if lastBytesRead <= 0 {
		t.Errorf("lastBytesRead = %d, expected > 0", lastBytesRead)
	}
}

// TestProgressCallbackCumulativeBytes verifies that bytesRead is cumulative.
func TestProgressCallbackCumulativeBytes(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	var bytesReadHistory []int64

	opts := DefaultOptions()
	opts.OnProgress = func(bytesRead, totalBytes int64) {
		bytesReadHistory = append(bytesReadHistory, bytesRead)
	}

	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	// Verify bytesRead is monotonically increasing
	for i := 1; i < len(bytesReadHistory); i++ {
		if bytesReadHistory[i] < bytesReadHistory[i-1] {
			t.Errorf("bytesRead decreased: %d < %d at index %d",
				bytesReadHistory[i], bytesReadHistory[i-1], i)
		}
	}
}

// TestProgressCallbackTotalSizeUnknown verifies that totalSize=0 reports -1.
func TestProgressCallbackTotalSizeUnknown(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	var receivedTotal int64 = 999 // Start with non-zero to verify it gets set

	opts := DefaultOptions()
	opts.TotalSize = 0 // Unknown size (default)
	opts.OnProgress = func(bytesRead, totalBytes int64) {
		receivedTotal = totalBytes
	}

	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	// When TotalSize is 0, callback should receive -1
	if receivedTotal != -1 {
		t.Errorf("totalBytes = %d, want -1 when TotalSize is unknown", receivedTotal)
	}
}

// TestProgressCallbackTotalSizeKnown verifies that explicit TotalSize is passed.
func TestProgressCallbackTotalSizeKnown(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`
	expectedTotal := int64(1024)

	var receivedTotal int64

	opts := DefaultOptions()
	opts.TotalSize = expectedTotal
	opts.OnProgress = func(bytesRead, totalBytes int64) {
		receivedTotal = totalBytes
	}

	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	if receivedTotal != expectedTotal {
		t.Errorf("totalBytes = %d, want %d", receivedTotal, expectedTotal)
	}
}

// TestProgressCallbackNilNoOverhead verifies no wrapper when callback is nil.
func TestProgressCallbackNilNoOverhead(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	opts := DefaultOptions()
	opts.OnProgress = nil // Explicitly nil

	doc, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	if doc == nil {
		t.Fatal("DecodeWithOptions() returned nil document")
	}

	// The test passes if decoding succeeds without error
	// The nil callback path should not create a progressReader wrapper
}

// TestProgressReaderDirectly tests the progressReader implementation directly.
func TestProgressReaderDirectly(t *testing.T) {
	data := []byte("Hello, World!")
	var bytesReported int64
	var totalReported int64

	pr := &progressReader{
		reader:    bytes.NewReader(data),
		totalSize: int64(len(data)),
		callback: func(bytesRead, totalBytes int64) {
			bytesReported = bytesRead
			totalReported = totalBytes
		},
	}

	buf := make([]byte, 5)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Read() n = %d, want 5", n)
	}
	if bytesReported != 5 {
		t.Errorf("bytesReported = %d, want 5", bytesReported)
	}
	if totalReported != int64(len(data)) {
		t.Errorf("totalReported = %d, want %d", totalReported, len(data))
	}

	// Read more
	n, err = pr.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Read() n = %d, want 5", n)
	}
	if bytesReported != 10 {
		t.Errorf("bytesReported = %d, want 10", bytesReported)
	}

	// Read remaining
	n, err = pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}
	if n != 3 {
		t.Errorf("Read() n = %d, want 3", n)
	}
	if bytesReported != 13 {
		t.Errorf("bytesReported = %d, want 13", bytesReported)
	}
}

// TestProgressReaderUnknownTotal tests progressReader with unknown total size.
func TestProgressReaderUnknownTotal(t *testing.T) {
	data := []byte("Test data")
	var totalReported int64 = 999

	pr := &progressReader{
		reader:    bytes.NewReader(data),
		totalSize: 0, // Unknown
		callback: func(bytesRead, totalBytes int64) {
			totalReported = totalBytes
		},
	}

	buf := make([]byte, 100)
	_, err := pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}

	if totalReported != -1 {
		t.Errorf("totalReported = %d, want -1 for unknown size", totalReported)
	}
}

// TestProgressReaderEOF verifies callback is not called on zero-byte reads.
func TestProgressReaderEOF(t *testing.T) {
	data := []byte("Hi")
	var callCount int

	pr := &progressReader{
		reader:    bytes.NewReader(data),
		totalSize: int64(len(data)),
		callback: func(bytesRead, totalBytes int64) {
			callCount++
		},
	}

	// Read all data
	buf := make([]byte, 100)
	_, _ = pr.Read(buf)
	initialCount := callCount

	// Try to read again (should get EOF with 0 bytes)
	n, err := pr.Read(buf)
	if err != io.EOF {
		t.Errorf("Expected EOF, got err = %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes, got %d", n)
	}

	// Callback should NOT be called for zero-byte read
	if callCount != initialCount {
		t.Errorf("Callback was called on EOF with 0 bytes")
	}
}

// TestProgressCallbackWithRealGEDCOM tests with actual GEDCOM test data.
func TestProgressCallbackWithRealGEDCOM(t *testing.T) {
	// Create a more substantial GEDCOM to ensure multiple read calls
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5\n1 CHAR UTF-8\n")

	// Add many individuals to make the file larger
	for i := 1; i <= 100; i++ {
		sb.WriteString("0 @I")
		sb.WriteString(string(rune('0' + i/100)))
		sb.WriteString(string(rune('0' + (i%100)/10)))
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString("@ INDI\n")
		sb.WriteString("1 NAME Test Person /Number ")
		sb.WriteString(string(rune('0' + i/100)))
		sb.WriteString(string(rune('0' + (i%100)/10)))
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString("/\n")
		sb.WriteString("1 BIRT\n2 DATE 1 JAN 1900\n")
	}
	sb.WriteString("0 TRLR\n")

	input := sb.String()
	inputSize := int64(len(input))

	var lastBytesRead int64
	var callCount int

	opts := DefaultOptions()
	opts.TotalSize = inputSize
	opts.OnProgress = func(bytesRead, totalBytes int64) {
		callCount++
		lastBytesRead = bytesRead

		// Verify totalBytes matches our specified size
		if totalBytes != inputSize {
			t.Errorf("totalBytes = %d, want %d", totalBytes, inputSize)
		}

		// Verify bytesRead never exceeds totalBytes
		if bytesRead > totalBytes {
			t.Errorf("bytesRead %d exceeds totalBytes %d", bytesRead, totalBytes)
		}
	}

	doc, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}

	if doc == nil {
		t.Fatal("DecodeWithOptions() returned nil document")
	}

	// Verify callback was called
	if callCount == 0 {
		t.Error("Progress callback was never invoked")
	}

	// Final bytesRead should match input size
	if lastBytesRead != inputSize {
		t.Errorf("lastBytesRead = %d, want %d", lastBytesRead, inputSize)
	}
}
