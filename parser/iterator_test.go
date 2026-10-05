package parser

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"testing"
)

// TestRecordsWithOffset_LineTooLong verifies that a single line longer
// than MaxLineBytes aborts offset-based iteration with the ErrLineTooLong
// sentinel — an inspectable identity (ADR-007), not an opaque error.
func TestRecordsWithOffset_LineTooLong(t *testing.T) {
	// A record whose NOTE line exceeds the 1 MiB cap.
	huge := strings.Repeat("x", MaxLineBytes+1)
	input := "0 @I1@ INDI\n1 NOTE " + huge + "\n"

	var gotErr error
	for _, err := range RecordsWithOffset(strings.NewReader(input)) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, ErrLineTooLong) {
		t.Fatalf("iteration error = %v, want ErrLineTooLong", gotErr)
	}
}

func TestRecordIterator_BasicIteration(t *testing.T) {
	input := `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	it := NewRecordIterator(strings.NewReader(input))

	// Record 1: HEAD
	if !it.Next() {
		t.Fatalf("Expected first record, got none. Err: %v", it.Err())
	}
	rec := it.Record()
	if rec.Type != "HEAD" {
		t.Errorf("First record Type = %q, want HEAD", rec.Type)
	}
	if rec.XRef != "" {
		t.Errorf("HEAD record should have no XRef, got %q", rec.XRef)
	}
	if len(rec.Lines) != 4 {
		t.Errorf("HEAD record has %d lines, want 4", len(rec.Lines))
	}

	// Record 2: @I1@ INDI
	if !it.Next() {
		t.Fatalf("Expected second record, got none. Err: %v", it.Err())
	}
	rec = it.Record()
	if rec.Type != "INDI" {
		t.Errorf("Second record Type = %q, want INDI", rec.Type)
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Second record XRef = %q, want @I1@", rec.XRef)
	}
	if len(rec.Lines) != 4 {
		t.Errorf("@I1@ record has %d lines, want 4", len(rec.Lines))
	}

	// Record 3: @I2@ INDI
	if !it.Next() {
		t.Fatalf("Expected third record, got none. Err: %v", it.Err())
	}
	rec = it.Record()
	if rec.Type != "INDI" {
		t.Errorf("Third record Type = %q, want INDI", rec.Type)
	}
	if rec.XRef != "@I2@" {
		t.Errorf("Third record XRef = %q, want @I2@", rec.XRef)
	}

	// Record 4: TRLR
	if !it.Next() {
		t.Fatalf("Expected fourth record, got none. Err: %v", it.Err())
	}
	rec = it.Record()
	if rec.Type != "TRLR" {
		t.Errorf("Fourth record Type = %q, want TRLR", rec.Type)
	}

	// No more records
	if it.Next() {
		t.Error("Expected no more records")
	}
	if it.Err() != nil {
		t.Errorf("Unexpected error: %v", it.Err())
	}
}

func TestRecordIterator_EmptyInput(t *testing.T) {
	it := NewRecordIterator(strings.NewReader(""))

	if it.Next() {
		t.Error("Expected no records for empty input")
	}
	if it.Err() != nil {
		t.Errorf("Unexpected error: %v", it.Err())
	}
}

func TestRecordIterator_SingleRecord(t *testing.T) {
	input := "0 HEAD\n1 SOUR Test"

	it := NewRecordIterator(strings.NewReader(input))

	if !it.Next() {
		t.Fatalf("Expected one record, got none. Err: %v", it.Err())
	}

	rec := it.Record()
	if rec.Type != "HEAD" {
		t.Errorf("Record Type = %q, want HEAD", rec.Type)
	}
	if len(rec.Lines) != 2 {
		t.Errorf("Record has %d lines, want 2", len(rec.Lines))
	}

	if it.Next() {
		t.Error("Expected no more records")
	}
}

func TestRecordIterator_LineNumbers(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 TRLR`

	it := NewRecordIterator(strings.NewReader(input))

	if !it.Next() {
		t.Fatal("Expected first record")
	}
	rec := it.Record()
	if rec.Lines[0].LineNumber != 1 {
		t.Errorf("First line number = %d, want 1", rec.Lines[0].LineNumber)
	}
	if rec.Lines[1].LineNumber != 2 {
		t.Errorf("Second line number = %d, want 2", rec.Lines[1].LineNumber)
	}

	if !it.Next() {
		t.Fatal("Expected second record")
	}
	rec = it.Record()
	if rec.Lines[0].LineNumber != 3 {
		t.Errorf("TRLR line number = %d, want 3", rec.Lines[0].LineNumber)
	}
}

func TestRecordIterator_CRLFLineEndings(t *testing.T) {
	input := "0 HEAD\r\n1 SOUR Test\r\n0 TRLR\r\n"

	it := NewRecordIterator(strings.NewReader(input))

	count := 0
	for it.Next() {
		count++
	}
	if it.Err() != nil {
		t.Fatalf("Unexpected error: %v", it.Err())
	}
	if count != 2 {
		t.Errorf("Got %d records, want 2", count)
	}
}

func TestRecordIterator_CROnlyLineEndings(t *testing.T) {
	input := "0 HEAD\r1 SOUR Test\r0 TRLR\r"

	it := NewRecordIterator(strings.NewReader(input))

	count := 0
	for it.Next() {
		count++
	}
	if it.Err() != nil {
		t.Fatalf("Unexpected error: %v", it.Err())
	}
	if count != 2 {
		t.Errorf("Got %d records, want 2", count)
	}
}

func TestRecordIterator_ParseError(t *testing.T) {
	// Invalid level number in subordinate line
	// The error occurs while reading subordinate lines of HEAD
	input := "0 HEAD\nX INVALID\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	// The parse error on "X INVALID" occurs while reading HEAD's subordinate lines
	// This causes Next() to return false with an error
	if it.Next() {
		t.Error("Expected iteration to stop on parse error")
	}
	if it.Err() == nil {
		t.Error("Expected error from invalid level")
	}
}

func TestRecordIterator_ParseError_SecondRecord(t *testing.T) {
	// Parse error in the second record (after successfully returning first)
	input := "0 HEAD\n0 @I1@ INDI\nX INVALID\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	// First record (HEAD) should be returned successfully
	if !it.Next() {
		t.Fatal("Expected first record")
	}
	if it.Record().Type != "HEAD" {
		t.Errorf("First record Type = %q, want HEAD", it.Record().Type)
	}

	// Second record should fail during subordinate line parsing
	if it.Next() {
		t.Error("Expected iteration to stop on parse error")
	}
	if it.Err() == nil {
		t.Error("Expected error from invalid level")
	}
}

// TestRecordIterator_SpacedXRef pins the streaming behavior for an XRef
// containing a space (issue #377). The iterator has no lenient mode, so — like
// [Parser.Parse] and every other parse error above — it stops and reports the
// error rather than silently mangling the record. Callers who need the
// recovered line must use ParseWithOptions in lenient mode.
func TestRecordIterator_SpacedXRef(t *testing.T) {
	input := "0 HEAD\n0 @NoTe ref@ NOTE mixed case and space\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	// As in TestRecordIterator_ParseError, the error surfaces while scanning
	// ahead for the next level-0 line, so even HEAD is not handed back.
	if it.Next() {
		t.Error("Expected iteration to stop on the spaced xref")
	}
	if it.Err() == nil || !strings.Contains(it.Err().Error(), "xref contains a space") {
		t.Errorf("Err() = %v, want a spaced-xref error", it.Err())
	}
}

// TestRecordIterator_UnterminatedXRef pins the same streaming behavior for an
// XRef with no closing "@" (issue #385): the iterator has no lenient mode, so
// it stops and reports rather than silently mangling the record.
func TestRecordIterator_UnterminatedXRef(t *testing.T) {
	input := "0 HEAD\n0 @I1 INDI\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	if it.Next() {
		t.Error("Expected iteration to stop on the unterminated xref")
	}
	if it.Err() == nil || !strings.Contains(it.Err().Error(), "xref is missing its closing @") {
		t.Errorf("Err() = %v, want an unterminated-xref error", it.Err())
	}
}

func TestRecordIterator_MatchesFullParse(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
2 GIVN John
2 SURN Smith
1 SEX M
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	// Full parse
	p := NewParser()
	fullLines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Full parse error: %v", err)
	}

	// Iterator
	it := NewRecordIterator(strings.NewReader(input))
	var iteratedLines []*Line
	for it.Next() {
		iteratedLines = append(iteratedLines, it.Record().Lines...)
	}
	if it.Err() != nil {
		t.Fatalf("Iterator error: %v", it.Err())
	}

	// Compare
	if len(iteratedLines) != len(fullLines) {
		t.Fatalf("Iterator got %d lines, full parse got %d", len(iteratedLines), len(fullLines))
	}

	for i := range fullLines {
		if iteratedLines[i].Level != fullLines[i].Level {
			t.Errorf("Line %d: Level = %d, want %d", i, iteratedLines[i].Level, fullLines[i].Level)
		}
		if iteratedLines[i].Tag != fullLines[i].Tag {
			t.Errorf("Line %d: Tag = %q, want %q", i, iteratedLines[i].Tag, fullLines[i].Tag)
		}
		if iteratedLines[i].Value != fullLines[i].Value {
			t.Errorf("Line %d: Value = %q, want %q", i, iteratedLines[i].Value, fullLines[i].Value)
		}
		if iteratedLines[i].XRef != fullLines[i].XRef {
			t.Errorf("Line %d: XRef = %q, want %q", i, iteratedLines[i].XRef, fullLines[i].XRef)
		}
		if iteratedLines[i].LineNumber != fullLines[i].LineNumber {
			t.Errorf("Line %d: LineNumber = %d, want %d", i, iteratedLines[i].LineNumber, fullLines[i].LineNumber)
		}
	}
}

func TestRecordIteratorWithOffset_ByteOffsets(t *testing.T) {
	// Each line is carefully crafted for predictable byte lengths
	// "0 HEAD\n" = 7 bytes
	// "1 SOUR Test\n" = 12 bytes
	// "0 TRLR\n" = 7 bytes
	input := "0 HEAD\n1 SOUR Test\n0 TRLR\n"

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	// First record: HEAD
	if !it.Next() {
		t.Fatalf("Expected first record. Err: %v", it.Err())
	}
	rec := it.Record()
	if rec.ByteOffset != 0 {
		t.Errorf("HEAD ByteOffset = %d, want 0", rec.ByteOffset)
	}
	// HEAD + SOUR = 7 + 12 = 19 bytes
	if rec.ByteLength != 19 {
		t.Errorf("HEAD ByteLength = %d, want 19", rec.ByteLength)
	}

	// Second record: TRLR
	if !it.Next() {
		t.Fatalf("Expected second record. Err: %v", it.Err())
	}
	rec = it.Record()
	if rec.ByteOffset != 19 {
		t.Errorf("TRLR ByteOffset = %d, want 19", rec.ByteOffset)
	}
	if rec.ByteLength != 7 {
		t.Errorf("TRLR ByteLength = %d, want 7", rec.ByteLength)
	}

	if it.Next() {
		t.Error("Expected no more records")
	}
}

func TestRecordIteratorWithOffset_EmptyInput(t *testing.T) {
	it := NewRecordIteratorWithOffset(strings.NewReader(""))

	if it.Next() {
		t.Error("Expected no records for empty input")
	}
	if it.Err() != nil {
		t.Errorf("Unexpected error: %v", it.Err())
	}
}

func TestRecordIteratorWithOffset_CRLFOffsets(t *testing.T) {
	// "0 HEAD\r\n" = 8 bytes
	// "1 SOUR Test\r\n" = 13 bytes
	// "0 TRLR\r\n" = 8 bytes
	input := "0 HEAD\r\n1 SOUR Test\r\n0 TRLR\r\n"

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	if !it.Next() {
		t.Fatalf("Expected first record. Err: %v", it.Err())
	}
	rec := it.Record()
	if rec.ByteOffset != 0 {
		t.Errorf("HEAD ByteOffset = %d, want 0", rec.ByteOffset)
	}
	// HEAD + SOUR = 8 + 13 = 21 bytes
	if rec.ByteLength != 21 {
		t.Errorf("HEAD ByteLength = %d, want 21", rec.ByteLength)
	}

	if !it.Next() {
		t.Fatalf("Expected second record. Err: %v", it.Err())
	}
	rec = it.Record()
	if rec.ByteOffset != 21 {
		t.Errorf("TRLR ByteOffset = %d, want 21", rec.ByteOffset)
	}
}

func TestRecordIteratorWithOffset_MultipleRecords(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR
`

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	records := make([]*RawRecord, 0)
	for it.Next() {
		// Make a copy since Record() may be reused
		rec := it.Record()
		records = append(records, &RawRecord{
			XRef:       rec.XRef,
			Type:       rec.Type,
			Lines:      rec.Lines,
			ByteOffset: rec.ByteOffset,
			ByteLength: rec.ByteLength,
		})
	}

	if it.Err() != nil {
		t.Fatalf("Unexpected error: %v", it.Err())
	}

	if len(records) != 4 {
		t.Fatalf("Got %d records, want 4", len(records))
	}

	// Verify records are contiguous
	var lastEnd int64
	for i, rec := range records {
		if rec.ByteOffset != lastEnd {
			t.Errorf("Record %d: ByteOffset = %d, expected %d (gap in offsets)", i, rec.ByteOffset, lastEnd)
		}
		lastEnd = rec.ByteOffset + rec.ByteLength
	}
}

func TestRecordIteratorWithOffset_ParseError(t *testing.T) {
	// Invalid line occurs while reading HEAD's subordinate lines
	input := "0 HEAD\nINVALID LINE\n0 TRLR\n"

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	// The parse error on "INVALID LINE" occurs while reading HEAD's subordinate lines
	// This causes Next() to return false with an error
	if it.Next() {
		t.Error("Expected iteration to stop on parse error")
	}
	if it.Err() == nil {
		t.Error("Expected parse error")
	}
}

func TestRecordIteratorWithOffset_ParseError_SecondRecord(t *testing.T) {
	// Parse error in the second record
	input := "0 HEAD\n0 @I1@ INDI\nINVALID LINE\n0 TRLR\n"

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	// First record (HEAD) should be returned successfully
	if !it.Next() {
		t.Fatal("Expected first record")
	}
	if it.Record().Type != "HEAD" {
		t.Errorf("First record Type = %q, want HEAD", it.Record().Type)
	}

	// Second record should fail during subordinate line parsing
	if it.Next() {
		t.Error("Expected iteration to stop on parse error")
	}
	if it.Err() == nil {
		t.Error("Expected parse error")
	}
}

func TestRawRecord_Fields(t *testing.T) {
	input := "0 @I1@ INDI\n1 NAME John /Doe/\n"

	it := NewRecordIterator(strings.NewReader(input))
	if !it.Next() {
		t.Fatal("Expected one record")
	}

	rec := it.Record()
	if rec.XRef != "@I1@" {
		t.Errorf("XRef = %q, want @I1@", rec.XRef)
	}
	if rec.Type != "INDI" {
		t.Errorf("Type = %q, want INDI", rec.Type)
	}
	if len(rec.Lines) != 2 {
		t.Errorf("Lines count = %d, want 2", len(rec.Lines))
	}
}

func TestRecordIterator_NoTrailingNewline(t *testing.T) {
	// File without trailing newline
	input := "0 HEAD\n1 SOUR Test\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	count := 0
	for it.Next() {
		count++
	}
	if it.Err() != nil {
		t.Fatalf("Unexpected error: %v", it.Err())
	}
	if count != 2 {
		t.Errorf("Got %d records, want 2", count)
	}
}

func TestRecordIteratorWithOffset_NoTrailingNewline(t *testing.T) {
	input := "0 HEAD\n1 SOUR Test\n0 TRLR"

	it := NewRecordIteratorWithOffset(strings.NewReader(input))

	count := 0
	for it.Next() {
		count++
	}
	if it.Err() != nil {
		t.Fatalf("Unexpected error: %v", it.Err())
	}
	if count != 2 {
		t.Errorf("Got %d records, want 2", count)
	}
}

func TestRecordIterator_RecordWithoutXRef(t *testing.T) {
	// Some valid GEDCOM records don't have XRefs (like HEAD, TRLR, SUBM without pointer)
	input := "0 HEAD\n1 CHAR UTF-8\n0 @S1@ SUBM\n1 NAME Submitter\n0 TRLR"

	it := NewRecordIterator(strings.NewReader(input))

	// HEAD - no XRef
	if !it.Next() {
		t.Fatal("Expected HEAD record")
	}
	rec := it.Record()
	if rec.XRef != "" {
		t.Errorf("HEAD XRef = %q, want empty", rec.XRef)
	}

	// SUBM - has XRef
	if !it.Next() {
		t.Fatal("Expected SUBM record")
	}
	rec = it.Record()
	if rec.XRef != "@S1@" {
		t.Errorf("SUBM XRef = %q, want @S1@", rec.XRef)
	}

	// TRLR - no XRef
	if !it.Next() {
		t.Fatal("Expected TRLR record")
	}
	rec = it.Record()
	if rec.XRef != "" {
		t.Errorf("TRLR XRef = %q, want empty", rec.XRef)
	}
}

// ============================================================================
// Tests for iter.Seq2 API: Records() and RecordsWithOffset()
// ============================================================================

func TestRecords_BasicIteration(t *testing.T) {
	input := `0 HEAD
1 SOUR TestSystem
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR`

	var records []*RawRecord
	for rec, err := range Records(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		records = append(records, rec)
	}

	if len(records) != 3 {
		t.Errorf("got %d records, want 3", len(records))
	}
	if records[0].Type != "HEAD" {
		t.Errorf("first record type = %q, want HEAD", records[0].Type)
	}
	if records[1].Type != "INDI" {
		t.Errorf("second record type = %q, want INDI", records[1].Type)
	}
	if records[1].XRef != "@I1@" {
		t.Errorf("second record XRef = %q, want @I1@", records[1].XRef)
	}
	if records[2].Type != "TRLR" {
		t.Errorf("third record type = %q, want TRLR", records[2].Type)
	}
}

func TestRecords_EmptyInput(t *testing.T) {
	var records []*RawRecord
	var gotError error
	for rec, err := range Records(strings.NewReader("")) {
		if err != nil {
			gotError = err
			break
		}
		records = append(records, rec)
	}

	if gotError != nil {
		t.Errorf("unexpected error for empty input: %v", gotError)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestRecords_ParseError(t *testing.T) {
	// Invalid level number in subordinate line
	input := "0 HEAD\nX INVALID\n0 TRLR"

	var gotError error
	for _, err := range Records(strings.NewReader(input)) {
		if err != nil {
			gotError = err
			break
		}
	}

	if gotError == nil {
		t.Error("expected error from invalid level, got nil")
	}

	// Verify error contains useful context
	errStr := gotError.Error()
	if !strings.Contains(errStr, "level") && !strings.Contains(errStr, "parse") {
		t.Errorf("error should contain useful context, got: %v", gotError)
	}
}

func TestRecords_EarlyTermination(t *testing.T) {
	// Generate input with many records
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 SOUR Test\n")
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 NAME Person%d\n", i, i)
	}
	sb.WriteString("0 TRLR\n")
	input := sb.String()

	count := 0
	for _, err := range Records(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		if count >= 5 {
			break // early termination
		}
	}

	if count != 5 {
		t.Errorf("got %d records before break, want 5", count)
	}
}

func TestRecordsWithOffset_ByteOffsets(t *testing.T) {
	// "0 HEAD\n" = 7 bytes
	// "1 SOUR Test\n" = 12 bytes
	// HEAD record = 19 bytes total
	// "0 TRLR\n" = 7 bytes
	input := "0 HEAD\n1 SOUR Test\n0 TRLR\n"

	var records []*RawRecord
	for rec, err := range RecordsWithOffset(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		records = append(records, rec)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	// First record: HEAD
	if records[0].ByteOffset != 0 {
		t.Errorf("HEAD ByteOffset = %d, want 0", records[0].ByteOffset)
	}
	if records[0].ByteLength != 19 {
		t.Errorf("HEAD ByteLength = %d, want 19", records[0].ByteLength)
	}

	// Second record: TRLR
	if records[1].ByteOffset != 19 {
		t.Errorf("TRLR ByteOffset = %d, want 19", records[1].ByteOffset)
	}
	if records[1].ByteLength != 7 {
		t.Errorf("TRLR ByteLength = %d, want 7", records[1].ByteLength)
	}
}

func TestRecordsWithOffset_EmptyInput(t *testing.T) {
	var records []*RawRecord
	var gotError error
	for rec, err := range RecordsWithOffset(strings.NewReader("")) {
		if err != nil {
			gotError = err
			break
		}
		records = append(records, rec)
	}

	if gotError != nil {
		t.Errorf("unexpected error for empty input: %v", gotError)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestRecordsWithOffset_ParseError(t *testing.T) {
	// Invalid line
	input := "0 HEAD\nINVALID LINE\n0 TRLR\n"

	var gotError error
	for _, err := range RecordsWithOffset(strings.NewReader(input)) {
		if err != nil {
			gotError = err
			break
		}
	}

	if gotError == nil {
		t.Error("expected parse error")
	}
}

func TestRecordsWithOffset_EarlyTermination(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 SOUR Test\n")
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 NAME Person%d\n", i, i)
	}
	sb.WriteString("0 TRLR\n")
	input := sb.String()

	count := 0
	for _, err := range RecordsWithOffset(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		if count >= 3 {
			break
		}
	}

	if count != 3 {
		t.Errorf("got %d records before break, want 3", count)
	}
}

func TestRecords_MatchesRecordIterator(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
2 GIVN John
2 SURN Smith
1 SEX M
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	// Use RecordIterator
	it := NewRecordIterator(strings.NewReader(input))
	var iterRecords []*RawRecord
	for it.Next() {
		rec := it.Record()
		iterRecords = append(iterRecords, &RawRecord{
			XRef: rec.XRef,
			Type: rec.Type,
		})
	}
	if it.Err() != nil {
		t.Fatalf("RecordIterator error: %v", it.Err())
	}

	// Use Records() iter.Seq2
	var seqRecords []*RawRecord
	for rec, err := range Records(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("Records error: %v", err)
		}
		seqRecords = append(seqRecords, rec)
	}

	// Compare
	if len(seqRecords) != len(iterRecords) {
		t.Fatalf("Records got %d, RecordIterator got %d", len(seqRecords), len(iterRecords))
	}

	for i := range iterRecords {
		if seqRecords[i].XRef != iterRecords[i].XRef {
			t.Errorf("Record %d: XRef = %q, want %q", i, seqRecords[i].XRef, iterRecords[i].XRef)
		}
		if seqRecords[i].Type != iterRecords[i].Type {
			t.Errorf("Record %d: Type = %q, want %q", i, seqRecords[i].Type, iterRecords[i].Type)
		}
	}
}

// A read that fails part-way through a line must be reported as the reader
// error it is, not as whatever the leftover fragment parses as (issue #382).
func TestRecordIterator_ReadErrorOutranksTruncatedTail(t *testing.T) {
	tests := []struct {
		name string
		// content is handed over in full, then the read fails.
		content string
		// wantRecords is the number of records completed before the fragment.
		wantRecords int
	}{
		{
			// The fragment is the first line the iterator sees, so the failure
			// lands in scanNextLine.
			name:        "fragment starts a record",
			content:     "0",
			wantRecords: 0,
		},
		{
			// The fragment is a subordinate line, so the failure lands in Next.
			name:        "fragment continues a record",
			content:     "0 HEAD\n1 SOUR TEST\n0 @I1@ INDI\n1",
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readErr := &locatedReadError{line: 42}
			it := NewRecordIterator(&failingReader{content: tt.content, err: readErr})

			count := 0
			for it.Next() {
				count++
			}
			if count != tt.wantRecords {
				t.Errorf("iterated %d records, want %d", count, tt.wantRecords)
			}
			if !errors.Is(it.Err(), readErr) {
				t.Errorf("Err() = %v, want the reader error %v", it.Err(), readErr)
			}
			var parseErr *ParseError
			if errors.As(it.Err(), &parseErr) {
				t.Errorf("Err() = %v, want the reader error, not a syntax error about the fragment", it.Err())
			}
		})
	}
}

// offsetFixtureRecords is a multi-record document used by the byte-offset
// tests, one slice of lines (without terminators) per record.
var offsetFixtureRecords = [][]string{
	{"0 HEAD", "1 SOUR TEST", "2 VERS 1.0", "1 GEDC", "2 VERS 5.5.1", "1 CHAR UTF-8"},
	{"0 @I1@ INDI", "1 NAME John /Smith/", "1 SEX M", "1 BIRT", "2 DATE 1 JAN 1900", "1 FAMS @F1@"},
	{"0 @I2@ INDI", "1 NAME Jane /Doe/", "1 SEX F", "1 FAMS @F1@"},
	{"0 @F1@ FAM", "1 HUSB @I1@", "1 WIFE @I2@"},
	{"0 TRLR"},
}

// offsetVariant chooses the terminator for line i of n lines in total.
type offsetVariant struct {
	name string
	term func(i, n int) string
}

var offsetVariants = []offsetVariant{
	{"LF", func(int, int) string { return "\n" }},
	{"CRLF", func(int, int) string { return "\r\n" }},
	{"CR", func(int, int) string { return "\r" }},
	{"mixed", func(i, _ int) string { return []string{"\n", "\r\n", "\r"}[i%3] }},
	{"no trailing newline", func(i, n int) string {
		if i == n-1 {
			return ""
		}
		return "\n"
	}},
}

// offsetTerm returns the terminator function of the named offsetVariant.
func offsetTerm(name string) func(i, n int) string {
	for _, v := range offsetVariants {
		if v.name == name {
			return v.term
		}
	}
	panic("unknown offset variant " + name)
}

// buildOffsetFixture renders offsetFixtureRecords with term's line endings.
// It returns the full input and the exact bytes of each record, terminators
// included.
func buildOffsetFixture(term func(i, n int) string) (input string, records []string) {
	n := 0
	for _, rec := range offsetFixtureRecords {
		n += len(rec)
	}
	var all strings.Builder
	i := 0
	for _, rec := range offsetFixtureRecords {
		var b strings.Builder
		for _, line := range rec {
			b.WriteString(line)
			b.WriteString(term(i, n))
			i++
		}
		records = append(records, b.String())
		all.WriteString(b.String())
	}
	return all.String(), records
}

// assertRecordSlices checks that each record's ByteOffset and ByteLength
// describe exactly the bytes in want, relative to input.
func assertRecordSlices(t *testing.T, input string, got []*RawRecord, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d", len(got), len(want))
	}
	for i, rec := range got {
		start, end := rec.ByteOffset, rec.ByteOffset+rec.ByteLength
		if start < 0 || rec.ByteLength < 0 || end > int64(len(input)) {
			t.Errorf("record %d (%s): offset %d length %d out of range for %d-byte input",
				i, rec.Type, rec.ByteOffset, rec.ByteLength, len(input))
			continue
		}
		if s := input[start:end]; s != want[i] {
			t.Errorf("record %d (%s): input[%d:%d] = %q, want %q", i, rec.Type, start, end, s, want[i])
		}
	}
}

func collectIterator(t *testing.T, it *RecordIterator) []*RawRecord {
	t.Helper()
	var out []*RawRecord
	for it.Next() {
		out = append(out, it.Record())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iteration error: %v", err)
	}
	return out
}

func collectSeq(t *testing.T, seq iter.Seq2[*RawRecord, error]) []*RawRecord {
	t.Helper()
	var out []*RawRecord
	for rec, err := range seq {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		out = append(out, rec)
	}
	return out
}

// Every iterator must report offsets and lengths that slice out exactly one
// record, whatever the line endings (issue #502).
func TestRecordIterators_ExactByteOffsets(t *testing.T) {
	for _, v := range offsetVariants {
		t.Run(v.name, func(t *testing.T) {
			input, want := buildOffsetFixture(v.term)

			plain := collectIterator(t, NewRecordIterator(strings.NewReader(input)))
			seq := collectSeq(t, Records(strings.NewReader(input)))
			withOffset := collectSeq(t, RecordsWithOffset(strings.NewReader(input)))

			t.Run("NewRecordIterator", func(t *testing.T) { assertRecordSlices(t, input, plain, want) })
			t.Run("Records", func(t *testing.T) { assertRecordSlices(t, input, seq, want) })
			t.Run("RecordsWithOffset", func(t *testing.T) { assertRecordSlices(t, input, withOffset, want) })

			if len(plain) != len(withOffset) {
				t.Fatalf("NewRecordIterator yielded %d records, RecordsWithOffset %d", len(plain), len(withOffset))
			}
			for i := range plain {
				if plain[i].ByteOffset != withOffset[i].ByteOffset || plain[i].ByteLength != withOffset[i].ByteLength {
					t.Errorf("record %d: NewRecordIterator (%d,%d) != RecordsWithOffset (%d,%d)", i,
						plain[i].ByteOffset, plain[i].ByteLength, withOffset[i].ByteOffset, withOffset[i].ByteLength)
				}
			}
		})
	}
}

// The same document in LF and CRLF differs by one byte per preceding line.
func TestRecordIterator_CRLFOffsetsShiftByLineCount(t *testing.T) {
	lfInput, _ := buildOffsetFixture(offsetTerm("LF"))
	crlfInput, _ := buildOffsetFixture(offsetTerm("CRLF"))

	lf := collectIterator(t, NewRecordIterator(strings.NewReader(lfInput)))
	crlf := collectIterator(t, NewRecordIterator(strings.NewReader(crlfInput)))
	if len(lf) != len(crlf) {
		t.Fatalf("LF yielded %d records, CRLF %d", len(lf), len(crlf))
	}

	linesBefore := int64(0)
	for i := range lf {
		if diff := crlf[i].ByteOffset - lf[i].ByteOffset; diff != linesBefore {
			t.Errorf("record %d (%s): CRLF offset - LF offset = %d, want %d", i, lf[i].Type, diff, linesBefore)
		}
		linesBefore += int64(len(offsetFixtureRecords[i]))
	}
}

// newRecordIteratorAt reports offsets relative to the file, not the reader.
func TestNewRecordIteratorAt_BaseOffset(t *testing.T) {
	input, want := buildOffsetFixture(offsetTerm("CRLF"))
	const prefix = "XXXXXXX"
	file := prefix + input

	got := collectIterator(t, newRecordIteratorAt(strings.NewReader(input), int64(len(prefix))))
	assertRecordSlices(t, file, got, want)
}
