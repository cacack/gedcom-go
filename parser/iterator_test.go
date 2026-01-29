package parser

import (
	"fmt"
	"strings"
	"testing"
)

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
		sb.WriteString(fmt.Sprintf("0 @I%d@ INDI\n1 NAME Person%d\n", i, i))
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
		sb.WriteString(fmt.Sprintf("0 @I%d@ INDI\n1 NAME Person%d\n", i, i))
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
