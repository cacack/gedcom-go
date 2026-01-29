package parser

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// stringReadSeeker wraps a string as an io.ReadSeeker
type stringReadSeeker struct {
	*strings.Reader
}

func newStringReadSeeker(s string) *stringReadSeeker {
	return &stringReadSeeker{strings.NewReader(s)}
}

func TestNewLazyParser(t *testing.T) {
	input := "0 HEAD\n0 TRLR\n"
	rs := newStringReadSeeker(input)

	lp := NewLazyParser(rs)
	if lp == nil {
		t.Fatal("NewLazyParser returned nil")
	}
	if lp.HasIndex() {
		t.Error("New LazyParser should not have index")
	}
}

func TestLazyParser_BuildIndex(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	if !lp.HasIndex() {
		t.Error("HasIndex should return true after BuildIndex")
	}

	if lp.Index() == nil {
		t.Error("Index() should not return nil after BuildIndex")
	}

	// Should have 4 records: HEAD, @I1@, @I2@, TRLR
	if lp.RecordCount() != 4 {
		t.Errorf("RecordCount() = %d, want 4", lp.RecordCount())
	}

	// Check XRefs
	xrefs := lp.XRefs()
	if len(xrefs) != 2 {
		t.Errorf("Got %d XRefs, want 2", len(xrefs))
	}
}

func TestLazyParser_FindRecord(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Find @I1@
	rec, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord(@I1@) error: %v", err)
	}

	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
	if rec.Type != "INDI" {
		t.Errorf("Record Type = %q, want INDI", rec.Type)
	}
	if len(rec.Lines) != 4 {
		t.Errorf("Record has %d lines, want 4", len(rec.Lines))
	}

	// Verify line content
	if rec.Lines[0].Tag != "INDI" {
		t.Errorf("First line Tag = %q, want INDI", rec.Lines[0].Tag)
	}
	if rec.Lines[1].Tag != "NAME" || rec.Lines[1].Value != "John /Doe/" {
		t.Errorf("NAME line = %q %q, want NAME John /Doe/", rec.Lines[1].Tag, rec.Lines[1].Value)
	}

	// Find @I2@
	rec, err = lp.FindRecord("@I2@")
	if err != nil {
		t.Fatalf("FindRecord(@I2@) error: %v", err)
	}
	if rec.XRef != "@I2@" {
		t.Errorf("Record XRef = %q, want @I2@", rec.XRef)
	}
}

func TestLazyParser_FindRecord_NotFound(t *testing.T) {
	input := "0 HEAD\n0 @I1@ INDI\n0 TRLR"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	_, err := lp.FindRecord("@I999@")
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
	if !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got: %v", err)
	}
}

func TestLazyParser_FindRecord_NoIndex(t *testing.T) {
	input := "0 HEAD\n0 TRLR"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	_, err := lp.FindRecord("@I1@")
	if err == nil {
		t.Error("Expected error without index")
	}
	if !errors.Is(err, ErrNoIndex) {
		t.Errorf("Expected ErrNoIndex, got: %v", err)
	}
}

func TestLazyParser_FindRecordByType(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Find HEAD
	rec, err := lp.FindRecordByType("HEAD")
	if err != nil {
		t.Fatalf("FindRecordByType(HEAD) error: %v", err)
	}

	if rec.Type != "HEAD" {
		t.Errorf("Record Type = %q, want HEAD", rec.Type)
	}
	if len(rec.Lines) != 4 {
		t.Errorf("HEAD has %d lines, want 4", len(rec.Lines))
	}

	// Find TRLR
	rec, err = lp.FindRecordByType("TRLR")
	if err != nil {
		t.Fatalf("FindRecordByType(TRLR) error: %v", err)
	}
	if rec.Type != "TRLR" {
		t.Errorf("Record Type = %q, want TRLR", rec.Type)
	}
}

func TestLazyParser_FindRecordByType_NotFound(t *testing.T) {
	input := "0 HEAD\n0 TRLR"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	_, err := lp.FindRecordByType("SUBM")
	if err == nil {
		t.Error("Expected error for non-existent type")
	}
	if !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got: %v", err)
	}
}

func TestLazyParser_SaveLoadIndex(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
1 NAME John
0 @I2@ INDI
1 NAME Jane
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Save index
	var buf bytes.Buffer
	if err := lp.SaveIndex(&buf); err != nil {
		t.Fatalf("SaveIndex error: %v", err)
	}

	// Create new parser and load index
	rs2 := newStringReadSeeker(input)
	lp2 := NewLazyParser(rs2)

	if err := lp2.LoadIndex(&buf); err != nil {
		t.Fatalf("LoadIndex error: %v", err)
	}

	if !lp2.HasIndex() {
		t.Error("Should have index after LoadIndex")
	}

	// Verify loaded index works
	rec, err := lp2.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord after LoadIndex error: %v", err)
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
}

func TestLazyParser_SaveIndex_NoIndex(t *testing.T) {
	rs := newStringReadSeeker("0 HEAD\n0 TRLR")
	lp := NewLazyParser(rs)

	var buf bytes.Buffer
	err := lp.SaveIndex(&buf)
	if err == nil {
		t.Error("Expected error when saving without index")
	}
	if !errors.Is(err, ErrNoIndex) {
		t.Errorf("Expected ErrNoIndex, got: %v", err)
	}
}

func TestLazyParser_Iterate(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	it := lp.Iterate()

	count := 0
	for it.Next() {
		count++
	}
	if it.Err() != nil {
		t.Fatalf("Iterator error: %v", it.Err())
	}

	if count != 3 {
		t.Errorf("Iterated %d records, want 3", count)
	}
}

func TestLazyParser_IterateAll(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	// Read some data first to move position
	buf := make([]byte, 10)
	_, _ = rs.Read(buf)

	// IterateAll should seek to beginning
	it, err := lp.IterateAll()
	if err != nil {
		t.Fatalf("IterateAll error: %v", err)
	}

	count := 0
	for it.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("Iterated %d records, want 3", count)
	}
}

func TestLazyParser_IterateFrom(t *testing.T) {
	// "0 HEAD\n" = 7 bytes
	// "1 SOUR Test\n" = 12 bytes
	// HEAD record ends at offset 19
	// "0 @I1@ INDI\n" starts at 19
	input := "0 HEAD\n1 SOUR Test\n0 @I1@ INDI\n1 NAME John\n0 TRLR\n"

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	// Build index to get offsets
	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	entry, _ := lp.Index().Lookup("@I1@")

	// Iterate from @I1@
	it, err := lp.IterateFrom(entry.ByteOffset)
	if err != nil {
		t.Fatalf("IterateFrom error: %v", err)
	}

	if !it.Next() {
		t.Fatal("Expected at least one record")
	}

	rec := it.Record()
	if rec.XRef != "@I1@" {
		t.Errorf("First record XRef = %q, want @I1@", rec.XRef)
	}
}

func TestLazyParser_XRefs_NoIndex(t *testing.T) {
	rs := newStringReadSeeker("0 HEAD\n0 TRLR")
	lp := NewLazyParser(rs)

	xrefs := lp.XRefs()
	if xrefs != nil {
		t.Errorf("XRefs without index should be nil, got %v", xrefs)
	}
}

func TestLazyParser_RecordCount_NoIndex(t *testing.T) {
	rs := newStringReadSeeker("0 HEAD\n0 TRLR")
	lp := NewLazyParser(rs)

	if lp.RecordCount() != 0 {
		t.Errorf("RecordCount without index = %d, want 0", lp.RecordCount())
	}
}

func TestLazyParser_FindRecordByType_NoIndex(t *testing.T) {
	rs := newStringReadSeeker("0 HEAD\n0 TRLR")
	lp := NewLazyParser(rs)

	_, err := lp.FindRecordByType("HEAD")
	if !errors.Is(err, ErrNoIndex) {
		t.Errorf("Expected ErrNoIndex, got: %v", err)
	}
}

func TestLazyParser_MultipleFinds(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John
0 @I2@ INDI
1 NAME Jane
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Find records in non-sequential order
	rec, err := lp.FindRecord("@F1@")
	if err != nil {
		t.Fatalf("FindRecord(@F1@) error: %v", err)
	}
	if rec.Type != "FAM" {
		t.Errorf("@F1@ Type = %q, want FAM", rec.Type)
	}

	rec, err = lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord(@I1@) error: %v", err)
	}
	if rec.Type != "INDI" {
		t.Errorf("@I1@ Type = %q, want INDI", rec.Type)
	}

	rec, err = lp.FindRecord("@I2@")
	if err != nil {
		t.Fatalf("FindRecord(@I2@) error: %v", err)
	}
	if rec.XRef != "@I2@" {
		t.Errorf("Record XRef = %q, want @I2@", rec.XRef)
	}
}

func TestLazyParser_IterateAfterFind(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
1 NAME John
0 @I2@ INDI
1 NAME Jane
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Find a record first
	_, err := lp.FindRecord("@I2@")
	if err != nil {
		t.Fatalf("FindRecord error: %v", err)
	}

	// Then iterate all
	it, err := lp.IterateAll()
	if err != nil {
		t.Fatalf("IterateAll error: %v", err)
	}

	count := 0
	for it.Next() {
		count++
	}
	if count != 4 {
		t.Errorf("Iterated %d records after Find, want 4", count)
	}
}

func TestLazyParser_FindRecordMatchesFullParse(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
2 GIVN John
2 SURN Smith
1 SEX M
1 BIRT
2 DATE 1 JAN 1950
0 TRLR`

	// Full parse to get expected record
	p := NewParser()
	fullLines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Full parse error: %v", err)
	}

	// Find @I1@ record lines from full parse
	var expectedLines []*Line
	inI1 := false
	for _, line := range fullLines {
		if line.Level == 0 {
			if line.XRef == "@I1@" {
				inI1 = true
			} else if inI1 {
				break
			}
		}
		if inI1 {
			expectedLines = append(expectedLines, line)
		}
	}

	// Lazy parse and find
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	rec, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord error: %v", err)
	}

	// Compare lines
	if len(rec.Lines) != len(expectedLines) {
		t.Fatalf("Got %d lines, want %d", len(rec.Lines), len(expectedLines))
	}

	for i := range expectedLines {
		if rec.Lines[i].Level != expectedLines[i].Level {
			t.Errorf("Line %d: Level = %d, want %d", i, rec.Lines[i].Level, expectedLines[i].Level)
		}
		if rec.Lines[i].Tag != expectedLines[i].Tag {
			t.Errorf("Line %d: Tag = %q, want %q", i, rec.Lines[i].Tag, expectedLines[i].Tag)
		}
		if rec.Lines[i].Value != expectedLines[i].Value {
			t.Errorf("Line %d: Value = %q, want %q", i, rec.Lines[i].Value, expectedLines[i].Value)
		}
	}
}

// errorSeeker is a ReadSeeker that returns errors on Seek
type errorSeeker struct {
	io.Reader
}

func (e *errorSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("seek error")
}

func TestLazyParser_BuildIndex_SeekError(t *testing.T) {
	es := &errorSeeker{Reader: strings.NewReader("0 HEAD\n0 TRLR\n")}
	lp := NewLazyParser(es)

	err := lp.BuildIndex()
	if err == nil {
		t.Error("Expected error from seek failure")
	}
}

func TestLazyParser_IterateFrom_SeekError(t *testing.T) {
	es := &errorSeeker{Reader: strings.NewReader("0 HEAD\n0 TRLR\n")}
	lp := NewLazyParser(es)

	_, err := lp.IterateFrom(10)
	if err == nil {
		t.Error("Expected error from seek failure")
	}
}

func TestLazyParser_LoadIndex_InvalidData(t *testing.T) {
	rs := newStringReadSeeker("0 HEAD\n0 TRLR\n")
	lp := NewLazyParser(rs)

	err := lp.LoadIndex(strings.NewReader("invalid gob data"))
	if err == nil {
		t.Error("Expected error from invalid index data")
	}
}

func TestLazyParser_EmptyFile(t *testing.T) {
	rs := newStringReadSeeker("")
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	if lp.RecordCount() != 0 {
		t.Errorf("RecordCount = %d, want 0", lp.RecordCount())
	}
}

func TestLazyParser_CRLFLineEndings(t *testing.T) {
	input := "0 HEAD\r\n1 SOUR Test\r\n0 @I1@ INDI\r\n1 NAME John\r\n0 TRLR\r\n"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Verify record lookup works with CRLF
	rec, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord error: %v", err)
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
	if len(rec.Lines) != 2 {
		t.Errorf("Record has %d lines, want 2", len(rec.Lines))
	}
}

func TestLazyParser_CROnlyLineEndings(t *testing.T) {
	input := "0 HEAD\r1 SOUR Test\r0 @I1@ INDI\r1 NAME John\r0 TRLR\r"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	rec, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord error: %v", err)
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
}

func TestLazyParser_MixedLineEndings(t *testing.T) {
	// Mix of CRLF, LF, and CR
	input := "0 HEAD\r\n1 SOUR Test\n0 @I1@ INDI\r1 NAME John\n0 TRLR\r\n"
	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	rec, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord error: %v", err)
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
}

// ============================================================================
// Tests for iter.Seq2 API: Records(), RecordsFrom(), AllRecords()
// ============================================================================

func TestLazyParser_Records(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	var records []*RawRecord
	for rec, err := range lp.Records() {
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
	if records[1].XRef != "@I1@" {
		t.Errorf("second record XRef = %q, want @I1@", records[1].XRef)
	}
}

func TestLazyParser_RecordsFrom(t *testing.T) {
	// "0 HEAD\n" = 7 bytes
	// "1 SOUR Test\n" = 12 bytes
	// HEAD record ends at offset 19
	// "0 @I1@ INDI\n" starts at 19
	input := "0 HEAD\n1 SOUR Test\n0 @I1@ INDI\n1 NAME John\n0 TRLR\n"

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	// Build index to get offsets
	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	entry, _ := lp.Index().Lookup("@I1@")

	// Iterate from @I1@ using iter.Seq2
	var records []*RawRecord
	for rec, err := range lp.RecordsFrom(entry.ByteOffset) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		records = append(records, rec)
	}

	if len(records) < 1 {
		t.Fatal("expected at least one record")
	}

	if records[0].XRef != "@I1@" {
		t.Errorf("first record XRef = %q, want @I1@", records[0].XRef)
	}

	// Should have @I1@ and TRLR
	if len(records) != 2 {
		t.Errorf("got %d records from offset, want 2", len(records))
	}
}

func TestLazyParser_AllRecords(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	// Read some data first to move position
	buf := make([]byte, 10)
	_, _ = rs.Read(buf)

	// AllRecords should seek to beginning
	var records []*RawRecord
	for rec, err := range lp.AllRecords() {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		records = append(records, rec)
	}

	if len(records) != 3 {
		t.Errorf("got %d records, want 3", len(records))
	}

	// Verify we got all records from the beginning
	if records[0].Type != "HEAD" {
		t.Errorf("first record type = %q, want HEAD", records[0].Type)
	}
}

func TestLazyParser_RecordsFrom_SeekError(t *testing.T) {
	es := &errorSeeker{Reader: strings.NewReader("0 HEAD\n0 TRLR\n")}
	lp := NewLazyParser(es)

	var gotError error
	for _, err := range lp.RecordsFrom(10) {
		if err != nil {
			gotError = err
			break
		}
	}

	if gotError == nil {
		t.Error("expected error from seek failure")
	}

	// Verify error contains "seek" context
	if !strings.Contains(gotError.Error(), "seek") {
		t.Errorf("error should mention seek, got: %v", gotError)
	}
}

func TestLazyParser_Records_EmptyFile(t *testing.T) {
	rs := newStringReadSeeker("")
	lp := NewLazyParser(rs)

	var count int
	for _, err := range lp.Records() {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("got %d records, want 0", count)
	}
}

func TestLazyParser_Records_EarlyTermination(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 @I2@ INDI
0 @I3@ INDI
0 @I4@ INDI
0 @I5@ INDI
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	count := 0
	for _, err := range lp.Records() {
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

func TestLazyParser_RecordsFrom_EarlyTermination(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 @I2@ INDI
0 @I3@ INDI
0 @I4@ INDI
0 @I5@ INDI
0 TRLR`

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	count := 0
	for _, err := range lp.RecordsFrom(0) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		if count >= 2 {
			break // early termination
		}
	}

	if count != 2 {
		t.Errorf("got %d records before break, want 2", count)
	}
}

func TestLazyParser_RecordsFrom_ParseError(t *testing.T) {
	// Parse error occurs during iteration
	input := "0 HEAD\n0 @I1@ INDI\nINVALID LINE\n0 TRLR\n"

	rs := newStringReadSeeker(input)
	lp := NewLazyParser(rs)

	var gotError error
	for _, err := range lp.RecordsFrom(0) {
		if err != nil {
			gotError = err
			break
		}
	}

	if gotError == nil {
		t.Error("expected parse error")
	}
}

func TestLazyParser_AllRecords_MatchesIterateAll(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John
0 @I2@ INDI
1 NAME Jane
0 TRLR`

	// Use IterateAll
	rs1 := newStringReadSeeker(input)
	lp1 := NewLazyParser(rs1)
	it, err := lp1.IterateAll()
	if err != nil {
		t.Fatalf("IterateAll error: %v", err)
	}

	var iterRecords []string
	for it.Next() {
		iterRecords = append(iterRecords, it.Record().Type)
	}
	if it.Err() != nil {
		t.Fatalf("Iterator error: %v", it.Err())
	}

	// Use AllRecords iter.Seq2
	rs2 := newStringReadSeeker(input)
	lp2 := NewLazyParser(rs2)

	var seqRecords []string
	for rec, err := range lp2.AllRecords() {
		if err != nil {
			t.Fatalf("AllRecords error: %v", err)
		}
		seqRecords = append(seqRecords, rec.Type)
	}

	// Compare
	if len(seqRecords) != len(iterRecords) {
		t.Fatalf("AllRecords got %d, IterateAll got %d", len(seqRecords), len(iterRecords))
	}

	for i := range iterRecords {
		if seqRecords[i] != iterRecords[i] {
			t.Errorf("Record %d: Type = %q, want %q", i, seqRecords[i], iterRecords[i])
		}
	}
}
