package decoder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

// manyIndividuals returns a GEDCOM 5.5.1 document holding n individuals.
func manyIndividuals(n int) string {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n")
	for i := range n {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 NAME Person%d /Surname/\n1 SEX M\n1 BIRT\n2 DATE 1 JAN 1900\n2 PLAC Somewhere, Some County, Some State\n", i, i)
	}
	sb.WriteString("0 TRLR\n")
	return sb.String()
}

// cancelOnRead cancels its context once n bytes have been read, and counts
// every byte read.
type cancelOnRead struct {
	r      io.Reader
	after  int
	read   int
	cancel context.CancelFunc
}

func (c *cancelOnRead) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	if c.read >= c.after {
		c.cancel()
	}
	return n, err
}

// TestDecodeStopsReadingOnCancel pins that a context cancelled mid-parse stops
// the read at once (#743): the decode returns context.Canceled having read
// little more than the bytes before the cancellation, not the whole input.
func TestDecodeStopsReadingOnCancel(t *testing.T) {
	input := manyIndividuals(20000) // about 2 MB
	for _, strict := range []bool{false, true} {
		t.Run(fmt.Sprintf("strict=%v", strict), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := &cancelOnRead{r: strings.NewReader(input), after: 64 << 10, cancel: cancel}

			result, err := DecodeWithDiagnostics(r, &DecodeOptions{Context: ctx, StrictMode: strict})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want context.Canceled", err)
			}
			if result != nil {
				t.Errorf("result = %+v, want nil on cancellation", result)
			}
			if limit := 256 << 10; r.read > limit {
				t.Errorf("read %d of %d bytes after cancelling at %d; want at most %d", r.read, len(input), r.after, limit)
			}
		})
	}
}

// TestPopulateEntitiesStopsOnCancel pins that entity building checks the
// context before each record: cancelled after the first check, the first
// record gets its entity and the rest are not built.
func TestPopulateEntitiesStopsOnCancel(t *testing.T) {
	doc := &gedcom.Document{Records: []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
		{XRef: "@I2@", Type: gedcom.RecordTypeIndividual},
		{XRef: "@I3@", Type: gedcom.RecordTypeIndividual},
	}}

	err := populateEntities(ctxcheck.CancelAfter(1), doc, &diagnosticCollector{discard: true})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if doc.Records[0].Entity == nil {
		t.Error("record @I1@ has no entity; the record before the cancellation should be built")
	}
	for _, r := range doc.Records[1:] {
		if r.Entity != nil {
			t.Errorf("record %s has entity %T; records after the cancellation should not be built", r.XRef, r.Entity)
		}
	}
}

// TestDecodeDeadlineOnLargeInput is the issue's probe (#743): a 50 ms deadline
// on a decode of tens of megabytes returns context.DeadlineExceeded promptly,
// where before it returned only once the whole input was parsed (seconds).
// The bound is loose so a loaded machine does not make it flaky.
func TestDecodeDeadlineOnLargeInput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a large input")
	}
	input := manyIndividuals(300000) // about 32 MB

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{Context: ctx})
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > time.Second {
		t.Errorf("decode returned after %v, want shortly after the 50ms deadline", elapsed)
	}
}
