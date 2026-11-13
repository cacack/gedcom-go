package parser

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

const bom = "\xEF\xBB\xBF"

// Every iterator skips a UTF-8 BOM at the start of the input and reports
// offsets that count it, so HEAD starts at 3 and each record slices out of the
// file exactly (issue #792).
func TestIterators_SkipLeadingUTF8BOM(t *testing.T) {
	body, records := buildOffsetFixture(offsetTerm("CRLF"))
	input := bom + body
	newLP := func() *LazyParser { return NewLazyParser(newStringReadSeeker(input)) }

	check := func(t *testing.T, got []*RawRecord) {
		t.Helper()
		assertRecordSlices(t, input, got, records)
		if got[0].Type != "HEAD" || got[0].ByteOffset != 3 {
			t.Errorf("first record = %s at %d, want HEAD at 3", got[0].Type, got[0].ByteOffset)
		}
	}

	t.Run("NewRecordIterator", func(t *testing.T) {
		check(t, collectIterator(t, NewRecordIterator(strings.NewReader(input))))
	})
	t.Run("Records", func(t *testing.T) {
		check(t, collectSeq(t, Records(strings.NewReader(input))))
	})
	t.Run("NewRecordIteratorWithOffset", func(t *testing.T) {
		it := NewRecordIteratorWithOffset(strings.NewReader(input))
		var got []*RawRecord
		for it.Next() {
			got = append(got, it.Record())
		}
		if err := it.Err(); err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		check(t, got)
	})
	t.Run("RecordsWithOffset", func(t *testing.T) {
		check(t, collectSeq(t, RecordsWithOffset(strings.NewReader(input))))
	})
	t.Run("LazyParser.IterateAll", func(t *testing.T) {
		it, err := newLP().IterateAll()
		if err != nil {
			t.Fatal(err)
		}
		check(t, collectIterator(t, it))
	})
	t.Run("LazyParser.Iterate", func(t *testing.T) {
		check(t, collectIterator(t, newLP().Iterate()))
	})
	t.Run("LazyParser.AllRecords", func(t *testing.T) {
		check(t, collectSeq(t, newLP().AllRecords()))
	})
	t.Run("LazyParser.Records", func(t *testing.T) {
		check(t, collectSeq(t, newLP().Records()))
	})
}

// BuildIndex succeeds on a BOM-prefixed file and FindRecord reads every
// indexed record back exactly (issue #792).
func TestLazyParser_BuildIndex_LeadingUTF8BOM(t *testing.T) {
	const input = bom + "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n1 NAME John /Doe/\r\n0 TRLR\r\n"
	lp := NewLazyParser(newStringReadSeeker(input))
	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	head, err := lp.FindRecordByType("HEAD")
	if err != nil {
		t.Fatalf("FindRecordByType(HEAD) error = %v", err)
	}
	if head.ByteOffset != 3 {
		t.Errorf("HEAD ByteOffset = %d, want 3", head.ByteOffset)
	}
	if got := input[head.ByteOffset : head.ByteOffset+head.ByteLength]; got != "0 HEAD\r\n1 CHAR UTF-8\r\n" {
		t.Errorf("HEAD bytes = %q", got)
	}
	if head.Lines[0].Tag != "HEAD" || head.Lines[1].Tag != "CHAR" || head.Lines[1].Value != "UTF-8" {
		t.Errorf("HEAD lines = %+v, %+v", head.Lines[0], head.Lines[1])
	}

	indi, err := lp.FindRecord("@I1@")
	if err != nil {
		t.Fatalf("FindRecord(@I1@) error = %v", err)
	}
	if got := input[indi.ByteOffset : indi.ByteOffset+indi.ByteLength]; got != "0 @I1@ INDI\r\n1 NAME John /Doe/\r\n" {
		t.Errorf("@I1@ bytes = %q", got)
	}
	if indi.Lines[1].Tag != "NAME" || indi.Lines[1].Value != "John /Doe/" {
		t.Errorf("@I1@ NAME line = %+v", indi.Lines[1])
	}
}

// Without a BOM the bytes peeked for one are replayed, including inputs
// shorter than a BOM and ones that only share its first byte.
func TestSkipUTF8BOM_NoBOM(t *testing.T) {
	for _, input := range []string{"", "0", "0 HEAD\n", "\xEF\xBB0 HEAD\n"} {
		r, n := skipUTF8BOM(strings.NewReader(input))
		var b bytes.Buffer
		if _, err := b.ReadFrom(r); err != nil {
			t.Fatalf("%q: read error %v", input, err)
		}
		if n != 0 || b.String() != input {
			t.Errorf("%q: skipped %d, replayed %q", input, n, b.String())
		}
	}
}

// A read error met while peeking for a BOM is reported after the bytes read
// before it, not swallowed.
// onceFailingReader returns content, then err once, then io.EOF.
type onceFailingReader struct {
	content string
	err     error
	calls   int
}

func (r *onceFailingReader) Read(p []byte) (int, error) {
	r.calls++
	switch r.calls {
	case 1:
		return copy(p, r.content), nil
	case 2:
		return 0, r.err
	default:
		return 0, io.EOF
	}
}

func TestSkipUTF8BOM_OneShotReadErrorKept(t *testing.T) {
	readErr := errors.New("transient failure")
	r, n := skipUTF8BOM(&onceFailingReader{content: "0", err: readErr})
	var b bytes.Buffer
	_, err := b.ReadFrom(r)
	if !errors.Is(err, readErr) {
		t.Errorf("read error = %v, want %v", err, readErr)
	}
	if n != 0 || b.String() != "0" {
		t.Errorf("skipped %d, replayed %q, want 0 and %q", n, b.String(), "0")
	}

	it := NewRecordIterator(&onceFailingReader{content: "0", err: readErr})
	for it.Next() {
	}
	if !errors.Is(it.Err(), readErr) {
		t.Errorf("Err() = %v, want %v", it.Err(), readErr)
	}
}

func TestSkipUTF8BOM_ReadErrorReplayed(t *testing.T) {
	readErr := errors.New("disk on fire")
	r, n := skipUTF8BOM(&failingReader{content: "0", err: readErr})
	var b bytes.Buffer
	_, err := b.ReadFrom(r)
	if !errors.Is(err, readErr) {
		t.Errorf("read error = %v, want %v", err, readErr)
	}
	if n != 0 || b.String() != "0" {
		t.Errorf("skipped %d, replayed %q, want 0 and %q", n, b.String(), "0")
	}

	it := NewRecordIterator(&failingReader{content: "0", err: readErr})
	if it.Next() {
		t.Fatalf("Next() = true, record %+v", it.Record())
	}
	if !errors.Is(it.Err(), readErr) {
		t.Errorf("Err() = %v, want %v", it.Err(), readErr)
	}
}
